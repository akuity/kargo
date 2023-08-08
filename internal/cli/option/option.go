package option

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
)

type Option struct {
	ServerURL      string
	UseLocalServer bool

	IOStreams  *genericclioptions.IOStreams
	PrintFlags *genericclioptions.PrintFlags
}

func NewScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, errors.Wrap(err, "add core v1 scheme")
	}
	if err := kubev1alpha1.AddToScheme(scheme); err != nil {
		return nil, errors.Wrap(err, "add kargo v1alpha1 scheme")
	}
	return scheme, nil
}
