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

func effectiveResourceVersionFromObjects[T metav1.Object](rv string, items []T) string {
	itemVersions := make([]string, len(items))
	for i, item := range items {
		itemVersions[i] = item.GetResourceVersion()
	}
	return effectiveResourceVersion(rv, itemVersions)
}

func buildWatchListOptions(
	namespace string,
	resourceVersion string,
	opts ...client.ListOption,
) []client.ListOption {
	watchOpts := []client.ListOption{client.InNamespace(namespace)}
	watchOpts = append(watchOpts, opts...)
	if resourceVersion != "" {
		watchOpts = append(watchOpts, &client.ListOptions{
			Raw: &metav1.ListOptions{ResourceVersion: resourceVersion},
		})
	}
	return watchOpts
}
