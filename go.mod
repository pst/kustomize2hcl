module github.com/kbst/kustomize2hcl

go 1.14

replace github.com/kbst/kustomize2hcl/pkg => ./pkg

require (
	github.com/hashicorp/terraform v0.12.24
	github.com/hashicorp/terraform-config-inspect v0.0.0-20191212124732-c6ae6269b9d7
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.6.1
	github.com/valyala/fasttemplate v1.2.1
	sigs.k8s.io/kustomize/api v0.6.3
	sigs.k8s.io/kustomize/kyaml v0.9.1
)
