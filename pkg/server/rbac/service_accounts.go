package rbac

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
)

type ServiceAccountDatabaseConfig struct {
	KargoNamespace string `envconfig:"KARGO_NAMESPACE" default:"kargo"`
}

func ServiceAccountDatabaseConfigFromEnv() ServiceAccountDatabaseConfig {
	cfg := ServiceAccountDatabaseConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// ServiceAccountsDatabase is an interface for the Kargo ServiceAccounts store.
// Note: Kargo ServiceAccounts are specially labeled Kubernetes ServiceAccounts
// with associated bearer tokens.
type ServiceAccountsDatabase interface {
	// Create creates a Kargo ServiceAccount.
	Create(context.Context, *corev1.ServiceAccount) (*corev1.ServiceAccount, error)
	// Delete deletes a Kargo ServiceAccount.
	Delete(ctx context.Context, project, name string) error
	// Get returns a Kargo ServiceAccount.
	Get(
		ctx context.Context,
		systemLevel bool,
		project string,
		name string,
	) (*corev1.ServiceAccount, error)
	// List returns a list of Kargo ServiceAccounts.
	List(
		ctx context.Context,
		systemLevel bool,
		project string,
	) ([]corev1.ServiceAccount, error)
	// CreateToken generates and returns a new bearer token for a Kargo
	// ServiceAccount in the form of a Kubernetes Secret.
	CreateToken(
		ctx context.Context,
		systemLevel bool,
		project string,
		saName string,
		tokenName string,
	) (*corev1.Secret, error)
	// DeleteToken deletes a bearer token associated with a Kargo ServiceAccount.
	DeleteToken(
		ctx context.Context,
		systemLevel bool,
		project string,
		name string,
	) error
	// GetToken returns a bearer token associated with a Kargo ServiceAccount.
	GetToken(
		ctx context.Context,
		systemLevel bool,
		project string,
		name string,
	) (*corev1.Secret, error)
	// ListTokens lists all bearer tokens associated with a specified Kargo
	// ServiceAccount.
	ListTokens(
		ctx context.Context,
		systemLevel bool,
		project string,
		saName string,
	) ([]corev1.Secret, error)
}

// serviceAccountsDatabase is an implementation of the ServiceAccountsDatabase
// interface that utilizes a Kubernetes controller runtime client to store and
// retrieve Kargo ServiceAccounts.
type serviceAccountsDatabase struct {
	client client.Client
	cfg    ServiceAccountDatabaseConfig
}

// NewKubernetesServiceAccountsDatabase returns an implementation of the
// ServiceAccountsDatabase interface that utilizes a Kubernetes controller
// runtime client to store and retrieve Kargo ServiceAccounts.
func NewKubernetesServiceAccountsDatabase(
	c client.Client,
	cfg ServiceAccountDatabaseConfig,
) ServiceAccountsDatabase {
	return &serviceAccountsDatabase{
		client: c,
		cfg:    cfg,
	}
}

// Create implements ServiceAccountsDatabase.
func (s *serviceAccountsDatabase) Create(
	ctx context.Context,
	sa *corev1.ServiceAccount,
) (*corev1.ServiceAccount, error) {
	if sa.Labels == nil {
		sa.Labels = make(map[string]string, 1)
	}
	sa.Labels[rbacapi.LabelKeyServiceAccount] = rbacapi.LabelValueTrue
	if sa.Annotations == nil {
		sa.Annotations = make(map[string]string, 1)
	}
	sa.Annotations[rbacapi.AnnotationKeyManaged] = rbacapi.AnnotationValueTrue
	if err := s.client.Create(ctx, sa); err != nil {
		return nil, fmt.Errorf("error creating ServiceAccount %q in namespace %q: %w",
			sa.Name, sa.Namespace, err,
		)
	}
	return sa, nil
}

// Delete implements ServiceAccountsDatabase.
func (s *serviceAccountsDatabase) Delete(
	ctx context.Context,
	project string,
	name string,
) error {
	sa := &corev1.ServiceAccount{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{
			Namespace: project,
			Name:      name,
		},
		sa,
	); err != nil {
		return fmt.Errorf(
			"error getting ServiceAccount %q in namespace %q: %w", name, project, err,
		)
	}
	if !isKargoServiceAccount(sa) {
		return apierrors.NewBadRequest(
			fmt.Sprintf(
				"Kubernetes ServiceAccount %q in namespace %q is not labeled as a "+
					"Kargo ServiceAccount",
				sa.Name, sa.Namespace,
			),
		)
	}
	if !isKargoManaged(sa) {
		return apierrors.NewBadRequest(
			fmt.Sprintf(
				"Kubernetes ServiceAccount %q in namespace %q is not annotated as "+
					"Kargo-managed",
				sa.Name, sa.Namespace,
			),
		)
	}
	if err := s.client.Delete(ctx, sa); err != nil {
		return fmt.Errorf(
			"error deleting ServiceAccount %q in namespace %q: %w",
			sa.Name, sa.Namespace, err,
		)
	}
	return nil
}

// Get implements ServiceAccountsDatabase.
func (s *serviceAccountsDatabase) Get(
	ctx context.Context,
	systemLevel bool,
	project string,
	name string,
) (*corev1.ServiceAccount, error) {
	namespace := project
	if systemLevel {
		namespace = s.cfg.KargoNamespace
	}
	sa := &corev1.ServiceAccount{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		},
		sa,
	); err != nil {
		return nil, fmt.Errorf(
			"error getting ServiceAccount %q in namespace %q: %w", name, namespace, err,
		)
	}
	if !isKargoServiceAccount(sa) {
		return nil, apierrors.NewBadRequest(
			fmt.Sprintf(
				"Kubernetes ServiceAccount %q in namespace %q is not labeled as a "+
					"Kargo ServiceAccount",
				sa.Name, sa.Namespace,
			),
		)
	}
	return sa, nil
}

