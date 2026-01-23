package promotion

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestStepEvaluator_BuildExprEnv(t *testing.T) {
	testClient := fake.NewClientBuilder().Build()

	tests := []struct {
		name     string
		promoCtx Context
		opts     []ExprEnvOption
		expected map[string]any
	}{
		{
			name: "basic context environment",
			promoCtx: Context{
				Project:   "test-project",
				Stage:     "test-stage",
				Promotion: "test-promotion",
				Actor:     "test-actor",
				TargetFreightRef: kargoapi.FreightReference{
					Name: "test-freight",
					Origin: kargoapi.FreightOrigin{
						Name: "test-warehouse",
					},
				},
			},
			expected: map[string]any{
				"ctx": map[string]any{
					"project":   "test-project",
					"promotion": "test-promotion",
					"stage":     "test-stage",
					"targetFreight": map[string]any{
						"name": "test-freight",
						"origin": map[string]any{
							"name": "test-warehouse",
						},
					},
					"meta": map[string]any{
						"promotion": map[string]any{
							"actor": "test-actor",
						},
					},
				},
			},
		},
		{
			name: "with variables option",
			promoCtx: Context{
				Project:   "test-project",
				Stage:     "test-stage",
				Promotion: "test-promotion",
				Actor:     "test-actor",
				TargetFreightRef: kargoapi.FreightReference{
					Name: "test-freight",
					Origin: kargoapi.FreightOrigin{
						Name: "test-warehouse",
					},
				},
			},
			opts: []ExprEnvOption{
				ExprEnvWithVars(map[string]any{
					"var1": "value1",
					"var2": 42,
					"var3": true,
				}),
			},
			expected: map[string]any{
				"ctx": map[string]any{
					"project":   "test-project",
					"promotion": "test-promotion",
					"stage":     "test-stage",
					"targetFreight": map[string]any{
						"name": "test-freight",
						"origin": map[string]any{
							"name": "test-warehouse",
						},
					},
					"meta": map[string]any{
						"promotion": map[string]any{
							"actor": "test-actor",
						},
					},
				},
				"vars": map[string]any{
					"var1": "value1",
					"var2": 42,
					"var3": true,
				},
			},
		},
		{
			name: "with step metas option",
			promoCtx: Context{
				Project:   "test-project",
				Stage:     "test-stage",
				Promotion: "test-promotion",
				Actor:     "test-actor",
				TargetFreightRef: kargoapi.FreightReference{
					Name: "test-freight",
					Origin: kargoapi.FreightOrigin{
						Name: "test-warehouse",
					},
				},
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{
					{
						Alias:  "step1",
						Status: kargoapi.PromotionStepStatusSucceeded,
					},
					{
						Alias:  "step2",
						Status: kargoapi.PromotionStepStatusRunning,
					},
				},
			},
			opts: []ExprEnvOption{
				ExprEnvWithStepMetas(Context{
					StepExecutionMetadata: kargoapi.StepExecutionMetadataList{
						{
							Alias:  "step1",
							Status: kargoapi.PromotionStepStatusSucceeded,
						},
						{
							Alias:  "step2",
							Status: kargoapi.PromotionStepStatusRunning,
						},
					},
				}),
			},
			expected: map[string]any{
				"ctx": map[string]any{
					"project":   "test-project",
					"promotion": "test-promotion",
					"stage":     "test-stage",
					"targetFreight": map[string]any{
						"name": "test-freight",
						"origin": map[string]any{
							"name": "test-warehouse",
						},
					},
					"meta": map[string]any{
						"promotion": map[string]any{
							"actor": "test-actor",
						},
					},
				},
				"stepMetas": map[string]any{
					"step1": kargoapi.StepExecutionMetadata{
						Alias:  "step1",
						Status: kargoapi.PromotionStepStatusSucceeded,
					},
					"step2": kargoapi.StepExecutionMetadata{
						Alias:  "step2",
						Status: kargoapi.PromotionStepStatusRunning,
					},
				},
			},
		},
		{
			name: "with outputs option",
			promoCtx: Context{
				Project:   "test-project",
				Stage:     "test-stage",
				Promotion: "test-promotion",
				Actor:     "test-actor",
				TargetFreightRef: kargoapi.FreightReference{
					Name: "test-freight",
					Origin: kargoapi.FreightOrigin{
						Name: "test-warehouse",
					},
				},
			},
			opts: []ExprEnvOption{
				ExprEnvWithOutputs(State{
					"output1": "value1",
					"output2": map[string]any{
						"nested": "value2",
					},
				}),
			},
			expected: map[string]any{
				"ctx": map[string]any{
					"project":   "test-project",
					"promotion": "test-promotion",
					"stage":     "test-stage",
					"targetFreight": map[string]any{
						"name": "test-freight",
						"origin": map[string]any{
							"name": "test-warehouse",
						},
					},
					"meta": map[string]any{
						"promotion": map[string]any{
							"actor": "test-actor",
						},
					},
				},
				"outputs": State{
					"output1": "value1",
					"output2": map[string]any{
						"nested": "value2",
					},
				},
			},
		},
		{
			name: "with task outputs option - has task outputs",
			promoCtx: Context{
				Project:   "test-project",
				Stage:     "test-stage",
				Promotion: "test-promotion",
				Actor:     "test-actor",
				TargetFreightRef: kargoapi.FreightReference{
					Name: "test-freight",
					Origin: kargoapi.FreightOrigin{
						Name: "test-warehouse",
					},
				},
			},
			opts: []ExprEnvOption{
				ExprEnvWithTaskOutputs("task::step1", State{
					"task::step1": map[string]any{
						"output1": "value1",
					},
					"task::step2": map[string]any{
						"output2": "value2",
					},
					"regular-output": "value3",
				}),
			},
			expected: map[string]any{
				"ctx": map[string]any{
					"project":   "test-project",
					"promotion": "test-promotion",
					"stage":     "test-stage",
					"targetFreight": map[string]any{
						"name": "test-freight",
						"origin": map[string]any{
							"name": "test-warehouse",
						},
					},
					"meta": map[string]any{
						"promotion": map[string]any{
							"actor": "test-actor",
						},
					},
				},
				"task": map[string]any{
					"outputs": State{
						"step1": map[string]any{
							"output1": "value1",
						},
						"step2": map[string]any{
							"output2": "value2",
						},
					},
				},
			},
		},
		{
			name: "with task outputs option - no task outputs",
			promoCtx: Context{
				Project:   "test-project",
				Stage:     "test-stage",
				Promotion: "test-promotion",
				Actor:     "test-actor",
				TargetFreightRef: kargoapi.FreightReference{
					Name: "test-freight",
					Origin: kargoapi.FreightOrigin{
						Name: "test-warehouse",
					},
				},
			},
			opts: []ExprEnvOption{
				ExprEnvWithTaskOutputs("regular-step", State{
					"regular-output": "value1",
				}),
			},
			expected: map[string]any{
				"ctx": map[string]any{
					"project":   "test-project",
					"promotion": "test-promotion",
					"stage":     "test-stage",
					"targetFreight": map[string]any{
						"name": "test-freight",
						"origin": map[string]any{
							"name": "test-warehouse",
						},
					},
					"meta": map[string]any{
						"promotion": map[string]any{
							"actor": "test-actor",
						},
					},
				},
			},
		},
		{
			name: "with multiple options",
			promoCtx: Context{
				Project:   "test-project",
				Stage:     "test-stage",
				Promotion: "test-promotion",
				Actor:     "test-actor",
				TargetFreightRef: kargoapi.FreightReference{
					Name: "test-freight",
					Origin: kargoapi.FreightOrigin{
						Name: "test-warehouse",
					},
				},
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{
					{
						Alias:  "prev-step",
						Status: kargoapi.PromotionStepStatusSucceeded,
					},
				},
			},
			opts: []ExprEnvOption{
				ExprEnvWithVars(map[string]any{
					"var1": "value1",
				}),
				ExprEnvWithStepMetas(Context{
					StepExecutionMetadata: kargoapi.StepExecutionMetadataList{
						{
							Alias:  "prev-step",
							Status: kargoapi.PromotionStepStatusSucceeded,
						},
					},
				}),
				ExprEnvWithOutputs(State{
					"output1": "value1",
				}),
				ExprEnvWithTaskOutputs("task::step1", State{
					"task::step1": map[string]any{
						"taskOutput": "taskValue",
					},
				}),
			},
			expected: map[string]any{
				"ctx": map[string]any{
					"project":   "test-project",
					"promotion": "test-promotion",
					"stage":     "test-stage",
					"targetFreight": map[string]any{
						"name": "test-freight",
						"origin": map[string]any{
							"name": "test-warehouse",
						},
					},
					"meta": map[string]any{
						"promotion": map[string]any{
							"actor": "test-actor",
						},
					},
				},
				"vars": map[string]any{
					"var1": "value1",
				},
				"stepMetas": map[string]any{
					"prev-step": kargoapi.StepExecutionMetadata{
						Alias:  "prev-step",
						Status: kargoapi.PromotionStepStatusSucceeded,
					},
				},
				"outputs": State{
					"output1": "value1",
				},
				"task": map[string]any{
					"outputs": State{
						"step1": map[string]any{
							"taskOutput": "taskValue",
						},
					},
				},
			},
		},
		{
			name: "empty context",
			promoCtx: Context{
				TargetFreightRef: kargoapi.FreightReference{},
			},
			expected: map[string]any{
				"ctx": map[string]any{
					"project":   "",
					"promotion": "",
					"stage":     "",
					"targetFreight": map[string]any{
						"name": "",
						"origin": map[string]any{
							"name": "",
						},
					},
					"meta": map[string]any{
						"promotion": map[string]any{
							"actor": "",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewStepEvaluator(testClient, nil)
			result := evaluator.BuildExprEnv(tt.promoCtx, tt.opts...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStepEvaluator_Vars(t *testing.T) {
	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	testClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&kargoapi.Warehouse{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-warehouse",
				Namespace: "fake-project",
			},
			Spec: kargoapi.WarehouseSpec{
				InternalSubscriptions: []kargoapi.RepoSubscription{
					{
						Git: &kargoapi.GitSubscription{
							RepoURL: "https://fake-git-repo",
						},
					},
				},
			},
		},
	).Build()

	tests := []struct {
		name         string
		promoCtx     Context
		step         Step
		expectedVars map[string]any
		expectErr    bool
	}{
		{
			name: "global vars with literal values",
			promoCtx: Context{
				Project: "fake-project",
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "str",
						Value: "foo",
					},
					{
						Name:  "bool",
						Value: "true",
					},
					{
						Name:  "num",
						Value: "42",
					},
				},
			},
			step: Step{},
			expectedVars: map[string]any{
				"str":  "foo",
				"bool": "true",
				"num":  "42",
			},
		},
		{
			name: "global vars with expressions",
			promoCtx: Context{
				Project: "fake-project",
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "str",
						Value: "${{ 'f' + 'o' + 'o' }}",
					},
					{
						Name:  "bool",
						Value: "${{ 1 == 1 }}",
					},
					{
						Name:  "num",
						Value: "${{ 40 + 2 }}",
					},
				},
			},
			step: Step{},
			expectedVars: map[string]any{
				"str":  "foo",
				"bool": true,
				"num":  float64(42),
			},
		},
		{
			name: "step vars with literal values",
			promoCtx: Context{
				Project: "fake-project",
			},
			step: Step{
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "str",
						Value: "foo",
					},
					{
						Name:  "bool",
						Value: "true",
					},
					{
						Name:  "num",
						Value: "42",
					},
				},
			},
			expectedVars: map[string]any{
				"str":  "foo",
				"bool": "true",
				"num":  "42",
			},
		},
		{
			name: "step vars with expressions",
			promoCtx: Context{
				Project: "fake-project",
			},
			step: Step{
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "str",
						Value: "${{ 'f' + 'o' + 'o' }}",
					},
					{
						Name:  "bool",
						Value: "${{ 1 == 1 }}",
					},
					{
						Name:  "num",
						Value: "${{ 40 + 2 }}",
					},
				},
			},
			expectedVars: map[string]any{
				"str":  "foo",
				"bool": true,
				"num":  float64(42),
			},
		},
		{
			name: "step vars referencing global vars",
			promoCtx: Context{
				Project: "fake-project",
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "global",
						Value: "global-value",
					},
				},
			},
			step: Step{
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "local",
						Value: "${{ vars.global + '-suffix' }}",
					},
				},
			},
			expectedVars: map[string]any{
				"global": "global-value",
				"local":  "global-value-suffix",
			},
		},
		{
			name: "step vars referencing other step vars",
			promoCtx: Context{
				Project: "fake-project",
			},
			step: Step{
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "v1",
						Value: "value1",
					},
					{
						Name:  "v2",
						Value: "${{ vars.v1 + '-suffix' }}",
					},
				},
			},
			expectedVars: map[string]any{
				"v1": "value1",
				"v2": "value1-suffix",
			},
		},
		{
			name: "step vars referencing outputs",
			promoCtx: Context{
				Project: "fake-project",
				State: State{
					"output1": "output-value",
				},
			},
			step: Step{
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "fromOutput",
						Value: "${{ outputs.output1 + '-suffix' }}",
					},
				},
			},
			expectedVars: map[string]any{
				"fromOutput": "output-value-suffix",
			},
		},
		{
			name: "step vars referencing task outputs",
			promoCtx: Context{
				Project: "fake-project",
				State: State{
					"task::alias": map[string]any{
						"foo": "baz",
					},
				},
			},
			step: Step{
				Alias: "task::other-alias",
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "fromTask",
						Value: "${{ task.outputs.alias.foo + '-suffix' }}",
					},
				},
			},
			expectedVars: map[string]any{
				"fromTask": "baz-suffix",
			},
		},
		{
			name: "invalid expression in global var",
			promoCtx: Context{
				Project: "fake-project",
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "invalid",
						Value: "${{ invalid.expression }}",
					},
				},
			},
			step:      Step{},
			expectErr: true,
		},
		{
			name: "invalid expression in step var",
			promoCtx: Context{
				Project: "fake-project",
			},
			step: Step{
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "invalid",
						Value: "${{ invalid.expression }}",
					},
				},
			},
			expectErr: true,
		},
		{
			name: "warehouse function in vars",
			promoCtx: Context{
				Project: "fake-project",
				FreightRequests: []kargoapi.FreightRequest{{
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "fake-warehouse",
					},
				}},
			},
			step: Step{
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "wh",
						Value: "${{ warehouse('fake-warehouse') }}",
					},
				},
			},
			expectedVars: map[string]any{
				"wh": map[string]any{
					"kind": string(kargoapi.FreightOriginKindWarehouse),
					"name": "fake-warehouse",
				},
			},
		},
		{
			name: "context properties in vars",
			promoCtx: Context{
				Project:   "fake-project",
				Stage:     "fake-stage",
				Promotion: "fake-promotion",
				Actor:     "fake-creator",
				TargetFreightRef: kargoapi.FreightReference{
					Name: "fake-freight",
					Origin: kargoapi.FreightOrigin{
						Name: "fake-warehouse",
					},
				},
			},
			step: Step{
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "proj",
						Value: "${{ ctx.project }}",
					},
					{
						Name:  "stage",
						Value: "${{ ctx.stage }}",
					},
					{
						Name:  "promo",
						Value: "${{ ctx.promotion }}",
					},
					{
						Name:  "actor",
						Value: "${{ ctx.meta.promotion.actor }}",
					},
					{
						Name:  "targetFreight",
						Value: "${{ ctx.targetFreight.origin.name }}",
					},
				},
			},
			expectedVars: map[string]any{
				"proj":          "fake-project",
				"stage":         "fake-stage",
				"promo":         "fake-promotion",
				"actor":         "fake-creator",
				"targetFreight": "fake-warehouse",
			},
		},
		{
			name: "freight functions in vars",
			promoCtx: Context{
				Project: "fake-project",
				FreightRequests: []kargoapi.FreightRequest{{
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "fake-warehouse",
					},
					Sources: kargoapi.FreightSources{
						Direct: true,
					},
				}},
				Freight: kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						"Warehouse/fake-warehouse": {
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "fake-warehouse",
							},
							Commits: []kargoapi.GitCommit{{
								RepoURL: "https://fake-git-repo",
								ID:      "fake-commit-id",
							}},
						},
					},
				},
			},
			step: Step{
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "commit",
						Value: "${{ commitFrom('https://fake-git-repo').ID }}",
					},
				},
			},
			expectedVars: map[string]any{
				"commit": "fake-commit-id",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewStepEvaluator(testClient, nil)
			vars, err := evaluator.Vars(
				context.Background(),
				tt.promoCtx,
				tt.step,
			)

			if tt.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectedVars, vars)
		})
	}
}

