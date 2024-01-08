package credentials

import "sigs.k8s.io/controller-runtime/pkg/client"

func WithArgoCDNamespace(namespace string) KubernetesDatabaseOption {
	return func(config *kubernetesDatabaseConfig) {
		config.ArgoCDNamespace = namespace
	}
}

func WithGlobalCredentialsNamespaces(namespaces []string) KubernetesDatabaseOption {
	return func(config *kubernetesDatabaseConfig) {
		config.GlobalCredentialsNamespaces = namespaces
	}
}

func WithArgoClient(argoClient client.Client) KubernetesDatabaseOption {
	return func(config *kubernetesDatabaseConfig) {
		config.argoClient = argoClient
	}
}