// List implements the ServiceAccountsDatabase interface.
func (s *serviceAccountsDatabase) List(
	ctx context.Context,
	systemLevel bool,
	project string,
) ([]corev1.ServiceAccount, error) {
	namespace := project
	if systemLevel {
		namespace = s.cfg.KargoNamespace
	}
	saList := &corev1.ServiceAccountList{}
	if err := s.client.List(
		ctx,
		saList,
		client.InNamespace(namespace),
		client.MatchingLabels{
			rbacapi.LabelKeyServiceAccount: rbacapi.LabelValueTrue,
		},
	); err != nil {
		return nil, fmt.Errorf(
			"error listing ServiceAccounts in namespace %q: %w", namespace, err,
		)
	}
	slices.SortFunc(saList.Items, func(a, b corev1.ServiceAccount) int {
		return strings.Compare(a.Name, b.Name)
	})
	return saList.Items, nil
}

// CreateToken implements ServiceAccountsDatabase.
func (s *serviceAccountsDatabase) CreateToken(
	ctx context.Context,
	systemLevel bool,
	project string,
	saName string,
	tokenName string,
) (*corev1.Secret, error) {
	namespace := project
	if systemLevel {
		namespace = s.cfg.KargoNamespace
	}
	sa, err := s.Get(ctx, systemLevel, project, saName)
	if err != nil {
		return nil, err
	}
	if !isKargoServiceAccount(sa) {
		return nil, apierrors.NewBadRequest(
			fmt.Sprintf(
				"Kubernetes ServiceAccount %q in namespace %q is not labeled as a "+
					"Kargo ServiceAccount",
				sa.Name, sa.Namespace,
			),
		)
	}
	tokenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      tokenName,
			Labels: map[string]string{
				rbacapi.LabelKeyServiceAccountToken: rbacapi.LabelValueTrue,
			},
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": saName,
				rbacapi.AnnotationKeyManaged:         rbacapi.AnnotationValueTrue,
			},
			// Make sure deleting the ServiceAccount cascades to associated tokens.
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "v1",
				Kind:       "ServiceAccount",
				Name:       saName,
				UID:        sa.UID,
			}},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}
	if err = s.client.Create(ctx, tokenSecret); err != nil {
		return nil, fmt.Errorf(
			"error creating token Secret %q for ServiceAccount %q in namespace %q: %w",
			tokenName, saName, namespace, err,
		)
	}

	// Retrieve Secret -- this is necessary to actually get the token. We wrap
	// this in a retry because token data is created asynchronously and we don't
	// want to prematurely return the Secret without its data.
	tokenSecret, err = s.waitForTokenData(
		ctx,
		namespace,
		tokenName,
		5, // Up to five attempts
	)
	if err != nil {
		return nil, err
	}

	return tokenSecret, nil
}

