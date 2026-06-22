package server

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// normalizeListResourceVersion returns a list ResourceVersion safe to seed a
// follow-up Watch with: the value as-is when it is a real version, or "" when
// it is the cached client's "0" sentinel.
//
// The list+watch seed endpoints read through listForWatchSeed's uncached
// reader, which always returns a real list-level resource version. The "0"
// sentinel only appears on the degraded path where no direct reader is wired
// (tests, or no rest.Config) and the cached client is used; there we return ""
// so the client opens an unseeded watch — the pre-change behavior.
func normalizeListResourceVersion(rv string) string {
	if rv == "0" {
		return ""
	}
	return rv
}

// buildWatchListOptions returns namespace-scoped list options for watch calls,
// including ResourceVersion when the caller has one.
func buildWatchListOptions(namespace string, resourceVersion string) []client.ListOption {
	watchOpts := []client.ListOption{client.InNamespace(namespace)}
	if resourceVersion != "" {
		watchOpts = append(watchOpts, &client.ListOptions{
			Raw: &metav1.ListOptions{ResourceVersion: resourceVersion},
		})
	}
	return watchOpts
}
