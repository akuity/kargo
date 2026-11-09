package subscription

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnknownFields(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		subType string
		cfg     string
		assert  func(*testing.T, []string, error)
	}{
		{
			name:    "type without a schema",
			subType: "fake-type",
			cfg:     `{"bogus":true}`,
			assert: func(t *testing.T, fields []string, err error) {
				require.NoError(t, err)
				require.Empty(t, fields)
			},
		},
		{
			name:    "malformed config",
			subType: "chart",
			cfg:     `{`,
			assert: func(t *testing.T, _ []string, err error) {
				require.ErrorContains(t, err, "error validating chart subscription")
			},
		},
		{
			name:    "no unknown fields",
			subType: "chart",
			cfg:     `{"repoURL":"oci://example.com/chart","semverConstraint":"<1.1.0","discoveryLimit":20}`,
			assert: func(t *testing.T, fields []string, err error) {
				require.NoError(t, err)
				require.Empty(t, fields)
			},
		},
		{
			// Constraints other than the set of allowed fields are enforced by
			// each Subscriber's ValidateSubscription
			name:    "other schema violations are ignored",
			subType: "chart",
			cfg:     `{"repoURL":"","discoveryLimit":1000}`,
			assert: func(t *testing.T, fields []string, err error) {
				require.NoError(t, err)
				require.Empty(t, fields)
			},
		},
		{
			name:    "chart subscription with image constraint field",
			subType: "chart",
			cfg:     `{"repoURL":"oci://example.com/chart","constraint":"<1.1.0"}`,
			assert: func(t *testing.T, fields []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"constraint"}, fields)
			},
		},
		{
			name:    "image subscription with chart constraint field",
			subType: "image",
			cfg:     `{"repoURL":"example.com/image","semverConstraint":"^1"}`,
			assert: func(t *testing.T, fields []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"semverConstraint"}, fields)
			},
		},
		{
			name:    "git subscription with several unknown fields",
			subType: "git",
			cfg:     `{"repoURL":"https://example.com/repo","zeta":1,"constraint":"^1"}`,
			assert: func(t *testing.T, fields []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"constraint", "zeta"}, fields)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			fields, err := UnknownFields(testCase.subType, []byte(testCase.cfg))
			testCase.assert(t, fields, err)
		})
	}
}
