package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func TestHelmPullRunner_Name(t *testing.T) {
	r := newHelmPullRunner(nil)
	require.Equal(t, "helm-pull", r.Name())
}

func TestHelmPullRunner_Run(t *testing.T) {
	tests := []struct {
		name       string
		cfg        builtin.HelmPullConfig
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "invalid config",
			cfg:  builtin.HelmPullConfig{},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "validation error")
			},
		},
		{
			name: "valid config structure",
			cfg: builtin.HelmPullConfig{
				Path: "./charts",
				Charts: []builtin.HelmPullChart{
					{
						Name:       "test-chart",
						Repository: "https://charts.example.com",
						Version:    "1.0.0",
						OutPath:    "test-chart",
					},
				},
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				// This test will fail because we don't have a real chart repository
				// but it validates the configuration structure
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to pull chart")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			r := newHelmPullRunner(nil)
			result, err := r.Run(
				context.Background(),
				&promotion.StepContext{
					WorkDir: workDir,
					Config:  tt.cfg,
					Project: "test-project",
				},
			)
			tt.assertions(t, workDir, result, err)
		})
	}
}

func TestHelmPullRunner_isOCIRepository(t *testing.T) {
	tests := []struct {
		name     string
		repoURL  string
		expected bool
	}{
		{
			name:     "OCI repository",
			repoURL:  "oci://registry.example.com/charts/my-chart",
			expected: true,
		},
		{
			name:     "HTTP repository",
			repoURL:  "https://charts.example.com",
			expected: false,
		},
		{
			name:     "HTTPS repository",
			repoURL:  "https://charts.example.com",
			expected: false,
		},
		{
			name:     "empty URL",
			repoURL:  "",
			expected: false,
		},
		{
			name:     "short URL",
			repoURL:  "oci://",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOCIRepository(tt.repoURL)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestHelmPullRunner_run_directoryCreation(t *testing.T) {
	workDir := t.TempDir()
	r := &helmPullRunner{}

	cfg := builtin.HelmPullConfig{
		Path: "charts/subdir",
		Charts: []builtin.HelmPullChart{
			{
				Name:       "test-chart",
				Repository: "https://charts.example.com",
				Version:    "1.0.0",
				OutPath:    "test-chart",
			},
		},
	}

	// This will fail at the chart pulling stage, but should create directories
	_, err := r.run(
		context.Background(),
		&promotion.StepContext{
			WorkDir: workDir,
			Project: "test-project",
		},
		cfg,
	)

	// Verify that the base directory was created
	expectedDir := filepath.Join(workDir, "charts", "subdir")
	_, err = os.Stat(expectedDir)
	require.NoError(t, err, "Base directory should be created")
}
