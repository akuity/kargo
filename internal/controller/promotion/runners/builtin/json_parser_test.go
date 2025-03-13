package builtin

import (
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runners/builtin"
)

func Test_jsonParser_validate(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]any
		expectedError string
	}{
		{
			name: "path not specified (missing path field)",
			config: map[string]any{
				"outputs": []map[string]any{
					{"name": "output1", "fromExpression": "$.data"},
				},
			},
			expectedError: "(root): path is required",
		},
		{
			name: "path is empty string",
			config: map[string]any{
				"path": "",
				"outputs": []map[string]any{
					{"name": "output1", "fromExpression": "$.data"},
				},
			},
			expectedError: "path: String length must be greater than or equal to 1",
		},
		{
			name: "outputs field is missing",
			config: map[string]any{
				"path": "valid.json",
			},
			expectedError: "(root): outputs is required",
		},
		{
			name: "outputs field is an empty array",
			config: map[string]any{
				"path":    "valid.json",
				"outputs": []map[string]any{},
			},
			expectedError: "outputs: Array must have at least 1 items",
		},
		{
			name: "name is not specified",
			config: map[string]any{
				"path": "valid.json",
				"outputs": []map[string]any{
					{"fromExpression": "$.data"},
				},
			},
			expectedError: "outputs.0: name is required",
		},
		{
			name: "name is empty",
			config: map[string]any{
				"path": "valid.json",
				"outputs": []map[string]any{
					{"name": "", "fromExpression": "$.data"},
				},
			},
			expectedError: "name: String length must be greater than or equal to 1",
		},
		{
			name: "FromExpression is not specified",
			config: map[string]any{
				"path": "valid.json",
				"outputs": []map[string]any{
					{"name": "output1"},
				},
			},
			expectedError: "outputs.0: fromExpression is required",
		},
		{
			name: "FromExpression is empty",
			config: map[string]any{
				"path": "valid.json",
				"outputs": []map[string]any{
					{"name": "output1", "fromExpression": ""},
				},
			},
			expectedError: "fromExpression: String length must be greater than or equal to 1",
		},
		{
			name: "valid configuration (path + outputs present)",
			config: map[string]any{
				"path": "valid.json",
				"outputs": []map[string]any{
					{"name": "output1", "fromExpression": "$.data"},
				},
			},
			expectedError: "",
		},
	}

	r := newJSONParser()
	runner, ok := r.(*jsonParser)
	require.True(t, ok)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := runner.validate(tc.config)
			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			}
		})
	}
}

func Test_jsonParser_run(t *testing.T) {
	tests := []struct {
		name       string
		stepCtx    *promotion.StepContext
		cfg        builtin.JSONParseConfig
		files      map[string]string
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "successful run with outputs",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.JSONParseConfig{
				Path: "config.json",
				Outputs: []builtin.JSONParse{
					{Name: "appVersion", FromExpression: "app.version"},
					{Name: "featureStatus", FromExpression: "features.newFeature"},
				},
			},
			files: map[string]string{
				"config.json": `{
					"app": {
						"version": "1.0.0"
					},
					"features": {
						"newFeature": false
					}
				}`,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{
					Status: kargoapi.PromotionPhaseSucceeded,
					Output: map[string]any{
						"appVersion":    "1.0.0",
						"featureStatus": false,
					},
				}, result)
				require.NoError(t, err)
			},
		},
		{
			name: "failed to extract outputs",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.JSONParseConfig{
				Path: "config.json",
				Outputs: []builtin.JSONParse{
					{Name: "invalidField", FromExpression: "nonexistent.path"},
				},
			},
			files: map[string]string{"config.json": `{ "app": { "version": "1.0.0" }}`},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionPhaseErrored}, result)
				assert.Contains(t, err.Error(), "failed to extract outputs")
			},
		},
		{
			name: "no outputs provided",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.JSONParseConfig{
				Path:    "config.json",
				Outputs: []builtin.JSONParse{},
			},
			files: map[string]string{
				"config.json": `{
					"app": {
						"version": "1.0.0"
					}
				}`,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{
					Status: kargoapi.PromotionPhaseErrored,
				}, result)
				assert.Contains(t, err.Error(), "outputs is required")
			},
		},
		{
			name: "handle empty JSON file",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.JSONParseConfig{
				Path: "config.json",
				Outputs: []builtin.JSONParse{
					{Name: "key", FromExpression: "app.key"},
				},
			},
			files: map[string]string{
				"config.json": ``,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionPhaseErrored}, result)
				assert.Contains(t, err.Error(), "could not parse JSON file")
			},
		},
		{
			name:    "path is empty",
			stepCtx: &promotion.StepContext{Project: "test-project"},
			cfg: builtin.JSONParseConfig{
				Path:    "",
				Outputs: []builtin.JSONParse{{Name: "key", FromExpression: "app.key"}},
			},
			files: map[string]string{},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionPhaseErrored}, result)
				assert.Contains(t, err.Error(), "JSON file path cannot be empty")
			},
		},
		{
			name:    "path is a directory instead of a file",
			stepCtx: &promotion.StepContext{Project: "test-project"},
			cfg: builtin.JSONParseConfig{
				Path: "config", Outputs: []builtin.JSONParse{{Name: "key", FromExpression: "app.key"}},
			},
			files: map[string]string{},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionPhaseErrored}, result)
				assert.Contains(t, err.Error(), "no such file or directory")
			},
		},
		{
			name: "valid JSON, valid expressions, valid path",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.JSONParseConfig{
				Path: "config.json",
				Outputs: []builtin.JSONParse{
					{Name: "appVersion", FromExpression: "app.version"},
					{Name: "isEnabled", FromExpression: "features.enabled"},
					{Name: "threshold", FromExpression: "config.threshold"},
				},
			},
			files: map[string]string{
				"config.json": `{
					"app": {
						"version": "2.0.1"
					},
					"features": {
						"enabled": true
					},
					"config": {
						"threshold": 10.0
					}
				}`,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{
					Status: kargoapi.PromotionPhaseSucceeded,
					Output: map[string]any{
						"appVersion": "2.0.1",
						"isEnabled":  true,
						"threshold":  10.0,
					},
				}, result)
			},
		},
	}
	runner := &jsonParser{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepCtx := tt.stepCtx

			stepCtx.WorkDir = t.TempDir()
			for p, c := range tt.files {
				require.NoError(t, os.MkdirAll(path.Join(stepCtx.WorkDir, path.Dir(p)), 0o700))
				require.NoError(t, os.WriteFile(path.Join(stepCtx.WorkDir, p), []byte(c), 0o600))
			}

			result, err := runner.run(context.Background(), stepCtx, tt.cfg)
			tt.assertions(t, stepCtx.WorkDir, result, err)
		})
	}
}

