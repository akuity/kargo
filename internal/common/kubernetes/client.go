package kubernetes

import (
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/akuityio/k8sta/internal/common/os"
)

// Client returns an implementation of kubernetes.Interface.
func Client() (kubernetes.Interface, error) {
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
	return kubernetes.NewForConfig(cfg)
}
