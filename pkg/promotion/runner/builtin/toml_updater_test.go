package builtin

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	inttoml "github.com/akuity/kargo/pkg/toml"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_tomlUpdater_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name:             "path is not specified",
			config:           promotion.Config{},
			expectedProblems: []string{"(root): path is required"},
		},
		{
			name: "path is empty",
			config: promotion.Config{
				"path": "",
			},
			expectedProblems: []string{"path: String length must be greater than or equal to 1"},
		},
		{
			name:             "updates is null",
			config:           promotion.Config{},
			expectedProblems: []string{"(root): updates is required"},
		},
		{
			name: "updates is empty",
			config: promotion.Config{
				"updates": []promotion.Config{},
			},
			expectedProblems: []string{"updates: Array must have at least 1 items"},
		},
		{
			name: "key not specified",
			config: promotion.Config{
				"updates": []promotion.Config{{}},
			},
			expectedProblems: []string{"updates.0: key is required"},
		},
		{
			name: "key is empty",
			config: promotion.Config{
				"updates": []promotion.Config{{"key": ""}},
			},
			expectedProblems: []string{"updates.0.key: String length must be greater than or equal to 1"},
		},
		{
			name: "value not specified",
			config: promotion.Config{
				"updates": []promotion.Config{{}},
			},
			expectedProblems: []string{"updates.0: value is required"},
		},
		{
			name: "valid config",
			config: promotion.Config{
				"path": "fake-path",
				"updates": []promotion.Config{
					{"key": "package.version", "value": "1.2.3"},
					{"key": "features.enabled", "value": true},
				},
			},
		},
	}

	r := newTOMLUpdater(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*tomlUpdater)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_tomlUpdater_run(t *testing.T) {
	tests := []struct {
		name       string
		stepCtx    *promotion.StepContext
		cfg        builtin.TOMLUpdateConfig
		files      map[string]string
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name:    "successful run with updates",
			stepCtx: &promotion.StepContext{Project: "test-project"},
			cfg: builtin.TOMLUpdateConfig{
				Path: "config.toml",
				Updates: []builtin.TomlUpdate{
					{Key: "package.version", Value: "1.0.1"},
					{Key: "features.newFeature", Value: true},
					{Key: "threshold", Value: 100},
				},
			},
			files: map[string]string{
				"config.toml": "threshold = 1\n\n[package]\nversion = \"1.0.0\"\n\n[features]\nnewFeature = false\n",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{
					Status: kargoapi.PromotionStepStatusSucceeded,
					Output: map[string]any{
						"commitMessage": "Updated config.toml\n\n" +
							"- package.version: '1.0.1'\n" +
							"- features.newFeature: true\n" +
							"- threshold: 100",
					},
				}, result)
				content, err := os.ReadFile(path.Join(workDir, "config.toml"))
				require.NoError(t, err)
				assert.Contains(t, string(content), "version = '1.0.1'")
				assert.Contains(t, string(content), "newFeature = true")
				assert.Contains(t, string(content), "threshold = 100")
			},
		},
		{
			name:    "failed to update file",
			stepCtx: &promotion.StepContext{Project: "test-project"},
			cfg: builtin.TOMLUpdateConfig{
				Path:    "non-existent/config.toml",
				Updates: []builtin.TomlUpdate{{Key: "package.version", Value: "1.0.1"}},
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				assert.Contains(t, err.Error(), "TOML file update failed")
			},
		},
		{
			name:    "no updates provided",
			stepCtx: &promotion.StepContext{Project: "test-project"},
			cfg:     builtin.TOMLUpdateConfig{Path: "config.toml", Updates: []builtin.TomlUpdate{}},
			files:   map[string]string{"config.toml": "[package]\nversion = \"1.0.0\"\n"},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)
				content, err := os.ReadFile(path.Join(workDir, "config.toml"))
				require.NoError(t, err)
				assert.Equal(t, "[package]\nversion = \"1.0.0\"\n", string(content))
			},
		},
	}

	runner := &tomlUpdater{}

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

func Test_tomlUpdater_updateFile(t *testing.T) {
	tests := []struct {
		name          string
		valuesContent string
		updates       []inttoml.Update
		assertions    func(*testing.T, string, error)
	}{
		{
			name:          "successful update",
			valuesContent: "title = \"value\"\n",
			updates:       []inttoml.Update{{Key: "title", Value: "newvalue"}},
			assertions: func(t *testing.T, valuesFilePath string, err error) {
				require.NoError(t, err)
				content, readErr := os.ReadFile(valuesFilePath)
				require.NoError(t, readErr)
				assert.Equal(t, "title = 'newvalue'\n", string(content))
			},
		},
		{
			name:          "file does not exist",
			valuesContent: "",
			updates:       []inttoml.Update{{Key: "title", Value: "value"}},
			assertions: func(t *testing.T, valuesFilePath string, err error) {
				require.ErrorContains(t, err, "no such file or directory")
				require.NoFileExists(t, valuesFilePath)
			},
		},
		{
			name:          "empty changes",
			valuesContent: "title = \"value\"\n",
			updates:       []inttoml.Update{},
			assertions: func(t *testing.T, valuesFilePath string, err error) {
				require.NoError(t, err)
				content, readErr := os.ReadFile(valuesFilePath)
				require.NoError(t, readErr)
				assert.Equal(t, "title = \"value\"\n", string(content))
			},
		},
	}

	runner := &tomlUpdater{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			valuesFile := path.Join(workDir, "values.toml")

			if tt.valuesContent != "" {
				err := os.WriteFile(valuesFile, []byte(tt.valuesContent), 0o600)
				require.NoError(t, err)
			}

			err := runner.updateFile(workDir, path.Base(valuesFile), tt.updates)
			tt.assertions(t, valuesFile, err)
		})
	}
}

func Test_tomlUpdater_generateCommitMessage(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		updates    []builtin.TomlUpdate
		assertions func(*testing.T, string)
	}{
		{
			name: "no changes",
			path: "values.toml",
			assertions: func(t *testing.T, result string) {
				assert.Empty(t, result)
			},
		},
		{
			name:    "single change",
			path:    "values.toml",
			updates: []builtin.TomlUpdate{{Key: "package.version", Value: "1.2.3"}},
			assertions: func(t *testing.T, result string) {
				assert.Equal(t, "Updated values.toml\n\n- package.version: '1.2.3'", result)
			},
		},
		{
			name: "multiple changes",
			path: "Cargo.toml",
			updates: []builtin.TomlUpdate{
				{Key: "package.version", Value: "1.2.3"},
				{Key: "features.enabled", Value: true},
				{Key: "threshold", Value: 42},
			},
			assertions: func(t *testing.T, result string) {
				assert.Equal(
					t,
					"Updated Cargo.toml\n\n- package.version: '1.2.3'\n- features.enabled: true\n- threshold: 42",
					result,
				)
			},
		},
	}

	runner := &tomlUpdater{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runner.generateCommitMessage(tt.path, tt.updates)
			tt.assertions(t, result)
		})
	}
}