func Test_jsonParser_readAndParseJSON(t *testing.T) {
	tempDir := t.TempDir()
	validJSON := `{"key": "value", "num": 42, "flag": true}`
	invalidJSON := `{key: "value"}`
	tests := []struct {
		name    string
		content string
		expects error
	}{
		{"Valid JSON", validJSON, nil},
		{"Invalid JSON syntax", invalidJSON, errors.New("could not parse JSON file")},
		{"Empty JSON file", "", errors.New("could not parse JSON file")},
	}

	r := newJSONParser()
	runner, ok := r.(*jsonParser)
	require.True(t, ok)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, "test.json")
			err := os.WriteFile(filePath, []byte(tt.content), 0600)
			if err != nil {
				t.Fatalf("failed to write file: %v", err)
			}
			_, err = runner.readAndParseJSON(tempDir, "test.json")
			if tt.expects != nil {
				assert.ErrorContains(t, err, tt.expects.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_jsonParser_extractValues(t *testing.T) {
	jp := &jsonParser{}

	tests := []struct {
		name           string
		data           map[string]any
		outputs        []builtin.JSONParse
		expected       map[string]any
		expectedErrMsg string
	}{
		{
			name: "valid json, valid expression",
			data: map[string]any{"key": "value"},
			outputs: []builtin.JSONParse{
				{Name: "result", FromExpression: "key"},
			},
			expected: map[string]any{"result": "value"},
		},
		{
			name: "valid json, expression points to missing key",
			data: map[string]any{"key": "value"},
			outputs: []builtin.JSONParse{
				{Name: "result", FromExpression: "missingKey"},
			},
			expectedErrMsg: "error compiling expression",
		},
		{
			name: "expression evaluates to a nested object",
			data: map[string]any{"nested": map[string]any{"key": "value"}},
			outputs: []builtin.JSONParse{
				{Name: "result", FromExpression: "nested"},
			},
			expected: map[string]any{"result": map[string]any{"key": "value"}},
		},
		{
			name: "expression evaluates to an array",
			data: map[string]any{"array": []any{1, 2, 3}},
			outputs: []builtin.JSONParse{
				{Name: "result", FromExpression: "array"},
			},
			expected: map[string]any{"result": []any{1, 2, 3}},
		},
		{
			name: "expression evaluates to a string",
			data: map[string]any{"key": "value"},
			outputs: []builtin.JSONParse{
				{Name: "result", FromExpression: "key"},
			},
			expected: map[string]any{"result": "value"},
		},
		{
			name: "expression evaluates to an integer",
			data: map[string]any{"number": 42},
			outputs: []builtin.JSONParse{
				{Name: "result", FromExpression: "number"},
			},
			expected: map[string]any{"result": 42},
		},
		{
			name: "expression compilation error",
			data: map[string]any{"key": "value"},
			outputs: []builtin.JSONParse{
				{Name: "result", FromExpression: "(1 + 2"},
			},
			expectedErrMsg: "error compiling expression",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := jp.extractValues(tt.data, tt.outputs)

			if tt.expectedErrMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
