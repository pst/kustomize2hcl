package writer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/kbst/kustomize2hcl/pkg/reader"
	"github.com/stretchr/testify/assert"
)

func TestTemplateWriterWriteExistingDir(t *testing.T) {
	src := filepath.Join(".", "fixtures", "nginx", "base")
	r := reader.KustomizeReader{}
	rm, _ := r.Read(src)

	// have WriterWrite write into existing dir
	out, _ := ioutil.TempDir(os.TempDir(), "kustomize2hcl-unit-test-*")
	defer os.RemoveAll(out)

	w := NewTemplateWriter(rm)
	err := w.Write(out)
	assert.Equal(t, nil, err, nil)

	m, diags := tfconfig.LoadModule(out)
	de := diags.HasErrors()
	assert.Equal(t, false, de, nil)

	assert.Contains(t, m.RequiredProviders, "kubernetes-alpha", nil)

	for _, exp := range []string{"kubernetes_manifest.ClusterRoleBinding_ingress-nginx-admission_dc24307", "kubernetes_manifest.ClusterRoleBinding_ingress-nginx_3cc309c", "kubernetes_manifest.ClusterRole_ingress-nginx-admission_2ae7abe", "kubernetes_manifest.ClusterRole_ingress-nginx_170ce36", "kubernetes_manifest.ConfigMap_ingress-nginx-controller_1e866b1", "kubernetes_manifest.Deployment_ingress-nginx-controller_ad48b54", "kubernetes_manifest.Job_ingress-nginx-admission-create_9803350", "kubernetes_manifest.Job_ingress-nginx-admission-patch_4386371", "kubernetes_manifest.Namespace_ingress-nginx_92f07d0", "kubernetes_manifest.RoleBinding_ingress-nginx-admission_2cf3a82", "kubernetes_manifest.RoleBinding_ingress-nginx_4b4ea88", "kubernetes_manifest.Role_ingress-nginx-admission_48b95cf", "kubernetes_manifest.Role_ingress-nginx_7a255c6", "kubernetes_manifest.ServiceAccount_ingress-nginx-admission_2fbe545", "kubernetes_manifest.ServiceAccount_ingress-nginx_50330e0", "kubernetes_manifest.Service_ingress-nginx-controller-admission_fae9390", "kubernetes_manifest.Service_ingress-nginx-controller_1a6a48a", "kubernetes_manifest.ValidatingWebhookConfiguration_ingress-nginx-admission_a8997eb"} {
		assert.Contains(t, m.ManagedResources, exp, nil)
	}
}

func TestTemplateWriterWriteNewDir(t *testing.T) {
	src := filepath.Join(".", "fixtures", "nginx", "base")
	r := reader.KustomizeReader{}
	rm, _ := r.Read(src)

	// have WriterWrite create a subdir under tdir
	tdir, _ := ioutil.TempDir(os.TempDir(), "kustomize2hcl-unit-test-*")
	defer os.RemoveAll(tdir)
	out := filepath.Join(tdir, "created")

	w := NewTemplateWriter(rm)
	err := w.Write(out)
	assert.Equal(t, nil, err, nil)

	m, diags := tfconfig.LoadModule(out)
	de := diags.HasErrors()
	assert.Equal(t, false, de, nil)

	assert.Contains(t, m.RequiredProviders, "kubernetes-alpha", nil)

	for _, exp := range []string{"kubernetes_manifest.ClusterRoleBinding_ingress-nginx-admission_dc24307", "kubernetes_manifest.ClusterRoleBinding_ingress-nginx_3cc309c", "kubernetes_manifest.ClusterRole_ingress-nginx-admission_2ae7abe", "kubernetes_manifest.ClusterRole_ingress-nginx_170ce36", "kubernetes_manifest.ConfigMap_ingress-nginx-controller_1e866b1", "kubernetes_manifest.Deployment_ingress-nginx-controller_ad48b54", "kubernetes_manifest.Job_ingress-nginx-admission-create_9803350", "kubernetes_manifest.Job_ingress-nginx-admission-patch_4386371", "kubernetes_manifest.Namespace_ingress-nginx_92f07d0", "kubernetes_manifest.RoleBinding_ingress-nginx-admission_2cf3a82", "kubernetes_manifest.RoleBinding_ingress-nginx_4b4ea88", "kubernetes_manifest.Role_ingress-nginx-admission_48b95cf", "kubernetes_manifest.Role_ingress-nginx_7a255c6", "kubernetes_manifest.ServiceAccount_ingress-nginx-admission_2fbe545", "kubernetes_manifest.ServiceAccount_ingress-nginx_50330e0", "kubernetes_manifest.Service_ingress-nginx-controller-admission_fae9390", "kubernetes_manifest.Service_ingress-nginx-controller_1a6a48a", "kubernetes_manifest.ValidatingWebhookConfiguration_ingress-nginx-admission_a8997eb"} {
		assert.Contains(t, m.ManagedResources, exp, nil)
	}
}

