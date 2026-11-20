package builtin

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/x/promotion/stepconfig"
)

func Test_BuildArgoCDAppLabelSelector(t *testing.T) {
	testCases := []struct {
		name      string
		selector  *stepconfig.ArgoCDAppSelector
		expectErr bool
	}{
		{
			name:      "empty selector",
			selector:  &stepconfig.ArgoCDAppSelector{},
			expectErr: true,
		},
		{
			name: "valid matchLabels",
			selector: &stepconfig.ArgoCDAppSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
			expectErr: false,
		},
		{
			name: "valid matchExpressions",
			selector: &stepconfig.ArgoCDAppSelector{
				MatchExpressions: []stepconfig.MatchExpression{
					{Key: "app", Operator: stepconfig.In, Values: []string{"a", "b"}},
				},
			},
			expectErr: false,
		},
		{
			name: "invalid operator",
			selector: &stepconfig.ArgoCDAppSelector{
				MatchExpressions: []stepconfig.MatchExpression{
					{Key: "app", Operator: "Invalid"},
				},
			},
			expectErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sel, err := BuildArgoCDAppLabelSelector(tc.selector)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, sel)
			}
		})
	}
}
