package authn

import (
	"context"
	"fmt"

	authnv1 "k8s.io/api/authentication/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/pkg/server/user"
)

// kubernetes recognizes tokens that Kubernetes itself can vouch for: a
// ServiceAccount's, such as a Kargo API token or a controller's, or one from
// the cluster's own identity provider.
type kubernetes struct {
	client client.Client
}

var _ Authenticator = (*kubernetes)(nil)

// NewKubernetes returns an Authenticator that asks Kubernetes about a token
// through a TokenReview. It requires the kargo-api ClusterRole to grant
// permission to create TokenReviews.
func NewKubernetes(kube client.Client) Authenticator {
	return &kubernetes{client: kube}
}

// Authenticate implements Authenticator. Every token is of this kind: it is
// the last resort for a token no other Authenticator recognized. Only JWTs are
// accepted: any other kind of token Kubernetes might recognize is increasingly
// unlikely, so it is rejected without spending a TokenReview on it.
func (k *kubernetes) Authenticate(
	ctx context.Context,
	rawToken string,
) (user.Identity, bool, error) {
	if _, ok := unverifiedIssuer(rawToken); !ok {
		return nil, true, ErrInvalidToken
	}
	review := &authnv1.TokenReview{
		Spec: authnv1.TokenReviewSpec{Token: rawToken},
	}
	if err := k.client.Create(ctx, review); err != nil {
		// This says nothing about the token, so it is left untyped to be
		// reported as an internal error rather than a rejected credential.
		return nil, true, fmt.Errorf("submit TokenReview: %w", err)
	}
	if review.Status.Error != "" {
		return nil, true, fmt.Errorf("%w: %s", ErrInvalidToken, review.Status.Error)
	}
	if !review.Status.Authenticated {
		return nil, true, ErrInvalidToken
	}
	return user.KubernetesUser{UserInfo: review.Status.User}, true, nil
}
