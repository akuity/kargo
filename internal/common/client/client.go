package client

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuityio/k8sta/internal/common/os"
)

// New returns an implementation of the controller runtime client.
func New(scheme *runtime.Scheme) (client.Client, error) {
	masterURL := os.GetEnvVar("KUBE_MASTER", "")
	kubeConfigPath := os.GetEnvVar("KUBE_CONFIG", "")

	var cfg *rest.Config
	var err error
	if masterURL == "" && kubeConfigPath == "" {
		cfg, err = rest.InClusterConfig()
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags(masterURL, kubeConfigPath)
	}
	if err != nil {
		return nil, errors.Wrap(
			err,
			"error getting Kubernetes configuration",
		)
	}
	return client.New(
		cfg,
		client.Options{
			Scheme: scheme,
		},
	)
}
