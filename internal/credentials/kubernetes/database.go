package kubernetes

import (
	"context"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/credentials/kubernetes/basic"
	"github.com/akuity/kargo/internal/credentials/kubernetes/ecr"
	"github.com/akuity/kargo/internal/credentials/kubernetes/gar"
	"github.com/akuity/kargo/internal/credentials/kubernetes/github"
	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/logging"
)

// database is an implementation of the credentials.Database interface that
// utilizes a Kubernetes controller runtime client to retrieve credentials
// stored in Kubernetes Secrets.
type database struct {
	kargoClient       client.Client
	credentialHelpers []credentials.Helper
	cfg               DatabaseConfig
}

// DatabaseConfig represents configuration for a Kubernetes based implementation
// of the credentials.Database interface.
type DatabaseConfig struct {
	GlobalCredentialsNamespaces []string `envconfig:"GLOBAL_CREDENTIALS_NAMESPACES" default:""`
}

func DatabaseConfigFromEnv() DatabaseConfig {
	cfg := DatabaseConfig{}
	envconfig.MustProcess("", &cfg)
	slices.Sort(cfg.GlobalCredentialsNamespaces)
	return cfg
}

// NewDatabase initializes and returns an implementation of the
// credentials.Database interface that utilizes a Kubernetes controller runtime
// client to retrieve Credentials stored in Kubernetes Secrets.
func NewDatabase(
	ctx context.Context,
	kargoClient client.Client,
	cfg DatabaseConfig,
) credentials.Database {
	credentialHelpers := []credentials.Helper{
		basic.SecretToCreds,
		ecr.NewAccessKeyCredentialHelper(),
		ecr.NewPodIdentityCredentialHelper(ctx),
		gar.NewServiceAccountKeyCredentialHelper(),
		gar.NewWorkloadIdentityFederationCredentialHelper(ctx),
		github.NewAppCredentialHelper(),
	}
	finalCredentialHelpers := make([]credentials.Helper, 0, len(credentialHelpers))
	for _, helper := range credentialHelpers {
		if helper != nil {
			finalCredentialHelpers = append(finalCredentialHelpers, helper)
		}
	}
	return &database{
		kargoClient:       kargoClient,
		credentialHelpers: finalCredentialHelpers,
		cfg:               cfg,
	}
}

func (k *database) Get(
	ctx context.Context,
	namespace string,
	credType credentials.Type,
	repoURL string,
) (credentials.Credentials, bool, error) {
	// If we are dealing with an insecure HTTP endpoint (of any type),
	// refuse to return any credentials
	if strings.HasPrefix(repoURL, "http://") {
		logging.LoggerFromContext(ctx).Info(
			"refused to get credentials for insecure HTTP endpoint",
			"repoURL", repoURL,
		)
		return credentials.Credentials{}, false, nil
	}

	var secret *corev1.Secret
	var err error

	// Check namespace for credentials
	if secret, err = k.getCredentialsSecret(
		ctx,
		namespace,
		credType,
		repoURL,
	); err != nil {
		return credentials.Credentials{}, false, err
	}

	if secret == nil {
		// Check global credentials namespaces for credentials
		for _, globalCredsNamespace := range k.cfg.GlobalCredentialsNamespaces {
			if secret, err = k.getCredentialsSecret(
				ctx,
				globalCredsNamespace,
				credType,
				repoURL,
			); err != nil {
				return credentials.Credentials{}, false, err
			}
			if secret != nil {
				break
			}
		}
	}

	for _, helper := range k.credentialHelpers {
		creds, err := helper(ctx, namespace, credType, repoURL, secret)
		if err != nil {
			return credentials.Credentials{}, false, err
		}
		if creds != nil {
			return *creds, true, nil
		}
	}

	return credentials.Credentials{}, false, nil
}

func (k *database) getCredentialsSecret(
	ctx context.Context,
	namespace string,
	credType credentials.Type,
	repoURL string,
) (*corev1.Secret, error) {
	// List all secrets in the namespace that are labeled with the credential
	// type.
	secrets := corev1.SecretList{}
	if err := k.kargoClient.List(
		ctx,
		&secrets,
		&client.ListOptions{
			Namespace: namespace,
			LabelSelector: labels.Set(map[string]string{
				kargoapi.CredentialTypeLabelKey: credType.String(),
			}).AsSelector(),
		},
	); err != nil {
		return nil, err
	}

	// Sort the secrets for consistent ordering every time this function is
	// called.
	slices.SortFunc(secrets.Items, func(a, b corev1.Secret) int {
		if a.Name < b.Name {
			return -1
		} else if a.Name > b.Name {
			return 1
		} else {
			return 0
		}
	})

	// Normalize the repository URL. These normalizations should be safe even
	// if not applicable to the URL type.
	repoURL = helm.NormalizeChartRepositoryURL(git.NormalizeURL(repoURL))

	logger := logging.LoggerFromContext(ctx)

	// Search for a matching Secret.
	var matchingSecret *corev1.Secret
	for _, secret := range secrets.Items {
		secret := secret

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
			if regex.MatchString(repoURL) {
				matchingSecret = &secret
				break
			}
		} else if repoURL == helm.NormalizeChartRepositoryURL(git.NormalizeURL(string(urlBytes))) {
			return &secret, nil
		}
	}
	return matchingSecret, nil
}
