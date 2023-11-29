package credentials

import "sigs.k8s.io/controller-runtime/pkg/client"

func WithArgoCDNamespace(namespace string) KubernetesDatabaseOption {
	return func(config *kubernetesDatabaseConfig) {
		config.argoCDNamespace = namespace
	}
}

func WithGlobalCredentialsNamespaces(namespaces []string) KubernetesDatabaseOption {
	return func(config *kubernetesDatabaseConfig) {
		config.globalCredentialsNamespaces = namespaces
	}
}

func WithArgoClient(argoClient client.Client) KubernetesDatabaseOption {
	return func(config *kubernetesDatabaseConfig) {
		config.argoClient = argoClient
	}
}
