package kubernetes

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

var scheme = runtime.NewScheme()

func init() {
	_ = corev1.AddToScheme(scheme)
	_ = kargoapi.AddToScheme(scheme)
}

// GetScheme returns a runtime.Scheme with the types of the Kargo API.
func GetScheme() *runtime.Scheme {
	return scheme
}
