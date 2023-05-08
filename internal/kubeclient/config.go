package kubeclient

import (
	"os"

	"k8s.io/client-go/tools/clientcmd"
)

var overrides = clientcmd.ConfigOverrides{}

func NewClientConfig() clientcmd.ClientConfig {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	return clientcmd.NewInteractiveDeferredLoadingClientConfig(rules, &overrides, os.Stdin)
}
