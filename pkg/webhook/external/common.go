package external

import (
	"os"

	v1 "k8s.io/api/core/v1"
)

func getSecretName(sharedSecretRef string, secretRef v1.LocalObjectReference) string {
	if sharedSecretRef != "" {
		return sharedSecretRef
	}
	return secretRef.Name
}

func getSecretNamespace(project, sharedSecretRef string) string {
	if sharedSecretRef != "" {
		return os.Getenv("SHARED_RESOURCES_NAMESPACE")
	}
	if project == "" {
		return os.Getenv("SYSTEM_RESOURCES_NAMESPACE")
	}
	return project
}
