package builtin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/io/fs"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_helmChartUpdater_convert(t *testing.T) {
	tests := []validationTestCase{
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
			name: "charts is null",
			config: promotion.Config{
				"path": "fake-path",
			},
		},
		{
			name: "charts is empty",
			config: promotion.Config{
				"charts": []promotion.Config{},
			},
			expectedProblems: []string{
				"charts: Array must have at least 1 items",
			},
		},
		{
			name: "repository not specified",
			config: promotion.Config{
				"charts": []promotion.Config{{}},
			},
			expectedProblems: []string{
				"charts.0: repository is required",
			},
		},
		{
			name: "repository is empty",
			config: promotion.Config{
				"charts": []promotion.Config{{
					"repository": "",
				}},
			},
			expectedProblems: []string{
				"charts.0.repository: String length must be greater than or equal to 1",
			},
		},
		{
			name: "name not specified",
			config: promotion.Config{
				"charts": []promotion.Config{{}},
			},
			expectedProblems: []string{
				"charts.0: name is required",
			},
		},
		{
			name: "name is empty",
			config: promotion.Config{
				"charts": []promotion.Config{{
					"name": "",
				}},
			},
			expectedProblems: []string{
				"charts.0.name: String length must be greater than or equal to 1",
			},
		},
		{
			name: "version not specified",
			config: promotion.Config{
				"charts": []promotion.Config{{}},
			},
			expectedProblems: []string{
				"charts.0: version is required",
			},
		},
		{
			name: "version is empty",
			config: promotion.Config{
				"charts": []promotion.Config{{
					"version": "",
				}},
			},
			expectedProblems: []string{
				"charts.0.version: String length must be greater than or equal to 1",
			},
		},
		{
			name: "valid kitchen sink",
			config: promotion.Config{
				"path": "fake-path",
				"charts": []promotion.Config{
					{
						"repository": "fake-repository",
						"name":       "fake-chart",
						"version":    "fake-version",
					},
				},
			},
		},
	}

	r := newHelmChartUpdater(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*helmChartUpdater)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_helmChartUpdater_run(t *testing.T) {
	tests := []struct {
		name            string
		context         *promotion.StepContext
		cfg             builtin.HelmUpdateChartConfig
		chartMetadata   *chart.Metadata
		setupRepository func(t *testing.T) (string, func())
		assertions      func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "successful run with HTTP repository",
			context: &promotion.StepContext{
				Project: "test-project",
				Freight: kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						"Warehouse/test-warehouse": {
							Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "test-warehouse"},
							Charts: []kargoapi.Chart{
								{RepoURL: "https://charts.example.com", Name: "examplechart", Version: "0.1.0"},
							},
						},
					},
				},
				FreightRequests: []kargoapi.FreightRequest{
					{
						Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "test-warehouse"},
					},
				},
			},
			cfg: builtin.HelmUpdateChartConfig{
				Path: "testchart",
				Charts: []builtin.Chart{
					{
						Repository: "https://charts.example.com",
						Name:       "examplechart",
						Version:    "0.1.0",
					},
				},
			},
			chartMetadata: &chart.Metadata{
				APIVersion: chart.APIVersionV2,
				Name:       "test-chart",
				Version:    "0.1.0",
				Dependencies: []*chart.Dependency{
					{
						Name:       "examplechart",
						Version:    ">=0.0.1",
						Repository: "https://charts.example.com",
					},
				},
			},
			setupRepository: func(t *testing.T) (string, func()) {
				httpRepositoryRoot := t.TempDir()
				require.NoError(t, fs.CopyFile(
					"../../../helm/testdata/charts/examplechart-0.1.0.tgz",
					filepath.Join(httpRepositoryRoot, "examplechart-0.1.0.tgz"),
				))
				httpRepository := httptest.NewServer(http.FileServer(http.Dir(httpRepositoryRoot)))

				repoIndex, err := repo.IndexDirectory(httpRepositoryRoot, httpRepository.URL)
				require.NoError(t, err)
				require.NoError(t, repoIndex.WriteFile(filepath.Join(httpRepositoryRoot, "index.yaml"), 0o600))

				return httpRepository.URL, httpRepository.Close
			},
			assertions: func(t *testing.T, tempDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{
					Status: kargoapi.PromotionStepStatusSucceeded,
					Output: map[string]any{
						"commitMessage": `Updated chart dependencies for testchart

- examplechart: 0.1.0`,
					},
				}, result)

				// Check if Chart.yaml was updated correctly
				updatedChartYaml, err := os.ReadFile(filepath.Join(tempDir, "testchart", "Chart.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(updatedChartYaml), "version: 0.1.0")

				// Check if the dependency was downloaded
				assert.FileExists(t, filepath.Join(tempDir, "testchart", "charts", "examplechart-0.1.0.tgz"))

				// Check if the Chart.lock file was created
				assert.FileExists(t, filepath.Join(tempDir, "testchart", "Chart.lock"))
			},
		},
		{
			name: "successful run with SemVer range",
			context: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.HelmUpdateChartConfig{
				Path: "testchart",
			},
			chartMetadata: &chart.Metadata{
				APIVersion: chart.APIVersionV2,
				Name:       "test-chart",
				Version:    "0.1.0",
				Dependencies: []*chart.Dependency{
					{
						Name:       "examplechart",
						Version:    ">=0.0.1",
						Repository: "https://charts.example.com",
					},
				},
			},
			setupRepository: func(t *testing.T) (string, func()) {
				httpRepositoryRoot := t.TempDir()
				require.NoError(t, fs.CopyFile(
					"../../../helm/testdata/charts/examplechart-0.1.0.tgz",
					filepath.Join(httpRepositoryRoot, "examplechart-0.1.0.tgz"),
				))
				httpRepository := httptest.NewServer(http.FileServer(http.Dir(httpRepositoryRoot)))

				repoIndex, err := repo.IndexDirectory(httpRepositoryRoot, httpRepository.URL)
				require.NoError(t, err)
				require.NoError(t, repoIndex.WriteFile(filepath.Join(httpRepositoryRoot, "index.yaml"), 0o600))

				return httpRepository.URL, httpRepository.Close
			},
			assertions: func(t *testing.T, tempDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{
					Status: kargoapi.PromotionStepStatusSucceeded,
					Output: map[string]any{
						"commitMessage": `Updated chart dependencies for testchart

- examplechart: 0.1.0`,
					},
				}, result)

				// Check if Chart.yaml was updated correctly
				updatedChartYaml, err := os.ReadFile(filepath.Join(tempDir, "testchart", "Chart.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(updatedChartYaml), "version: 0.1.0")

				// Check if the dependency was downloaded
				assert.FileExists(t, filepath.Join(tempDir, "testchart", "charts", "examplechart-0.1.0.tgz"))

				// Check if the Chart.lock file was created
				assert.FileExists(t, filepath.Join(tempDir, "testchart", "Chart.lock"))
			},
		},
	}

	runner := &helmChartUpdater{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up a fake Helm cache directory to ensure it is not used
			h := t.TempDir()
			t.Setenv(helmpath.CacheHomeEnvVar, h)

			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))

			stepCtx := tt.context
			stepCtx.WorkDir = t.TempDir()
			chartMetadata := tt.chartMetadata

			if tt.setupRepository != nil {
				repoURL, cleanup := tt.setupRepository(t)
				defer cleanup()

				// Update the repository URL in the configuration and the
				// chart metadata
				for i := range tt.cfg.Charts {
					tt.cfg.Charts[i].Repository = repoURL
				}
				for _, freight := range stepCtx.Freight.Freight {
					for i := range freight.Charts {
						freight.Charts[i].RepoURL = repoURL
					}
				}
				for _, dep := range chartMetadata.Dependencies {
					dep.Repository = repoURL
				}
			}

			if chartMetadata != nil {
				chartPath := filepath.Join(stepCtx.WorkDir, tt.cfg.Path)
				require.NoError(t, os.MkdirAll(chartPath, 0o700))

				b, err := yaml.Marshal(chartMetadata)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(chartPath, "Chart.yaml"), b, 0o600))
			}

			result, err := runner.run(context.Background(), stepCtx, tt.cfg)
			tt.assertions(t, stepCtx.WorkDir, result, err)

			// Assert that the Helm cache directory was not used
			assert.NoDirExistsf(t, helmpath.CachePath("repository"), "Helm home directory was used")
		})
	}
}

