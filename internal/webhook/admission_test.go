package webhook

import (
	"testing"

	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	authnv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/authentication/serviceaccount"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestIsRequestFromKargoControlplane(t *testing.T) {
	testCases := map[string]struct {
		knownServiceAccounts map[types.NamespacedName]struct{}
		userInfo             authnv1.UserInfo
		expected             bool
	}{
		"no known service accounts": {
			userInfo: authnv1.UserInfo{
				Username: serviceaccount.ServiceAccountUsernamePrefix + "kargo:kargo-api",
			},
			expected: false,
		},
		"unknown service account": {
			knownServiceAccounts: map[types.NamespacedName]struct{}{
				{Namespace: "kargo", Name: "kargo-api"}: {},
			},
			userInfo: authnv1.UserInfo{
				Username: serviceaccount.ServiceAccountUsernamePrefix + "some-namespace:unknown-service-account",
			},
			expected: false,
		},
		"known service account": {
			knownServiceAccounts: map[types.NamespacedName]struct{}{
				{Namespace: "kargo", Name: "kargo-api"}: {},
			},
			userInfo: authnv1.UserInfo{
				Username: serviceaccount.ServiceAccountUsernamePrefix + "kargo:kargo-api",
			},
			expected: true,
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					UserInfo: tc.userInfo,
				},
			}
			actual := IsRequestFromKargoControlplane(tc.knownServiceAccounts)(req)
			require.Equal(t, tc.expected, actual)
		})
	}
}
