package directives

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
	helmregistry "helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
	intyaml "github.com/akuity/kargo/internal/yaml"
)

func Test_helmChartUpdater_validate(t *testing.T) {
	testCases := []struct {
		name             string
		config           Config
		expectedProblems []string
	}{
		{
			name:   "path is not specified",
			config: Config{},
			expectedProblems: []string{
				"(root): path is required",
			},
		},
		{
			name: "path is empty",
			config: Config{
				"path": "",
			},
			expectedProblems: []string{
				"path: String length must be greater than or equal to 1",
			},
		},
		{
			name:   "charts is null",
			config: Config{},
			expectedProblems: []string{
				"(root): charts is required",
			},
		},
		{
			name: "charts is empty",
			config: Config{
				"charts": []Config{},
			},
			expectedProblems: []string{
				"charts: Array must have at least 1 items",
			},
		},
		{
			name: "repository not specified",
			config: Config{
				"charts": []Config{{}},
			},
			expectedProblems: []string{
				"charts.0: repository is required",
			},
		},
		{
			name: "repository is empty",
			config: Config{
				"charts": []Config{{
					"repository": "",
				}},
			},
			expectedProblems: []string{
				"charts.0.repository: String length must be greater than or equal to 1",
			},
		},
		{
			name: "name not specified",
			config: Config{
				"charts": []Config{{}},
			},
			expectedProblems: []string{
				"charts.0: name is required",
			},
		},
		{
			name: "name is empty",
			config: Config{
				"charts": []Config{{
					"name": "",
				}},
			},
			expectedProblems: []string{
				"charts.0.name: String length must be greater than or equal to 1",
			},
		},
		{
			name: "valid kitchen sink",
			config: Config{
				"path": "fake-path",
				"charts": []Config{
					{
						"repository": "fake-repository",
						"name":       "fake-chart-0",
					},
					{
						"repository": "fake-repository",
						"name":       "fake-chart-1",
						"version":    "",
					},
					{
						"repository": "fake-repository",
						"name":       "fake-chart-2",
						"fromOrigin": Config{
							"kind": Warehouse,
							"name": "fake-warehouse",
						},
					},
					{
						"repository": "fake-repository",
						"name":       "fake-chart-3",
						"version":    "",
						"fromOrigin": Config{
							"kind": Warehouse,
							"name": "fake-warehouse",
						},
					},
					{
						"repository": "fake-repository",
						"name":       "fake-chart-4",
						"version":    "fake-version",
					},
				},
			},
		},
	}

	r := newHelmChartUpdater()
	runner, ok := r.(*helmChartUpdater)
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

func Test_helmChartUpdater_runPromotionStep(t *testing.T) {
	tests := []struct {
		name            string
		context         *PromotionStepContext
		cfg             HelmUpdateChartConfig
		chartMetadata   *chart.Metadata
		setupRepository func(t *testing.T) (string, func())
		assertions      func(*testing.T, string, PromotionStepResult, error)
	}{
		{
			name: "successful run with HTTP repository",
			context: &PromotionStepContext{
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
			cfg: HelmUpdateChartConfig{
				Path: "testchart",
				Charts: []Chart{
					{
						Repository: "https://charts.example.com",
						Name:       "examplechart",
						FromOrigin: &ChartFromOrigin{
							Kind: "Warehouse",
							Name: "test-warehouse",
						},
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
				require.NoError(t, copyFile(
					"testdata/helm/charts/examplechart-0.1.0.tgz",
					filepath.Join(httpRepositoryRoot, "examplechart-0.1.0.tgz"),
				))
				httpRepository := httptest.NewServer(http.FileServer(http.Dir(httpRepositoryRoot)))

				repoIndex, err := repo.IndexDirectory(httpRepositoryRoot, httpRepository.URL)
				require.NoError(t, err)
				require.NoError(t, repoIndex.WriteFile(filepath.Join(httpRepositoryRoot, "index.yaml"), 0o600))

				return httpRepository.URL, httpRepository.Close
			},
			assertions: func(t *testing.T, tempDir string, result PromotionStepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, PromotionStepResult{
					Status: kargoapi.PromotionPhaseSucceeded,
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

			result, err := runner.runPromotionStep(context.Background(), stepCtx, tt.cfg)
			tt.assertions(t, stepCtx.WorkDir, result, err)
		})
	}
}

func Test_helmChartUpdater_processChartUpdates(t *testing.T) {
	tests := []struct {
		name              string
		objects           []client.Object
		context           *PromotionStepContext
		cfg               HelmUpdateChartConfig
		chartDependencies []chartDependency
		assertions        func(*testing.T, []intyaml.Update, error)
	}{
		{
			name: "finds chart update",
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-warehouse",
						Namespace: "test-project",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{
							{
								Chart: &kargoapi.ChartSubscription{
									RepoURL: "https://charts.example.com",
									Name:    "test-chart",
								},
							},
						},
					},
				},
			},
			context: &PromotionStepContext{
				Project: "test-project",
				Freight: kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						"Warehouse/test-warehouse": {
							Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "test-warehouse"},
							Charts: []kargoapi.Chart{
								{RepoURL: "https://charts.example.com", Name: "test-chart", Version: "1.0.0"},
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
			cfg: HelmUpdateChartConfig{
				Charts: []Chart{
					{Repository: "https://charts.example.com", Name: "test-chart"},
				},
			},
			chartDependencies: []chartDependency{
				{Repository: "https://charts.example.com", Name: "test-chart"},
			},
			assertions: func(t *testing.T, updates []intyaml.Update, err error) {
				assert.NoError(t, err)
				assert.Equal(
					t,
					[]intyaml.Update{{Key: "dependencies.0.version", Value: "1.0.0"}},
					updates,
				)
			},
		},
		{
			name: "chart not found",
			context: &PromotionStepContext{
				Project:         "test-project",
				Freight:         kargoapi.FreightCollection{},
				FreightRequests: []kargoapi.FreightRequest{},
			},
			cfg: HelmUpdateChartConfig{
				Charts: []Chart{
					{Repository: "https://charts.example.com", Name: "non-existent-chart"},
				},
			},
			chartDependencies: []chartDependency{
				{Repository: "https://charts.example.com", Name: "non-existent-chart"},
			},
			assertions: func(t *testing.T, _ []intyaml.Update, err error) {
				assert.ErrorContains(t, err, "not found in referenced Freight")
			},
		},
		{
			name: "multiple charts, one not found",
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-warehouse",
						Namespace: "test-project",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{
							{
								Chart: &kargoapi.ChartSubscription{
									RepoURL: "https://charts.example.com",
									Name:    "chart1",
								},
							},
						},
					},
				},
			},
			context: &PromotionStepContext{
				Project: "test-project",
				Freight: kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						"Warehouse/test-warehouse": {
							Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "test-warehouse"},
							Charts: []kargoapi.Chart{
								{RepoURL: "https://charts.example.com", Name: "chart1", Version: "1.0.0"},
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
			cfg: HelmUpdateChartConfig{
				Charts: []Chart{
					{Repository: "https://charts.example.com", Name: "chart1"},
					{Repository: "https://charts.example.com", Name: "chart2"},
				},
			},
			chartDependencies: []chartDependency{
				{Repository: "https://charts.example.com", Name: "chart1"},
				{Repository: "https://charts.example.com", Name: "chart2"},
			},
			assertions: func(t *testing.T, _ []intyaml.Update, err error) {
				assert.ErrorContains(t, err, "not found in referenced Freight")
			},
		},
		{
			name: "chart with FromOrigin specified",
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-warehouse",
						Namespace: "test-project",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{
							{
								Chart: &kargoapi.ChartSubscription{
									RepoURL: "https://charts.example.com",
									Name:    "origin-chart",
								},
							},
						},
					},
				},
			},
			context: &PromotionStepContext{
				Project: "test-project",
				Freight: kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						"Warehouse/test-warehouse": {
							Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "test-warehouse"},
							Charts: []kargoapi.Chart{
								{RepoURL: "https://charts.example.com", Name: "origin-chart", Version: "2.0.0"},
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
			cfg: HelmUpdateChartConfig{
				Charts: []Chart{
					{
						Repository: "https://charts.example.com",
						Name:       "origin-chart",
						FromOrigin: &ChartFromOrigin{Kind: "Warehouse", Name: "test-warehouse"},
					},
				},
			},
			chartDependencies: []chartDependency{
				{Repository: "https://charts.example.com", Name: "origin-chart"},
			},
			assertions: func(t *testing.T, updates []intyaml.Update, err error) {
				assert.NoError(t, err)
				assert.Equal(
					t,
					[]intyaml.Update{{Key: "dependencies.0.version", Value: "2.0.0"}},
					updates,
				)
			},
		},
		{
			name: "chart with version specified",
			context: &PromotionStepContext{
				Project: "test-project",
			},
			cfg: HelmUpdateChartConfig{
				Charts: []Chart{
					{
						Repository: "https://charts.example.com",
						Name:       "origin-chart",
						Version:    "fake-version",
					},
				},
			},
			chartDependencies: []chartDependency{
				{Repository: "https://charts.example.com", Name: "origin-chart"},
			},
			assertions: func(t *testing.T, updates []intyaml.Update, err error) {
				assert.NoError(t, err)
				assert.Equal(
					t,
					[]intyaml.Update{{Key: "dependencies.0.version", Value: "fake-version"}},
					updates,
				)
			},
		},
		{
			name: "update specified for non-existent chart dependency",
			context: &PromotionStepContext{
				Project: "test-project",
			},
			cfg: HelmUpdateChartConfig{
				Charts: []Chart{
					{
						Repository: "https://charts.example.com",
						Name:       "origin-chart",
						Version:    "fake-version",
					},
				},
			},
			chartDependencies: []chartDependency{},
			assertions: func(t *testing.T, _ []intyaml.Update, err error) {
				assert.ErrorContains(t, err, "no dependency in Chart.yaml matched update")
			},
		},
	}

	runner := &helmChartUpdater{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))

			stepCtx := tt.context
			stepCtx.KargoClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build()

			updates, err := runner.processChartUpdates(context.Background(), stepCtx, tt.cfg, tt.chartDependencies)
			tt.assertions(t, updates, err)
		})
	}
}

