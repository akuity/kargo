package builtin

import (
	"sync/atomic"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/controller/health"
)

var initialized atomic.Uint32

// Initialize registers all built-in health.Checkers with the health package's
// internal Checker registry.
func Initialize(argocdClient client.Client) {
	if !initialized.CompareAndSwap(0, 1) {
		panic("built-in health checkers already initialized")
	}
	health.RegisterChecker(newArgocdChecker(argocdClient))
}