func TestStepEvaluator_ShouldSkip(t *testing.T) {
	tests := []struct {
		name       string
		step       Step
		promoCtx   Context
		assertions func(*testing.T, bool, error)
	}{
		{
			name: "no if condition with failures",
			step: Step{},
			promoCtx: Context{
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{{
					Status: kargoapi.PromotionStepStatusFailed,
				}},
			},
			assertions: func(t *testing.T, b bool, err error) {
				assert.True(t, b)
				assert.NoError(t, err)
			},
		},
		{
			name:     "no if condition without failures",
			step:     Step{},
			promoCtx: Context{},
			assertions: func(t *testing.T, b bool, err error) {
				assert.False(t, b)
				assert.NoError(t, err)
			},
		},
		{
			name: "if condition uses vars",
			step: Step{
				If: "${{ vars.foo == 'bar' }}",
			},
			promoCtx: Context{
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "foo",
						Value: "bar",
					},
				},
			},
			assertions: func(t *testing.T, b bool, err error) {
				assert.NoError(t, err)
				assert.False(t, b)
			},
		},
		{
			name: "if condition uses outputs",
			promoCtx: Context{
				State: State{
					"foo": "bar",
				},
			},
			step: Step{
				If: "${{ outputs.foo == 'bar' }}",
			},
			assertions: func(t *testing.T, b bool, err error) {
				assert.NoError(t, err)
				assert.False(t, b)
			},
		},
		{
			name: "if condition uses task outputs",
			promoCtx: Context{
				State: State{
					"task::alias": map[string]any{
						"foo": "baz",
					},
				},
			},
			step: Step{
				Alias: "task::other-alias",
				If:    "${{ task.outputs.alias.foo == 'bar' }}",
			},
			assertions: func(t *testing.T, b bool, err error) {
				assert.NoError(t, err)
				assert.True(t, b)
			},
		},
		{
			name: "if condition uses expression function",
			step: Step{
				If: "${{ commitFrom('https://git.example.com', warehouse('fake-warehouse')).ID == 'foo' }}",
			},
			promoCtx: Context{
				Project: "fake-project",
				Freight: kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						"Warehouse/fake-warehouse": {
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "fake-warehouse",
							},
							Commits: []kargoapi.GitCommit{{
								RepoURL: "https://git.example.com",
								ID:      "foo",
							}},
						},
					},
				},
			},
			assertions: func(t *testing.T, b bool, err error) {
				assert.NoError(t, err)
				assert.False(t, b)
			},
		},
		{
			name: "if condition does not evaluate to a boolean",
			step: Step{
				If: "invalid condition",
			},
			promoCtx: Context{},
			assertions: func(t *testing.T, b bool, err error) {
				assert.ErrorContains(t, err, "must evaluate to a boolean")
				assert.False(t, b)
			},
		},
		{
			name: "if condition evaluates to false - step should be skipped",
			step: Step{
				If: "${{ false }}",
			},
			promoCtx: Context{},
			assertions: func(t *testing.T, b bool, err error) {
				assert.NoError(t, err)
				assert.True(t, b) // Should skip when condition is false
			},
		},
		{
			name: "if condition evaluates to true - step should not be skipped",
			step: Step{
				If: "${{ true }}",
			},
			promoCtx: Context{},
			assertions: func(t *testing.T, b bool, err error) {
				assert.NoError(t, err)
				assert.False(t, b) // Should not skip when condition is true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewStepEvaluator(
				fake.NewClientBuilder().Build(),
				nil,
			)
			got, err := evaluator.ShouldSkip(
				context.Background(),
				tt.promoCtx,
				tt.step,
			)
			tt.assertions(t, got, err)
		})
	}
}

