package promotion

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	pkgPromotion "github.com/akuity/kargo/pkg/promotion"
)

func TestStep_GetTimeout(t *testing.T) {
	tests := []struct {
		name       string
		step       *Step
		runner     promotion.StepRunner
		assertions func(t *testing.T, result *time.Duration)
	}{
		{
			name: "returns 0 with no retry config",
			step: &Step{
				Retry: nil,
			},
			assertions: func(t *testing.T, result *time.Duration) {
				assert.Equal(t, ptr.To(time.Duration(0)), result)
			},
		},
		{
			name: "returns configured timeout for non-retryable runner",
			step: &Step{
				Retry: &kargoapi.PromotionStepRetry{
					Timeout: &metav1.Duration{
						Duration: time.Duration(5),
					},
				},
			},
			runner: nil,
			assertions: func(t *testing.T, result *time.Duration) {
				assert.Equal(t, ptr.To(time.Duration(5)), result)
			},
		},
		{
			name: "returns configured timeout for retryable runner",
			step: &Step{
				Retry: &kargoapi.PromotionStepRetry{
					Timeout: &metav1.Duration{
						Duration: time.Duration(5),
					},
				},
			},
			runner: pkgPromotion.NewRetryableStepRunner(nil, ptr.To(time.Duration(3)), 0),
			assertions: func(t *testing.T, result *time.Duration) {
				assert.Equal(t, ptr.To(time.Duration(5)), result)
			},
		},
		{
			name: "returns default timeout when retry config returns nil",
			step: &Step{
				Retry: &kargoapi.PromotionStepRetry{},
			},
			runner: pkgPromotion.NewRetryableStepRunner(nil, ptr.To(time.Duration(3)), 0),
			assertions: func(t *testing.T, result *time.Duration) {
				assert.Equal(t, ptr.To(time.Duration(3)), result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.step.GetTimeout(tt.runner)
			tt.assertions(t, result)
		})
	}
}

func TestStep_GetConfig(t *testing.T) {
	testScheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	testClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&kargoapi.Warehouse{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-warehouse",
				Namespace: "fake-project",
			},
			Spec: kargoapi.WarehouseSpec{
				Subscriptions: []kargoapi.RepoSubscription{
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

	testCases := []struct {
		name        string
		promoCtx    Context
		promoState  promotion.State
		rawCfg      []byte
		expectedCfg promotion.Config
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
			rawCfg: []byte(`{
				"project": "${{ ctx.project }}",
				"stage": "${{ ctx.stage }}",
				"promotion": "${{ ctx.promotion }}",
				"actor": "${{ ctx.meta.promotion.actor }}",
				"targetFreight": {
					"name": "${{ ctx.targetFreight.name }}",
					"origin": "${{ ctx.targetFreight.origin.name }}"
				}
			}`),
			expectedCfg: promotion.Config{
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
			name: "test secrets",
			// Test that expressions can reference secrets
			promoCtx: Context{
				Secrets: map[string]map[string]string{
					"secret1": {
						"key1": "value1",
						"key2": "value2",
					},
					"secret2": {
						"key3": "value3",
						"key4": "value4",
					},
				},
			},
			rawCfg: []byte(`{
				"secret1-1": "${{ secrets.secret1.key1 }}",
				"secret1-2": "${{ secrets.secret1.key2 }}",
				"secret2-3": "${{ secrets.secret2.key3 }}",
				"secret2-4": "${{ secrets.secret2.key4 }}"
			}`),
			expectedCfg: promotion.Config{
				"secret1-1": "value1",
				"secret1-2": "value2",
				"secret2-3": "value3",
				"secret2-4": "value4",
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
			rawCfg: []byte(`{
				"strVar": "${{ vars.strVar }}",
				"boolVar": "${{ vars.boolVar }}",
				"numVar": "${{ vars.numVar }}"
			}`),
			expectedCfg: promotion.Config{
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
			rawCfg: []byte(`{
				"strVar": "${{ vars.strVar }}",
				"boolVar": "${{ vars.boolVar }}",
				"numVar": "${{ vars.numVar }}"
			}`),
			expectedCfg: promotion.Config{
				"strVar":  "foo",
				"boolVar": true,
				"numVar":  42,
			},
		},
		{
			name: "test outputs",
			// Test that expressions can reference outputs
			promoState: promotion.State{
				"strOutput":  "foo",
				"boolOutput": true,
				"numOutput":  42,
			},
			rawCfg: []byte(`{
				"strOutput": "${{ outputs.strOutput }}",
				"boolOutput": "${{ outputs.boolOutput }}",
				"numOutput": "${{ outputs.numOutput }}"
			}`),
			expectedCfg: promotion.Config{
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
			rawCfg: []byte(`{
				"origin1": "${{ warehouse('fake-warehouse') }}",
				"origin2": "${{ warehouse(vars.warehouseName) }}",
				"origin3": "${{ warehouse(ctx.targetFreight.origin.name) }}"
			}`),
			expectedCfg: promotion.Config{
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
			// Two ways to use the commitFrom() function:
			// 1. Pass a git repo URL
			// 2. Pass a git repo URL and origin
			rawCfg: []byte(`{
				"commitID1": "${{ commitFrom('https://fake-git-repo').ID }}",
				"commitID2": "${{ commitFrom(vars.repoURL).ID }}",
				"commitID3": "${{ commitFrom('https://fake-git-repo', warehouse('fake-warehouse')).ID }}",
				"commitID4": "${{ commitFrom(vars.repoURL, warehouse(vars.warehouseName)).ID }}"
			}`),
			expectedCfg: promotion.Config{
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
			// Two ways to use the imageFrom() function:
			// 1. Pass an image repo URL
			// 2. Pass an image repo URL and origin
			rawCfg: []byte(`{
				"imageTag1": "${{ imageFrom('fake-image-repo').Tag }}",
				"imageTag2": "${{ imageFrom(vars.repoURL).Tag }}",
				"imageTag3": "${{ imageFrom('fake-image-repo', warehouse('fake-warehouse')).Tag }}",
				"imageTag4": "${{ imageFrom(vars.repoURL, warehouse(vars.warehouseName)).Tag }}"
			}`),
			expectedCfg: promotion.Config{
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
			// Four ways to use the chartFrom() function:
			// 1. Pass an OCI chart repo URL
			// 2. Pass an OCI chart repo URL and origin
			// 3. Pass an HTTPS chart repo URL and chart name
			// 4. Pass an HTTPS chart repo URL, chart name, and origin
			// nolint: lll
			rawCfg: []byte(`{
				"chartVersion1": "${{ chartFrom('oci://fake-oci-repo/fake-chart').Version }}",
				"chartVersion2": "${{ chartFrom(vars.ociRepoURL).Version }}",
				"chartVersion3": "${{ chartFrom('oci://fake-oci-repo/fake-chart', warehouse('fake-warehouse')).Version }}",
				"chartVersion4": "${{ chartFrom(vars.ociRepoURL, warehouse(vars.warehouseName)).Version }}",
				"chartVersion5": "${{ chartFrom('https://fake-chart-repo', 'fake-chart').Version }}",
				"chartVersion6": "${{ chartFrom(vars.repoURL, vars.chartName).Version }}",
				"chartVersion7": "${{ chartFrom('https://fake-chart-repo', 'fake-chart', warehouse('fake-warehouse')).Version }}",
				"chartVersion8": "${{ chartFrom(vars.repoURL, vars.chartName, warehouse(vars.warehouseName)).Version }}"
			}`),
			expectedCfg: promotion.Config{
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
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			promoStep := Step{
				Config: testCase.rawCfg,
			}
			stepCfg, err := promoStep.GetConfig(
				context.Background(),
				testClient,
				nil,
				testCase.promoCtx,
				testCase.promoState,
			)
			require.NoError(t, err)
			require.Equal(t, testCase.expectedCfg, stepCfg)
		})
	}
}

func TestStep_GetVars(t *testing.T) {
	testScheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	testClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&kargoapi.Warehouse{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-warehouse",
				Namespace: "fake-project",
			},
			Spec: kargoapi.WarehouseSpec{
				Subscriptions: []kargoapi.RepoSubscription{
					{
						Git: &kargoapi.GitSubscription{
							RepoURL: "https://fake-git-repo",
						},
					},
				},
			},
		},
	).Build()

	testCases := []struct {
		name         string
		promoCtx     Context
		promoState   promotion.State
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
			},
			promoState: promotion.State{
				"output1": "output-value",
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
			},
			promoState: promotion.State{
				"task::alias": map[string]any{
					"foo": "baz",
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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			vars, err := testCase.step.GetVars(
				context.Background(),
				testClient,
				nil,
				testCase.promoCtx,
				testCase.promoState,
			)

			if testCase.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, testCase.expectedVars, vars)
		})
	}
}

func TestStep_Skip(t *testing.T) {
	tests := []struct {
		name       string
		step       *Step
		ctx        Context
		state      promotion.State
		assertions func(*testing.T, bool, error)
	}{
		{
			name: "no if condition with failures",
			step: &Step{},
			ctx: Context{
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
			name: "no if condition without failures",
			step: &Step{},
			assertions: func(t *testing.T, b bool, err error) {
				assert.False(t, b)
				assert.NoError(t, err)
			},
		},
		{
			name: "if condition uses vars",
			step: &Step{
				If: "${{ vars.foo == 'bar' }}",
			},
			ctx: Context{
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
			step: &Step{
				If: "${{ outputs.foo == 'bar' }}",
			},
			state: promotion.State{
				"foo": "bar",
			},
			assertions: func(t *testing.T, b bool, err error) {
				assert.NoError(t, err)
				assert.False(t, b)
			},
		},
		{
			name: "if condition uses task outputs",
			step: &Step{
				Alias: "task::other-alias",
				If:    "${{ task.outputs.alias.foo == 'bar' }}",
			},
			state: promotion.State{
				"task::alias": map[string]any{
					"foo": "baz",
				},
			},
			assertions: func(t *testing.T, b bool, err error) {
				assert.NoError(t, err)
				assert.True(t, b)
			},
		},
		{
			name: "if condition uses expression function",
			step: &Step{
				If: "${{ commitFrom('https://git.example.com', warehouse('fake-warehouse')).ID == 'foo' }}",
			},
			ctx: Context{
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
			step: &Step{
				If: "invalid condition",
			},
			assertions: func(t *testing.T, b bool, err error) {
				assert.ErrorContains(t, err, "must evaluate to a boolean")
				assert.False(t, b)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.step.Skip(
				context.Background(),
				fake.NewClientBuilder().Build(),
				nil,
				tt.ctx,
				tt.state,
			)
			tt.assertions(t, got, err)
		})
	}
}
