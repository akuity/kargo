package directives

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

type mockRetryableRunner struct {
	defaultAttempts int64
}

func (m mockRetryableRunner) DefaultAttempts() int64 {
	return m.defaultAttempts
}

func TestPromotionStep_GetAttempts(t *testing.T) {
	tests := []struct {
		name       string
		step       *PromotionStep
		state      State
		assertions func(t *testing.T, result int64)
	}{
		{
			name: "returns 0 when state does not contain alias entry",
			step: &PromotionStep{
				Alias: "step-1",
			},
			state: State{},
			assertions: func(t *testing.T, result int64) {
				assert.Equal(t, int64(0), result)
			},
		},
		{
			name: "returns 0 when state has alias entry but no attempts key",
			step: &PromotionStep{
				Alias: "custom-alias",
			},
			state: State{
				"custom-alias": map[string]any{
					"someOtherKey": "value",
				},
			},
			assertions: func(t *testing.T, result int64) {
				assert.Equal(t, int64(0), result)
			},
		},
		{
			name: "correctly converts float64 attempts",
			step: &PromotionStep{
				Alias: "step-1",
			},
			state: State{
				"step-1": map[string]any{
					stateKeyAttempts: float64(3),
				},
			},
			assertions: func(t *testing.T, result int64) {
				assert.Equal(t, int64(3), result)
			},
		},
		{
			name: "correctly handles int64 attempts",
			step: &PromotionStep{
				Alias: "step-2",
			},
			state: State{
				"step-2": map[string]any{
					stateKeyAttempts: int64(1),
				},
			},
			assertions: func(t *testing.T, result int64) {
				assert.Equal(t, int64(1), result)
			},
		},
		{
			name: "returns 0 when attempts value is not a number",
			step: &PromotionStep{
				Alias: "invalid",
			},
			state: State{
				"invalid": map[string]any{
					stateKeyAttempts: "not a number",
				},
			},
			assertions: func(t *testing.T, result int64) {
				assert.Equal(t, int64(0), result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.step.GetAttempts(tt.state)
			tt.assertions(t, result)
		})
	}
}

func TestPromotionStep_GetMaxAttempts(t *testing.T) {
	tests := []struct {
		name       string
		step       *PromotionStep
		runner     any
		assertions func(t *testing.T, result int64)
	}{
		{
			name: "returns 1 with no retry config",
			step: &PromotionStep{
				Retry: nil,
			},
			assertions: func(t *testing.T, result int64) {
				assert.Equal(t, int64(1), result)
			},
		},
		{
			name: "returns configured attempts for non-retryable runner",
			step: &PromotionStep{
				Retry: &kargoapi.PromotionRetry{
					Attempts: 5,
				},
			},
			runner: nil,
			assertions: func(t *testing.T, result int64) {
				assert.Equal(t, int64(5), result)
			},
		},
		{
			name: "returns configured attempts for retryable runner",
			step: &PromotionStep{
				Retry: &kargoapi.PromotionRetry{
					Attempts: 5,
				},
			},
			runner: mockRetryableRunner{defaultAttempts: 3},
			assertions: func(t *testing.T, result int64) {
				assert.Equal(t, int64(5), result)
			},
		},
		{
			name: "returns default attempts when retry config returns 0",
			step: &PromotionStep{
				Retry: &kargoapi.PromotionRetry{
					Attempts: 0,
				},
			},
			runner: mockRetryableRunner{defaultAttempts: 3},
			assertions: func(t *testing.T, result int64) {
				assert.Equal(t, int64(3), result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.step.GetMaxAttempts(tt.runner)
			tt.assertions(t, result)
		})
	}
}

func TestPromotionStep_RecordAttempt(t *testing.T) {
	tests := []struct {
		name       string
		step       *PromotionStep
		state      State
		output     map[string]any
		assertions func(*testing.T, map[string]any)
	}{
		{
			name: "increment attempt counter",
			step: &PromotionStep{
				Alias: "foo",
			},
			state: State{
				"foo": map[string]any{"attempts": 2},
			},
			assertions: func(t *testing.T, output map[string]any) {
				assert.Equal(t, int64(3), output["attempts"])
			},
		},
		{
			name: "initialize attempt counter",
			step: &PromotionStep{
				Alias: "foo",
			},
			assertions: func(t *testing.T, output map[string]any) {
				assert.Equal(t, int64(1), output["attempts"])
			},
		},
		{
			name: "preserve existing output",
			step: &PromotionStep{
				Alias: "foo",
			},
			output: map[string]any{"existing": "value"},
			assertions: func(t *testing.T, output map[string]any) {
				assert.Equal(t, int64(1), output["attempts"])
				assert.Equal(t, "value", output["existing"])
			},
		},
		{
			name: "overwrites attempts value modified by runner",
			step: &PromotionStep{
				Alias: "foo",
			},
			state: State{
				"foo": map[string]any{"attempts": 2},
			},
			output: map[string]any{"attempts": 5},
			assertions: func(t *testing.T, output map[string]any) {
				assert.Equal(t, int64(3), output["attempts"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.step.RecordAttempt(tt.state, tt.output)
			tt.assertions(t, output)
		})
	}
}

func TestPromotionStep_GetConfig(t *testing.T) {
	promoCtx := PromotionContext{
		Project:   "fake-project",
		Stage:     "fake-stage",
		Promotion: "fake-promotion",
		Vars: []kargoapi.PromotionVariable{
			{
				Name:  "strVar",
				Value: "foo",
			},
			{
				Name:  "concatStrVar",
				Value: "${{ vars.strVar }}bar",
			},
			{
				Name:  "boolVar",
				Value: "true",
			},
			{
				Name: "boolStrVar",
				// Prove boolVar evaluated as a boolean
				Value: "${{ quote(!vars.boolVar) }}",
			},
			{
				Name:  "numVar",
				Value: "42",
			},
			{
				Name: "numStrVar",
				// Prove numVar evaluated as a number
				Value: "${{ quote(vars.numVar + 1) }}",
			},
		},
		FreightRequests: []kargoapi.FreightRequest{
			{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "fake-warehouse",
				},
				Sources: kargoapi.FreightSources{
					Direct: true,
				},
			},
		},
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
					Images: []kargoapi.Image{{
						RepoURL: "fake-image-repo",
						Tag:     "fake-image-tag",
					}},
					Charts: []kargoapi.Chart{{
						RepoURL: "https://fake-chart-repo",
						Name:    "fake-chart",
						Version: "fake-chart-version",
					}},
				},
			},
		},
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
	}
	promoState := State{
		"strOutput":  "foo",
		"boolOutput": true,
		"numOutput":  42,
	}
	promoStep := PromotionStep{
		// nolint: lll
		Config: []byte(`{
			"project": "${{ ctx.project }}",
			"stage": "${{ ctx.stage }}",
			"promotion": "${{ ctx.promotion }}",
			"staticStr": "foo",
			"staticBool": true,
			"staticNum": 42,
			"strVar": "${{ vars.strVar }}",
			"concatStrVar": "${{ vars.concatStrVar }}",
			"boolVar": "${{ vars.boolVar }}",
			"boolStrVar": "${{ quote(vars.boolStrVar) }}",
			"numVar": "${{ vars.numVar }}",
			"numStrVar": "${{ quote(vars.numStrVar) }}",
			"commitID": "${{ commitFrom(\"https://fake-git-repo\", warehouse(\"fake-warehouse\")).ID }}",
			"imageTag": "${{ imageFrom(\"fake-image-repo\", warehouse(\"fake-warehouse\")).Tag }}",
			"chartVersion": "${{ chartFrom(\"https://fake-chart-repo\", \"fake-chart\", warehouse(\"fake-warehouse\")).Version }}",
			"secret1-1": "${{ secrets.secret1.key1 }}",
			"secret1-2": "${{ secrets.secret1.key2 }}",
			"secret2-3": "${{ secrets.secret2.key3 }}",
			"secret2-4": "${{ secrets.secret2.key4 }}",
			"strOutput": "${{ outputs.strOutput }}",
			"strOutputConcat": "${{ outputs.strOutput }}${{ outputs.strOutput }}",
			"boolOutput": "${{ outputs.boolOutput }}",
			"boolStrOutput": "${{ quote(!outputs.boolOutput) }}",
			"numOutput": "${{ outputs.numOutput }}",
			"numStrOutput": "${{ quote(outputs.numOutput + 1) }}"
		}`),
	}
	stepCfg, err := promoStep.GetConfig(
		context.Background(),
		nil, // We can get away with a nil Kubernetes client because we're specifying origins
		promoCtx,
		promoState,
	)
	require.NoError(t, err)
	require.Equal(
		t,
		Config{
			"project":         "fake-project",
			"stage":           "fake-stage",
			"promotion":       "fake-promotion",
			"staticStr":       "foo",
			"staticBool":      true,
			"staticNum":       42,
			"strVar":          "foo",
			"concatStrVar":    "foobar",
			"boolVar":         true,
			"boolStrVar":      "false",
			"numVar":          42,
			"numStrVar":       "43",
			"commitID":        "fake-commit-id",
			"imageTag":        "fake-image-tag",
			"chartVersion":    "fake-chart-version",
			"secret1-1":       "value1",
			"secret1-2":       "value2",
			"secret2-3":       "value3",
			"secret2-4":       "value4",
			"strOutput":       "foo",
			"strOutputConcat": "foofoo",
			"boolOutput":      true,
			"boolStrOutput":   "false",
			"numOutput":       42,
			"numStrOutput":    "43",
		},
		stepCfg,
	)
}
