package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runners/builtin"
)

func Test_jsonUpdater_validate(t *testing.T) {
	testCases := []struct {
		name             string
		config           promotion.Config
		expectedProblems []string
	}{
		{
			name:   "path is not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): path is required",
			},
		},
		{
			name: "path is empty",
			config: promotion.Config{
				"path": "",
			},
			expectedProblems: []string{
				"path: String length must be greater than or equal to 1",
			},
		},
		{
			name:   "updates is null",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): updates is required",
			},
		},
		{
			name: "updates is empty",
			config: promotion.Config{
				"updates": []promotion.Config{},
			},
			expectedProblems: []string{
				"updates: Array must have at least 1 items",
			},
		},
		{
			name: "key not specified",
			config: promotion.Config{
				"updates": []promotion.Config{{}},
			},
			expectedProblems: []string{
				"updates.0: key is required",
			},
		},
		{
			name: "key is empty",
			config: promotion.Config{
				"updates": []promotion.Config{{
					"key": "",
				}},
			},
			expectedProblems: []string{
				"updates.0.key: String length must be greater than or equal to 1",
			},
		},
		{
			name: "value not specified",
			config: promotion.Config{
				"updates": []promotion.Config{{}},
			},
			expectedProblems: []string{
				"updates.0: value is required",
			},
		},
		{
			name: "valid config",
			config: promotion.Config{
				"path": "fake-path",
				"updates": []promotion.Config{
					{
						"key":   "fake-key",
						"value": "fake-value",
					},
					{
						"key":   "another-fake-key",
						"value": "another-fake-value",
					},
				},
			},
		},
	}

	r := newJSONUpdater()
	runner, ok := r.(*jsonUpdater)
	require.True(t, ok)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := runner.validate(testCase.config)
			if len(testCase.expectedProblems) == 0 {
				require.NoError(t, err)
			} else {
				for _, problem := range testCase.expectedProblems {
					require.ErrorContains(t, err, problem)
				}
			}
		})
	}
}

func Test_jsonUpdater_updateValuesFile(t *testing.T) {
	tests := []struct {
		name          string
		valuesContent string
		changes       []builtin.JSONUpdate
		assertions    func(*testing.T, string, error)
	}{
		{
			name:          "successful update",
			valuesContent: `{"key": "value"}`,
			changes: []builtin.JSONUpdate{{
				Key:   "key",
				Value: "newvalue",
			}},
			assertions: func(t *testing.T, valuesFilePath string, err error) {
				require.NoError(t, err)

				require.FileExists(t, valuesFilePath)
				content, err := os.ReadFile(valuesFilePath)
				require.NoError(t, err)

				var result map[string]any
				err = json.Unmarshal(content, &result)
				require.NoError(t, err)
				assert.Equal(t, "newvalue", result["key"])
			},
		},
		{
			name:          "file does not exist",
			valuesContent: "",
			changes: []builtin.JSONUpdate{{
				Key:   "key",
				Value: "value",
			}},
			assertions: func(t *testing.T, valuesFilePath string, err error) {
				require.ErrorContains(t, err, "no such file or directory")
				require.NoFileExists(t, valuesFilePath)
			},
		},
		{
			name:          "empty changes",
			valuesContent: `{"key": "value"}`,
			changes:       []builtin.JSONUpdate{},
			assertions: func(t *testing.T, valuesFilePath string, err error) {
				require.NoError(t, err)
				require.FileExists(t, valuesFilePath)

				content, err := os.ReadFile(valuesFilePath)
				require.NoError(t, err)

				assert.JSONEq(t, `{"key": "value"}`, string(content))
			},
		},
		{
			name: "preserve formatting after update",
			valuesContent: `{
				"key": "value",
				"nested": {
					"key1": "value1",
					"key2": "value2"
				}
			}`,
			changes: []builtin.JSONUpdate{{
				Key:   "key",
				Value: "newvalue",
			}},
			assertions: func(t *testing.T, valuesFilePath string, err error) {
				require.NoError(t, err)

				require.FileExists(t, valuesFilePath)
				content, err := os.ReadFile(valuesFilePath)
				require.NoError(t, err)

				updatedContent := `{
					"key": "newvalue",
					"nested": {
						"key1": "value1",
						"key2": "value2"
					}
				}`

				assert.JSONEq(t, updatedContent, string(content))

				var result map[string]any
				err = json.Unmarshal(content, &result)
				require.NoError(t, err)
				assert.Equal(t, "newvalue", result["key"])
			},
		},
	}

	runner := &jsonUpdater{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			valuesFile := path.Join(workDir, "values.json")

			if tt.valuesContent != "" {
				err := os.WriteFile(valuesFile, []byte(tt.valuesContent), 0o600)
				require.NoError(t, err)
			}

			err := runner.updateFile(workDir, path.Base(valuesFile), tt.changes)
			tt.assertions(t, valuesFile, err)
		})
	}
}