// waitForTokenData retrieves a token Secret with retry logic. It retries when:
//
//  1. The Secret exists but token data hasn't been populated yet
//  2. Transient errors occur (timeouts, rate limits, server errors, conflicts)
//
// It does NOT retry on permanent errors like NotFound, BadRequest, Forbidden,
// etc.
func (s *serviceAccountsDatabase) waitForTokenData(
	ctx context.Context,
	namespace string,
	tokenName string,
	maxAttempts int,
) (*corev1.Secret, error) {
	var tokenSecret *corev1.Secret
	backoff := retry.DefaultBackoff
	backoff.Steps = maxAttempts

	if err := retry.OnError(
		backoff,
		func(innerErr error) bool {
			if innerErr == nil {
				return false // Stop retrying if no error
			}
			// Retry on transient errors
			_, isTokenNotPopulatedErr := innerErr.(*errTokenNotPopulated)
			return isTokenNotPopulatedErr ||
				apierrors.IsServerTimeout(innerErr) ||
				apierrors.IsTimeout(innerErr) ||
				apierrors.IsTooManyRequests(innerErr) ||
				apierrors.IsServiceUnavailable(innerErr) ||
				apierrors.IsInternalError(innerErr) ||
				apierrors.IsConflict(innerErr)
		},
		func() error {
			tokenSecret = &corev1.Secret{}
			if innerErr := s.client.Get(
				ctx,
				client.ObjectKey{
					Namespace: namespace,
					Name:      tokenName,
				},
				tokenSecret,
			); innerErr != nil {
				return innerErr
			}
			if _, gotToken := tokenSecret.Data["token"]; !gotToken {
				return &errTokenNotPopulated{}
			}
			return nil
		},
	); err != nil {
		return nil, fmt.Errorf(
			"error while waiting for token Secret %q in namespace %q to be "+
				"populated: %w",
			tokenName, namespace, err,
		)
	}

	return tokenSecret, nil
}

// DeleteToken implements ServiceAccountsDatabase.
func (s *serviceAccountsDatabase) DeleteToken(
	ctx context.Context,
	systemLevel bool,
	project string,
	name string,
) error {
	namespace := project
	if systemLevel {
		namespace = s.cfg.KargoNamespace
	}
	tokenSecret := &corev1.Secret{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		},
		tokenSecret,
	); err != nil {
		return fmt.Errorf(
			"error getting token Secret %q in namespace %q: %w", name, namespace, err,
		)
	}
	if !isKargoServiceAccountToken(tokenSecret) {
		return apierrors.NewBadRequest(
			fmt.Sprintf(
				"Kubernetes Secret %q in namespace %q is not labeled as a Kargo "+
					"ServiceAccount token",
				tokenSecret.Name, tokenSecret.Namespace,
			),
		)
	}
	if !isKargoManaged(tokenSecret) {
		return apierrors.NewBadRequest(
			fmt.Sprintf(
				"Kubernetes Secret %q in namespace %q is not annotated as Kargo-managed",
				tokenSecret.Name, tokenSecret.Namespace,
			),
		)
	}
	if err := s.client.Delete(ctx, tokenSecret); err != nil {
		return fmt.Errorf(
			"error deleting token Secret %q in namespace %q: %w",
			tokenSecret.Name, tokenSecret.Namespace, err,
		)
	}
	return nil
}

