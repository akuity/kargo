package argocd

import "os"

func Namespace() string {
	value := os.Getenv("ARGOCD_NAMESPACE")
	if value == "" {
		return "argocd"
	}
	return value
}
