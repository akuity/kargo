package kubeclient

import (
	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/pkg/errors"
	authzv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubescheme "k8s.io/client-go/kubernetes/scheme"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

var kargoSchemes = runtime.SchemeBuilder{
	corev1.AddToScheme,
	kargoapi.AddToScheme,
}

func newScheme(extraSchemes ...func(*runtime.Scheme) error) (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	for _, addToScheme := range extraSchemes {
		if err := addToScheme(scheme); err != nil {
			return nil, errors.Wrap(err, "add api to scheme")
		}
	}
	return scheme, nil
}
func NewAPIScheme() (*runtime.Scheme, error) {
	return newScheme(kargoSchemes...)
}

func NewAppManagerScheme() (*runtime.Scheme, error) {
	return newScheme(
		corev1.AddToScheme,
		argocd.AddToScheme,
	)
}

func NewCLIScheme() (*runtime.Scheme, error) {
	return newScheme(kargoSchemes...)
}

func NewKargoScheme() (*runtime.Scheme, error) {
	return newScheme(kargoSchemes...)
}

func NewKargoManagerScheme() (*runtime.Scheme, error) {
	return newScheme(kargoSchemes...)
}

func NewKubernetesScheme() (*runtime.Scheme, error) {
	return newScheme(kubescheme.AddToScheme, kargoapi.AddToScheme)
}

func NewGarbageCollectorScheme() (*runtime.Scheme, error) {
	return newScheme(kargoSchemes...)
}

func NewWebhooksServerScheme() (*runtime.Scheme, error) {
	return newScheme(append(kargoSchemes, authzv1.AddToScheme)...)
}
