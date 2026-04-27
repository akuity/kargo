package builtin

import (
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

func Test_tomlParser_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name: "path not specified",
			config: promotion.Config{
				"outputs": []map[string]any{{"name": "output1", "fromExpression": "app.version"}},
			},
			expectedProblems: []string{"(root): path is required"},
		},
		{
			name: "path is empty string",
			config: promotion.Config{
				"path":    "",
				"outputs": []map[string]any{{"name": "output1", "fromExpression": "app.version"}},
			},
			expectedProblems: []string{"path: String length must be greater than or equal to 1"},
		},
		{
			name: "outputs field is missing",
			config: promotion.Config{
				"path": "valid.toml",
			},
			expectedProblems: []string{"(root): outputs is required"},
		},
		{
			name: "outputs field is an empty array",
			config: promotion.Config{
				"path":    "valid.toml",
				"outputs": []map[string]any{},
			},
			expectedProblems: []string{"outputs: Array must have at least 1 items"},
		},
		{
			name: "name is not specified",
			config: promotion.Config{
				"path":    "valid.toml",
				"outputs": []map[string]any{{"fromExpression": "app.version"}},
			},
			expectedProblems: []string{"outputs.0: name is required"},
		},
		{
			name: "name is empty",
			config: promotion.Config{
				"path":    "valid.toml",
				"outputs": []map[string]any{{"name": "", "fromExpression": "app.version"}},
			},
			expectedProblems: []string{"name: String length must be greater than or equal to 1"},
		},
		{
			name: "fromExpression is not specified",
			config: promotion.Config{
				"path":    "valid.toml",
				"outputs": []map[string]any{{"name": "output1"}},
			},
			expectedProblems: []string{"outputs.0: fromExpression is required"},
		},
		{
			name: "fromExpression is empty",
			config: promotion.Config{
				"path":    "valid.toml",
				"outputs": []map[string]any{{"name": "output1", "fromExpression": ""}},
			},
			expectedProblems: []string{"fromExpression: String length must be greater than or equal to 1"},
		},
		{
			name: "valid configuration",
			config: promotion.Config{
				"path":    "valid.toml",
				"outputs": []map[string]any{{"name": "output1", "fromExpression": "app.version"}},
			},
		},
	}

	r := newTOMLParser(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*tomlParser)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_tomlParser_run(t *testing.T) {
	tests := []struct {
		name       string
		stepCtx    *promotion.StepContext
		cfg        builtin.TOMLParseConfig
		files      map[string]string
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name:    "successful run with outputs",
			stepCtx: &promotion.StepContext{Project: "test-project"},
			cfg: builtin.TOMLParseConfig{
				Path: "config.toml",
				Outputs: []builtin.TomlParse{
					{Name: "appVersion", FromExpression: "app.version"},
					{Name: "featureStatus", FromExpression: "features.newFeature"},
				},
			},
			files: map[string]string{
				"config.toml": "[app]\nversion = \"1.0.0\"\n\n[features]\nnewFeature = false\n",
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
			},
		},
		{
			name:    "failed to extract outputs",
			stepCtx: &promotion.StepContext{Project: "test-project"},
			cfg: builtin.TOMLParseConfig{
				Path:    "config.toml",
				Outputs: []builtin.TomlParse{{Name: "invalidField", FromExpression: "nonexistent.path"}},
			},
			files: map[string]string{"config.toml": "[app]\nversion = \"1.0.0\"\n"},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				assert.Contains(t, err.Error(), "failed to extract outputs")
			},
		},
		{
			name:    "no outputs provided",
			stepCtx: &promotion.StepContext{Project: "test-project"},
			cfg:     builtin.TOMLParseConfig{Path: "config.toml", Outputs: []builtin.TomlParse{}},
			files:   map[string]string{"config.toml": "[app]\nversion = \"1.0.0\"\n"},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				assert.Contains(t, err.Error(), "outputs is required")
			},
		},
		{
			name:    "handle empty TOML file",
			stepCtx: &promotion.StepContext{Project: "test-project"},
			cfg: builtin.TOMLParseConfig{
				Path:    "config.toml",
				Outputs: []builtin.TomlParse{{Name: "key", FromExpression: "app.key"}},
			},
			files: map[string]string{"config.toml": ""},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				assert.Contains(t, err.Error(), "could not parse empty TOML file")
			},
		},
		{
			name:    "path is empty",
			stepCtx: &promotion.StepContext{Project: "test-project"},
			cfg: builtin.TOMLParseConfig{
				Path:    "",
				Outputs: []builtin.TomlParse{{Name: "key", FromExpression: "app.key"}},
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				assert.Contains(t, err.Error(), "TOML file path cannot be empty")
			},
		},
		{
			name:    "path is a directory instead of a file",
			stepCtx: &promotion.StepContext{Project: "test-project"},
			cfg: builtin.TOMLParseConfig{
				Path:    "config",
				Outputs: []builtin.TomlParse{{Name: "key", FromExpression: "app.key"}},
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				assert.Contains(t, err.Error(), "no such file or directory")
			},
		},
	}

	runner := &tomlParser{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepCtx := tt.stepCtx
			stepCtx.WorkDir = t.TempDir()
			for p, c := range tt.files {
				require.NoError(t, os.MkdirAll(path.Join(stepCtx.WorkDir, path.Dir(p)), 0o700))
				require.NoError(t, os.WriteFile(path.Join(stepCtx.WorkDir, p), []byte(c), 0o600))
			}

			result, err := runner.run(t.Context(), stepCtx, tt.cfg)
			tt.assertions(t, stepCtx.WorkDir, result, err)
		})
	}
}

func Test_tomlParser_readAndParseTOML(t *testing.T) {
	tp := &tomlParser{}

	tests := []struct {
		name           string
		content        string
		expected       any
		expectedErrMsg string
	}{
		{
			name:    "valid TOML with nested structure",
			content: "title = \"example\"\nnum = 42\nflag = true\nitems = [1, 2]\n[app]\nversion = \"1.2.3\"\n",
			expected: map[string]any{
				"title": "example",
				"num":   int64(42),
				"flag":  true,
				"items": []any{int64(1), int64(2)},
				"app": map[string]any{
					"version": "1.2.3",
				},
			},
		},
		{
			name:           "invalid TOML syntax",
			content:        "title = \n",
			expectedErrMsg: "could not parse TOML file",
		},
		{
			name:           "empty TOML file",
			content:        "",
			expectedErrMsg: "could not parse empty TOML file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			filePath := filepath.Join(tempDir, "test.toml")

			err := os.WriteFile(filePath, []byte(tt.content), 0o600)
			require.NoError(t, err)

			result, err := tp.readAndParseTOML(tempDir, "test.toml")

			if tt.expectedErrMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				return
			}

			assert.NoError(t, err)
			assert.EqualValues(t, tt.expected, result)
		})
	}
}