func Test_helmChartUpdater_updateDependencies(t *testing.T) {
	runner := &helmChartUpdater{}

	t.Run("updates dependencies", func(t *testing.T) {
		// Set up the HTTP repository
		httpRepositoryRoot := t.TempDir()
		require.NoError(t, copyFile(
			"testdata/helm/charts/examplechart-0.1.0.tgz",
			filepath.Join(httpRepositoryRoot, "examplechart-0.1.0.tgz"),
		))
		httpRepository := httptest.NewServer(http.FileServer(http.Dir(httpRepositoryRoot)))
		t.Cleanup(httpRepository.Close)

		repoIndex, err := repo.IndexDirectory(httpRepositoryRoot, httpRepository.URL)
		require.NoError(t, err)
		require.NoError(t, repoIndex.WriteFile(filepath.Join(httpRepositoryRoot, "index.yaml"), 0o600))

		// Set up the OCI registry
		ociRegistry := httptest.NewServer(registry.New())
		t.Cleanup(ociRegistry.Close)

		ociClient, err := helm.NewRegistryClient(t.TempDir())
		require.NoError(t, err)

		b, err := os.ReadFile("testdata/helm/charts/demo-0.1.0.tgz")
		require.NoError(t, err)
		repositoryRef := strings.TrimPrefix(ociRegistry.URL, "http://")
		_, err = ociClient.Push(b, repositoryRef+"/demo:0.1.0")
		require.NoError(t, err)

		// Prepare the dependant chart with a Chart.yaml file
		chartPath := t.TempDir()
		metadata := chart.Metadata{
			APIVersion: chart.APIVersionV2,
			Name:       "test-chart",
			Version:    "0.1.0",
			Dependencies: []*chart.Dependency{
				{
					Name:       "examplechart",
					Version:    "0.1.0",
					Repository: httpRepository.URL,
				},
				{
					Name:       "demo",
					Version:    "0.1.0",
					Repository: "oci://" + repositoryRef,
				},
			},
		}
		b, err = yaml.Marshal(metadata)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(chartPath, "Chart.yaml"), b, 0o600))

		// Run the promotion step and assert the dependencies are updated
		newVersions, err := runner.updateDependencies(
			context.Background(),
			&PromotionStepContext{},
			t.TempDir(),
			chartPath,
			nil,
		)
		require.NoError(t, err)
		require.DirExists(t, filepath.Join(chartPath, "charts"))
		assert.FileExists(t, filepath.Join(chartPath, "charts", "examplechart-0.1.0.tgz"))
		assert.FileExists(t, filepath.Join(chartPath, "charts", "demo-0.1.0.tgz"))
		assert.Equal(t, map[string]string{
			"examplechart": "0.1.0",
			"demo":         "0.1.0",
		}, newVersions)
	})

	t.Run("updates dependencies with credentials", func(t *testing.T) {
		// Set up the OCI registry
		ociRegistry := newAuthRegistryServer("username", "password")
		ociRegistry.Start()
		t.Cleanup(ociRegistry.Close)

		ociClient, err := helm.NewRegistryClient(t.TempDir())
		require.NoError(t, err)

		b, err := os.ReadFile("testdata/helm/charts/demo-0.1.0.tgz")
		require.NoError(t, err)

		repositoryRef := strings.TrimPrefix(ociRegistry.URL, "http://")
		require.NoError(t, ociClient.Login(
			repositoryRef,
			helmregistry.LoginOptBasicAuth("username", "password"),
		))
		_, err = ociClient.Push(b, repositoryRef+"/demo:0.1.0")
		require.NoError(t, err)

		// Prepare the dependant chart with a Chart.yaml file
		chartPath := t.TempDir()
		metadata := chart.Metadata{
			APIVersion: chart.APIVersionV2,
			Name:       "test-chart",
			Version:    "0.1.0",
			Dependencies: []*chart.Dependency{
				{
					Name:       "demo",
					Version:    "0.1.0",
					Repository: "oci://" + repositoryRef,
				},
			},
		}
		b, err = yaml.Marshal(metadata)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(chartPath, "Chart.yaml"), b, 0o600))

		// Prepare the credentials database
		credentialsDB := &credentials.FakeDB{
			GetFn: func(context.Context, string, credentials.Type, string) (credentials.Credentials, bool, error) {
				return credentials.Credentials{
					Username: "username",
					Password: "password",
				}, true, nil
			},
		}

		// Run the promotion step and assert the dependency is updated
		newVersions, err := runner.updateDependencies(context.Background(), &PromotionStepContext{
			CredentialsDB: credentialsDB,
		}, t.TempDir(), chartPath, []chartDependency{
			{
				Name:       "demo",
				Repository: "oci://" + repositoryRef,
			},
		})
		require.NoError(t, err)
		require.DirExists(t, filepath.Join(chartPath, "charts"))
		assert.FileExists(t, filepath.Join(chartPath, "charts", "demo-0.1.0.tgz"))
		assert.Equal(t, map[string]string{
			"demo": "0.1.0",
		}, newVersions)
	})

	tests := []struct {
		name              string
		credentialsDB     credentials.Database
		chartDependencies []chartDependency
		assertions        func(*testing.T, string, string, error)
	}{
		{
			name: "error loading dependency credentials",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(context.Context, string, credentials.Type, string) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false, fmt.Errorf("something went wrong")
				},
			},
			chartDependencies: []chartDependency{
				{
					Name:       "dep1",
					Repository: "https://charts.example.com",
				},
			},
			assertions: func(t *testing.T, _, _ string, err error) {
				require.ErrorContains(t, err, "failed to obtain credentials")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "writes repository file",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(context.Context, string, credentials.Type, string) (credentials.Credentials, bool, error) {
					return credentials.Credentials{
						Username: "username",
						Password: "password",
					}, true, nil
				},
			},
			chartDependencies: []chartDependency{
				{
					Name:       "dep1",
					Repository: "https://charts.example.com",
				},
			},
			assertions: func(t *testing.T, helmHome, _ string, _ error) {
				repoFilePath := filepath.Join(helmHome, "repositories.yaml")
				require.FileExists(t, repoFilePath)

				repoFile, err := repo.LoadFile(filepath.Join(helmHome, "repositories.yaml"))
				require.NoError(t, err)
				require.Len(t, repoFile.Repositories, 1)
				assert.Equal(t, "https://charts.example.com", repoFile.Repositories[0].URL)
			},
		},
		{
			name: "error updating dependencies on empty chart",
			assertions: func(t *testing.T, _ string, _ string, err error) {
				require.ErrorContains(t, err, "failed to update chart dependencies")
				require.ErrorContains(t, err, "Chart.yaml file is missing")
			},
		},
		{
			name: "error validating file dependency",
			chartDependencies: []chartDependency{
				{
					Name:       "dep1",
					Repository: "file:///absolute/path",
				},
			},
			assertions: func(t *testing.T, _, _ string, err error) {
				assert.ErrorContains(t, err, "dependency path must be relative")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helmHome, chartPath := t.TempDir(), t.TempDir()
			_, err := runner.updateDependencies(context.Background(), &PromotionStepContext{
				CredentialsDB: tt.credentialsDB,
			}, helmHome, chartPath, tt.chartDependencies)
			tt.assertions(t, helmHome, chartPath, err)
		})
	}
}

