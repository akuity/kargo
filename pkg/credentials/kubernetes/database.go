package kubernetes

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/component"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/urls"
)

// database is an implementation of the credentials.Database interface that
// utilizes a Kubernetes controller runtime client to retrieve credentials
// stored in Kubernetes Secrets.
type database struct {
	controlPlaneClient          client.Client
	localClusterClient          client.Client
	credentialProvidersRegistry credentials.ProviderRegistry
	cfg                         DatabaseConfig
}

// DatabaseConfig represents configuration for a Kubernetes based implementation
// of the credentials.Database interface.
type DatabaseConfig struct {
	SharedResourcesNamespace string `envconfig:"SHARED_RESOURCES_NAMESPACE" default:""`
	AllowCredentialsOverHTTP bool   `envconfig:"ALLOW_CREDENTIALS_OVER_HTTP" default:"false"`
}

func DatabaseConfigFromEnv() DatabaseConfig {
	cfg := DatabaseConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// NewDatabase initializes and returns an implementation of the
// credentials.Database interface that utilizes a Kubernetes controller runtime
// client to retrieve Credentials stored in Kubernetes Secrets.
func NewDatabase(
	controlPlaneClient client.Client,
	localClusterClient client.Client,
	credentialProvidersRegistry credentials.ProviderRegistry,
	cfg DatabaseConfig,
) credentials.Database {
	return &database{
		controlPlaneClient:          controlPlaneClient,
		localClusterClient:          localClusterClient,
		credentialProvidersRegistry: credentialProvidersRegistry,
		cfg:                         cfg,
	}
}

func (k *database) Get(
	ctx context.Context,
	namespace string,
	credType credentials.Type,
	repoURL string,
) (*credentials.Credentials, error) {
	// If we are dealing with an insecure HTTP endpoint (of any type),
	// refuse to return any credentials
	if !k.cfg.AllowCredentialsOverHTTP && strings.HasPrefix(repoURL, "http://") {
		logging.LoggerFromContext(ctx).Info(
			"refused to get credentials for insecure HTTP endpoint",
			"repoURL", repoURL,
		)
		return nil, nil
	}

	clients := make([]client.Client, 1, 2)
	clients[0] = k.controlPlaneClient
	if k.localClusterClient != nil {
		clients = append(clients, k.localClusterClient)
	}

	var secret *corev1.Secret
	var err error
clientLoop:
	for _, c := range clients {
		// Check namespace for credentials
		if secret, err = k.getCredentialsSecret(
			ctx,
			c,
			namespace,
			credType,
			repoURL,
		); err != nil {
			return nil, fmt.Errorf("failed to get %s creds for %s in namespace %q: %w",
				credType.String(),
				repoURL,
				namespace,
				err,
			)
		}
		if secret != nil {
			break clientLoop
		}

		// check shared resources namespace for credentials
		if secret, err = k.getCredentialsSecret(
			ctx,
			c,
			k.cfg.SharedResourcesNamespace,
			credType,
			repoURL,
		); err != nil {
			return nil, fmt.Errorf("failed to get %s creds for %s in shared namespace %q: %w",
				credType.String(),
				repoURL,
				k.cfg.SharedResourcesNamespace,
				err,
			)
		}
		if secret != nil {
			break clientLoop
		}
	}

	var data map[string][]byte
	var metadata map[string]string

	if secret != nil {
		data = secret.Data
		metadata = secret.Annotations
	}

	req := credentials.Request{
		Project:  namespace,
		Type:     credType,
		RepoURL:  normalizeRepoURL(credType, repoURL),
		Data:     data,
		Metadata: metadata,
	}

	providerReg, err := k.credentialProvidersRegistry.Get(ctx, req)
	if err != nil {
		if !component.IsNotFoundError(err) {
			return nil, err
		}
		// If no provider was found, treat it as no credentials found.
		return nil, nil
	}

	// The registration's value is a Provider
	provider := providerReg.Value

	return provider.GetCredentials(ctx, req)
}

func (k *database) getCredentialsSecret(
	ctx context.Context,
	c client.Client,
	namespace string,
	credType credentials.Type,
	repoURL string,
) (*corev1.Secret, error) {
	// List all secrets in the namespace that are labeled with the credential
	// type.
	secrets := corev1.SecretList{}
	if err := c.List(
		ctx,
		&secrets,
		&client.ListOptions{
			Namespace: namespace,
			LabelSelector: labels.Set(map[string]string{
				kargoapi.LabelKeyCredentialType: credType.String(),
			}).AsSelector(),
		},
	); err != nil {
		return nil, err
	}

	// Sort the secrets for consistent ordering every time this function is
	// called.
	slices.SortFunc(secrets.Items, func(lhs, rhs corev1.Secret) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	normalizedRepoURL := normalizeRepoURL(credType, repoURL)

	logger := logging.LoggerFromContext(ctx)

	// Search for a matching Secret.
	for _, secret := range secrets.Items {
		if secret.Data == nil {
			continue
		}

		isRegex := string(secret.Data[credentials.FieldRepoURLIsRegex]) == "true"
		urlBytes, ok := secret.Data[credentials.FieldRepoURL]
		if !ok {
			continue
		}

		if isRegex {
			regex, err := regexp.Compile(string(urlBytes))
			if err != nil {
				logger.Error(
					err, "failed to compile regex for credential secret",
					"namespace", namespace,
					"secret", secret.Name,
				)
				continue
			}
			// We can't normalize the regex pattern so we need to check
			// the original repoURL and the normalized one.
			// For more details see: https://github.com/akuity/kargo/issues/4833
			if regex.MatchString(repoURL) || regex.MatchString(normalizedRepoURL) {
				return &secret, nil
			}
			continue
		}

		// Not a regex
		if normalizeRepoURL(credType, string(urlBytes)) == normalizedRepoURL {
			return &secret, nil
		}
	}
	return nil, nil
}

func normalizeRepoURL(credType credentials.Type, repoURL string) string {
	// Note: We formerly applied these normalizations to any URL, thinking them
	// generally safe. We no longer do this as it was discovered that an image
	// repository URL with a port number could be mistaken for an SCP-style URL of
	// the form host.xz:path/to/repo
	switch credType {
	case credentials.TypeGit:
		return urls.NormalizeGit(repoURL)
	case credentials.TypeImage:
		return urls.NormalizeImage(repoURL)
	case credentials.TypeHelm:
		return urls.NormalizeChart(repoURL)
	default:
		return repoURL
	}
}
