package credentials

import (
	"context"
	"strings"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/git"
)

const (
	// authorizedProjectsAnnotationKey is the key for an annotation used by owners
	// of Secrets in Argo CD's namespace to indicate consent to be borrowed by
	// specific Kargo projects.
	authorizedProjectsAnnotationKey = "kargo.akuity.io/authorized-projects"

	// kargoSecretTypeLabelKey is the key for a label used to identify the type
	// of credentials stored in a Secret.
	kargoSecretTypeLabelKey = "kargo.akuity.io/secret-type" // nolint: gosec
	// argoCDSecretTypeLabelKey is the key for a label used to identify the type
	// of credentials stored in a Secret within Argo CD's namespace.
	argoCDSecretTypeLabelKey = "argocd.argoproj.io/secret-type" // nolint: gosec
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
	argoClient  client.Client // nil if credential borrowing is not enabled
	cfg         KubernetesDatabaseConfig
}

// KubernetesDatabaseConfig represents configuration for a Kubernetes based
// implementation of the Database interface.
type KubernetesDatabaseConfig struct {
	ArgoCDNamespace             string   `envconfig:"ARGOCD_NAMESPACE" default:"argocd"`
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
	argoClient client.Client,
	cfg KubernetesDatabaseConfig,
) Database {
	return &kubernetesDatabase{
		kargoClient: kargoClient,
		argoClient:  argoClient,
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
	// Found creds in namespace
	if secret != nil {
		return secretToCreds(secret), true, nil
	}

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

	// Found template creds in namespace
	if secret != nil {
		return secretToCreds(secret), true, nil
	}

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
		// Found creds in global creds namespace
		if secret != nil {
			return secretToCreds(secret), true, nil
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
			return secretToCreds(secret), true, nil
		}
	}

	if k.argoClient == nil {
		// We cannot borrow creds from from Argo CD
		return creds, false, nil
	}

	// Check Argo CD's namespace for credentials
	if secret, err = getCredentialsSecret(
		ctx,
		k.argoClient,
		k.cfg.ArgoCDNamespace,
		labels.Set(map[string]string{
			argoCDSecretTypeLabelKey: repositorySecretTypeLabelValue,
		}).AsSelector(),
		credType,
		repoURL,
		false, // repoURL is not a prefix
	); err != nil {
		return creds, false, err
	}

	if secret == nil {
		// Check Argo CD's namespace for credentials template
		if secret, err = getCredentialsSecret(
			ctx,
			k.argoClient,
			k.cfg.ArgoCDNamespace,
			labels.Set(map[string]string{
				argoCDSecretTypeLabelKey: repoCredsSecretTypeLabelValue,
			}).AsSelector(),
			credType,
			repoURL,
			true, // repoURL is a prefix
		); err != nil || secret == nil {
			return creds, false, err
		}
	}

	if secret == nil {
		return creds, false, nil
	}

	// This Secret represents credentials borrowed from Argo CD. We need to look
	// at its annotations to see if this is authorized by the Secret's owner.
	// If it's not annotated properly, we'll treat it as we didn't find it.
	allowedProjectsStr, ok := secret.Annotations[authorizedProjectsAnnotationKey]
	if !ok {
		return creds, false, nil
	}
	allowedProjects := strings.Split(allowedProjectsStr, ",")
	for _, allowedProject := range allowedProjects {
		if strings.TrimSpace(allowedProject) == namespace {
			return secretToCreds(secret), true, nil
		}
	}

	return creds, false, nil
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
	if credType == TypeGit {
		// This is important. We don't want the presence or absence of ".git" at the
		// end of the URL to affect credential lookups.
		repoURL = git.NormalizeGitURL(repoURL)
	}

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
		url := string(urlBytes)
		if acceptPrefixMatch && strings.HasPrefix(repoURL, url) {
			return &secret, nil
		}
		if !acceptPrefixMatch && git.NormalizeGitURL(url) == repoURL {
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
