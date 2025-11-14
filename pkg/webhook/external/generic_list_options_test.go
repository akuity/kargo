package external

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_buildListOptionsTarget(t *testing.T) {
	tests := []struct {
		name     string
		project  string
		target   kargoapi.GenericWebhookTarget
		expected []client.ListOption
		env      map[string]any
		err      error
	}{
		{
			name:    "no selectors defined",
			project: "demo-project",
			target: kargoapi.GenericWebhookTarget{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
			},
			env: map[string]any{},
			expected: []client.ListOption{
				// only the default selectors should be present
				client.InNamespace("demo-project"),
				client.MatchingLabelsSelector{
					Selector: labels.Everything(),
				},
			},
			err: nil,
		},
		{
			name:    "index selector with invalid expression",
			project: "test-project",
			target: kargoapi.GenericWebhookTarget{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				IndexSelector: kargoapi.IndexSelector{
					MatchIndices: []kargoapi.IndexSelectorRequirement{
						{
							Key:      "env",
							Operator: kargoapi.IndexSelectorRequirementOperatorEqual,
							Value:    "${{ invalid expression }}",
						},
					},
				},
			},
			env:      map[string]any{},
			expected: nil,
			err:      errors.New("failed to evaluate values expression"),
		},
		{
			name:    "index selector with non-string expression result",
			project: "test-project",
			target: kargoapi.GenericWebhookTarget{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				IndexSelector: kargoapi.IndexSelector{
					MatchIndices: []kargoapi.IndexSelectorRequirement{
						{
							Key:      "env",
							Operator: kargoapi.IndexSelectorRequirementOperatorEqual,
							Value:    "${{ 12345 }}",
						},
					},
				},
			},
			env:      map[string]any{},
			expected: nil,
			err:      errors.New("expression result"),
		},
		{
			name:    "index selector with unsupported operator",
			project: "test-project",
			target: kargoapi.GenericWebhookTarget{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				IndexSelector: kargoapi.IndexSelector{
					MatchIndices: []kargoapi.IndexSelectorRequirement{
						{
							Key:      "env",
							Operator: "UnsupportedOperator",
							Value:    "prod",
						},
					},
				},
			},
			env:      map[string]any{},
			expected: nil,
			err:      errors.New("unsupported operator"),
		},
		{
			name:    "index selector success",
			project: "sample-project",
			target: kargoapi.GenericWebhookTarget{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				IndexSelector: kargoapi.IndexSelector{
					MatchIndices: []kargoapi.IndexSelectorRequirement{
						{
							Key:      "region",
							Operator: kargoapi.IndexSelectorRequirementOperatorEqual,
							Value:    "us-west-1",
						},
						{
							Key:      "status",
							Operator: kargoapi.IndexSelectorRequirementOperatorNotEqual,
							Value:    "inactive",
						},
					},
				},
			},
			env: map[string]any{},
			expected: []client.ListOption{
				client.InNamespace("sample-project"),
				client.MatchingFieldsSelector{
					Selector: fields.OneTermEqualSelector("region", "us-west-1"),
				},
				client.MatchingFieldsSelector{
					Selector: fields.OneTermNotEqualSelector("status", "inactive"),
				},
			},
			err: nil,
		},
		{
			name:    "label selector match expressions with invalid operator",
			project: "test-project",
			target: kargoapi.GenericWebhookTarget{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				LabelSelector: metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "env",
							Operator: "InvalidOperator",
							Values:   []string{"prod", "staging"},
						},
					},
				},
			},
			env:      map[string]any{},
			expected: nil,
			err:      errors.New("unsupported LabelSelectorOperator: \"InvalidOperator\""),
		},
		{
			name:    "label selector match expressions with invalid requirement (empty values for In operator)",
			project: "example-project",
			target: kargoapi.GenericWebhookTarget{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				LabelSelector: metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "env",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{},
						},
					},
				},
			},
			env:      map[string]any{},
			expected: nil,
			err:      errors.New("Invalid value: []: for 'in', 'notin' operators, values set can't be empty"),
		},
		{
			name:    "label selector match expressions with invalid expression result",
			project: "example-project",
			target: kargoapi.GenericWebhookTarget{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				LabelSelector: metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "env",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{"${{ undefined() }}"},
						},
					},
				},
			},
			env:      map[string]any{},
			expected: nil,
			err:      errors.New("failed to evaluate expression"),
		},
		{
			name:    "label selector match expressions with non-string result",
			project: "example-project",
			target: kargoapi.GenericWebhookTarget{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				LabelSelector: metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "env",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{"${{ 12345 }}"},
						},
					},
				},
			},
			env:      map[string]any{},
			expected: nil,
			err: errors.New(
				"failed to parse matchExpression values: failed to evaluate value \"${{ 12345 }}\" as string",
			),
		},
		{
			name:    "label selector match labels invalid requirement (empty key)",
			project: "sample-project",
			target: kargoapi.GenericWebhookTarget{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"": ""},
				},
			},
			env:      map[string]any{},
			expected: nil,
			err:      errors.New("key: Invalid value: \"\": name part must be non-empty;"),
		},
		{
			name:    "simple label selector",
			project: "test-project",
			target: kargoapi.GenericWebhookTarget{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "my-app",
						"foo": "bar",
					},
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "env",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{"prod", "staging"},
						},
						{
							Key:      "tier",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{"backend", "frontend"},
						},
					},
				},
			},
			env: map[string]any{},
			expected: []client.ListOption{
				client.InNamespace("test-project"),
				client.MatchingLabelsSelector{
					Selector: labels.NewSelector().Add(
						newLabelRequirement(t, "app", selection.Equals, []string{"my-app"}),
						newLabelRequirement(t, "foo", selection.Equals, []string{"bar"}),
						newLabelRequirement(t, "env", selection.In, []string{"prod", "staging"}),
						newLabelRequirement(t, "tier", selection.In, []string{"backend", "frontend"}),
					),
				},
			},
			err: nil,
		},
		{
			name:    "combined index and label selectors",
			project: "combined-project",
			target: kargoapi.GenericWebhookTarget{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				IndexSelector: kargoapi.IndexSelector{
					MatchIndices: []kargoapi.IndexSelectorRequirement{
						{
							Key:      "region",
							Operator: kargoapi.IndexSelectorRequirementOperatorEqual,
							Value:    "us-east-1",
						},
					},
				},
				LabelSelector: metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "app",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{"combined-app"},
						},
					},
					MatchLabels: map[string]string{
						"env": "staging",
					},
				},
			},
			env: map[string]any{},
			expected: []client.ListOption{
				client.InNamespace("combined-project"),
				client.MatchingFieldsSelector{
					Selector: fields.OneTermEqualSelector("region", "us-east-1"),
				},
				client.MatchingLabelsSelector{
					Selector: labels.NewSelector().Add(
						newLabelRequirement(t, "app", selection.In, []string{"combined-app"}),
						newLabelRequirement(t, "env", selection.Equals, []string{"staging"}),
					),
				},
			},
			err: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildListOptionsForTarget(tt.project, tt.target, tt.env)
			if tt.err != nil {
				require.ErrorContains(t, err, tt.err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, got)
		})
	}
}

