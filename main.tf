resource "kubernetes_manifest" "batch_v1_job_ingress-nginx_ingress-nginx-admission-patch" {
  provider = kubernetes_alpha

  manifest = {
    "apiVersion" = "batch/v1"
    "kind"       = "Job"
    "metadata" = {
      "annotations" = {
        "app.kubernetes.io/version"      = "v0.40.2"
        "catalog.kubestack.com/heritage" = "kubestack.com/catalog/nginx"
        "catalog.kubestack.com/variant"  = "base"
        "helm.sh/hook"                   = "post-install,post-upgrade"
        "helm.sh/hook-delete-policy"     = "before-hook-creation,hook-succeeded"
      }
      "labels" = {
        "app.kubernetes.io/component"  = "ingress-controller"
        "app.kubernetes.io/instance"   = "ingress-nginx"
        "app.kubernetes.io/managed-by" = "kubestack"
        "app.kubernetes.io/name"       = "nginx"
        "app.kubernetes.io/version"    = "0.40.2"
        "helm.sh/chart"                = "ingress-nginx-3.4.1"
      }
      "name"      = "ingress-nginx-admission-patch"
      "namespace" = "ingress-nginx"
    }
    "spec" = {
      "template" = {
        "metadata" = {
          "annotations" = {
            "app.kubernetes.io/version"      = "v0.40.2"
            "catalog.kubestack.com/heritage" = "kubestack.com/catalog/nginx"
            "catalog.kubestack.com/variant"  = "base"
          }
          "labels" = {
            "app.kubernetes.io/component"  = "ingress-controller"
            "app.kubernetes.io/instance"   = "ingress-nginx"
            "app.kubernetes.io/managed-by" = "kubestack"
            "app.kubernetes.io/name"       = "nginx"
            "app.kubernetes.io/version"    = "0.40.2"
            "helm.sh/chart"                = "ingress-nginx-3.4.1"
          }
          "name" = "ingress-nginx-admission-patch"
        }
        "spec" = {
          "containers" = [
            {
              "args" = [
                "patch",
                "--webhook-name=ingress-nginx-admission",
                "--namespace=$(POD_NAMESPACE)",
                "--patch-mutating=false",
                "--secret-name=ingress-nginx-admission",
                "--patch-failure-policy=Fail",
              ]
              "env" = [
                {
                  "name" = "POD_NAMESPACE"
                  "valueFrom" = {
                    "fieldRef" = {
                      "fieldPath" = "metadata.namespace"
                    }
                  }
                },
              ]
              "image"           = "docker.io/jettech/kube-webhook-certgen:v1.3.0"
              "imagePullPolicy" = "IfNotPresent"
              "name"            = "patch"
            },
          ]
          "restartPolicy" = "OnFailure"
          "securityContext" = {
            "runAsNonRoot" = true
            "runAsUser"    = 2000
          }
          "serviceAccountName" = "ingress-nginx-admission"
        }
      }
    }
  }

  depends_on = [test]
}
