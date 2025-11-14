package builtin

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_yamlParser_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name: "path not specified (missing path field)",
			config: map[string]any{
				"outputs": []map[string]any{
					{"name": "output1", "fromExpression": "$.data"},
				},
			},
			expectedProblems: []string{
				"(root): path is required",
			},
		},
		{
			name: "path is empty string",
			config: map[string]any{
				"path": "",
				"outputs": []map[string]any{
					{"name": "output1", "fromExpression": "$.data"},
				},
			},
			expectedProblems: []string{
				"path: String length must be greater than or equal to 1",
			},
		},
		{
			name: "outputs field is missing",
			config: map[string]any{
				"path": "valid.yaml",
			},
			expectedProblems: []string{
				"(root): outputs is required",
			},
		},
		{
			name: "outputs field is an empty array",
			config: map[string]any{
				"path":    "valid.yaml",
				"outputs": []map[string]any{},
			},
			expectedProblems: []string{
				"outputs: Array must have at least 1 items",
			},
		},
		{
			name: "name is not specified",
			config: map[string]any{
				"path": "valid.yaml",
				"outputs": []map[string]any{
					{"fromExpression": "$.data"},
				},
			},
			expectedProblems: []string{
				"outputs.0: name is required",
			},
		},
		{
			name: "name is empty",
			config: map[string]any{
				"path": "valid.yaml",
				"outputs": []map[string]any{
					{"name": "", "fromExpression": "$.data"},
				},
			},
			expectedProblems: []string{
				"name: String length must be greater than or equal to 1",
			},
		},
		{
			name: "FromExpression is not specified",
			config: map[string]any{
				"path": "valid.yaml",
				"outputs": []map[string]any{
					{"name": "output1"},
				},
			},
			expectedProblems: []string{
				"outputs.0: fromExpression is required",
			},
		},
		{
			name: "FromExpression is empty",
			config: map[string]any{
				"path": "valid.yaml",
				"outputs": []map[string]any{
					{"name": "output1", "fromExpression": ""},
				},
			},
			expectedProblems: []string{
				"fromExpression: String length must be greater than or equal to 1",
			},
		},
		{
			name: "valid configuration (path + outputs present)",
			config: map[string]any{
				"path": "valid.yaml",
				"outputs": []map[string]any{
					{"name": "output1", "fromExpression": "$.data"},
				},
			},
			expectedProblems: nil,
		},
	}

	r := newYAMLParser(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*yamlParser)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_yamlParser_run(t *testing.T) {
	tests := []struct {
		name       string
		stepCtx    *promotion.StepContext
		cfg        builtin.YAMLParseConfig
		files      map[string]string
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "successful run with outputs",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.YAMLParseConfig{
				Path: "config.yaml",
				Outputs: []builtin.YAMLParse{
					{Name: "appVersion", FromExpression: "app.version"},
					{Name: "featureStatus", FromExpression: "features.newFeature"},
				},
			},
			files: map[string]string{
				"config.yaml": `
app:
  version: "1.0.0"
features:
  newFeature: false
`,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{
					Status: kargoapi.PromotionStepStatusSucceeded,
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
			cfg: builtin.YAMLParseConfig{
				Path: "config.yaml",
				Outputs: []builtin.YAMLParse{
					{Name: "invalidField", FromExpression: "nonexistent.path"},
				},
			},
			files: map[string]string{
				"config.yaml": `
app:
  version: "1.0.0"
`,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				assert.Contains(t, err.Error(), "failed to extract outputs")
			},
		},
		{
			name: "no outputs provided",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.YAMLParseConfig{
				Path:    "config.yaml",
				Outputs: []builtin.YAMLParse{},
			},
			files: map[string]string{
				"config.yaml": `
app:
  version: "1.0.0"
`,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				}, result)
				assert.Contains(t, err.Error(), "outputs is required")
			},
		},
		{
			name: "handle empty YAML file",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.YAMLParseConfig{
				Path: "config.yaml",
				Outputs: []builtin.YAMLParse{
					{Name: "key", FromExpression: "app.key"},
				},
			},
			files: map[string]string{
				"config.yaml": ``,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				assert.Contains(t, err.Error(), "could not parse empty YAML file")
			},
		},
		{
			name:    "path is empty",
			stepCtx: &promotion.StepContext{Project: "test-project"},
			cfg: builtin.YAMLParseConfig{
				Path:    "",
				Outputs: []builtin.YAMLParse{{Name: "key", FromExpression: "app.key"}},
			},
			files: map[string]string{},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				assert.Contains(t, err.Error(), "YAML file path cannot be empty")
			},
		},
		{
			name:    "path is a directory instead of a file",
			stepCtx: &promotion.StepContext{Project: "test-project"},
			cfg: builtin.YAMLParseConfig{
				Path:    "config",
				Outputs: []builtin.YAMLParse{{Name: "key", FromExpression: "app.key"}},
			},
			files: map[string]string{},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				assert.Contains(t, err.Error(), "no such file or directory")
			},
		},
		{
			name: "valid YAML, valid expressions, valid path",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.YAMLParseConfig{
				Path: "config.yaml",
				Outputs: []builtin.YAMLParse{
					{Name: "appVersion", FromExpression: "app.version"},
					{Name: "isEnabled", FromExpression: "features.enabled"},
					{Name: "threshold", FromExpression: "config.threshold"},
				},
			},
			files: map[string]string{
				"config.yaml": `
app:
  version: "2.0.1"
features:
  enabled: true
config:
  threshold: 10.0
`,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{
					Status: kargoapi.PromotionStepStatusSucceeded,
					Output: map[string]any{
						"appVersion": "2.0.1",
						"isEnabled":  true,
						"threshold":  10.0,
					},
				}, result)
			},
		},
	}

	runner := &yamlParser{}

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

