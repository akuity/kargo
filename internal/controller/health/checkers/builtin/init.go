package builtin

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/controller/health"
)

// Initialize registers all built-in health.Checkers with the health package's
// internal Checker registry.
func Initialize(argocdClient client.Client) {
	health.RegisterChecker(newArgocdChecker(argocdClient))
}
