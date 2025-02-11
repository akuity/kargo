package directives

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

type mockRetryableRunner struct {
	defaultTimeout        *time.Duration
	defaultErrorThreshold uint32
}

func (m mockRetryableRunner) DefaultTimeout() *time.Duration {
	return m.defaultTimeout
}

func (m mockRetryableRunner) DefaultErrorThreshold() uint32 {
	return m.defaultErrorThreshold
}

func TestPromotionStep_GetTimeout(t *testing.T) {
	tests := []struct {
		name       string
		step       *PromotionStep
		runner     any
		assertions func(t *testing.T, result *time.Duration)
	}{
		{
			name: "returns 0 with no retry config",
			step: &PromotionStep{
				Retry: nil,
			},
			assertions: func(t *testing.T, result *time.Duration) {
				assert.Equal(t, ptr.To(time.Duration(0)), result)
			},
		},
		{
			name: "returns configured timeout for non-retryable runner",
			step: &PromotionStep{
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
			step: &PromotionStep{
				Retry: &kargoapi.PromotionStepRetry{
					Timeout: &metav1.Duration{
						Duration: time.Duration(5),
					},
				},
			},
			runner: mockRetryableRunner{defaultTimeout: ptr.To(time.Duration(3))},
			assertions: func(t *testing.T, result *time.Duration) {
				assert.Equal(t, ptr.To(time.Duration(5)), result)
			},
		},
		{
			name: "returns default timeout when retry config returns nil",
			step: &PromotionStep{
				Retry: &kargoapi.PromotionStepRetry{},
			},
			runner: mockRetryableRunner{defaultTimeout: ptr.To(time.Duration(3))},
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

func TestPromotionStep_GetErrorThreshold(t *testing.T) {
	tests := []struct {
		name       string
		step       *PromotionStep
		runner     any
		assertions func(t *testing.T, result uint32)
	}{
		{
			name: "returns 1 with no retry config",
			step: &PromotionStep{
				Retry: nil,
			},
			assertions: func(t *testing.T, result uint32) {
				assert.Equal(t, uint32(1), result)
			},
		},
		{
			name: "returns configured threshold for non-retryable runner",
			step: &PromotionStep{
				Retry: &kargoapi.PromotionStepRetry{
					ErrorThreshold: 5,
				},
			},
			runner: nil,
			assertions: func(t *testing.T, result uint32) {
				assert.Equal(t, uint32(5), result)
			},
		},
		{
			name: "returns configured threshold for retryable runner",
			step: &PromotionStep{
				Retry: &kargoapi.PromotionStepRetry{
					ErrorThreshold: 5,
				},
			},
			runner: mockRetryableRunner{defaultErrorThreshold: 3},
			assertions: func(t *testing.T, result uint32) {
				assert.Equal(t, uint32(5), result)
			},
		},
		{
			name: "returns default threshold when retry config returns 0",
			step: &PromotionStep{
				Retry: &kargoapi.PromotionStepRetry{
					ErrorThreshold: 0,
				},
			},
			runner: mockRetryableRunner{defaultErrorThreshold: 3},
			assertions: func(t *testing.T, result uint32) {
				assert.Equal(t, uint32(3), result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.step.GetErrorThreshold(tt.runner)
			tt.assertions(t, result)
		})
	}
}

func TestPromotionStep_GetConfig(t *testing.T) {
	testScheme := k8sruntime.NewScheme()
	err := kargoapi.AddToScheme(testScheme)
	require.NoError(t, err)
	testClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&kargoapi.Warehouse{
			ObjectMeta: v1.ObjectMeta{
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
		promoCtx    PromotionContext
		promoState  State
		rawCfg      []byte
		expectedCfg Config
	}{
		{
			name: "test context",
			// Test that expressions can reference promotion context
			promoCtx: PromotionContext{
				Project:   "fake-project",
				Stage:     "fake-stage",
				Promotion: "fake-promotion",
			},
			rawCfg: []byte(`{
				"project": "${{ ctx.project }}",
				"stage": "${{ ctx.stage }}",
				"promotion": "${{ ctx.promotion }}"
			}`),
			expectedCfg: Config{
				"project":   "fake-project",
				"stage":     "fake-stage",
				"promotion": "fake-promotion",
			},
		},
		{
			name: "test secrets",
			// Test that expressions can reference secrets
			promoCtx: PromotionContext{
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
			expectedCfg: Config{
				"secret1-1": "value1",
				"secret1-2": "value2",
				"secret2-3": "value3",
				"secret2-4": "value4",
			},
		},
		{
			name: "test vars with literal values",
			// Test that vars can be assigned literal values
			promoCtx: PromotionContext{
				Vars: []kargoapi.PromotionVariable{
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
			expectedCfg: Config{
				"strVar":  "foo",
				"boolVar": true,
				"numVar":  42,
			},
		},
		{
			name: "test vars with expressions",
			// Test using expressions to define the value of vars
			promoCtx: PromotionContext{
				Vars: []kargoapi.PromotionVariable{
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
			expectedCfg: Config{
				"strVar":  "foo",
				"boolVar": true,
				"numVar":  42,
			},
		},
		{
			name: "test outputs",
			// Test that expressions can reference outputs
			promoState: State{
				"strOutput":  "foo",
				"boolOutput": true,
				"numOutput":  42,
			},
			rawCfg: []byte(`{
				"strOutput": "${{ outputs.strOutput }}",
				"boolOutput": "${{ outputs.boolOutput }}",
				"numOutput": "${{ outputs.numOutput }}"
			}`),
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
			promoCtx: PromotionContext{
				Vars: []kargoapi.PromotionVariable{{
					Name:  "warehouseName",
					Value: "fake-warehouse",
				}},
			},
			rawCfg: []byte(`{
				"origin1": "${{ warehouse('fake-warehouse') }}",
				"origin2": "${{ warehouse(vars.warehouseName) }}"
			}`),
			expectedCfg: Config{
				"origin1": map[string]any{
					"kind": "Warehouse",
					"name": "fake-warehouse",
				},
				"origin2": map[string]any{
					"kind": "Warehouse",
					"name": "fake-warehouse",
				},
			},
		},
		{
			name: "test commitFrom function",
			// Test different ways to use the commitFrom() function
			promoCtx: PromotionContext{
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
				Vars: []kargoapi.PromotionVariable{
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
			promoCtx: PromotionContext{
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
				Vars: []kargoapi.PromotionVariable{
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
			promoCtx: PromotionContext{
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
				Vars: []kargoapi.PromotionVariable{
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
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			promoStep := PromotionStep{
				Config: testCase.rawCfg,
			}
			stepCfg, err := promoStep.GetConfig(
				context.Background(),
				testClient,
				testCase.promoCtx,
				testCase.promoState,
			)
			require.NoError(t, err)
			require.Equal(t, testCase.expectedCfg, stepCfg)
		})
	}
}
func TestPromotionStep_Skip(t *testing.T) {
	tests := []struct {
		name       string
		step       *PromotionStep
		ctx        PromotionContext
		state      State
		assertions func(*testing.T, bool, error)
	}{
		{
			name: "no if condition",
			step: &PromotionStep{},
			assertions: func(t *testing.T, b bool, err error) {
				assert.False(t, b)
				assert.NoError(t, err)
			},
		},
		{
			name: "if condition uses vars",
			step: &PromotionStep{
				If: "${{ vars.foo == 'bar' }}",
			},
			ctx: PromotionContext{
				Vars: []kargoapi.PromotionVariable{
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
			step: &PromotionStep{
				If: "${{ outputs.foo == 'bar' }}",
			},
			state: State{
				"foo": "bar",
			},
			assertions: func(t *testing.T, b bool, err error) {
				assert.NoError(t, err)
				assert.False(t, b)
			},
		},
		{
			name: "if condition uses task outputs",
			step: &PromotionStep{
				Alias: "task::other-alias",
				If:    "${{ task.outputs.alias.foo == 'bar' }}",
			},
			state: State{
				"task::alias": map[string]any{
					"foo": "baz",
				},
			},
			assertions: func(t *testing.T, b bool, err error) {
				assert.NoError(t, err)
				assert.True(t, b)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.step.Skip(tt.ctx, tt.state)
			tt.assertions(t, got, err)
		})
	}
}
