package credentials

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/logging"
)

const (
	FieldRepoURL        = "repoURL"
	FieldRepoURLIsRegex = "repoURLIsRegex"
	FieldUsername       = "username"
	FieldPassword       = "password"
)

// Type is a string type used to represent a type of Credentials.
type Type string

// String returns the string representation of a Type.
func (t Type) String() string {
	return string(t)
}

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
	sort.StringSlice(cfg.GlobalCredentialsNamespaces).Sort()
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
	if secret, err = k.getCredentialsSecret(
		ctx,
		namespace,
		credType,
		repoURL,
	); err != nil {
		return creds, false, err
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

func (k *kubernetesDatabase) getCredentialsSecret(
	ctx context.Context,
	namespace string,
	credType Type,
	repoURL string,
) (*corev1.Secret, error) {
	repoURL = helm.NormalizeChartRepositoryURL( // This should be safe even on non-chart repo URLs
		git.NormalizeGitURL(repoURL), // This should be safe even on non-Git URLs
	)

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

	// Sort the secrets so they're considered in the same order every time this
	// function is called.
	sort.Slice(secrets.Items, func(i, j int) bool {
		return secrets.Items[i].Name < secrets.Items[j].Name
	})

	// Scan for an exact match
	for _, secret := range secrets.Items {
		if secret.Data == nil {
			continue
		}
		if isRegexBytes := secret.Data[FieldRepoURLIsRegex]; string(isRegexBytes) == "true" {
			continue
		}
		urlBytes, ok := secret.Data[FieldRepoURL]
		if !ok {
			continue
		}
		url := helm.NormalizeChartRepositoryURL( // This should be safe even on non-chart repo URLs
			git.NormalizeGitURL( // This should be safe even on non-Git URLs
				string(urlBytes),
			),
		)
		if url == repoURL {
			return &secret, nil
		}
	}

	logger := logging.LoggerFromContext(ctx)

	// Scan for a pattern match
	for _, secret := range secrets.Items {
		if secret.Data == nil {
			continue
		}
		if isRegexBytes := secret.Data[FieldRepoURLIsRegex]; string(isRegexBytes) != "true" {
			continue
		}
		patternBytes, ok := secret.Data[FieldRepoURL]
		if !ok {
			continue
		}
		regex, err := regexp.Compile(string(patternBytes))
		if err != nil {
			logger.WithFields(log.Fields{
				"namespace": namespace,
				"secret":    secret.Name,
			}).Warn("failed to compile regex for credential secret")
			continue
		}
		if regex.MatchString(repoURL) {
			return &secret, nil
		}
	}

	return nil, nil
}

func secretToCreds(secret *corev1.Secret) Credentials {
	return Credentials{
		Username:      string(secret.Data["username"]),
		Password:      string(secret.Data["password"]),
		SSHPrivateKey: string(secret.Data["sshPrivateKey"]),
	}
}
