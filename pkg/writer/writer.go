package writer

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/repl"
	"github.com/valyala/fasttemplate"
	"sigs.k8s.io/kustomize/api/filters/namespace"
	"sigs.k8s.io/kustomize/api/resid"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/kyaml/filtersutil"
)

var _ Writer = &TemplateWriter{}

type Writer interface {
	Write(path string) (err error)
}

type terraformReferences struct {
	Namespaces map[string]string
	Variables  map[string]string
}

type TemplateWriter struct {
	ResMap              resmap.ResMap
	TerraformReferences terraformReferences
	ProviderAlias       string
	ProviderResource    string
}

func NewTemplateWriter(r resmap.ResMap) TemplateWriter {
	return TemplateWriter{
		ResMap: r,
		TerraformReferences: terraformReferences{
			Namespaces: make(map[string]string),
			Variables:  make(map[string]string),
		},
		ProviderAlias:    "kubernetes-alpha",
		ProviderResource: "kubernetes_manifest",
	}
}

func (tw *TemplateWriter) Write(path string) (err error) {
	tfr, err := tw.resmapToHCL()
	if err != nil {
		return err
	}

	err = os.Mkdir(path, os.ModePerm)
	if err != nil {
		return err
	}

	for n, c := range tfr {
		fp := filepath.Join(path, fmt.Sprintf("%s.tf", n))
		cb := []byte(c)
		err := ioutil.WriteFile(fp, cb, 0644)
		if err != nil {
			return err
		}
	}

	return
}

func (tw *TemplateWriter) resmapToHCL() (tfr map[string]string, err error) {
	tfr = make(map[string]string)
	tfr["_main"] = staticMain
	tfr["_variables"] = staticVariables

	for _, r := range tw.ResMap.Resources() {
		n := tw.resourceName(r)

		refs := map[string]interface{}{}

		nsref, err := tw.namespaceRef(r)
		if err == nil {
			refs["namespace"] = nsref
		}

		hclT, err := tw.toHCL(n, r)
		if err != nil {
			return tfr, err
		}

		t := fasttemplate.New(hclT, "\"%%", "%%\"")
		hcl := t.ExecuteString(refs)

		tfr[n] = hcl
	}

	return tfr, nil
}

// TF resource names need to be unique and stable across releases.
//
// But Kubernetes kinds are only unqiue per apiGroup and apiVersion.
// Kustomize resid's include group, version, kind, namespace and name,
// but they are not valid TF resource names and are hard to read.
//
// This function provides friendly but stable TF resource names
// by concatenating kind, name and a hash of Kustomize's resid
func (tw *TemplateWriter) resourceName(r *resource.Resource) string {
	kind := strings.ToLower(strings.ReplaceAll(r.GetKind(), ".", "-"))
	name := strings.ToLower(strings.ReplaceAll(r.GetOriginalName(), ".", "-"))

	h := sha512.New()
	id := r.OrgId().String()
	h.Write([]byte(id))
	hash := hex.EncodeToString(h.Sum(nil))[0:7]

	return fmt.Sprintf("%s_%s_%s", kind, name, hash)
}

func (tw *TemplateWriter) toHCL(n string, r *resource.Resource) (hcl string, err error) {
	// we marshal and unmarshal to JSON
	// to get an interface that works
	// with FormatResult
	j, err := r.MarshalJSON()
	if err != nil {
		return hcl, err
	}

	var k interface{}
	json.Unmarshal(j, &k)

	m, err := repl.FormatResult(k)
	if err != nil {
		return hcl, err
	}

	hcl = fmt.Sprintf("resource %q %q {\n", tw.ProviderResource, n)
	hcl += fmt.Sprintf("  provider = %v\n\n", tw.ProviderAlias)
	hcl += fmt.Sprintf("  manifest = %v\n\n", strings.ReplaceAll(m, "\n", "\n  "))
	hcl += fmt.Sprintf("}\n\n")

	return hcl, nil
}

func (tw *TemplateWriter) namespaceRef(r *resource.Resource) (string, error) {
	ns := r.GetNamespace()
	if ns == "" {
		return "", fmt.Errorf("'%s', has no namespace", r.CurId())
	}

	err := filtersutil.ApplyToJSON(namespace.Filter{
		Namespace: "%%namespace%%",
		FsSlice:   nil,
	}, r)
	if err != nil {
		return "", err
	}
	matches := tw.ResMap.GetMatchingResourcesByCurrentId(r.CurId().Equals)
	if len(matches) != 1 {
		return "", fmt.Errorf(
			"namespace transformation produces ID conflict: %+v", matches)
	}

	gvk := resid.Gvk{Group: "", Version: "v1", Kind: "Namespace"}
	rid := resid.NewResId(gvk, ns)
	nsr, err := tw.ResMap.GetById(rid)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"%s.%s",
		tw.ProviderResource,
		tw.resourceName(nsr),
	), nil
}
