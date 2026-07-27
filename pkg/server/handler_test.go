package server

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func mustNewScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		panic(fmt.Errorf("add core v1 to scheme: %w", err))
	}
	if err := kargoapi.AddToScheme(scheme); err != nil {
		panic(fmt.Errorf("add kargo v1alpha1 scheme: %w", err))
	}
	return scheme
}
