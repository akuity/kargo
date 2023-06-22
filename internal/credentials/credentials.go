package credentials

import (
	"context"
	"fmt"
	"strings"

	"github.com/argoproj/argo-cd/v2/applicationset/utils"
	"github.com/argoproj/argo-cd/v2/common"
	"github.com/argoproj/argo-cd/v2/util/git"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const authorizedProjectsAnnotationKey = "kargo.akuity.io/authorized-projects"

// Type is a string type used to represent a type of Credentials.
type Type string

const (
	// TypeGit represents credentials for a Git repository.
	TypeGit Type = "git"
	// TypeHelm represents credentials for a Helm chart repository.
	TypeHelm Type = "helm"
	// TypeImage represents credentials for an image repository.
	TypeImage Type = "image"

	// secretsByRepo is the name of the index that credentialsDB uses for indexing
	// credentials stored in Kubernetes Secrets by repository type + URL.
	secretsByRepo = "repo"

	kargoSecretTypeLabel = "kargo.akuity.io/secret-type" // nolint: gosec
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
// utilizes a Kubernetes controller runtime client to index and retrieve
// credentials stored in Kubernetes Secrets.
type kubernetesDatabase struct {
	argoCDNamespace string
	kargoClient     client.Client
	argoClient      client.Client
}

// NewKubernetesDatabase initializes and returns an implementation of the
// Database interface that utilizes a Kubernetes controller runtime client to
// index and retrieve Credentials stored in Kubernetes Secrets. This function
// carries out the important task of indexing Credentials stored in Kubernetes
// Secrets by repository type + URL.
func NewKubernetesDatabase(
	ctx context.Context,
	argoCDNamespace string,
	kargoMgr manager.Manager,
	argoMgr manager.Manager,
) (Database, error) {
	k := &kubernetesDatabase{
		argoCDNamespace: argoCDNamespace,
		kargoClient:     kargoMgr.GetClient(),
	}
	if argoMgr != nil {
		k.argoClient = argoMgr.GetClient()
	}
	if err := kargoMgr.GetFieldIndexer().IndexField(
		ctx,
		&corev1.Secret{},
		secretsByRepo,
		k.index,
	); err != nil {
		return k, errors.Wrap(err, "error indexing Secrets by repo")
	}
	if argoMgr != nil && argoMgr != kargoMgr {
		if err := argoMgr.GetFieldIndexer().IndexField(
			ctx,
			&corev1.Secret{},
			secretsByRepo,
			k.index,
		); err != nil {
			return k, errors.Wrap(err, "error indexing Secrets by repo")
		}
	}
	return k, nil
}

func (k *kubernetesDatabase) index(obj client.Object) []string {
	secret := obj.(*corev1.Secret) // nolint: forcetypeassert
	// Refuse to index this secret if it has no labels or data.
	if secret.Labels == nil || secret.Data == nil {
		return nil
	}
	if secret.Namespace == k.argoCDNamespace {
		// If the Secret is in Argo CD's namespace, expect that it should be
		// labeled like an Argo CD Secret. If it isn't refuse to index this
		// Secret. We're also not interested in indexing Secrets that represent
		// credential templates, because we will always have to iterate over
		// such secrets to find what we're looking for anyway.
		if secType, ok :=
			secret.Labels[utils.ArgoCDSecretTypeLabel]; !ok ||
			secType != common.LabelValueSecretTypeRepository {
			return nil
		}
	} else {
		// If the Secret is in any other namespace, expect that it should be
		// labeled like a Kargo Secret. If it isn't refuse to index this Secret.
		// We're also not interested in indexing Secrets that represent
		// credential templates, because we will always have to iterate over
		// such secrets to find what we're looking for anyway.
		if secType, ok := secret.Labels[kargoSecretTypeLabel]; !ok ||
			secType != common.LabelValueSecretTypeRepository {
			return nil
		}
	}
	var credsType Type
	if credsTypeBytes, ok := secret.Data["type"]; ok {
		credsType = Type(credsTypeBytes)
	} else {
		// If not specified, assume these credentials are for a Git repo.
		credsType = TypeGit
	}
	// Refuse to index this Secret if we don't recognize what type of
	// repository these credentials are supposed to be for.
	switch credsType {
	case TypeGit, TypeHelm, TypeImage:
	default:
		return nil
	}
	var repoURL string
	if repoURLBytes, ok := secret.Data["url"]; ok {
		repoURL = string(repoURLBytes)
		if credsType == TypeGit {
			// This is important. We don't want the presence or absence of ".git"
			// at the end of the URL to affect credential lookups.
			repoURL = git.NormalizeGitURL(string(repoURLBytes))
		}
	} else {
		// No URL. Refuse to index this Secret.
		return nil
	}
	return []string{credsSecretIndexVal(credsType, repoURL)}
}

func (k *kubernetesDatabase) Get(
	ctx context.Context,
	namespace string,
	credType Type,
	repoURL string,
) (Credentials, bool, error) {
	if credType == TypeGit {
		// This is important. We don't want the presence or absence of ".git" at the
		// end of the URL to affect credential lookups.
		repoURL = git.NormalizeGitURL(repoURL)
	}
	creds := Credentials{}

	var secret *corev1.Secret
	var err error

	// Check namespace for credentials
	if secret, err = getCredentialsSecret(
		ctx,
		k.kargoClient,
		namespace,
		labels.Set(map[string]string{
			kargoSecretTypeLabel: common.LabelValueSecretTypeRepository,
		}).AsSelector(),
		fields.Set(map[string]string{
			secretsByRepo: credsSecretIndexVal(credType, repoURL),
		}).AsSelector(),
	); err != nil {
		return creds, false, err
	}

	if secret == nil {
		// Check namespace for credentials template
		if secret, err = getCredentialsTemplateSecret(
			ctx,
			k.kargoClient,
			namespace,
			labels.Set(map[string]string{
				kargoSecretTypeLabel: common.LabelValueSecretTypeRepoCreds,
			}).AsSelector(),
			repoURL,
		); err != nil {
			return creds, false, err
		}
	}

	if secret != nil {
		creds.Username = string(secret.Data["username"])
		creds.Password = string(secret.Data["password"])
		creds.SSHPrivateKey = string(secret.Data["sshPrivateKey"])
		return creds, true, nil
	}

	if k.argoClient == nil {
		// We cannot borrow creds from from Argo CD
		return creds, false, nil
	}

	// Check Argo CD's namespace for credentials
	if secret, err = getCredentialsSecret(
		ctx,
		k.argoClient,
		k.argoCDNamespace,
		labels.Set(map[string]string{
			utils.ArgoCDSecretTypeLabel: common.LabelValueSecretTypeRepository,
		}).AsSelector(),
		fields.Set(map[string]string{
			secretsByRepo: credsSecretIndexVal(credType, repoURL),
		}).AsSelector(),
	); err != nil {
		return creds, false, err
	}

	if secret == nil {
		// Check Argo CD's namespace for credentials template
		if secret, err = getCredentialsTemplateSecret(
			ctx,
			k.argoClient,
			k.argoCDNamespace,
			labels.Set(map[string]string{
				utils.ArgoCDSecretTypeLabel: common.LabelValueSecretTypeRepoCreds,
			}).AsSelector(),
			repoURL,
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
			creds.Username = string(secret.Data["username"])
			creds.Password = string(secret.Data["password"])
			creds.SSHPrivateKey = string(secret.Data["sshPrivateKey"])
			return creds, true, nil
		}
	}

	return creds, false, nil
}

func getCredentialsSecret(
	ctx context.Context,
	kubeClient client.Client,
	namespace string,
	labelSelector labels.Selector,
	fieldSelector fields.Selector,
) (*corev1.Secret, error) {
	secrets := corev1.SecretList{}
	if err := kubeClient.List(
		ctx,
		&secrets,
		&client.ListOptions{
			Namespace:     namespace,
			LabelSelector: labelSelector,
			FieldSelector: fieldSelector,
		},
	); err != nil {
		return nil, err
	}
	if len(secrets.Items) == 0 {
		return nil, nil
	}
	// We know any secret we find has properly formatted data because that was
	// a condition for indexing it.
	return &(secrets.Items[0]), nil
}

func getCredentialsTemplateSecret(
	ctx context.Context,
	kubeClient client.Client,
	namespace string,
	labelSelector labels.Selector,
	repoURL string,
) (*corev1.Secret, error) {
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
		if url, ok := secret.Data["url"]; ok &&
			strings.HasPrefix(repoURL, string(url)) {
			return &secret, nil
		}
	}
	return nil, nil
}

func credsSecretIndexVal(credsType Type, repoURL string) string {
	return fmt.Sprintf("%s:%s", credsType, repoURL)
}
