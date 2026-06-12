package server

import (
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// effectiveResourceVersion returns rv if it is a real, non-default Kubernetes
// resource version (non-empty and non-"0"). Otherwise it computes the maximum
// numeric resource version from the provided item versions as a best-effort
// starting point for a subsequent Watch.
//
// Background: the controller-runtime cached client returns "0" for the
// list-level ResourceVersion, which causes a Kubernetes Watch to replay all
// existing objects as ADDED events. Kubernetes resource versions are officially
// opaque, so this fallback is deliberately conservative: it is only used when
// item versions parse as positive integers, which matches the resource versions
// returned by the API server today. Non-numeric item versions are ignored. If
// no usable item version exists, we return an empty string and preserve the
// previous watch behavior.
//
// In production the max-item fallback is effectively unreachable: the list+watch
// seed endpoints read through listForWatchSeed's uncached reader, which always
// returns a real list-level resource version, so the early return below wins.
// The fallback only runs on the degraded path where no direct reader is wired
// (tests, or no rest.Config) and the cached client reports "0"/"". There it is a
// best effort: the max item version may be older than the apiserver's watch
// window, in which case the follow-up watch simply restarts with a fresh list,
// i.e. the pre-change behavior.
func effectiveResourceVersion(rv string, itemVersions []string) string {
	if rv != "" && rv != "0" {
		return rv
	}
	var maxRV int64
	for _, v := range itemVersions {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n <= 0 {
			continue
		}
		if n > maxRV {
			maxRV = n
		}
	}
	if maxRV == 0 {
		return ""
	}
	return strconv.FormatInt(maxRV, 10)
}

// effectiveResourceVersionForList returns an effective list resourceVersion for
// a typed list: the list-level resourceVersion when usable, otherwise the
// maximum item resourceVersion (see effectiveResourceVersion). itemRV extracts
// the resourceVersion from each (value-typed) list item, which a plain generic
// constraint cannot do because metav1.Object's methods have pointer receivers.
func effectiveResourceVersionForList[T any](rv string, items []T, itemRV func(T) string) string {
	itemVersions := make([]string, len(items))
	for i := range items {
		itemVersions[i] = itemRV(items[i])
	}
	return effectiveResourceVersion(rv, itemVersions)
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
