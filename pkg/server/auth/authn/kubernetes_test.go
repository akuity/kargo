package authn

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	authnv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/user"
)

func TestKubernetes_Authenticate(t *testing.T) {
	t.Parallel()
	testToken := tokenFrom(t, "https://kubernetes.default.svc")
	testCases := []struct {
		name         string
		token        string
		reviewStatus authnv1.TokenReviewStatus
		createErr    error
		assert       func(*testing.T, user.Identity, error)
	}{
		{
			// Rejected without a TokenReview, which the interceptor would
			// fail on because the token is not the one it expects.
			name:  "not a JWT",
			token: "test-bearer-token",
			assert: func(t *testing.T, id user.Identity, err error) {
				require.ErrorIs(t, err, ErrInvalidToken)
				requireErrorStatus(t, err, http.StatusUnauthorized)
				require.Nil(t, id)
			},
		},
		{
			// The check could not be carried out, which says nothing about the
			// token, so the error must carry no status code for the
			// error-handling middleware to report to the client.
			name:      "TokenReview call fails",
			createErr: errors.New("connection refused"),
			assert: func(t *testing.T, id user.Identity, err error) {
				require.ErrorContains(t, err, "submit TokenReview")
				require.ErrorContains(t, err, "connection refused")
				var httpErr *libhttp.HTTPError
				require.False(t, errors.As(err, &httpErr))
				require.Nil(t, id)
			},
		},
		{
			name:         "Kubernetes reports the token as invalid",
			reviewStatus: authnv1.TokenReviewStatus{Authenticated: false},
			assert: func(t *testing.T, id user.Identity, err error) {
				requireErrorStatus(t, err, http.StatusUnauthorized)
				require.ErrorIs(t, err, ErrInvalidToken)
				require.Nil(t, id)
			},
		},
		{
			name:         "Kubernetes reports an error verifying the token",
			reviewStatus: authnv1.TokenReviewStatus{Error: "some verification error"},
			assert: func(t *testing.T, id user.Identity, err error) {
				requireErrorStatus(t, err, http.StatusUnauthorized)
				require.ErrorContains(t, err, "some verification error")
				require.Nil(t, id)
			},
		},
		{
			name: "Kubernetes authenticates the token",
			reviewStatus: authnv1.TokenReviewStatus{
				Authenticated: true,
				User: authnv1.UserInfo{
					Username: "system:serviceaccount:kargo-demo:ci-bot",
					UID:      "abc-123",
				},
			},
			assert: func(t *testing.T, id user.Identity, err error) {
				require.NoError(t, err)
				require.Equal(t, user.KubernetesUser{UserInfo: authnv1.UserInfo{
					Username: "system:serviceaccount:kargo-demo:ci-bot",
					UID:      "abc-123",
				}}, id)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			scheme := runtime.NewScheme()
			require.NoError(t, authnv1.AddToScheme(scheme))
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
						review, ok := obj.(*authnv1.TokenReview)
						require.True(t, ok)
						require.Equal(t, testToken, review.Spec.Token)
						review.Status = testCase.reviewStatus
						return nil
					},
				},
			).Build()
			token := testCase.token
			if token == "" {
				token = testToken
			}
			id, ok, err := NewKubernetes(kube).Authenticate(t.Context(), token)
			// Every token is of this kind.
			require.True(t, ok)
			testCase.assert(t, id, err)
		})
	}
}

// requireErrorStatus asserts that err carries the expected HTTP status code
// for the error-handling middleware to act on.
func requireErrorStatus(t *testing.T, err error, code int) {
	t.Helper()
	var httpErr *libhttp.HTTPError
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, code, httpErr.Code())
}
