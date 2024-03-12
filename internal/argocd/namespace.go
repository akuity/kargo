package argocd

import "github.com/akuity/kargo/internal/os"

var namespace = os.GetEnv("ARGOCD_NAMESPACE", "argocd")

func Namespace() string {
	return namespace
}
