package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOIDCClaimsFromAnnotationValues(t *testing.T) {
	for _, test := range []struct {
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
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := OIDCClaimsFromAnnotationValues(test.annotations)
			if test.shouldErrOut {
				require.Error(t, err)
				require.ErrorContains(t, err, test.expectedErrMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expected, got)
		})
	}
}