// GetToken implements ServiceAccountsDatabase.
func (s *serviceAccountsDatabase) GetToken(
	ctx context.Context,
	systemLevel bool,
	project string,
	name string,
) (*corev1.Secret, error) {
	namespace := project
	if systemLevel {
		namespace = s.cfg.KargoNamespace
	}
	tokenSecret := &corev1.Secret{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		},
		tokenSecret,
	); err != nil {
		return nil, fmt.Errorf(
			"error getting token Secret %q in namespace %q: %w", name, namespace, err,
		)
	}
	if !isKargoServiceAccountToken(tokenSecret) {
		return nil, apierrors.NewBadRequest(
			fmt.Sprintf(
				"Kubernetes Secret %q in namespace %q is not labeled as a Kargo "+
					"ServiceAccount token",
				tokenSecret.Name, tokenSecret.Namespace,
			),
		)
	}
	redactTokenData(tokenSecret)
	return tokenSecret, nil
}

// ListTokens implements ServiceAccountsDatabase.
func (s *serviceAccountsDatabase) ListTokens(
	ctx context.Context,
	systemLevel bool,
	project string,
	saName string,
) ([]corev1.Secret, error) {
	namespace := project
	if systemLevel {
		namespace = s.cfg.KargoNamespace
	}
	if saName != "" {
		sa, err := s.Get(ctx, systemLevel, project, saName)
		if err != nil {
			return nil, err
		}
		if !isKargoServiceAccount(sa) {
			return nil, apierrors.NewBadRequest(
				fmt.Sprintf(
					"Kubernetes ServiceAccount %q in namespace %q is not labeled as a "+
						"Kargo ServiceAccount",
					sa.Name, sa.Namespace,
				),
			)
		}
	}
	tokenSecretList := &corev1.SecretList{}
	if err := s.client.List(
		ctx,
		tokenSecretList,
		client.InNamespace(namespace),
		client.MatchingLabels{
			rbacapi.LabelKeyServiceAccountToken: rbacapi.LabelValueTrue,
		},
	); err != nil {
		return nil, fmt.Errorf(
			"error listing token Secrets for ServiceAccount %q in namespace %q: %w",
			saName, namespace, err,
		)
	}
	var tokenSecrets []corev1.Secret
	for _, tokenSecret := range tokenSecretList.Items {
		if isKargoServiceAccountToken(&tokenSecret) &&
			(saName == "" || tokenSecret.Annotations["kubernetes.io/service-account.name"] == saName) {
			redactTokenData(&tokenSecret)
			tokenSecrets = append(tokenSecrets, tokenSecret)
		}
	}
	return tokenSecrets, nil
}

func isKargoServiceAccount(sa *corev1.ServiceAccount) bool {
	return sa.Labels[rbacapi.LabelKeyServiceAccount] == rbacapi.LabelValueTrue
}

func isKargoServiceAccountToken(secret *corev1.Secret) bool {
	return secret.Type == corev1.SecretTypeServiceAccountToken &&
		secret.Labels[rbacapi.LabelKeyServiceAccountToken] == rbacapi.LabelValueTrue
}

func redactTokenData(tokenSecret *corev1.Secret) {
	if _, ok := tokenSecret.Data["token"]; ok {
		tokenSecret.Data["token"] = []byte("*** REDACTED ***")
	}
}

type errTokenNotPopulated struct{}

func (e *errTokenNotPopulated) Error() string {
	return "did not find token data"
}