func Test_labelOpToSelectionOp(t *testing.T) {
	tests := []struct {
		name     string
		operator metav1.LabelSelectorOperator
		expected selection.Operator
		err      error
	}{
		{
			name:     "in operator",
			operator: metav1.LabelSelectorOpIn,
			expected: selection.In,
			err:      nil,
		},
		{
			name:     "not in operator",
			operator: metav1.LabelSelectorOpNotIn,
			expected: selection.NotIn,
			err:      nil,
		},
		{
			name:     "exists operator",
			operator: metav1.LabelSelectorOpExists,
			expected: selection.Exists,
			err:      nil,
		},
		{
			name:     "does not exist operator",
			operator: metav1.LabelSelectorOpDoesNotExist,
			expected: selection.DoesNotExist,
			err:      nil,
		},
		{
			name:     "greater than operator",
			operator: "GreaterThan",
			expected: "",
			err:      errors.New("unsupported LabelSelectorOperator: \"GreaterThan\""),
		},
		{
			name:     "less than operator",
			operator: "LessThan",
			expected: "",
			err:      errors.New("unsupported LabelSelectorOperator: \"LessThan\""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := labelOpToSelectionOp(tt.operator)
			if tt.err != nil {
				require.ErrorContains(t, err, tt.err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, got)
		})
	}
}

func newLabelRequirement(
	t *testing.T,
	key string,
	operator selection.Operator,
	values []string,
) labels.Requirement {
	req, err := labels.NewRequirement(key, operator, values)
	require.NoError(t, err)
	return *req
}
