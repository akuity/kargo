package controller

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

// credentialsType is a string type used to represent a type of credentials.
type credentialsType string

const (
	// credentialsTypeGit represents credentials for a Git repository.
	credentialsTypeGit credentialsType = "git"
	// credentialsTypeHelm represents credentials for a Helm chart repository.
	credentialsTypeHelm credentialsType = "helm"
	// credentialsTypeImage represents credentials for an image repository.
	credentialsTypeImage credentialsType = "image"

	// secretsByRepo is the name of the index that credentialsDB uses for indexing
	// credentials stored in Kubernetes Secrets by repository type + URL.
	secretsByRepo = "repo"

	kargoSecretTypeLabel = "kargo.akuity.io/secret-type"
)

// credentials generically represents any type of repository credential.
type credentials struct {
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

// credentialsDB is an interface for a credentials store.
type credentialsDB interface {
	get(
		ctx context.Context,
		namespace string,
		credType credentialsType,
		repo string,
	) (credentials, bool, error)
}

// kubernetesCredentialsDB is an implementation of the credentialsDB
// interface that utilizes a Kubernetes controller runtime client to index and
// retrieve credentials stored in Kubernetes Secrets.
type kubernetesCredentialsDB struct {
	argoCDNamespace string
	client          client.Client
}

// newKubernetesCredentialsDB initializes a new instance of
// kubernetesCredentialsDB. This function carries out the important task of
// indexing credentials stored in Kubernetes Secrets by repository type + URL.
func newKubernetesCredentialsDB(
	ctx context.Context,
	argoCDNamespace string,
	mgr manager.Manager,
) (credentialsDB, error) {
	k := &kubernetesCredentialsDB{
		argoCDNamespace: argoCDNamespace,
		client:          mgr.GetClient(),
	}
	err := mgr.GetFieldIndexer().IndexField(
		ctx,
		&corev1.Secret{},
		secretsByRepo,
		k.index,
	)
	return k, errors.Wrap(err, "error indexing Secrets by repo")
}

func (k *kubernetesCredentialsDB) index(obj client.Object) []string {
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
	var credsType credentialsType
	if credsTypeBytes, ok := secret.Data["type"]; ok {
		credsType = credentialsType(credsTypeBytes)
	} else {
		// If not specified, assume these credentials are for a Git repo.
		credsType = credentialsTypeGit
	}
	// Refuse to index this Secret if we don't recognize what type of
	// repository these credentials are supposed to be for.
	switch credsType {
	case credentialsTypeGit, credentialsTypeHelm, credentialsTypeImage:
	default:
		return nil
	}
	var repoURL string
	if repoURLBytes, ok := secret.Data["url"]; ok {
		repoURL = string(repoURLBytes)
		if credsType == credentialsTypeGit {
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

func (k *kubernetesCredentialsDB) get(
	ctx context.Context,
	namespace string,
	credType credentialsType,
	repoURL string,
) (credentials, bool, error) {
	if credType == credentialsTypeGit {
		// This is important. We don't want the presence or absence of ".git" at the
		// end of the URL to affect credential lookups.
		repoURL = git.NormalizeGitURL(repoURL)
	}
	creds := credentials{}

	var secret *corev1.Secret
	var err error

	// Check namespace for credentials
	if secret, err = k.getCredentialsSecret(
		ctx,
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
		if secret, err = k.getCredentialsTemplateSecret(
			ctx,
			namespace,
			labels.Set(map[string]string{
				kargoSecretTypeLabel: common.LabelValueSecretTypeRepoCreds,
			}).AsSelector(),
			repoURL,
		); err != nil {
			return creds, false, err
		}
	}

	if secret == nil {
		// Check Argo CD's namespace for credentials
		if secret, err = k.getCredentialsSecret(
			ctx,
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
	}

	if secret == nil {
		// Check Argo CD's namespace for credentials template
		if secret, err = k.getCredentialsTemplateSecret(
			ctx,
			k.argoCDNamespace,
			labels.Set(map[string]string{
				utils.ArgoCDSecretTypeLabel: common.LabelValueSecretTypeRepoCreds,
			}).AsSelector(),
			repoURL,
		); err != nil || secret == nil {
			return creds, false, err
		}
	}

	creds.Username = string(secret.Data["username"])
	creds.Password = string(secret.Data["password"])
	creds.SSHPrivateKey = string(secret.Data["sshPrivateKey"])
	return creds, true, nil
}

func (k *kubernetesCredentialsDB) getCredentialsSecret(
	ctx context.Context,
	namespace string,
	labelSelector labels.Selector,
	fieldSelector fields.Selector,
) (*corev1.Secret, error) {
	secrets := corev1.SecretList{}
	if err := k.client.List(
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

func (k *kubernetesCredentialsDB) getCredentialsTemplateSecret(
	ctx context.Context,
	namespace string,
	labelSelector labels.Selector,
	repoURL string,
) (*corev1.Secret, error) {
	secrets := corev1.SecretList{}
	if err := k.client.List(
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

func credsSecretIndexVal(credsType credentialsType, repoURL string) string {
	return fmt.Sprintf("%s:%s", credsType, repoURL)
}
