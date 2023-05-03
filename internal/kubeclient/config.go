package kubeclient

import (
	"os"

	"k8s.io/client-go/tools/clientcmd"
)

var overrides = clientcmd.ConfigOverrides{}

var (
	explicitPath string
)

func NewClientConfig() clientcmd.ClientConfig {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	rules.ExplicitPath = explicitPath
	return clientcmd.NewInteractiveDeferredLoadingClientConfig(rules, &overrides, os.Stdin)
}
