package controller

import (
	appclientset "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/akuityio/k8sta/internal/common/os"
)

func argocdClient() (*appclientset.Clientset, error) {
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
		return nil, errors.Wrap(err, "error getting kubernetes configuration")
	}
	return appclientset.NewForConfig(cfg)
}