func TestTemplateWriterWriteNotDir(t *testing.T) {
	tfile, _ := ioutil.TempFile(os.TempDir(), "kustomize2hcl-unit-test-*")
	defer os.RemoveAll(tfile.Name())

	w := templateWriter{}
	err := w.Write(tfile.Name())
	assert.EqualError(t, err, fmt.Sprintf("'%s', must be a directory", tfile.Name()), nil)
}

func TestTemplateWriterWriteNotEmpty(t *testing.T) {
	out, _ := ioutil.TempDir(os.TempDir(), "kustomize2hcl-unit-test-*")
	defer os.RemoveAll(out)
	ioutil.TempFile(out, "kustomize2hcl-unit-test-*")

	w := templateWriter{}
	err := w.Write(out)
	assert.EqualError(t, err, fmt.Sprintf("'%s', must be empty", out), nil)
}

func TestCleanResourceNameCharsFirstValid(t *testing.T) {
	tw := templateWriter{}

	for _, ch := range "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_" {
		out := tw.cleanResourceNameChars(string(ch))

		assert.Equal(t, string(ch), out, nil)
	}
}

func TestCleanResourceNameCharsFirstInvalid(t *testing.T) {
	tw := templateWriter{}

	out := tw.cleanResourceNameChars("0")

	assert.Equal(t, "_0", out, nil)
}

func TestCleanResourceNameNotFirstValid(t *testing.T) {
	tw := templateWriter{}

	for _, ch := range "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_0123456789" {
		in := "_" + string(ch)
		out := tw.cleanResourceNameChars(in)

		assert.Equal(t, in, out, nil)
	}
}

func TestCleanResourceNameNotFirstInvalid(t *testing.T) {
	tw := templateWriter{}

	for _, ch := range ".:,;\\'\"${}[]()%" {
		in := "_" + string(ch)
		out := tw.cleanResourceNameChars(in)

		assert.Equal(t, "_-", out, nil)
	}
}

func TestTemplateNamespaceRef(t *testing.T) {
	src := filepath.Join(".", "fixtures", "nginx", "base")
	r := reader.KustomizeReader{}
	rm, _ := r.Read(src)

	w := NewTemplateWriter(rm)

	// the second resource in the nginx fixture is service account
	// ~G_v1_ServiceAccount|ingress-nginx|ingress-nginx
	rs := rm.Resources()[1]
	k, v, err := w.namespaceRef(rs)
	assert.Equal(t, nil, err, nil)
	assert.Equal(t, "NamespaceRef_ingress-nginx", k, nil)
	assert.Equal(t, "kubernetes_manifest.Namespace_ingress-nginx_92f07d0.manifest.metadata.name", v, nil)
}

func TestTemplateNamespaceRefNoNS(t *testing.T) {
	src := filepath.Join(".", "fixtures", "nginx", "base")
	r := reader.KustomizeReader{}
	rm, _ := r.Read(src)

	w := NewTemplateWriter(rm)

	// the first resource in the nginx fixture is a namespace
	// ~G_v1_Namespace|~X|ingress-nginx
	rs := rm.Resources()[0]
	k, v, err := w.namespaceRef(rs)
	assert.Equal(t, "", k, nil)
	assert.Equal(t, "", v, nil)
	assert.Equal(t, nil, err, nil)
}

func TestTemplateNamespaceRefNSNotInResMap(t *testing.T) {
	src := filepath.Join(".", "fixtures", "nginx", "base")
	r := reader.KustomizeReader{}
	rm, _ := r.Read(src)

	w := NewTemplateWriter(rm)

	// the second resource in the nginx fixture is service account
	// ~G_v1_ServiceAccount|ingress-nginx|ingress-nginx
	rs := rm.Resources()[1]

	// overwrite ns to one that is not in the resmap
	rs.SetNamespace("default")

	k, v, err := w.namespaceRef(rs)
	assert.Equal(t, "", k, nil)
	assert.Equal(t, "", v, nil)
	assert.Equal(t, nil, err, nil)
}