func Test_helmChartUpdater_validateFileDependency(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) (workDir, chartPath, dependencyPath string)
		assertions func(t *testing.T, err error)
	}{
		{
			name: "valid file dependency",
			setup: func(t *testing.T) (string, string, string) {
				workDir := absoluteTempDir(t)

				chartPath := filepath.Join(workDir, "chart")
				require.NoError(t, os.Mkdir(chartPath, 0o700))

				dependencyPath := filepath.Join(workDir, "valid-dep")
				require.NoError(t, os.Mkdir(dependencyPath, 0o700))

				return workDir, chartPath, "../valid-dep"
			},
			assertions: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "dependency outside work directory",
			setup: func(t *testing.T) (string, string, string) {
				tmpDir := absoluteTempDir(t)

				workDir := filepath.Join(tmpDir, "work")
				chartPath := filepath.Join(workDir, "chart")
				require.NoError(t, os.MkdirAll(chartPath, 0o700))

				dependencyPath := filepath.Join(tmpDir, "outside-dep")
				require.NoError(t, os.Mkdir(dependencyPath, 0o700))

				return workDir, chartPath, "../../outside-dep"
			},
			assertions: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "dependency path is outside of the work directory")
			},
		},
		{
			name: "valid symlink within work directory",
			setup: func(t *testing.T) (string, string, string) {
				workDir := absoluteTempDir(t)

				chartPath := filepath.Join(workDir, "chart")
				require.NoError(t, os.Mkdir(chartPath, 0o700))

				dependencyPath := filepath.Join(workDir, "valid-dep")
				require.NoError(t, os.Mkdir(dependencyPath, 0o700))

				symlinkPath := filepath.Join(workDir, "symlink-dep")
				require.NoError(t, os.Symlink(dependencyPath, symlinkPath))

				return workDir, chartPath, "../symlink-dep"
			},
			assertions: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "symlink pointing outside work directory",
			setup: func(t *testing.T) (string, string, string) {
				tmpDir := absoluteTempDir(t)

				workDir := filepath.Join(tmpDir, "work")
				chartPath := filepath.Join(workDir, "chart")
				require.NoError(t, os.MkdirAll(chartPath, 0o700))

				dependencyPath := filepath.Join(tmpDir, "outside-dep")
				require.NoError(t, os.Mkdir(dependencyPath, 0o700))

				symlinkPath := filepath.Join(workDir, "symlink-dep")
				require.NoError(t, os.Symlink(dependencyPath, symlinkPath))

				return workDir, chartPath, "../symlink-dep"
			},
			assertions: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "dependency path is outside of the work directory")
			},
		},
		{
			name: "non-existent dependency path",
			setup: func(t *testing.T) (string, string, string) {
				workDir := absoluteTempDir(t)
				chartPath := filepath.Join(workDir, "chart")
				return workDir, chartPath, "../non-existent-dep"
			},
			assertions: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "failed to resolve dependency path: lstat non-existent-dep")
			},
		},
		{
			name: "absolute dependency path",
			setup: func(t *testing.T) (string, string, string) {
				workDir := absoluteTempDir(t)
				chartPath := filepath.Join(workDir, "chart")
				return workDir, chartPath, "/absolute-dep"
			},
			assertions: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "dependency path must be relative")
			},
		},
	}

	runner := &helmChartUpdater{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir, chartPath, dependencyPath := tt.setup(t)
			err := runner.validateFileDependency(workDir, chartPath, dependencyPath)
			tt.assertions(t, err)
		})
	}
}

