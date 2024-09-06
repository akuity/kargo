package directives

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/repo/repotest"

	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
)

func Test_helmUpdateChartDirective_loadDependencyCredentials(t *testing.T) {
	tests := []struct {
		name              string
		credentialsDB     credentials.Database
		repositoryFile    *repo.File
		newRegistryClient func(*testing.T) (string, *registry.Client)
		buildDependencies func(string) []chartDependency
		withOCIServer     bool
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
			newRegistryClient: func(*testing.T) (string, *registry.Client) {
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
			newRegistryClient: func(t *testing.T) (string, *registry.Client) {
				home := t.TempDir()
				c, err := helm.NewRegistryClient(home)
				require.NoError(t, err)
				return home, c
			},
			withOCIServer: true,
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
			newRegistryClient: func(*testing.T) (string, *registry.Client) {
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
			newRegistryClient: func(*testing.T) (string, *registry.Client) {
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
			newRegistryClient: func(*testing.T) (string, *registry.Client) {
				return "", nil
			},
			assertions: func(t *testing.T, _, _ string, _ *repo.File, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helmHome, registryClient := tt.newRegistryClient(t)

			var registryURL string
			if tt.withOCIServer {
				tempDir := t.TempDir()
				srv, err := repotest.NewOCIServer(t, tempDir)
				require.NoError(t, err)
				t.Cleanup(func() {
					_ = srv.Shutdown(context.Background())
				})
				go srv.ListenAndServe() // nolint:errcheck
				registryURL = srv.RegistryURL
			}

			dependencies := tt.buildDependencies(registryURL)

			d := &helmUpdateChartDirective{}
			err := d.loadDependencyCredentials(
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

func Test_loadChartDependencies(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*testing.T) string
		assert func(*testing.T, []chartDependency, error)
	}{
		{
			name: "valid chart.yaml",
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()

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
				chartPath := filepath.Join(tempDir, "Chart.yaml")
				require.NoError(t, os.WriteFile(chartPath, []byte(chartYAML), 0o600))

				return chartPath
			},
			assert: func(t *testing.T, dependencies []chartDependency, err error) {
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
				tempDir := t.TempDir()

				const chartYAML = `---
this is not a valid chart.yaml
`
				chartPath := filepath.Join(tempDir, "Chart.yaml")
				require.NoError(t, os.WriteFile(chartPath, []byte(chartYAML), 0o600))

				return chartPath
			},
			assert: func(t *testing.T, dependencies []chartDependency, err error) {
				require.ErrorContains(t, err, "failed to unmarshal")
				assert.Nil(t, dependencies)
			},
		},
		{
			name: "missing Chart.yaml",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "Chart.yaml")
			},
			assert: func(t *testing.T, dependencies []chartDependency, err error) {
				require.ErrorContains(t, err, "failed to read file")
				assert.Nil(t, dependencies)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chartPath := tt.setup(t)
			dependencies, err := loadChartDependencies(chartPath)
			tt.assert(t, dependencies, err)
		})
	}
}
