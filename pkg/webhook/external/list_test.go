package external

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_buildListOptionsForTarget(t *testing.T) {
	tests := []struct {
		name                    string
		project                 string
		targetSelectionCriteria kargoapi.GenericWebhookTargetSelectionCriteria
		expected                []client.ListOption
		env                     map[string]any
		err                     error
	}{
		{
			name:    "no selectors defined",
			project: "demo-project",
			targetSelectionCriteria: kargoapi.GenericWebhookTargetSelectionCriteria{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
			},
			env: map[string]any{},
			expected: []client.ListOption{
				// only the default selector should be present
				client.InNamespace("demo-project"),
			},
			err: nil,
		},
		{
			name:    "index selector with invalid expression",
			project: "test-project",
			targetSelectionCriteria: kargoapi.GenericWebhookTargetSelectionCriteria{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				IndexSelector: kargoapi.IndexSelector{
					MatchIndices: []kargoapi.IndexSelectorRequirement{
						{
							Key:      "env",
							Operator: kargoapi.IndexSelectorOperatorEqual,
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
			targetSelectionCriteria: kargoapi.GenericWebhookTargetSelectionCriteria{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				IndexSelector: kargoapi.IndexSelector{
					MatchIndices: []kargoapi.IndexSelectorRequirement{
						{
							Key:      "env",
							Operator: kargoapi.IndexSelectorOperatorEqual,
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
			targetSelectionCriteria: kargoapi.GenericWebhookTargetSelectionCriteria{
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
			targetSelectionCriteria: kargoapi.GenericWebhookTargetSelectionCriteria{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				IndexSelector: kargoapi.IndexSelector{
					MatchIndices: []kargoapi.IndexSelectorRequirement{
						{
							Key:      "region",
							Operator: kargoapi.IndexSelectorOperatorEqual,
							Value:    "us-west-1",
						},
						{
							Key:      "status",
							Operator: kargoapi.IndexSelectorOperatorNotEqual,
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
			targetSelectionCriteria: kargoapi.GenericWebhookTargetSelectionCriteria{
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
			err:      errors.New("\"InvalidOperator\" is not a valid label selector operator"),
		},
		{
			name:    "label selector match expressions with invalid requirement (empty values for In operator)",
			project: "example-project",
			targetSelectionCriteria: kargoapi.GenericWebhookTargetSelectionCriteria{
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
			err:      errors.New("values set can't be empty"),
		},
		{
			name:    "label selector match expressions with invalid expression result",
			project: "example-project",
			targetSelectionCriteria: kargoapi.GenericWebhookTargetSelectionCriteria{
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
			targetSelectionCriteria: kargoapi.GenericWebhookTargetSelectionCriteria{
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
				"expression result %!q(float64=12345) evaluated to float64; not a string",
			),
		},
		{
			name:    "label selector match labels invalid requirement (empty key)",
			project: "sample-project",
			targetSelectionCriteria: kargoapi.GenericWebhookTargetSelectionCriteria{
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
			targetSelectionCriteria: kargoapi.GenericWebhookTargetSelectionCriteria{
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
			targetSelectionCriteria: kargoapi.GenericWebhookTargetSelectionCriteria{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				IndexSelector: kargoapi.IndexSelector{
					MatchIndices: []kargoapi.IndexSelectorRequirement{
						{
							Key:      "region",
							Operator: kargoapi.IndexSelectorOperatorEqual,
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
			g := &genericWebhookReceiver{
				baseWebhookReceiver: &baseWebhookReceiver{
					project: tt.project,
				},
			}
			got, err := g.buildListOptionsForTarget(tt.targetSelectionCriteria, tt.env)
			if tt.err != nil {
				require.ErrorContains(t, err, tt.err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, got)
		})
	}
}

func Test_listTargetObjects(t *testing.T) {
	testProject := "test-project"
	testSecret := "fake-secret"
	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	tests := []struct {
		name                    string
		client                  client.Client
		targetSelectionCriteria kargoapi.GenericWebhookTargetSelectionCriteria
		config                  *kargoapi.GenericWebhookReceiverConfig
		assertions              func(*testing.T, []client.Object, error)
	}{
		{
			name:   "failed to build list options for target",
			client: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			config: nil,
			targetSelectionCriteria: kargoapi.GenericWebhookTargetSelectionCriteria{
				IndexSelector: kargoapi.IndexSelector{
					MatchIndices: []kargoapi.IndexSelectorRequirement{
						{
							Key:      "env",
							Operator: "InvalidOperator",
							Value:    "prod",
						},
					},
				},
			},
			assertions: func(t *testing.T, objects []client.Object, err error) {
				require.Nil(t, objects)
				require.Error(t, err)
				require.ErrorContains(t, err, "failed to build list options")
			},
		},
		{
			name: "eval error for expression derived name",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "some-warehouse",
					},
				},
			).Build(),
			config: nil,
			targetSelectionCriteria: kargoapi.GenericWebhookTargetSelectionCriteria{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				Name: "${{ 1  + 1 }}",
			},
			assertions: func(t *testing.T, objects []client.Object, err error) {
				require.Nil(t, objects)
				require.Error(t, err)
				require.ErrorContains(t, err, "failed to evaluate target name as string")
			},
		},
		{
			name: "eval error for expression derived label key",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "some-warehouse",
						Labels: map[string]string{
							"env": "prod",
						},
					},
				},
			).Build(),
			config: nil,
			targetSelectionCriteria: kargoapi.GenericWebhookTargetSelectionCriteria{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"env": "${{ undefined() }}",
					},
				},
			},
			assertions: func(t *testing.T, objects []client.Object, err error) {
				require.Nil(t, objects)
				require.Error(t, err)
				require.ErrorContains(t, err, "failed to evaluate expression")
			},
		},
		{
			name: "no targets found",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "some-warehouse",
					},
				},
			).Build(),
			config: nil,
			targetSelectionCriteria: kargoapi.GenericWebhookTargetSelectionCriteria{
				Kind: kargoapi.GenericWebhookTargetKindWarehouse,
				Name: "non-existent-warehouse",
			},
			assertions: func(t *testing.T, objects []client.Object, err error) {
				require.NoError(t, err)
				require.Empty(t, objects)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := genericWebhookReceiver{
				baseWebhookReceiver: &baseWebhookReceiver{
					client:     tt.client,
					project:    testProject,
					secretName: testSecret,
				},
				config: tt.config,
			}
			objects, err := g.listTargetObjects(
				t.Context(),
				tt.targetSelectionCriteria,
				nil,
			)
			tt.assertions(t, objects, err)
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