func TestStepEvaluator_Config(t *testing.T) {
	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	testClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&kargoapi.Warehouse{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-warehouse",
				Namespace: "fake-project",
			},
			Spec: kargoapi.WarehouseSpec{
				InternalSubscriptions: []kargoapi.RepoSubscription{
					{
						Git: &kargoapi.GitSubscription{
							RepoURL: "https://fake-git-repo",
						},
					},
					{
						Image: &kargoapi.ImageSubscription{
							RepoURL: "fake-image-repo",
						},
					},
					{
						Chart: &kargoapi.ChartSubscription{
							RepoURL: "https://fake-chart-repo",
							Name:    "fake-chart",
						},
					},
					{
						Chart: &kargoapi.ChartSubscription{
							RepoURL: "oci://fake-oci-repo/fake-chart",
						},
					},
				},
			},
		},
	).Build()

	tests := []struct {
		name        string
		promoCtx    Context
		step        Step
		expectedCfg Config
	}{
		{
			name: "test context",
			// Test that expressions can reference promotion context
			promoCtx: Context{
				Project:   "fake-project",
				Stage:     "fake-stage",
				Promotion: "fake-promotion",
				Actor:     "fake-creator",
				TargetFreightRef: kargoapi.FreightReference{
					Name: "fake-freight",
					Origin: kargoapi.FreightOrigin{
						Name: "fake-warehouse",
					},
				},
			},
			step: Step{
				Config: []byte(`{
					"project": "${{ ctx.project }}",
					"stage": "${{ ctx.stage }}",
					"promotion": "${{ ctx.promotion }}",
					"actor": "${{ ctx.meta.promotion.actor }}",
					"targetFreight": {
						"name": "${{ ctx.targetFreight.name }}",
						"origin": "${{ ctx.targetFreight.origin.name }}"
					}
				}`),
			},
			expectedCfg: Config{
				"project":   "fake-project",
				"stage":     "fake-stage",
				"promotion": "fake-promotion",
				"actor":     "fake-creator",
				"targetFreight": map[string]any{
					"name":   "fake-freight",
					"origin": "fake-warehouse",
				},
			},
		},
		{
			name: "test vars with literal values",
			// Test that vars can be assigned literal values
			promoCtx: Context{
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "strVar",
						Value: "foo",
					},
					{
						Name:  "boolVar",
						Value: "true",
					},
					{
						Name:  "numVar",
						Value: "42",
					},
				},
			},
			step: Step{
				Config: []byte(`{
					"strVar": "${{ vars.strVar }}",
					"boolVar": "${{ vars.boolVar }}",
					"numVar": "${{ vars.numVar }}"
				}`),
			},
			expectedCfg: Config{
				"strVar":  "foo",
				"boolVar": true,
				"numVar":  42,
			},
		},
		{
			name: "test vars with expressions",
			// Test using expressions to define the value of vars
			promoCtx: Context{
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "strVar",
						Value: "${{ 'f' + 'o' + 'o' }}",
					},
					{
						Name:  "boolVar",
						Value: "${{ vars.strVar == 'foo' }}",
					},
					{
						Name:  "numVar",
						Value: "${{ 40 + 2 }}",
					},
				},
			},
			step: Step{
				Config: []byte(`{
					"strVar": "${{ vars.strVar }}",
					"boolVar": "${{ vars.boolVar }}",
					"numVar": "${{ vars.numVar }}"
				}`),
			},
			expectedCfg: Config{
				"strVar":  "foo",
				"boolVar": true,
				"numVar":  42,
			},
		},
		{
			name: "test outputs",
			// Test that expressions can reference outputs
			promoCtx: Context{
				State: State{
					"strOutput":  "foo",
					"boolOutput": true,
					"numOutput":  42,
				},
			},
			step: Step{
				Config: []byte(`{
					"strOutput": "${{ outputs.strOutput }}",
					"boolOutput": "${{ outputs.boolOutput }}",
					"numOutput": "${{ outputs.numOutput }}"
				}`),
			},
			expectedCfg: Config{
				"strOutput":  "foo",
				"boolOutput": true,
				"numOutput":  42,
			},
		},
		{
			name: "test warehouse function",
			// Test that the warehouse() function can be used to reference freight
			// origins
			promoCtx: Context{
				TargetFreightRef: kargoapi.FreightReference{
					Name: "fake-freight",
					Origin: kargoapi.FreightOrigin{
						Name: "fake-origin-warehouse",
					},
				},
				Vars: []kargoapi.ExpressionVariable{{
					Name:  "warehouseName",
					Value: "fake-warehouse",
				}},
			},
			step: Step{
				Config: []byte(`{
					"origin1": "${{ warehouse('fake-warehouse') }}",
					"origin2": "${{ warehouse(vars.warehouseName) }}",
					"origin3": "${{ warehouse(ctx.targetFreight.origin.name) }}"
				}`),
			},
			expectedCfg: Config{
				"origin1": map[string]any{
					"kind": "Warehouse",
					"name": "fake-warehouse",
				},
				"origin2": map[string]any{
					"kind": "Warehouse",
					"name": "fake-warehouse",
				},
				"origin3": map[string]any{
					"kind": "Warehouse",
					"name": "fake-origin-warehouse",
				},
			},
		},
		{
			name: "test commitFrom function",
			// Test different ways to use the commitFrom() function
			promoCtx: Context{
				Project: "fake-project",
				FreightRequests: []kargoapi.FreightRequest{{
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "fake-warehouse",
					},
					Sources: kargoapi.FreightSources{
						Direct: true,
					},
				}},
				Freight: kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						"Warehouse/fake-warehouse": {
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "fake-warehouse",
							},
							Commits: []kargoapi.GitCommit{{
								RepoURL: "https://fake-git-repo",
								ID:      "fake-commit-id",
							}},
						},
					},
				},
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "warehouseName",
						Value: "fake-warehouse",
					},
					{
						Name:  "repoURL",
						Value: "https://fake-git-repo",
					},
				},
			},
			step: Step{
				// Two ways to use the commitFrom() function:
				// 1. Pass a git repo URL
				// 2. Pass a git repo URL and origin
				Config: []byte(`{
					"commitID1": "${{ commitFrom('https://fake-git-repo').ID }}",
					"commitID2": "${{ commitFrom(vars.repoURL).ID }}",
					"commitID3": "${{ commitFrom('https://fake-git-repo', warehouse('fake-warehouse')).ID }}",
					"commitID4": "${{ commitFrom(vars.repoURL, warehouse(vars.warehouseName)).ID }}"
				}`),
			},
			expectedCfg: Config{
				"commitID1": "fake-commit-id",
				"commitID2": "fake-commit-id",
				"commitID3": "fake-commit-id",
				"commitID4": "fake-commit-id",
			},
		},
		{
			name: "test imageFrom function",
			// Test different ways to use the imageFrom() function
			promoCtx: Context{
				Project: "fake-project",
				FreightRequests: []kargoapi.FreightRequest{{
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "fake-warehouse",
					},
					Sources: kargoapi.FreightSources{
						Direct: true,
					},
				}},
				Freight: kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						"Warehouse/fake-warehouse": {
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "fake-warehouse",
							},
							Images: []kargoapi.Image{{
								RepoURL: "fake-image-repo",
								Tag:     "fake-image-tag",
							}},
						},
					},
				},
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "warehouseName",
						Value: "fake-warehouse",
					},
					{
						Name:  "repoURL",
						Value: "fake-image-repo",
					},
				},
			},
			step: Step{
				// Two ways to use the imageFrom() function:
				// 1. Pass an image repo URL
				// 2. Pass an image repo URL and origin
				Config: []byte(`{
					"imageTag1": "${{ imageFrom('fake-image-repo').Tag }}",
					"imageTag2": "${{ imageFrom(vars.repoURL).Tag }}",
					"imageTag3": "${{ imageFrom('fake-image-repo', warehouse('fake-warehouse')).Tag }}",
					"imageTag4": "${{ imageFrom(vars.repoURL, warehouse(vars.warehouseName)).Tag }}"
				}`),
			},
			expectedCfg: Config{
				"imageTag1": "fake-image-tag",
				"imageTag2": "fake-image-tag",
				"imageTag3": "fake-image-tag",
				"imageTag4": "fake-image-tag",
			},
		},
		{
			name: "test chartFrom function",
			// Test different ways to use the chartFrom() function
			promoCtx: Context{
				Project: "fake-project",
				FreightRequests: []kargoapi.FreightRequest{{
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "fake-warehouse",
					},
					Sources: kargoapi.FreightSources{
						Direct: true,
					},
				}},
				Freight: kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						"Warehouse/fake-warehouse": {
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "fake-warehouse",
							},
							Charts: []kargoapi.Chart{
								{
									RepoURL: "https://fake-chart-repo",
									Name:    "fake-chart",
									Version: "fake-chart-version",
								},
								{
									RepoURL: "oci://fake-oci-repo/fake-chart",
									Version: "fake-oci-chart-version",
								},
							},
						},
					},
				},
				Vars: []kargoapi.ExpressionVariable{
					{
						Name:  "warehouseName",
						Value: "fake-warehouse",
					},
					{
						Name:  "repoURL",
						Value: "https://fake-chart-repo",
					},
					{
						Name:  "chartName",
						Value: "fake-chart",
					},
					{
						Name:  "ociRepoURL",
						Value: "oci://fake-oci-repo/fake-chart",
					},
				},
			},
			step: Step{
				// Four ways to use the chartFrom() function:
				// 1. Pass an OCI chart repo URL
				// 2. Pass an OCI chart repo URL and origin
				// 3. Pass an HTTPS chart repo URL and chart name
				// 4. Pass an HTTPS chart repo URL, chart name, and origin
				// nolint: lll
				Config: []byte(`{
					"chartVersion1": "${{ chartFrom('oci://fake-oci-repo/fake-chart').Version }}",
					"chartVersion2": "${{ chartFrom(vars.ociRepoURL).Version }}",
					"chartVersion3": "${{ chartFrom('oci://fake-oci-repo/fake-chart', warehouse('fake-warehouse')).Version }}",
					"chartVersion4": "${{ chartFrom(vars.ociRepoURL, warehouse(vars.warehouseName)).Version }}",
					"chartVersion5": "${{ chartFrom('https://fake-chart-repo', 'fake-chart').Version }}",
					"chartVersion6": "${{ chartFrom(vars.repoURL, vars.chartName).Version }}",
					"chartVersion7": "${{ chartFrom('https://fake-chart-repo', 'fake-chart', warehouse('fake-warehouse')).Version }}",
					"chartVersion8": "${{ chartFrom(vars.repoURL, vars.chartName, warehouse(vars.warehouseName)).Version }}"
				}`),
			},
			expectedCfg: Config{
				"chartVersion1": "fake-oci-chart-version",
				"chartVersion2": "fake-oci-chart-version",
				"chartVersion3": "fake-oci-chart-version",
				"chartVersion4": "fake-oci-chart-version",
				"chartVersion5": "fake-chart-version",
				"chartVersion6": "fake-chart-version",
				"chartVersion7": "fake-chart-version",
				"chartVersion8": "fake-chart-version",
			},
		},
		{
			name:     "test success function",
			promoCtx: Context{},
			step: Step{
				Config: []byte(`{"wasSuccessful": "${{ success() }}"}`),
			},
			expectedCfg: Config{"wasSuccessful": true},
		},
		{
			name:     "test failure function",
			promoCtx: Context{},
			step: Step{
				Config: []byte(`{"wasFailure": "${{ failure() }}"}`),
			},
			expectedCfg: Config{"wasFailure": false},
		},
		{
			name:     "test always function",
			promoCtx: Context{},
			step: Step{
				Config: []byte(`{"alwaysTrue": "${{ always() }}"}`),
			},
			expectedCfg: Config{"alwaysTrue": true},
		},
		{
			name: "test status function",
			promoCtx: Context{
				StepExecutionMetadata: kargoapi.StepExecutionMetadataList{
					{
						Alias:  "test-step",
						Status: kargoapi.PromotionStepStatusFailed,
					},
				},
			},
			step: Step{
				Config: []byte(`{"status": "${{ status(\"test-step\") }}"}`),
			},
			expectedCfg: Config{"status": "Failed"},
		},
		{
			name:     "test nil config",
			promoCtx: Context{},
			step: Step{
				Config: nil,
			},
			expectedCfg: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewStepEvaluator(testClient, nil)
			stepCfg, err := evaluator.Config(
				context.Background(),
				tt.promoCtx,
				tt.step,
			)
			require.NoError(t, err)
			require.Equal(t, tt.expectedCfg, stepCfg)
		})
	}
}

