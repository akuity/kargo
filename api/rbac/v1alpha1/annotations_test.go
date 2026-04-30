package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOIDCClaimsFromAnnotationValues(t *testing.T) {
	testCases := []struct {
		name           string
		annotations    map[string]string
		expected       map[string][]string
		shouldErrOut   bool
		expectedErrMsg string
	}{
		{
			name: "new style",
			annotations: map[string]string{
				AnnotationKeyOIDCClaims: `{"groups": ["foo", "bar"], "email": ["foo@bar.com"]}`,
			},
			expected: map[string][]string{
				"groups": {"bar", "foo"},
				"email":  {"foo@bar.com"},
			},
		},
		{
			name: "new style with bug",
			// Some versions incorrectly set the default claim annotations upon
			// programmatic Project creation.
			annotations: map[string]string{
				AnnotationKeyOIDCClaims: `{"groups": ["foo", "bar"], "email": "foo@bar.com"}`,
			},
			expected: map[string][]string{
				"groups": {"bar", "foo"},
				"email":  {"foo@bar.com"},
			},
		},
		{
			name: "new style with dupes",
			annotations: map[string]string{
				AnnotationKeyOIDCClaims: `{"groups": ["foo", "foo"], "email": ["foo@bar.com", "foo@bar.com"]}`,
			},
			expected: map[string][]string{
				"groups": {"foo"},
				"email":  {"foo@bar.com"},
			},
		},
		{
			name: "old style",
			annotations: map[string]string{
				AnnotationKeyOIDCClaimNamePrefix + "groups": "bar,foo",
				AnnotationKeyOIDCClaimNamePrefix + "email":  "foo@bar.com",
			},
			expected: map[string][]string{
				"groups": {"bar", "foo"},
				"email":  {"foo@bar.com"},
			},
		},
		{
			name: "old style with dupes",
			annotations: map[string]string{
				AnnotationKeyOIDCClaimNamePrefix + "groups": "foo,foo",
				AnnotationKeyOIDCClaimNamePrefix + "email":  "foo@bar.com,foo@bar.com",
			},
			expected: map[string][]string{
				"groups": {"foo"},
				"email":  {"foo@bar.com"},
			},
		},
		{
			name: "mixed",
			annotations: map[string]string{
				AnnotationKeyOIDCClaimNamePrefix + "groups": "bar,foo",
				AnnotationKeyOIDCClaimNamePrefix + "email":  "foo@bar.com",
				AnnotationKeyOIDCClaims:                     `{"groups": ["baz"], "email": ["bar@baz"]}`,
			},
			expected: map[string][]string{
				"groups": {"bar", "baz", "foo"},
				"email":  {"bar@baz", "foo@bar.com"},
			},
		},
		{
			name: "mixed with dupes",
			annotations: map[string]string{
				AnnotationKeyOIDCClaimNamePrefix + "groups": "bar,baz,foo",
				AnnotationKeyOIDCClaimNamePrefix + "email":  "bar@baz",
				AnnotationKeyOIDCClaims:                     `{"groups": ["bar", "baz", "foo"], "email": ["bar@baz"]}`,
			},
			expected: map[string][]string{
				"groups": {"bar", "baz", "foo"},
				"email":  {"bar@baz"},
			},
		},
		{
			name: "invalid new style - invalid json",
			annotations: map[string]string{
				AnnotationKeyOIDCClaims: "invalid",
			},
			shouldErrOut:   true,
			expectedErrMsg: "unmarshaling OIDC claims from annotation value",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := OIDCClaimsFromAnnotationValues(testCase.annotations)
			if testCase.shouldErrOut {
				require.Error(t, err)
				require.ErrorContains(t, err, testCase.expectedErrMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.expected, got)
		})
	}
}

func TestSetOIDCClaimsAnnotation(t *testing.T) {
	for _, test := range []struct {
		name     string
		claims   map[string][]string
		sa       *corev1.ServiceAccount
		expected map[string]string
	}{
		{
			name:   "new claims should overwrite existing claims and delete old style claims",
			claims: map[string][]string{"bar": {"baz"}},
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationKeyOIDCClaims:                  `{"foo": ["bar"]}`,
						AnnotationKeyOIDCClaimNamePrefix + "foo": "bar",
					},
				},
			},
			expected: map[string]string{
				AnnotationKeyOIDCClaims: `{"bar":["baz"]}`,
			},
		},
		{
			name:   "nil annotations should not panic",
			claims: map[string][]string{"bar": {"baz"}},
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{Annotations: nil},
			},
			expected: map[string]string{
				AnnotationKeyOIDCClaims: `{"bar":["baz"]}`,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			}
			err := SetOIDCClaimsAnnotation(sa, test.claims)
			require.NoError(t, err)
			require.Equal(t, test.expected, sa.Annotations)
		})
	}
}