func Test_helmChartUpdater_generateCommitMessage(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		newVersions map[string]string
		assertions  func(*testing.T, string)
	}{
		{
			name:        "empty newVersions",
			path:        "charts/mychart",
			newVersions: map[string]string{},
			assertions: func(t *testing.T, got string) {
				assert.Empty(t, got)
			},
		},
		{
			name: "single update",
			path: "charts/mychart",
			newVersions: map[string]string{
				"chart1": "1.0.0 -> 1.1.0",
			},
			assertions: func(t *testing.T, got string) {
				assert.Contains(t, got, "Updated chart dependencies for charts/mychart")
				assert.Contains(t, got, "- chart1: 1.0.0 -> 1.1.0")
				assert.Equal(t, 2, strings.Count(got, "\n"))
			},
		},
		{
			name: "multiple updates",
			path: "charts/mychart",
			newVersions: map[string]string{
				"chart1": "1.0.0 -> 1.1.0",
				"chart2": "2.0.0 -> 2.1.0",
				"chart3": "3.0.0 -> 3.1.0",
			},
			assertions: func(t *testing.T, got string) {
				assert.Contains(t, got, "Updated chart dependencies for charts/mychart")
				assert.Contains(t, got, "- chart1: 1.0.0 -> 1.1.0")
				assert.Contains(t, got, "- chart2: 2.0.0 -> 2.1.0")
				assert.Contains(t, got, "- chart3: 3.0.0 -> 3.1.0")
				assert.Equal(t, 4, strings.Count(got, "\n"))
			},
		},
		{
			name: "updates and removals",
			path: "charts/mychart",
			newVersions: map[string]string{
				"chart1": "1.0.0 -> 1.1.0",
				"chart2": "",
				"chart3": "3.0.0 -> 3.1.0",
			},
			assertions: func(t *testing.T, got string) {
				assert.Contains(t, got, "Updated chart dependencies for charts/mychart")
				assert.Contains(t, got, "- chart1: 1.0.0 -> 1.1.0")
				assert.Contains(t, got, "- chart2: removed")
				assert.Contains(t, got, "- chart3: 3.0.0 -> 3.1.0")
				assert.Equal(t, 4, strings.Count(got, "\n"))
			},
		},
		{
			name: "only removals",
			path: "charts/mychart",
			newVersions: map[string]string{
				"chart1": "",
				"chart2": "",
			},
			assertions: func(t *testing.T, got string) {
				assert.Contains(t, got, "Updated chart dependencies for charts/mychart")
				assert.Contains(t, got, "- chart1: removed")
				assert.Contains(t, got, "- chart2: removed")
				assert.Equal(t, 3, strings.Count(got, "\n"))
			},
		},
		{
			name: "new additions",
			path: "charts/mychart",
			newVersions: map[string]string{
				"chart1": "1.0.0",
				"chart2": "2.0.0",
			},
			assertions: func(t *testing.T, got string) {
				assert.Contains(t, got, "Updated chart dependencies for charts/mychart")
				assert.Contains(t, got, "- chart1: 1.0.0")
				assert.Contains(t, got, "- chart2: 2.0.0")
				assert.Equal(t, 3, strings.Count(got, "\n"))
			},
		},
		{
			name: "mixed updates, removals, and additions",
			path: "charts/mychart",
			newVersions: map[string]string{
				"chart1": "1.0.0 -> 1.1.0",
				"chart2": "",
				"chart3": "3.0.0",
				"chart4": "4.0.0 -> 4.1.0",
			},
			assertions: func(t *testing.T, got string) {
				assert.Contains(t, got, "Updated chart dependencies for charts/mychart")
				assert.Contains(t, got, "- chart1: 1.0.0 -> 1.1.0")
				assert.Contains(t, got, "- chart2: removed")
				assert.Contains(t, got, "- chart3: 3.0.0")
				assert.Contains(t, got, "- chart4: 4.0.0 -> 4.1.0")
				assert.Equal(t, 5, strings.Count(got, "\n"))
			},
		},
	}

	runner := &helmChartUpdater{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runner.generateCommitMessage(tt.path, tt.newVersions)
			tt.assertions(t, got)
		})
	}
}
