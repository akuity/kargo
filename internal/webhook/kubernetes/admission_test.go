package webhook

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	authnv1 "k8s.io/api/authentication/v1"
	"k8s.io/apiserver/pkg/authentication/serviceaccount"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestIsRequestFromKargoControlplane(t *testing.T) {
	testCases := map[string]struct {
		regex    *regexp.Regexp
		userInfo authnv1.UserInfo
		expected bool
	}{
		"no expression provided": {
			userInfo: authnv1.UserInfo{
				Username: serviceaccount.ServiceAccountUsernamePrefix + "kargo:kargo-api",
			},
			expected: false,
		},
		"no match": {
			regex: regexp.MustCompile("^fake-user$"),
			userInfo: authnv1.UserInfo{
				Username: serviceaccount.ServiceAccountUsernamePrefix + "kargo:kargo-api",
			},
			expected: false,
		},
		"unknown service account": {
			regex: regexp.MustCompile("^system:serviceaccount:kargo:kargo-api$"),
			userInfo: authnv1.UserInfo{
				Username: serviceaccount.ServiceAccountUsernamePrefix + "some-namespace:unknown-service-account",
			},
			expected: false,
		},
		"known service account": {
			regex: regexp.MustCompile("^system:serviceaccount:kargo:kargo-api$"),
			userInfo: authnv1.UserInfo{
				Username: serviceaccount.ServiceAccountUsernamePrefix + "kargo:kargo-api",
			},
			expected: true,
		},
		"one of known service accounts": {
			regex: regexp.MustCompile("^system:serviceaccount:kargo:[a-z0-9]([-a-z0-9]*[a-z0-9])?$"),
			userInfo: authnv1.UserInfo{
				Username: serviceaccount.ServiceAccountUsernamePrefix + "kargo:kargo-api",
			},
			expected: true,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					UserInfo: tc.userInfo,
				},
			}
			actual := IsRequestFromKargoControlplane(tc.regex)(req)
			require.Equal(t, tc.expected, actual)
		})
	}
}