func Test_yamlParser_readAndParseYAML(t *testing.T) {
	yp := &yamlParser{}

	tests := []struct {
		name           string
		content        string
		expected       any
		expectedErrMsg string
	}{
		{
			name: "Valid YAML with map root",
			content: `
key: value
num: 42
flag: true
`,
			expected: map[string]any{
				"key":  "value",
				"num":  float64(42),
				"flag": true,
			},
		},
		{
			name: "Valid YAML with root list",
			content: `
- item1
- item2
- item3
`,
			expected: []any{"item1", "item2", "item3"},
		},
		{
			name: "Valid YAML with nested structure",
			content: `
app:
  name: test-app
  version: 1.2.3
  config:
    debug: false
    port: 8080
items:
  - name: first
    value: 100
  - name: second
    value: 200
`,
			expected: map[string]any{
				"app": map[string]any{
					"name":    "test-app",
					"version": "1.2.3",
					"config": map[string]any{
						"debug": false,
						"port":  float64(8080),
					},
				},
				"items": []any{
					map[string]any{"name": "first", "value": float64(100)},
					map[string]any{"name": "second", "value": float64(200)},
				},
			},
		},
		{
			name: "Valid YAML with scalar root",
			content: `simple string value
`,
			expected: "simple string value",
		},
		{
			name:           "Invalid YAML syntax",
			content:        "key: : value\n  num: 42",
			expectedErrMsg: "could not parse YAML file",
		},
		{
			name:           "Empty YAML file",
			content:        "",
			expectedErrMsg: "could not parse empty YAML file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			filePath := filepath.Join(tempDir, "test.yaml")

			err := os.WriteFile(filePath, []byte(tt.content), 0600)
			require.NoError(t, err)

			result, err := yp.readAndParseYAML(tempDir, "test.yaml")

			if tt.expectedErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func Test_yamlParser_extractValues(t *testing.T) {
	yp := &yamlParser{}

	tests := []struct {
		name           string
		data           any
		outputs        []builtin.YAMLParse
		expected       map[string]any
		expectedErrMsg string
	}{
		{
			name: "valid expression",
			data: map[string]any{"key": "value"},
			outputs: []builtin.YAMLParse{
				{Name: "result", FromExpression: "key"},
			},
			expected: map[string]any{"result": "value"},
		},
		{
			name: "root list - access by index",
			data: []any{"first", "second", "third"},
			outputs: []builtin.YAMLParse{
				{Name: "result", FromExpression: "$env[0]"},
			},
			expected: map[string]any{"result": "first"},
		},
		{
			name: "root list - get length",
			data: []any{"first", "second", "third"},
			outputs: []builtin.YAMLParse{
				{Name: "result", FromExpression: "len($env)"},
			},
			expected: map[string]any{"result": 3},
		},
		{
			name: "root list of objects - access nested property",
			data: []any{
				map[string]any{"name": "item1", "value": 10},
				map[string]any{"name": "item2", "value": 20},
			},
			outputs: []builtin.YAMLParse{
				{Name: "result", FromExpression: "$env[1].name"},
			},
			expected: map[string]any{"result": "item2"},
		},
		{
			name: "expression points to missing key",
			data: map[string]any{"key": "value"},
			outputs: []builtin.YAMLParse{
				{Name: "result", FromExpression: "missingKey"},
			},
			expected: map[string]any{"result": nil},
		},
		{
			name: "expression evaluates to a nested object",
			data: map[string]any{"nested": map[string]any{"key": "value"}},
			outputs: []builtin.YAMLParse{
				{Name: "result", FromExpression: "nested"},
			},
			expected: map[string]any{"result": map[string]any{"key": "value"}},
		},
		{
			name: "expression evaluates to an array",
			data: map[string]any{"array": []any{1, 2, 3}},
			outputs: []builtin.YAMLParse{
				{Name: "result", FromExpression: "array"},
			},
			expected: map[string]any{"result": []any{1, 2, 3}},
		},
		{
			name: "expression evaluates to a string",
			data: map[string]any{"key": "value"},
			outputs: []builtin.YAMLParse{
				{Name: "result", FromExpression: "key"},
			},
			expected: map[string]any{"result": "value"},
		},
		{
			name: "expression evaluates to an integer",
			data: map[string]any{"number": 42},
			outputs: []builtin.YAMLParse{
				{Name: "result", FromExpression: "number"},
			},
			expected: map[string]any{"result": 42},
		},
		{
			name: "expression compilation error",
			data: map[string]any{"key": "value"},
			outputs: []builtin.YAMLParse{
				{Name: "result", FromExpression: "(1 + 2"},
			},
			expectedErrMsg: "error compiling expression",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := yp.extractValues(tt.data, tt.outputs)

			if tt.expectedErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}