func Test_helmChartUpdater_loadDependencyCredentials(t *testing.T) {
	tests := []struct {
		name              string
		credentialsDB     credentials.Database
		repositoryFile    *repo.File
		newRegistryClient func(*testing.T) (string, *helmregistry.Client)
		newOCIServer      func(*testing.T) string
		buildDependencies func(string) []chartDependency
		assertions        func(*testing.T, string, string, *repo.File, error)
	}{
		{
			name: "HTTP credentials",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(context.Context, string, credentials.Type, string) (credentials.Credentials, bool, error) {
					return credentials.Credentials{
						Username: "username",
						Password: "password",
					}, true, nil
				},
			},
			repositoryFile: repo.NewFile(),
			newRegistryClient: func(*testing.T) (string, *helmregistry.Client) {
				return "", nil
			},
			buildDependencies: func(string) []chartDependency {
				return []chartDependency{
					{
						Name:       "dep1",
						Repository: "https://charts.example.com",
					},
				}
			},
			assertions: func(t *testing.T, _, _ string, repositoryFile *repo.File, err error) {
				require.NoError(t, err)
				require.Len(t, repositoryFile.Repositories, 1)
				assert.Equal(t, "https://charts.example.com", repositoryFile.Repositories[0].URL)
				assert.Equal(t, "username", repositoryFile.Repositories[0].Username)
				assert.Equal(t, "password", repositoryFile.Repositories[0].Password)
			},
		},
		{
			name: "OCI credentials",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(context.Context, string, credentials.Type, string) (credentials.Credentials, bool, error) {
					return credentials.Credentials{
						Username: "username",
						Password: "password",
					}, true, nil
				},
			},
			buildDependencies: func(registryURL string) []chartDependency {
				return []chartDependency{
					{
						Name:       "dep1",
						Repository: "oci://" + registryURL,
					},
				}
			},
			newRegistryClient: func(t *testing.T) (string, *helmregistry.Client) {
				home := t.TempDir()
				c, err := helm.NewRegistryClient(home)
				require.NoError(t, err)
				return home, c
			},
			newOCIServer: func(t *testing.T) string {
				srv := newAuthRegistryServer("username", "password")
				srv.Start()
				t.Cleanup(srv.Close)
				return srv.URL
			},
			assertions: func(t *testing.T, home, registryURL string, _ *repo.File, err error) {
				require.NoError(t, err)

				require.FileExists(t, filepath.Join(home, ".docker", "config.json"))
				b, _ := os.ReadFile(filepath.Join(home, ".docker", "config.json"))
				assert.Contains(t, string(b), registryURL)
			},
		},
		{
			name: "multiple dependencies",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(context.Context, string, credentials.Type, string) (credentials.Credentials, bool, error) {
					return credentials.Credentials{
						Username: "username",
						Password: "password",
					}, true, nil
				},
			},
			repositoryFile: repo.NewFile(),
			newRegistryClient: func(*testing.T) (string, *helmregistry.Client) {
				return "", nil
			},
			buildDependencies: func(string) []chartDependency {
				return []chartDependency{
					{
						Name:       "dep1",
						Repository: "https://charts.example.com",
					},
					{
						Name:       "dep2",
						Repository: "https://example.com/repository/",
					},
				}
			},
			assertions: func(t *testing.T, _, _ string, repositoryFile *repo.File, err error) {
				require.NoError(t, err)
				require.Len(t, repositoryFile.Repositories, 2)
				assert.Equal(t, "https://charts.example.com", repositoryFile.Repositories[0].URL)
				assert.Equal(t, "username", repositoryFile.Repositories[0].Username)
				assert.Equal(t, "password", repositoryFile.Repositories[0].Password)
				assert.Equal(t, "https://example.com/repository/", repositoryFile.Repositories[1].URL)
				assert.Equal(t, "username", repositoryFile.Repositories[1].Username)
				assert.Equal(t, "password", repositoryFile.Repositories[1].Password)
			},
		},
		{
			name: "error getting credentials",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(context.Context, string, credentials.Type, string) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false, fmt.Errorf("something went wrong")
				},
			},
			buildDependencies: func(string) []chartDependency {
				return []chartDependency{
					{
						Name:       "dep1",
						Repository: "https://charts.example.com",
					},
				}
			},
			newRegistryClient: func(*testing.T) (string, *helmregistry.Client) {
				return "", nil
			},
			assertions: func(t *testing.T, _, _ string, _ *repo.File, err error) {
				require.ErrorContains(t, err, "failed to obtain credentials")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "unauthenticated repository",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(context.Context, string, credentials.Type, string) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false, nil
				},
			},
			buildDependencies: func(string) []chartDependency {
				return []chartDependency{
					{
						Name:       "dep1",
						Repository: "https://charts.example.com",
					},
				}
			},
			newRegistryClient: func(*testing.T) (string, *helmregistry.Client) {
				return "", nil
			},
			assertions: func(t *testing.T, _, _ string, _ *repo.File, err error) {
				require.NoError(t, err)
			},
		},
	}

	runner := &helmChartUpdater{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helmHome, registryClient := tt.newRegistryClient(t)

			var registryURL string
			if tt.newOCIServer != nil {
				registryURL = tt.newOCIServer(t)
			}

			dependencies := tt.buildDependencies(registryURL)

			err := runner.loadDependencyCredentials(
				context.Background(),
				tt.credentialsDB,
				registryClient,
				tt.repositoryFile,
				"fake-project",
				dependencies,
			)
			tt.assertions(t, helmHome, registryURL, tt.repositoryFile, err)
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

func Test_normalizeChartReference(t *testing.T) {
	tests := []struct {
		name            string
		repoURL         string
		chartName       string
		expectedRepoURL string
		expectedChart   string
	}{
		{
			name:            "OCI repository",
			repoURL:         "oci://example.com/charts",
			chartName:       "mychart",
			expectedRepoURL: "oci://example.com/charts/mychart",
			expectedChart:   "",
		},
		{
			name:            "OCI repository with trailing slash",
			repoURL:         "oci://example.com/charts/",
			chartName:       "mychart",
			expectedRepoURL: "oci://example.com/charts/mychart",
			expectedChart:   "",
		},
		{
			name:            "HTTP repository",
			repoURL:         "https://charts.example.com",
			chartName:       "mychart",
			expectedRepoURL: "https://charts.example.com",
			expectedChart:   "mychart",
		},
		{
			name:            "HTTP repository with path",
			repoURL:         "https://example.com/charts",
			chartName:       "mychart",
			expectedRepoURL: "https://example.com/charts",
			expectedChart:   "mychart",
		},
		{
			name:            "local path",
			repoURL:         "./charts",
			chartName:       "mychart",
			expectedRepoURL: "./charts",
			expectedChart:   "mychart",
		},
		{
			name:            "empty repo URL",
			repoURL:         "",
			chartName:       "mychart",
			expectedRepoURL: "",
			expectedChart:   "mychart",
		},
		{
			name:            "empty chart name",
			repoURL:         "https://charts.example.com",
			chartName:       "",
			expectedRepoURL: "https://charts.example.com",
			expectedChart:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoURL, chart := normalizeChartReference(tt.repoURL, tt.chartName)
			assert.Equal(t, tt.expectedRepoURL, repoURL)
			assert.Equal(t, tt.expectedChart, chart)
		})
	}
}

