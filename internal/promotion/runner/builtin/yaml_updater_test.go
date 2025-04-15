package builtin

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/yaml"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_yamlUpdater_validate(t *testing.T) {
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

	r := newYAMLUpdater()
	runner, ok := r.(*yamlUpdater)
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

func Test_yamlUpdater_run(t *testing.T) {
	tests := []struct {
		name       string
		stepCtx    *promotion.StepContext
		cfg        builtin.YAMLUpdateConfig
		files      map[string]string
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "successful run with updates",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.YAMLUpdateConfig{
				Path: "values.yaml",
				Updates: []builtin.YAMLUpdate{
					{Key: "image.tag", Value: "fake-tag"},
				},
			},
			files: map[string]string{
				"values.yaml": "image:\n  tag: oldtag\n",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{
					Status: kargoapi.PromotionStepPhaseSucceeded,
					Output: map[string]any{
						"commitMessage": "Updated values.yaml\n\n- image.tag: \"fake-tag\"",
					},
				}, result)
				content, err := os.ReadFile(path.Join(workDir, "values.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(content), "tag: fake-tag")
			},
		},
		{
			name: "failed to update file",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.YAMLUpdateConfig{
				Path: "non-existent/values.yaml",
				Updates: []builtin.YAMLUpdate{
					{Key: "image.tag", Value: "fake-tag"},
				},
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepPhaseErrored}, result)
				assert.Contains(t, err.Error(), "values file update failed")
			},
		},
	}

	runner := &yamlUpdater{}

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

func Test_yamlUpdater_updateValuesFile(t *testing.T) {
	tests := []struct {
		name          string
		valuesContent string
		updates       []yaml.Update
		assertions    func(*testing.T, string, error)
	}{
		{
			name:          "successful update",
			valuesContent: "key: value\n",
			updates:       []yaml.Update{{Key: "key", Value: "newvalue"}},
			assertions: func(t *testing.T, valuesFilePath string, err error) {
				require.NoError(t, err)

				require.FileExists(t, valuesFilePath)
				content, err := os.ReadFile(valuesFilePath)
				require.NoError(t, err)
				assert.Contains(t, string(content), "key: newvalue")
			},
		},
		{
			name:          "file does not exist",
			valuesContent: "",
			updates:       []yaml.Update{{Key: "key", Value: "value"}},
			assertions: func(t *testing.T, valuesFilePath string, err error) {
				require.ErrorContains(t, err, "no such file or directory")
				require.NoFileExists(t, valuesFilePath)
			},
		},
		{
			name:          "empty changes",
			valuesContent: "key: value\n",
			updates:       []yaml.Update{},
			assertions: func(t *testing.T, valuesFilePath string, err error) {
				require.NoError(t, err)
				require.FileExists(t, valuesFilePath)
				content, err := os.ReadFile(valuesFilePath)
				require.NoError(t, err)
				assert.Equal(t, "key: value\n", string(content))
			},
		},
	}

	runner := &yamlUpdater{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			valuesFile := path.Join(workDir, "values.yaml")

			if tt.valuesContent != "" {
				err := os.WriteFile(valuesFile, []byte(tt.valuesContent), 0o600)
				require.NoError(t, err)
			}

			err := runner.updateFile(workDir, path.Base(valuesFile), tt.updates)
			tt.assertions(t, valuesFile, err)
		})
	}
}

func Test_yamlUpdater_generateCommitMessage(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		updates    []builtin.YAMLUpdate
		assertions func(*testing.T, string)
	}{
		{
			name: "no changes",
			path: "values.yaml",
			assertions: func(t *testing.T, result string) {
				assert.Empty(t, result)
			},
		},
		{
			name:    "single change",
			path:    "values.yaml",
			updates: []builtin.YAMLUpdate{{Key: "image", Value: "repo/image:tag1"}},
			assertions: func(t *testing.T, result string) {
				assert.Equal(t, `Updated values.yaml

- image: "repo/image:tag1"`, result)
			},
		},
		{
			name: "multiple changes",
			path: "chart/values.yaml",
			updates: []builtin.YAMLUpdate{
				{Key: "image1", Value: "repo1/image1:tag1"},
				{Key: "image2", Value: "repo2/image2:tag2"},
			},
			assertions: func(t *testing.T, result string) {
				assert.Equal(t, `Updated chart/values.yaml

- image1: "repo1/image1:tag1"
- image2: "repo2/image2:tag2"`, result)
			},
		},
	}

	runner := &yamlUpdater{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runner.generateCommitMessage(tt.path, tt.updates)
			tt.assertions(t, result)
		})
	}
}
