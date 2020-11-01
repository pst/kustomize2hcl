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
	"unicode"

	"github.com/hashicorp/terraform/repl"
	"github.com/valyala/fasttemplate"
	"sigs.k8s.io/kustomize/api/filters/namespace"
	"sigs.k8s.io/kustomize/api/resid"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/kyaml/filtersutil"
)

const tplBegin string = "###"
const tplClose string = "###"

var _ Writer = &templateWriter{}

type Writer interface {
	Write(path string) (err error)
}

type terraformReferences struct {
	Namespaces map[string]string
	Variables  map[string]string
}

type templateWriter struct {
	ResMap              resmap.ResMap
	TerraformReferences terraformReferences
	ProviderAlias       string
	ProviderResource    string
	TemplateValues      map[string]interface{}
}

func NewTemplateWriter(r resmap.ResMap) templateWriter {
	return templateWriter{
		ResMap: r,
		TerraformReferences: terraformReferences{
			Namespaces: make(map[string]string),
			Variables:  make(map[string]string),
		},
		ProviderAlias:    "kubernetes-alpha",
		ProviderResource: "kubernetes_manifest",
		TemplateValues:   make(map[string]interface{}),
	}
}

func (tw *templateWriter) Write(path string) (err error) {
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		// path must be a directory
		return fmt.Errorf("'%s', must be a directory", path)
	} else if err != nil {
		// if path does not exist, try creating it
		err = os.Mkdir(path, os.ModePerm)
		if err != nil {
			return err
		}
	}

	if files, err := ioutil.ReadDir(path); err == nil && len(files) != 0 {
		// directory must be empty
		return fmt.Errorf("'%s', must be empty", path)
	} else if err != nil {
		return err
	}

	tfr, err := tw.resmapToHCL()
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

func (tw *templateWriter) resmapToHCL() (tfr map[string]string, err error) {
	tfr = make(map[string]string)
	tfr["_main"] = staticMain
	tfr["_variables"] = staticVariables

	for _, r := range tw.ResMap.Resources() {
		n := tw.resourceName(r)

		nsRefs, err := tw.namespaceRef(r)
		if err != nil {
			return tfr, err
		}
		for _, r := range nsRefs {
			if r.k != "" && r.v != "" {
				tw.TemplateValues[r.k] = r.v
			}
		}

		mldRefs, err := tw.multiLineDataRef(r)
		if err != nil {
			return tfr, err
		}
		for _, r := range mldRefs {
			if r.k != "" && r.v != "" {
				tw.TemplateValues[r.k] = r.v
			}
		}

		hcl, err := tw.toHCL(n, r)
		if err != nil {
			return tfr, err
		}

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
func (tw *templateWriter) resourceName(r *resource.Resource) string {
	kind := r.GetKind()
	name := r.GetOriginalName()

	h := sha512.New()
	id := r.OrgId().String()
	h.Write([]byte(id))
	hash := hex.EncodeToString(h.Sum(nil))[0:7]

	cs := fmt.Sprintf("%s_%s_%s", kind, name, hash)

	return tw.cleanResourceNameChars(cs)
}

// Ensure strings taken from Kubernetes YAML meet Terraforms name requirements
//
// A name must start with a letter or underscore and may contain only letters,
// digits, underscores, and dashes.
func (tw *templateWriter) cleanResourceNameChars(in string) string {
	var chars []rune
	for p, ch := range in {
		if 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch >= 0x80 && unicode.IsLetter(ch) {
			chars = append(chars, ch)
			continue
		}

		// if this is the first ch and it didn't pass the test above
		// we prefix an underscore
		if p == 0 {
			chars = append([]rune{'_'}, chars...)
		}

		// any ch not first ch can also be a digit
		if '0' <= ch && ch <= '9' || ch >= 0x80 && unicode.IsDigit(ch) {
			chars = append(chars, ch)
			continue
		}

		// if the ch did not meet any of the above,
		//replace it with a dash
		chars = append(chars, '-')
	}

	return string(chars)
}

func (tw *templateWriter) toHCL(n string, r *resource.Resource) (hcl string, err error) {
	// we marshal and unmarshal to JSON
	// to get an interface that works
	// with FormatResult
	j, err := r.MarshalJSON()
	if err != nil {
		return hcl, err
	}

	var k interface{}
	err = json.Unmarshal(j, &k)
	if err != nil {
		return hcl, err
	}

	m, err := repl.FormatResult(k)
	if err != nil {
		return hcl, err
	}

	t := fasttemplate.New(
		m,
		fmt.Sprintf("\"%s", tplBegin),
		fmt.Sprintf("%s\"", tplClose),
	)
	tm := t.ExecuteString(tw.TemplateValues)

	hcl = fmt.Sprintf("resource %q %q {\n", tw.ProviderResource, n)
	hcl += fmt.Sprintf("  provider = %v\n\n", tw.ProviderAlias)
	hcl += fmt.Sprintf("  manifest = %v\n\n", tm)
	hcl += fmt.Sprintf("}\n\n")

	return hcl, nil
}

type tplRef struct {
	k string
	v string
}

func (tw *templateWriter) namespaceRef(r *resource.Resource) (refs []tplRef, err error) {
	ns := r.GetNamespace()
	if ns == "" {
		// the resource has no namespace
		return refs, nil
	}

	gvk := resid.Gvk{Group: "", Version: "v1", Kind: "Namespace"}
	rid := resid.NewResId(gvk, ns)
	nsr, err := tw.ResMap.GetById(rid)
	if err != nil {
		// the resource's namespace is not in the resmap
		return refs, nil
	}

	tplRef := tplRef{}
	tplRef.k = fmt.Sprintf("NamespaceRef_%s", ns)
	tplRef.v = fmt.Sprintf(
		"%s.%s.manifest.metadata.name",
		tw.ProviderResource,
		tw.resourceName(nsr),
	)
	refs = append(refs, tplRef)

	err = filtersutil.ApplyToJSON(namespace.Filter{
		Namespace: fmt.Sprintf("%s%s%s", tplBegin, tplRef.k, tplClose),
		FsSlice:   nil,
	}, r)
	if err != nil {
		return refs, err
	}
	matches := tw.ResMap.GetMatchingResourcesByCurrentId(r.CurId().Equals)
	if len(matches) != 1 {
		return refs, fmt.Errorf(
			"namespace transformation produces ID conflict: %+v", matches)
	}

	return refs, nil
}

func (tw *templateWriter) multiLineDataRef(r *resource.Resource) (refs []tplRef, err error) {
	d, err := r.GetFieldValue("data")
	if err != nil {
		if err.Error() == "no field named 'data'" {
			// the resource does not have data
			return refs, nil
		}
		return refs, err
	}

	if d == nil {
		return refs, nil
	}

	data := d.(map[string]interface{})
	for k, v := range data {
		if strings.Index(v.(string), "\n") >= 0 || strings.Index(v.(string), "\r") >= 0 {
			tplRef := tplRef{}

			h := sha512.New()
			h.Write([]byte(v.(string)))

			tplRef.k = fmt.Sprintf("MultiLineDataRef_%s", hex.EncodeToString(h.Sum(nil)))

			// escape terraform ${ %{ interpolation
			s := strings.ReplaceAll(v.(string), "${", "$${")
			s = strings.ReplaceAll(s, "%{", "%%{")

			tplRef.v = fmt.Sprintf("<<EOF\n%sEOF", s)

			// replace value in resource with template placeholder
			data[k] = fmt.Sprintf("%s%s%s", tplBegin, tplRef.k, tplClose)

			refs = append(refs, tplRef)
		}
	}

	return refs, nil
}