func Test_readChartDependencies(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*testing.T) string
		assertions func(*testing.T, []chartDependency, error)
	}{
		{
			name: "valid chart.yaml",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()

				const chartYAML = `---
apiVersion: v2
name: test-chart
version: 0.1.0
dependencies:
- name: dep1
  version: 1.0.0
  repository: https://charts.example.com
- name: dep2
  version: 2.0.0
  repository: oci://registry.example.com/charts
`
				chartPath := filepath.Join(tmpDir, "Chart.yaml")
				require.NoError(t, os.WriteFile(chartPath, []byte(chartYAML), 0o600))

				return chartPath
			},
			assertions: func(t *testing.T, dependencies []chartDependency, err error) {
				require.NoError(t, err)
				assert.Len(t, dependencies, 2)

				assert.Equal(t, "dep1", dependencies[0].Name)
				assert.Equal(t, "https://charts.example.com", dependencies[0].Repository)
				assert.Equal(t, "dep2", dependencies[1].Name)
				assert.Equal(t, "oci://registry.example.com/charts", dependencies[1].Repository)
			},
		},
		{
			name: "invalid Chart.yaml",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()

				const chartYAML = `---
this is not a valid chart.yaml
`
				chartPath := filepath.Join(tmpDir, "Chart.yaml")
				require.NoError(t, os.WriteFile(chartPath, []byte(chartYAML), 0o600))

				return chartPath
			},
			assertions: func(t *testing.T, dependencies []chartDependency, err error) {
				require.ErrorContains(t, err, "failed to unmarshal")
				assert.Nil(t, dependencies)
			},
		},
		{
			name: "missing Chart.yaml",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "Chart.yaml")
			},
			assertions: func(t *testing.T, dependencies []chartDependency, err error) {
				require.ErrorContains(t, err, "failed to read file")
				assert.Nil(t, dependencies)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chartPath := tt.setup(t)
			dependencies, err := readChartDependencies(chartPath)
			tt.assertions(t, dependencies, err)
		})
	}
}

