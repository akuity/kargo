package credentials

import (
	"context"
	"strings"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/logging"
)

const (
	// kargoSecretTypeLabelKey is the key for a label used to identify the type
	// of credentials stored in a Secret.
	kargoSecretTypeLabelKey = "kargo.akuity.io/secret-type" // nolint: gosec
	// repositorySecretTypeLabelValue denotes that a secret contains credentials
	// for a repository that is an exact match on the normalized URL.
	repositorySecretTypeLabelValue = "repository"
	// repoCredsSecretTypeLabelValue denotes that a secret contains credentials
	// for any repository whose URL begins with a specific prefix.
	repoCredsSecretTypeLabelValue = "repo-creds" // nolint: gosec
)

// Type is a string type used to represent a type of Credentials.
type Type string

const (
	// TypeGit represents credentials for a Git repository.
	TypeGit Type = "git"
	// TypeHelm represents credentials for a Helm chart repository.
	TypeHelm Type = "helm"
	// TypeImage represents credentials for an image repository.
	TypeImage Type = "image"
)

// Credentials generically represents any type of repository credential.
type Credentials struct {
	// Username identifies a principal, which combined with the value of the
	// Password field, can be used for access to some repository.
	Username string
	// Password, when combined with the principal identified by the Username
	// field, can be used for access to some repository.
	Password string
	// SSHPrivateKey is a private key that can be used for access to some remote
	// repository. This is primarily applicable for Git repositories.
	SSHPrivateKey string
}

// Database is an interface for a Credentials store.
type Database interface {
	Get(
		ctx context.Context,
		namespace string,
		credType Type,
		repo string,
	) (Credentials, bool, error)
}

// kubernetesDatabase is an implementation of the Database interface that
// utilizes a Kubernetes controller runtime client to retrieve credentials
// stored in Kubernetes Secrets.
type kubernetesDatabase struct {
	kargoClient client.Client
	cfg         KubernetesDatabaseConfig
}

// KubernetesDatabaseConfig represents configuration for a Kubernetes based
// implementation of the Database interface.
type KubernetesDatabaseConfig struct {
	GlobalCredentialsNamespaces []string `envconfig:"GLOBAL_CREDENTIALS_NAMESPACES" default:""`
}

func KubernetesDatabaseConfigFromEnv() KubernetesDatabaseConfig {
	cfg := KubernetesDatabaseConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// NewKubernetesDatabase initializes and returns an implementation of the
// Database interface that utilizes a Kubernetes controller runtime client to
// retrieve Credentials stored in Kubernetes Secrets.
func NewKubernetesDatabase(
	kargoClient client.Client,
	cfg KubernetesDatabaseConfig,
) Database {
	return &kubernetesDatabase{
		kargoClient: kargoClient,
		cfg:         cfg,
	}
}

func (k *kubernetesDatabase) Get(
	ctx context.Context,
	namespace string,
	credType Type,
	repoURL string,
) (Credentials, bool, error) {
	creds := Credentials{}

	// If we are dealing with an insecure HTTP endpoint (of any type),
	// refuse to return any credentials
	if strings.HasPrefix(repoURL, "http://") {
		logger := logging.LoggerFromContext(ctx).WithField("repoURL", repoURL)
		logger.Warnf("refused to get credentials for insecure HTTP endpoint")

		return creds, false, nil
	}

	var secret *corev1.Secret
	var err error

	// Check namespace for credentials
	if secret, err = getCredentialsSecret(
		ctx,
		k.kargoClient,
		namespace,
		labels.Set(map[string]string{
			kargoSecretTypeLabelKey: repositorySecretTypeLabelValue,
		}).AsSelector(),
		credType,
		repoURL,
		false, // repoURL is not a prefix
	); err != nil {
		return creds, false, err
	}

	if secret == nil {
		// Check namespace for credentials template
		if secret, err = getCredentialsSecret(
			ctx,
			k.kargoClient,
			namespace,
			labels.Set(map[string]string{
				kargoSecretTypeLabelKey: repoCredsSecretTypeLabelValue,
			}).AsSelector(),
			credType,
			repoURL,
			true, // repoURL is a prefix
		); err != nil {
			return creds, false, err
		}
	}

	if secret == nil {
		// Check global credentials namespaces for credentials
		for _, globalCredsNamespace := range k.cfg.GlobalCredentialsNamespaces {
			// Check shared creds namespace for credentials
			if secret, err = getCredentialsSecret(
				ctx,
				k.kargoClient,
				globalCredsNamespace,
				labels.Set(map[string]string{
					kargoSecretTypeLabelKey: repositorySecretTypeLabelValue,
				}).AsSelector(),
				credType,
				repoURL,
				false, // repoURL is not a prefix
			); err != nil {
				return creds, false, err
			}
			if secret != nil {
				break
			}

			// Check shared creds namespace for credentials template
			if secret, err = getCredentialsSecret(
				ctx,
				k.kargoClient,
				globalCredsNamespace,
				labels.Set(map[string]string{
					kargoSecretTypeLabelKey: repoCredsSecretTypeLabelValue,
				}).AsSelector(),
				credType,
				repoURL,
				true, // repoURL is a prefix
			); err != nil {
				return creds, false, err
			}
			if secret != nil {
				break
			}
		}
	}

	if secret == nil {
		return creds, false, nil
	}

	return secretToCreds(secret), true, nil
}

func getCredentialsSecret(
	ctx context.Context,
	kubeClient client.Client,
	namespace string,
	labelSelector labels.Selector,
	credType Type,
	repoURL string,
	acceptPrefixMatch bool,
) (*corev1.Secret, error) {
	repoURL = normalizeChartRepositoryURL( // This should be safe even on non-chart repo URLs
		git.NormalizeGitURL(repoURL), // This should be safe even on non-Git URLs
	)

	secrets := corev1.SecretList{}
	if err := kubeClient.List(
		ctx,
		&secrets,
		&client.ListOptions{
			Namespace:     namespace,
			LabelSelector: labelSelector,
		},
	); err != nil {
		return nil, err
	}
	// Scan for the credentials we're looking for
	for _, secret := range secrets.Items {
		if secret.Data == nil {
			continue
		}
		if typeBytes, ok := secret.Data["type"]; !ok || Type(typeBytes) != credType {
			continue
		}
		urlBytes, ok := secret.Data["url"]
		if !ok {
			continue
		}
		url := normalizeChartRepositoryURL( // This should be safe even on non-chart repo URLs
			git.NormalizeGitURL( // This should be safe even on non-Git URLs
				string(urlBytes),
			),
		)
		if acceptPrefixMatch && strings.HasPrefix(repoURL, url) {
			return &secret, nil
		}
		if !acceptPrefixMatch && url == repoURL {
			return &secret, nil
		}
	}
	return nil, nil
}

// NormalizeURL normalizes a chart repository URL for purposes of comparison.
// Crucially, this function removes the oci:// prefix from the URL if there is
// one.
func normalizeChartRepositoryURL(repo string) string {
	return strings.TrimPrefix(
		strings.ToLower(
			strings.TrimSpace(repo),
		),
		"oci://",
	)
}

func secretToCreds(secret *corev1.Secret) Credentials {
	return Credentials{
		Username:      string(secret.Data["username"]),
		Password:      string(secret.Data["password"]),
		SSHPrivateKey: string(secret.Data["sshPrivateKey"]),
	}
}
