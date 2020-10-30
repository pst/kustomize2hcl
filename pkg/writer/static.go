package writer

var staticMain = `terraform {
  required_providers {
    kubernetes-alpha = {
      source  = "hashicorp/kubernetes-alpha"
    }
  }
}

provider "kubernetes-alpha" {
  config_path = "~/.kube/config"
}
`

var staticVariables = ``