func Test_readChartLock(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*testing.T) string
		assertions func(*testing.T, map[string]string, error)
	}{
		{
			name: "valid Chart.lock",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()

				const chartLock = `---
dependencies:
- name: dep1
  version: 1.0.0
  repository: https://charts.example.com
- name: dep2
  version: 2.0.0
  repository: oci://registry.example.com/charts
`
				lockPath := filepath.Join(tmpDir, "Chart.lock")
				require.NoError(t, os.WriteFile(lockPath, []byte(chartLock), 0o600))
				return lockPath
			},
			assertions: func(t *testing.T, charts map[string]string, err error) {
				require.NoError(t, err)

				assert.Len(t, charts, 2)
				assert.Equal(t, "1.0.0", charts["dep1"])
				assert.Equal(t, "2.0.0", charts["dep2"])
			},
		},
		{
			name: "invalid Chart.lock",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()

				const chartLock = `---
this is not a valid Chart.lock
`
				lockPath := filepath.Join(tmpDir, "Chart.lock")
				require.NoError(t, os.WriteFile(lockPath, []byte(chartLock), 0o600))
				return lockPath
			},
			assertions: func(t *testing.T, charts map[string]string, err error) {
				require.ErrorContains(t, err, "failed to parse Chart.lock")
				assert.Empty(t, charts)
			},
		},
		{
			name: "missing Chart.lock",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "Chart.lock")
			},
			assertions: func(t *testing.T, charts map[string]string, err error) {
				require.NoError(t, err)
				assert.Empty(t, charts)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chartPath := tt.setup(t)
			charts, err := readChartLock(chartPath)
			tt.assertions(t, charts, err)
		})
	}
}