func Test_jsonUpdater_generateCommitMessage(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		changes    []builtin.JSONUpdate
		assertions func(*testing.T, string)
	}{
		{
			name: "no changes",
			path: "values.json",
			assertions: func(t *testing.T, result string) {
				assert.Empty(t, result)
			},
		},
		{
			name: "single change",
			path: "values.json",
			changes: []builtin.JSONUpdate{{
				Key:   "image",
				Value: "repo/image:tag1",
			}},
			assertions: func(t *testing.T, result string) {
				assert.Equal(t, `Updated values.json

- image: "repo/image:tag1"`, result)
			},
		},
		{
			name: "multiple changes",
			path: "chart/values.json",
			changes: []builtin.JSONUpdate{
				{
					Key:   "image1",
					Value: "repo1/image1:tag1",
				},
				{
					Key:   "image2",
					Value: "repo2/image2:tag2",
				},
			},
			assertions: func(t *testing.T, result string) {
				assert.Equal(t, `Updated chart/values.json

- image1: "repo1/image1:tag1"
- image2: "repo2/image2:tag2"`, result)
			},
		},
	}

	runner := &jsonUpdater{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runner.generateCommitMessage(tt.path, tt.changes)
			tt.assertions(t, result)
		})
	}
}

func Test_jsonUpdater_run(t *testing.T) {
	tests := []struct {
		name       string
		stepCtx    *promotion.StepContext
		cfg        builtin.JSONUpdateConfig
		files      map[string]string
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "successful run with updates",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.JSONUpdateConfig{
				Path: "config.json",
				Updates: []builtin.JSONUpdate{
					{Key: "app.version", Value: "1.0.1"},
					{Key: "features.newFeature", Value: true},
					{Key: "threshold", Value: 100},
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
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{
					Status: kargoapi.PromotionPhaseSucceeded,
					Output: map[string]any{
						"commitMessage": "Updated config.json\n\n" +
							"- app.version: \"1.0.1\"\n" +
							"- features.newFeature: true\n" +
							"- threshold: 100",
					},
				}, result)
				content, err := os.ReadFile(path.Join(workDir, "config.json"))
				require.NoError(t, err)
				assert.Contains(t, string(content), `"version": "1.0.1"`)
				assert.Contains(t, string(content), `"newFeature": true`)
				assert.Contains(t, string(content), `"threshold":100`)
			},
		},
		{
			name: "failed to update file",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.JSONUpdateConfig{
				Path: "non-existent/config.json",
				Updates: []builtin.JSONUpdate{
					{Key: "app.version", Value: "1.0.1"},
				},
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionPhaseErrored}, result)
				assert.Contains(t, err.Error(), "JSON file update failed")
			},
		},
		{
			name: "no updates provided",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.JSONUpdateConfig{
				Path:    "config.json",
				Updates: []builtin.JSONUpdate{},
			},
			files: map[string]string{
				"config.json": `{
					"app": {
						"version": "1.0.0"
					}
				}`,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{
					Status: kargoapi.PromotionPhaseSucceeded,
				}, result)
				content, err := os.ReadFile(path.Join(workDir, "config.json"))
				require.NoError(t, err)
				assert.JSONEq(t, `{
					"app": {
						"version": "1.0.0"
					}
				}`, string(content))
			},
		},
		{
			name: "handle empty JSON file",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.JSONUpdateConfig{
				Path: "config.json",
				Updates: []builtin.JSONUpdate{
					{Key: "app.version", Value: "1.0.1"},
				},
			},
			files: map[string]string{
				"config.json": ``,
			},
			assertions: func(t *testing.T, workDir string, _ promotion.StepResult, err error) {
				assert.NoError(t, err)
				content, err := os.ReadFile(path.Join(workDir, "config.json"))
				require.NoError(t, err)
				assert.JSONEq(t, `{"app": {"version": "1.0.1"}}`, string(content))
			},
		},
		{
			name: "add new key to JSON",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.JSONUpdateConfig{
				Path: "config.json",
				Updates: []builtin.JSONUpdate{
					{Key: "settings.newKey", Value: "added"},
				},
			},
			files: map[string]string{
				"config.json": `{"settings": {}}`,
			},
			assertions: func(t *testing.T, workDir string, _ promotion.StepResult, err error) {
				assert.NoError(t, err)
				content, err := os.ReadFile(path.Join(workDir, "config.json"))
				require.NoError(t, err)
				assert.JSONEq(t, `{"settings": {"newKey": "added"}}`, string(content))
			},
		},
		{
			name: "update numeric value",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.JSONUpdateConfig{
				Path: "config.json",
				Updates: []builtin.JSONUpdate{
					{Key: "threshold", Value: 425},
				},
			},
			files: map[string]string{
				"config.json": `{"threshold": 10}`,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)
				content, err := os.ReadFile(path.Join(workDir, "config.json"))
				require.NoError(t, err)
				assert.JSONEq(t, `{"threshold": 425}`, string(content))
			},
		},
		{
			name: "update boolean value to false",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.JSONUpdateConfig{
				Path: "config.json",
				Updates: []builtin.JSONUpdate{
					{Key: "features.existingFeature", Value: false},
				},
			},
			files: map[string]string{
				"config.json": `{"features": {"existingFeature": true}}`,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, result.Status)
				content, err := os.ReadFile(path.Join(workDir, "config.json"))
				require.NoError(t, err)
				assert.JSONEq(t, `{"features": {"existingFeature": false}}`, string(content))
			},
		},
	}

	runner := &jsonUpdater{}

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
