package builtin

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_buildArgoCDAppLabelSelector(t *testing.T) {
	testCases := []struct {
		name      string
		selector  *builtin.ArgoCDAppSelector
		expectErr bool
	}{
		{
			name:      "empty selector",
			selector:  &builtin.ArgoCDAppSelector{},
			expectErr: true,
		},
		{
			name: "valid matchLabels",
			selector: &builtin.ArgoCDAppSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
			expectErr: false,
		},
		{
			name: "valid matchExpressions",
			selector: &builtin.ArgoCDAppSelector{
				MatchExpressions: []builtin.MatchExpression{
					{Key: "app", Operator: builtin.In, Values: []string{"a", "b"}},
				},
			},
			expectErr: false,
		},
		{
			name: "invalid operator",
			selector: &builtin.ArgoCDAppSelector{
				MatchExpressions: []builtin.MatchExpression{
					{Key: "app", Operator: "Invalid"},
				},
			},
			expectErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sel, err := buildArgoCDAppLabelSelector(tc.selector)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, sel)
			}
		})
	}
}