func Test_compareChartVersions(t *testing.T) {
	tests := []struct {
		name   string
		before map[string]string
		after  map[string]string
		want   map[string]string
	}{
		{
			name:   "No changes",
			before: map[string]string{"chart1": "1.0.0", "chart2": "2.0.0"},
			after:  map[string]string{"chart1": "1.0.0", "chart2": "2.0.0"},
			want:   map[string]string{},
		},
		{
			name:   "version update",
			before: map[string]string{"chart1": "1.0.0", "chart2": "2.0.0"},
			after:  map[string]string{"chart1": "1.1.0", "chart2": "2.0.0"},
			want:   map[string]string{"chart1": "1.0.0 -> 1.1.0"},
		},
		{
			name:   "new chart added",
			before: map[string]string{"chart1": "1.0.0"},
			after:  map[string]string{"chart1": "1.0.0", "chart2": "2.0.0"},
			want:   map[string]string{"chart2": "2.0.0"},
		},
		{
			name:   "chart removed",
			before: map[string]string{"chart1": "1.0.0", "chart2": "2.0.0"},
			after:  map[string]string{"chart1": "1.0.0"},
			want:   map[string]string{"chart2": ""},
		},
		{
			name:   "multiple changes",
			before: map[string]string{"chart1": "1.0.0", "chart2": "2.0.0", "chart3": "3.0.0"},
			after:  map[string]string{"chart1": "1.1.0", "chart2": "2.0.0", "chart4": "4.0.0"},
			want:   map[string]string{"chart1": "1.0.0 -> 1.1.0", "chart3": "", "chart4": "4.0.0"},
		},
		{
			name:   "empty before",
			before: map[string]string{},
			after:  map[string]string{"chart1": "1.0.0", "chart2": "2.0.0"},
			want:   map[string]string{"chart1": "1.0.0", "chart2": "2.0.0"},
		},
		{
			name:   "empty after",
			before: map[string]string{"chart1": "1.0.0", "chart2": "2.0.0"},
			after:  map[string]string{},
			want:   map[string]string{"chart1": "", "chart2": ""},
		},
		{
			name:   "both empty",
			before: map[string]string{},
			after:  map[string]string{},
			want:   map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, compareChartVersions(tt.before, tt.after))
		})
	}
}

func Test_checkSymlinks(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) (root string, dirPath string)
		maxDepth   int
		assertions func(*testing.T, map[string]struct{}, error)
	}{
		{
			name: "no symlinks",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("content"), 0o600))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.NoError(t, err)
				assert.Len(t, visited, 1)
			},
		},
		{
			name: "symlink within root",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("content"), 0o600))
				require.NoError(t, os.Symlink(filepath.Join(root, "file.txt"), filepath.Join(root, "link.txt")))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.NoError(t, err)
				assert.Len(t, visited, 2)
			},
		},
		{
			name: "symlink outside root",
			setup: func(t *testing.T) (string, string) {
				dir := absoluteTempDir(t)
				require.NoError(t, os.WriteFile(filepath.Join(dir, "outside.txt"), []byte("content"), 0o600))
				root := filepath.Join(dir, "root")
				require.NoError(t, os.Mkdir(root, 0o700))
				require.NoError(t, os.Symlink(filepath.Join(dir, "outside.txt"), filepath.Join(root, "link.txt")))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.ErrorContains(t, err, "symlink at link.txt points outside the path boundary")
				assert.Len(t, visited, 1)
			},
		},
		{
			name: "symlink to directory",
			setup: func(t *testing.T) (string, string) {
				dir := absoluteTempDir(t)
				subDir := filepath.Join(dir, "subdir")
				require.NoError(t, os.Mkdir(subDir, 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("content"), 0o600))
				require.NoError(t, os.Symlink(subDir, filepath.Join(dir, "link")))
				return dir, dir
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.NoError(t, err)
				assert.Len(t, visited, 2)
			},
		},
		{
			name: "circular symlink",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				subDir := filepath.Join(root, "subdir")
				require.NoError(t, os.Mkdir(subDir, 0o700))
				require.NoError(t, os.Symlink(root, filepath.Join(subDir, "parent")))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.NoError(t, err)
				assert.Len(t, visited, 2)
			},
		},
		{
			name: "symlink with relative path",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("content"), 0o600))
				require.NoError(t, os.Symlink("file.txt", filepath.Join(root, "link.txt")))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.NoError(t, err)
				assert.Len(t, visited, 2)
			},
		},
		{
			name: "symlink to non-existent file",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				require.NoError(t, os.Symlink(filepath.Join(root, "non-existent.txt"), filepath.Join(root, "link.txt")))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.Error(t, err)
				assert.Len(t, visited, 1)
			},
		},
		{
			name: "symlink chain within root",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("content"), 0o600))
				require.NoError(t, os.Symlink(filepath.Join(root, "file.txt"), filepath.Join(root, "link1.txt")))
				require.NoError(t, os.Symlink(filepath.Join(root, "link1.txt"), filepath.Join(root, "link2.txt")))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.NoError(t, err)
				assert.Len(t, visited, 2)
			},
		},
		{
			name: "symlink to directory outside root",
			setup: func(t *testing.T) (string, string) {
				dir := absoluteTempDir(t)
				outsideDir := filepath.Join(dir, "outside")
				require.NoError(t, os.Mkdir(outsideDir, 0o700))
				root := filepath.Join(dir, "root")
				require.NoError(t, os.Mkdir(root, 0o700))
				require.NoError(t, os.Symlink(outsideDir, filepath.Join(root, "link")))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.ErrorContains(t, err, "symlink at link points outside the path boundary")
				assert.Len(t, visited, 1)
			},
		},
		{
			name: "invalid symlink target",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				require.NoError(t, os.Symlink("non-existent.txt", filepath.Join(root, "invalidLink.txt")))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.ErrorContains(t, err, "failed to resolve symlink: lstat non-existent.txt")
				assert.Len(t, visited, 1)
			},
		},
		{
			name: "multiple links to same target",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("content"), 0o600))
				require.NoError(t, os.Symlink(filepath.Join(root, "file.txt"), filepath.Join(root, "link1.txt")))
				require.NoError(t, os.Symlink(filepath.Join(root, "file.txt"), filepath.Join(root, "link2.txt")))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.NoError(t, err)
				assert.Len(t, visited, 2)
			},
		},
		{
			name:     "recursion depth limit exceeded",
			maxDepth: 5,
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				subDir := root
				// Create a deep directory structure
				for i := 0; i < 10; i++ { // Exceeds depth limit of 5
					subDir = filepath.Join(subDir, fmt.Sprintf("level%d", i))
					require.NoError(t, os.Mkdir(subDir, 0o700))
				}
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.ErrorContains(t, err, "maximum recursion depth exceeded")
				assert.Len(t, visited, 6) // Root and 5 levels of subdirectories
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, dirPath := tt.setup(t)
			visited := make(map[string]struct{})
			maxDepth := tt.maxDepth
			if maxDepth <= 0 {
				maxDepth = 100
			}
			err := checkSymlinks(root, dirPath, visited, 0, maxDepth)
			tt.assertions(t, visited, err)
		})
	}
}

