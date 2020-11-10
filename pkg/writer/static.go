package writer

var staticMain = `terraform {
  required_providers {
    kubernetes-alpha = {
      source = "hashicorp/kubernetes-alpha"
    }
  }
}

provider "kubernetes-alpha" {
  config_path = "/home/runner/.kube/config"

  # set server side planning to false
  # TODO: remove when this is clear upstream
  server_side_planning = false
}
`

var staticVariables = ``
