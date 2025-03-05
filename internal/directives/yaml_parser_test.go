package directives

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
	"github.com/akuity/kargo/pkg/x/directive/builtin"
)

func Test_yamlParser_validate(t *testing.T) {
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
				"path": "valid.yaml",
			},
			expectedError: "(root): outputs is required",
		},
		{
			name: "outputs field is an empty array",
			config: map[string]any{
				"path":    "valid.yaml",
				"outputs": []map[string]any{},
			},
			expectedError: "outputs: Array must have at least 1 items",
		},
		{
			name: "name is not specified",
			config: map[string]any{
				"path": "valid.yaml",
				"outputs": []map[string]any{
					{"fromExpression": "$.data"},
				},
			},
			expectedError: "outputs.0: name is required",
		},
		{
			name: "name is empty",
			config: map[string]any{
				"path": "valid.yaml",
				"outputs": []map[string]any{
					{"name": "", "fromExpression": "$.data"},
				},
			},
			expectedError: "name: String length must be greater than or equal to 1",
		},
		{
			name: "FromExpression is not specified",
			config: map[string]any{
				"path": "valid.yaml",
				"outputs": []map[string]any{
					{"name": "output1"},
				},
			},
			expectedError: "outputs.0: fromExpression is required",
		},
		{
			name: "FromExpression is empty",
			config: map[string]any{
				"path": "valid.yaml",
				"outputs": []map[string]any{
					{"name": "output1", "fromExpression": ""},
				},
			},
			expectedError: "fromExpression: String length must be greater than or equal to 1",
		},
		{
			name: "valid configuration (path + outputs present)",
			config: map[string]any{
				"path": "valid.yaml",
				"outputs": []map[string]any{
					{"name": "output1", "fromExpression": "$.data"},
				},
			},
			expectedError: "",
		},
	}

	r := newYAMLParser()
	runner, ok := r.(*yamlParser)
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

func Test_yamlParser_runPromotionStep(t *testing.T) {
	tests := []struct {
		name       string
		stepCtx    *PromotionStepContext
		cfg        builtin.YAMLParseConfig
		files      map[string]string
		assertions func(*testing.T, string, PromotionStepResult, error)
	}{
		{
			name: "successful run with outputs",
			stepCtx: &PromotionStepContext{
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
			assertions: func(t *testing.T, _ string, result PromotionStepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, PromotionStepResult{
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
			stepCtx: &PromotionStepContext{
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
			assertions: func(t *testing.T, _ string, result PromotionStepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, result)
				assert.Contains(t, err.Error(), "failed to extract outputs")
			},
		},
		{
			name: "no outputs provided",
			stepCtx: &PromotionStepContext{
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
			assertions: func(t *testing.T, _ string, result PromotionStepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, PromotionStepResult{
					Status: kargoapi.PromotionPhaseErrored,
				}, result)
				assert.Contains(t, err.Error(), "outputs is required")
			},
		},
		{
			name: "handle empty YAML file",
			stepCtx: &PromotionStepContext{
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
			assertions: func(t *testing.T, _ string, result PromotionStepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, result)
				assert.Contains(t, err.Error(), "could not parse empty YAML file")
			},
		},
		{
			name:    "path is empty",
			stepCtx: &PromotionStepContext{Project: "test-project"},
			cfg: builtin.YAMLParseConfig{
				Path:    "",
				Outputs: []builtin.YAMLParse{{Name: "key", FromExpression: "app.key"}},
			},
			files: map[string]string{},
			assertions: func(t *testing.T, _ string, result PromotionStepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, result)
				assert.Contains(t, err.Error(), "YAML file path cannot be empty")
			},
		},
		{
			name:    "path is a directory instead of a file",
			stepCtx: &PromotionStepContext{Project: "test-project"},
			cfg: builtin.YAMLParseConfig{
				Path:    "config",
				Outputs: []builtin.YAMLParse{{Name: "key", FromExpression: "app.key"}},
			},
			files: map[string]string{},
			assertions: func(t *testing.T, _ string, result PromotionStepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, result)
				assert.Contains(t, err.Error(), "no such file or directory")
			},
		},
		{
			name: "valid YAML, valid expressions, valid path",
			stepCtx: &PromotionStepContext{
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
			assertions: func(t *testing.T, _ string, result PromotionStepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, PromotionStepResult{
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

	runner := &yamlParser{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepCtx := tt.stepCtx

			stepCtx.WorkDir = t.TempDir()
			for p, c := range tt.files {
				require.NoError(t, os.MkdirAll(path.Join(stepCtx.WorkDir, path.Dir(p)), 0o700))
				require.NoError(t, os.WriteFile(path.Join(stepCtx.WorkDir, p), []byte(c), 0o600))
			}

			result, err := runner.runPromotionStep(context.Background(), stepCtx, tt.cfg)
			tt.assertions(t, stepCtx.WorkDir, result, err)
		})
	}
}

func Test_yamlParser_readAndParseYAML(t *testing.T) {
	tempDir := t.TempDir()
	validYAML := `
key: value
num: 42
flag: true
`
	invalidYAML := `
key: : value
  num: 42
`
	tests := []struct {
		name    string
		content string
		expects error
	}{
		{"Valid YAML", validYAML, nil},
		{"Invalid YAML syntax", invalidYAML, errors.New("could not parse YAML file")},
		{"Empty YAML file", "", errors.New("could not parse empty YAML file")},
	}

	r := newYAMLParser()
	runner, ok := r.(*yamlParser)
	require.True(t, ok)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, "test.yaml")
			err := os.WriteFile(filePath, []byte(tt.content), 0600)
			if err != nil {
				t.Fatalf("failed to write file: %v", err)
			}
			_, err = runner.readAndParseYAML(tempDir, "test.yaml")
			if tt.expects != nil {
				assert.ErrorContains(t, err, tt.expects.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_yamlParser_extractValues(t *testing.T) {
	yp := &yamlParser{}

	tests := []struct {
		name           string
		data           map[string]any
		outputs        []builtin.YAMLParse
		expected       map[string]any
		expectedErrMsg string
	}{
		{
			name: "valid yaml, valid expression",
			data: map[string]any{"key": "value"},
			outputs: []builtin.YAMLParse{
				{Name: "result", FromExpression: "key"},
			},
			expected: map[string]any{"result": "value"},
		},
		{
			name: "valid yaml, expression points to missing key",
			data: map[string]any{"key": "value"},
			outputs: []builtin.YAMLParse{
				{Name: "result", FromExpression: "missingKey"},
			},
			expectedErrMsg: "error compiling expression",
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
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
