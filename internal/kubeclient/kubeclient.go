package kubeclient

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ClientSet interface {
	kubernetes.Interface
}

func NewUserProxyClient() (ClientSet, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// Explicitly invalidates BearerToken
	cfg.BearerToken = ""
	cfg.BearerTokenFile = ""
	hc, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, err
	}

	hc.Transport = newAuthRoundTripper(hc.Transport)
	return kubernetes.NewForConfigAndClient(cfg, hc)
}