func Test_isSubPath(t *testing.T) {
	tests := []struct {
		name   string
		parent string
		child  string
		want   bool
	}{
		{
			name:   "child is direct subdirectory",
			parent: "a/b",
			child:  "a/b/c",
			want:   true,
		},
		{
			name:   "child is nested subdirectory",
			parent: "a/b",
			child:  "a/b/c/d",
			want:   true,
		},
		{
			name:   "child is parent directory",
			parent: "a/b/c",
			child:  "a",
			want:   false,
		},
		{
			name:   "child is same as parent",
			parent: "a/b",
			child:  "a/b",
			want:   true,
		},
		{
			name:   "child is sibling directory",
			parent: "a/b1",
			child:  "a/b2",
			want:   false,
		},
		{
			name:   "parent is root",
			parent: ".",
			child:  "a/b",
			want:   true,
		},
		{
			name:   "child contains parent as prefix",
			parent: "a/b",
			child:  "a/bc/d",
			want:   false,
		},
		{
			name:   "parent and child on different roots",
			parent: "x/y",
			child:  "a/b",
			want:   false,
		},
		{
			name:   "child is parent with trailing separator",
			parent: "a/b",
			child:  "a/b/",
			want:   true,
		},
		{
			name:   "parent and child with relative paths",
			parent: "a",
			child:  "a/b",
			want:   true,
		},
		{
			name:   "complex nested structure",
			parent: "a/b/c",
			child:  "a/b/c/d/e/f",
			want:   true,
		},
		{
			name:   "parent is empty string",
			parent: "",
			child:  "a",
			want:   true,
		},
		{
			name:   "child is empty string",
			parent: "a",
			child:  "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert slashes to OS-specific separators
			parent := filepath.FromSlash(tt.parent)
			child := filepath.FromSlash(tt.child)

			got := isSubPath(parent, child)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_backupFile(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*testing.T) (string, string)
		assertions func(*testing.T, string, string, error)
	}{
		{
			name: "successful backup",
			setup: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				src := filepath.Join(tmpDir, "test_src.txt")
				dst := filepath.Join(tmpDir, "test_dst.txt")
				require.NoError(t, os.WriteFile(src, []byte("test content"), 0o600))
				return src, dst
			},
			assertions: func(t *testing.T, src, dst string, err error) {
				require.NoError(t, err)
				require.FileExists(t, dst)

				// Compare contents
				srcContent, err := os.ReadFile(src)
				require.NoError(t, err)
				dstContent, err := os.ReadFile(dst)
				require.NoError(t, err)
				assert.Equal(t, srcContent, dstContent)

				// Compare permissions
				srcInfo, err := os.Stat(src)
				require.NoError(t, err)
				dstInfo, err := os.Stat(dst)
				require.NoError(t, err)
				assert.Equal(t, srcInfo.Mode(), dstInfo.Mode())
			},
		},
		{
			name: "source file does not exist",
			setup: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				src := filepath.Join(tmpDir, "nonexistent.txt")
				dst := filepath.Join(tmpDir, "test_dst.txt")
				return src, dst
			},
			assertions: func(t *testing.T, _, _ string, err error) {
				assert.ErrorIs(t, err, os.ErrNotExist)
			},
		},
		{
			name: "destination file already exists",
			setup: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()

				src := filepath.Join(tmpDir, "test_src.txt")
				dst := filepath.Join(tmpDir, "test_dst.txt")

				require.NoError(t, os.WriteFile(src, []byte("test content"), 0o600))
				require.NoError(t, os.WriteFile(dst, []byte("existing content"), 0o600))
				return src, dst
			},
			assertions: func(t *testing.T, _, _ string, err error) {
				assert.ErrorIs(t, err, os.ErrExist)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst := tt.setup(t)
			tt.assertions(t, src, dst, backupFile(src, dst))
		})
	}
}

// absoluteTempDir returns the absolute path of a temporary directory created
// by t.TempDir(). This is useful when working with symlinks, as the temporary
// directory path may actually be a symlink on some platforms like macOS.
func absoluteTempDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	absDir, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	return absDir
}

func copyFile(src, dst string) error {
	srcF, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("error opening source file: %v", err)
	}
	defer srcF.Close()

	dstF, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("error creating destination file: %v", err)
	}
	defer dstF.Close()

	if _, err = io.Copy(dstF, srcF); err != nil {
		return fmt.Errorf("error copying file: %v", err)
	}

	srcI, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("error getting source file info: %v", err)
	}

	return os.Chmod(dst, srcI.Mode())
}
