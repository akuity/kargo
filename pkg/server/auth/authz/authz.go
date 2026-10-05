// Package authz decides whether a Kubernetes subject may perform an operation.
// Kargo keeps no permissions of its own: the answer is whatever the cluster's
// RBAC says, asked for through a SubjectAccessReview.
package authz

import (
	"context"
	"fmt"

	authv1 "k8s.io/api/authorization/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/pkg/server/user"
)

// Authorizer decides whether a subject may perform an operation.
type Authorizer interface {
	// Authorize reports whether the subject may perform the operation. An
	// error means the question could not be answered, not that the answer is
	// no.
	Authorize(
		ctx context.Context,
		subject user.Subject,
		ra authv1.ResourceAttributes,
	) (bool, error)
}

// subjectAccessReviewer asks Kubernetes through a SubjectAccessReview.
type subjectAccessReviewer struct {
	client client.Client
}

var _ Authorizer = (*subjectAccessReviewer)(nil)

// NewAuthorizer returns an Authorizer that asks Kubernetes, through a
// SubjectAccessReview, on every call. It requires the client's identity to be
// permitted to create SubjectAccessReviews.
func NewAuthorizer(kube client.Client) Authorizer {
	return &subjectAccessReviewer{client: kube}
}

// Authorize implements Authorizer.
func (r *subjectAccessReviewer) Authorize(
	ctx context.Context,
	subject user.Subject,
	ra authv1.ResourceAttributes,
) (bool, error) {
	review := &authv1.SubjectAccessReview{
		Spec: authv1.SubjectAccessReviewSpec{
			ResourceAttributes: &ra,
			User:               subject.Username,
			UID:                subject.UID,
			Groups:             subject.Groups,
			Extra:              subject.Extra,
		},
	}
	if err := r.client.Create(ctx, review); err != nil {
		return false, fmt.Errorf("submit SubjectAccessReview: %w", err)
	}
	return review.Status.Allowed, nil
}
