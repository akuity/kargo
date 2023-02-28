package images

import (
	"context"

	"github.com/argoproj-labs/argocd-image-updater/pkg/image"
	"github.com/argoproj-labs/argocd-image-updater/pkg/kube"
	"github.com/argoproj-labs/argocd-image-updater/pkg/registry"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

func getRegistryCredentials(
	ctx context.Context,
	kubeClient kubernetes.Interface,
	repoURL string,
	pullSecret string,
	registryEndpoint *registry.RegistryEndpoint,
) (image.Credential, error) {
	creds := image.Credential{}

	kc := kube.NewKubernetesClient(ctx, kubeClient, nil, "")

	// NB: This function call modifies the registryEndpoint
	if err := registryEndpoint.SetEndpointCredentials(kc); err != nil {
		return creds, errors.Wrapf(
			err,
			"error setting endpoint credentials for image %q",
			repoURL,
		)
	}

	if pullSecret == "" {
		return creds, nil
	}

	credSrc := image.CredentialSource{
		Type:            image.CredentialSourcePullSecret,
		SecretNamespace: "argo-cd", // TODO: Do not hard-code this
		SecretName:      pullSecret,
		SecretField:     ".dockerconfigjson",
	}
	credsPtr, err := credSrc.FetchCredentials(
		registryEndpoint.RegistryAPI,
		kc,
	)
	if err != nil {
		return creds, errors.Wrapf(
			err,
			"error getting credentials for image %q from image pull secret",
			repoURL,
		)
	}
	if credsPtr == nil {
		return creds, errors.Errorf(
			"could not find image pull secret for image %q",
			repoURL,
		)
	}

	return *credsPtr, nil
}
