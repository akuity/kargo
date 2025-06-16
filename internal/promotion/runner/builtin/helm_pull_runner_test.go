package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func TestHelmPullRunner_Name(t *testing.T) {
	runner := newHelmPullRunner(nil)
	assert.Equal(t, "helm-pull", runner.Name())
}

func TestHelmPullRunner_Run(t *testing.T) {
	tests := []struct {
		name        string
		config      builtin.HelmPullConfig
		freight     kargoapi.FreightCollection
		assertions  func(*testing.T, promotion.StepResult, error)
		setupMocks  func(*testing.T) credentials.Database
	}{
		{
			name: "invalid config",
			config: builtin.HelmPullConfig{
				// Missing required outPath
			},
			assertions: func(t *testing.T, result promotion.StepResult, err error) {
				require.Error(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
			},
			setupMocks: func(*testing.T) credentials.Database {
				return &credentials.FakeDB{}
			},
		},
		{
			name: "missing chart specification",
			config: builtin.HelmPullConfig{
				OutPath: "charts/",
				// Neither chart nor chartFromFreight specified
			},
			assertions: func(t *testing.T, result promotion.StepResult, err error) {
				require.Error(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
				assert.Contains(t, err.Error(), "either 'chart' or 'chartFromFreight' must be specified")
			},
			setupMocks: func(*testing.T) credentials.Database {
				return &credentials.FakeDB{}
			},
		},
		{
			name: "chart from freight not found",
			config: builtin.HelmPullConfig{
				OutPath: "charts/",
				ChartFromFreight: &builtin.HelmPullChartFromFreight{
					RepoURL: "oci://registry.example.com/charts",
					Name:    "nonexistent-chart",
				},
			},
			freight: kargoapi.FreightCollection{
				Freight: map[string]kargoapi.FreightReference{
					"warehouse/test": {
						Charts: []kargoapi.Chart{
							{
								RepoURL: "oci://registry.example.com/other-charts",
								Name:    "other-chart",
								Version: "1.0.0",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, result promotion.StepResult, err error) {
				require.Error(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
				assert.Contains(t, err.Error(), "chart not found in freight")
			},
			setupMocks: func(*testing.T) credentials.Database {
				return &credentials.FakeDB{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary work directory
			workDir := t.TempDir()

			// Set up mocks
			credsDB := tt.setupMocks(t)

			// Create runner
			runner := newHelmPullRunner(credsDB)

			// Create step context
			stepCtx := &promotion.StepContext{
				WorkDir: workDir,
				Config:  promotion.ConfigToMap(tt.config),
				Freight: tt.freight,
				Project: "test-project",
			}

			// Run the step
			result, err := runner.Run(context.Background(), stepCtx)

			// Assert results
			tt.assertions(t, result, err)
		})
	}
}

func TestHelmPullRunner_getChartDetails(t *testing.T) {
	runner := &helmPullRunner{}

	tests := []struct {
		name           string
		config         builtin.HelmPullConfig
		freight        kargoapi.FreightCollection
		expectedRepo   string
		expectedName   string
		expectedVer    string
		expectedError  string
	}{
		{
			name: "explicit chart configuration",
			config: builtin.HelmPullConfig{
				Chart: &builtin.HelmPullChart{
					RepoURL: "oci://registry.example.com/charts",
					Name:    "my-chart",
					Version: "1.2.3",
				},
			},
			expectedRepo: "oci://registry.example.com/charts",
			expectedName: "my-chart",
			expectedVer:  "1.2.3",
		},
		{
			name: "chart from freight - exact match",
			config: builtin.HelmPullConfig{
				ChartFromFreight: &builtin.HelmPullChartFromFreight{
					RepoURL: "oci://registry.example.com/charts",
					Name:    "my-chart",
				},
			},
			freight: kargoapi.FreightCollection{
				Freight: map[string]kargoapi.FreightReference{
					"warehouse/test": {
						Charts: []kargoapi.Chart{
							{
								RepoURL: "oci://registry.example.com/charts",
								Name:    "my-chart",
								Version: "1.2.3",
							},
						},
					},
				},
			},
			expectedRepo: "oci://registry.example.com/charts",
			expectedName: "my-chart",
			expectedVer:  "1.2.3",
		},
		{
			name: "chart from freight - repo match without name",
			config: builtin.HelmPullConfig{
				ChartFromFreight: &builtin.HelmPullChartFromFreight{
					RepoURL: "oci://registry.example.com/charts",
				},
			},
			freight: kargoapi.FreightCollection{
				Freight: map[string]kargoapi.FreightReference{
					"warehouse/test": {
						Charts: []kargoapi.Chart{
							{
								RepoURL: "oci://registry.example.com/charts",
								Name:    "any-chart",
								Version: "2.0.0",
							},
						},
					},
				},
			},
			expectedRepo: "oci://registry.example.com/charts",
			expectedName: "any-chart",
			expectedVer:  "2.0.0",
		},
		{
			name: "chart from freight - not found",
			config: builtin.HelmPullConfig{
				ChartFromFreight: &builtin.HelmPullChartFromFreight{
					RepoURL: "oci://registry.example.com/charts",
					Name:    "missing-chart",
				},
			},
			freight: kargoapi.FreightCollection{
				Freight: map[string]kargoapi.FreightReference{
					"warehouse/test": {
						Charts: []kargoapi.Chart{
							{
								RepoURL: "oci://registry.example.com/charts",
								Name:    "different-chart",
								Version: "1.0.0",
							},
						},
					},
				},
			},
			expectedError: "chart not found in freight",
		},
		{
			name: "neither chart nor chartFromFreight specified",
			config: builtin.HelmPullConfig{
				OutPath: "charts/",
			},
			expectedError: "either 'chart' or 'chartFromFreight' must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepCtx := &promotion.StepContext{
				Freight: tt.freight,
			}

			repoURL, name, version, err := runner.getChartDetails(stepCtx, tt.config)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedRepo, repoURL)
				assert.Equal(t, tt.expectedName, name)
				assert.Equal(t, tt.expectedVer, version)
			}
		})
	}
}

func TestHelmPullRunner_setupCredentials(t *testing.T) {
	runner := &helmPullRunner{
		credsDB: &credentials.FakeDB{
			GetFn: func(
				context.Context,
				string,
				credentials.Type,
				string,
			) (*credentials.Credentials, error) {
				return &credentials.Credentials{
					Username: "testuser",
					Password: "testpass",
				}, nil
			},
		},
	}

	tests := []struct {
		name        string
		repoURL     string
		chartName   string
		expectedURL string
	}{
		{
			name:        "OCI repository",
			repoURL:     "oci://registry.example.com/charts",
			chartName:   "my-chart",
			expectedURL: "oci://registry.example.com/charts/my-chart",
		},
		{
			name:        "classic repository",
			repoURL:     "https://charts.example.com",
			chartName:   "my-chart",
			expectedURL: "https://charts.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runner.setupCredentials(
				context.Background(),
				"test-project",
				tt.repoURL,
				tt.chartName,
				nil, // registry client not needed for this test
			)
			require.NoError(t, err)
		})
	}
}

func TestHelmPullRunner_Integration(t *testing.T) {
	// This test would require setting up a real or mock Helm registry
	// For now, we'll skip it but it shows the structure for integration testing
	t.Skip("Integration test requires Helm registry setup")

	workDir := t.TempDir()
	outPath := filepath.Join(workDir, "charts")

	config := builtin.HelmPullConfig{
		Chart: &builtin.HelmPullChart{
			RepoURL: "oci://ghcr.io/akuity/kargo-charts",
			Name:    "kargo",
			Version: "0.1.0",
		},
		OutPath: "charts/",
	}

	runner := newHelmPullRunner(&credentials.FakeDB{})
	stepCtx := &promotion.StepContext{
		WorkDir: workDir,
		Config:  promotion.ConfigToMap(config),
		Project: "test-project",
	}

	result, err := runner.Run(context.Background(), stepCtx)

	require.NoError(t, err)
	assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

	// Verify chart was extracted
	chartPath := filepath.Join(outPath, "kargo")
	_, err = os.Stat(chartPath)
	assert.NoError(t, err, "Chart directory should exist")

	// Verify Chart.yaml exists
	chartYamlPath := filepath.Join(chartPath, "Chart.yaml")
	_, err = os.Stat(chartYamlPath)
	assert.NoError(t, err, "Chart.yaml should exist")
}