func TestStepEvaluator_BuildStepContext(t *testing.T) {
	testClient := fake.NewClientBuilder().Build()

	tests := []struct {
		name       string
		promoCtx   Context
		step       Step
		assertions func(*testing.T, *StepContext, error)
	}{
		{
			name: "builds step context with all fields",
			promoCtx: Context{
				UIBaseURL: "https://kargo.example.com",
				WorkDir:   "/tmp/workdir",
				Project:   "test-project",
				Stage:     "test-stage",
				Promotion: "test-promotion",
				Actor:     "test-actor",
				State: State{
					"key": "value",
				},
				FreightRequests: []kargoapi.FreightRequest{
					{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: "test-warehouse",
						},
					},
				},
				Freight: kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						"test": {
							Name: "test-freight",
						},
					},
				},
				TargetFreightRef: kargoapi.FreightReference{
					Name: "target-freight",
					Origin: kargoapi.FreightOrigin{
						Name: "target-warehouse",
					},
				},
			},
			step: Step{
				Alias:  "test-step",
				Config: []byte(`{"key": "value"}`),
			},
			assertions: func(t *testing.T, stepCtx *StepContext, err error) {
				require.NoError(t, err)
				require.NotNil(t, stepCtx)

				assert.Equal(t, "https://kargo.example.com", stepCtx.UIBaseURL)
				assert.Equal(t, "/tmp/workdir", stepCtx.WorkDir)
				assert.Equal(t, "test-step", stepCtx.Alias)
				assert.Equal(t, "test-project", stepCtx.Project)
				assert.Equal(t, "test-stage", stepCtx.Stage)
				assert.Equal(t, "test-promotion", stepCtx.Promotion)
				assert.Equal(t, "test-actor", stepCtx.PromotionActor)
				assert.Equal(t, "target-freight", stepCtx.TargetFreightRef.Name)
				assert.Equal(t, "target-warehouse", stepCtx.TargetFreightRef.Origin.Name)

				// Verify state is deep copied
				assert.Equal(t, State{"key": "value"}, stepCtx.SharedState)

				// Verify freight requests are deep copied
				require.Len(t, stepCtx.FreightRequests, 1)
				assert.Equal(t, kargoapi.FreightOriginKindWarehouse, stepCtx.FreightRequests[0].Origin.Kind)
				assert.Equal(t, "test-warehouse", stepCtx.FreightRequests[0].Origin.Name)

				// Verify config is evaluated
				assert.Equal(t, Config{"key": "value"}, stepCtx.Config)
			},
		},
		{
			name: "handles nil freight requests",
			promoCtx: Context{
				WorkDir:         "/tmp/workdir",
				FreightRequests: nil,
			},
			step: Step{
				Alias: "test-step",
			},
			assertions: func(t *testing.T, stepCtx *StepContext, err error) {
				require.NoError(t, err)
				require.NotNil(t, stepCtx)
				assert.Nil(t, stepCtx.FreightRequests)
			},
		},
		{
			name: "handles step config evaluation error",
			promoCtx: Context{
				WorkDir: "/tmp/workdir",
			},
			step: Step{
				Alias:  "test-step",
				Config: []byte(`{"key": "${{ invalid.expression }}"}`),
			},
			assertions: func(t *testing.T, stepCtx *StepContext, err error) {
				require.ErrorContains(t, err, "failed to get step config")
				assert.Nil(t, stepCtx)
			},
		},
		{
			name: "handles nil step config",
			promoCtx: Context{
				WorkDir: "/tmp/workdir",
			},
			step: Step{
				Alias:  "test-step",
				Config: nil,
			},
			assertions: func(t *testing.T, stepCtx *StepContext, err error) {
				require.NoError(t, err)
				require.NotNil(t, stepCtx)
				assert.Nil(t, stepCtx.Config)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewStepEvaluator(testClient, nil)
			stepCtx, err := evaluator.BuildStepContext(
				context.Background(),
				tt.promoCtx,
				tt.step,
			)
			tt.assertions(t, stepCtx, err)
		})
	}
}
