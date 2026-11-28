package authz

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/akuity/kargo/pkg/server/user"
)

func TestSubjectAccessReviewer_Authorize(t *testing.T) {
	t.Parallel()
	subject := user.Subject{
		Username: "alice",
		UID:      "1234",
		Groups:   []string{"platform"},
		Extra:    map[string]authv1.ExtraValue{"scopes": {"read"}},
	}
	ra := authv1.ResourceAttributes{
		Verb:      "promote",
		Group:     "kargo.akuity.io",
		Resource:  "stages",
		Namespace: "kargo-demo",
		Name:      "uat",
	}
	testCases := []struct {
		name      string
		allowed   bool
		createErr error
		assert    func(*testing.T, bool, error)
	}{
		{
			name:      "review cannot be submitted",
			createErr: errors.New("connection refused"),
			assert: func(t *testing.T, allowed bool, err error) {
				require.ErrorContains(t, err, "submit SubjectAccessReview")
				require.ErrorContains(t, err, "connection refused")
				require.False(t, allowed)
			},
		},
		{
			name: "denied",
			assert: func(t *testing.T, allowed bool, err error) {
				require.NoError(t, err)
				require.False(t, allowed)
			},
		},
		{
			name:    "allowed",
			allowed: true,
			assert: func(t *testing.T, allowed bool, err error) {
				require.NoError(t, err)
				require.True(t, allowed)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			scheme := runtime.NewScheme()
			require.NoError(t, authv1.AddToScheme(scheme))
			kube := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(
				interceptor.Funcs{
					Create: func(
						_ context.Context,
						_ client.WithWatch,
						obj client.Object,
						_ ...client.CreateOption,
					) error {
						if testCase.createErr != nil {
							return testCase.createErr
						}
						review, ok := obj.(*authv1.SubjectAccessReview)
						require.True(t, ok)
						// The whole subject is reviewed, not just its name.
						require.Equal(t, authv1.SubjectAccessReviewSpec{
							ResourceAttributes: &ra,
							User:               subject.Username,
							UID:                subject.UID,
							Groups:             subject.Groups,
							Extra:              subject.Extra,
						}, review.Spec)
						review.Status.Allowed = testCase.allowed
						return nil
					},
				},
			).Build()
			allowed, err := NewAuthorizer(kube).Authorize(t.Context(), subject, ra)
			testCase.assert(t, allowed, err)
		})
	}
}
