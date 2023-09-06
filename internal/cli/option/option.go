package option

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

type Option struct {
	InsecureTLS        bool
	LocalServerAddress string
	UseLocalServer     bool

	Project Optional[string]

	IOStreams  *genericclioptions.IOStreams
	PrintFlags *genericclioptions.PrintFlags
}

func NewOption() *Option {
	return &Option{
		Project: OptionalString(),
	}
}

func NewScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, errors.Wrap(err, "add core v1 scheme")
	}
	if err := kargoapi.AddToScheme(scheme); err != nil {
		return nil, errors.Wrap(err, "add kargo v1alpha1 scheme")
	}
	return scheme, nil
}
