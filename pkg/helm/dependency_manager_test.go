package helm

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"

	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/io/fs"
)

func TestNewEphemeralDependencyManager(t *testing.T) {
	workDir := t.TempDir()
	credsDB := &credentials.FakeDB{}

	em, err := NewEphemeralDependencyManager(
		credsDB,
		"fake-project",
		workDir,
	)
	require.NoError(t, err)
	assert.NotNil(t, em)
	t.Cleanup(func() {
		_ = em.Teardown()
	})

	assert.Equal(t, em.credsDB, credsDB)
	assert.Equal(t, em.workDir, workDir)
	assert.NotEmpty(t, em.helmHome)
	assert.DirExists(t, em.helmHome)
}

func TestEphemeralDependencyManager_Teardown(t *testing.T) {
	t.Run("teardown removes helm home directory", func(t *testing.T) {
		workDir := t.TempDir()
		em, err := NewEphemeralDependencyManager(
			&credentials.FakeDB{},
			"fake-project",
			workDir,
		)
		require.NoError(t, err)
		require.NotNil(t, em)

		assert.DirExists(t, em.helmHome)
		assert.NoError(t, em.Teardown())

		assert.NoDirExists(t, em.helmHome)
		assert.NoError(t, em.Teardown())
	})
}

func TestEphemeralDependencyManager_update(t *testing.T) {
	const (
		testHTTPHostReplace = "<HTTP_REPO>"
		testOCIHostReplace  = "<OCI_HOST>"
	)
	tests := []struct {
		name              string
		chartMetadata     *chart.Metadata
		setupHTTPRegistry func(t *testing.T) string
		setupOCIRegistry  func(t *testing.T) string
		existingLock      func(serverURL string) string
		assertions        func(t *testing.T, chartDir string, updates map[string]string, err error)
	}{
		{
			name: "update with HTTP and OCI repositories",
			setupHTTPRegistry: func(t *testing.T) string {
				root := t.TempDir()
				require.NoError(
					t,
					fs.CopyFile(
						"testdata/charts/examplechart-0.1.0.tgz",
						filepath.Join(root, "examplechart-0.1.0.tgz"),
					),
				)

				server := httptest.NewServer(http.FileServer(http.Dir(root)))
				t.Cleanup(server.Close)

				index, err := repo.IndexDirectory(root, server.URL)
				require.NoError(t, err)
				require.NoError(t, index.WriteFile(filepath.Join(root, "index.yaml"), 0o600))
				return server.URL
			},
			setupOCIRegistry: func(t *testing.T) string {
				server := httptest.NewUnstartedServer(
					registry.New(
						registry.WithBlobHandler(registry.NewInMemoryBlobHandler()),
					),
				)
				server.Start()
				t.Cleanup(server.Close)

				b, err := os.ReadFile("testdata/charts/demo-0.1.0.tgz")
				require.NoError(t, err)

				client, err := NewRegistryClient(NewEphemeralAuthorizer().Client)
				require.NoError(t, err)
				_, err = client.Push(b, hostForRepositoryURL(server.URL)+"/demo:0.1.0")
				require.NoError(t, err)

				return server.URL
			},
			chartMetadata: &chart.Metadata{
				APIVersion: chart.APIVersionV2,
				Name:       "test-chart",
				Version:    "1.0.0",
				Dependencies: []*chart.Dependency{
					{
						Name:       "examplechart",
						Repository: testHTTPHostReplace,
						Version:    "0.1.*",
					},
					{
						Name:       "demo",
						Repository: fmt.Sprintf("oci://%s", testOCIHostReplace),
						Version:    "*",
					},
				},
			},
			assertions: func(t *testing.T, chartDir string, updates map[string]string, err error) {
				assert.NoError(t, err)
				assert.Len(t, updates, 2)

				// Verify the updates
				expectedUpdates := map[string]string{
					"examplechart": "0.1.0",
					"demo":         "0.1.0",
				}
				assert.Equal(t, expectedUpdates, updates)

				// Verify the charts were downloaded
				assert.FileExists(t, filepath.Join(chartDir, "charts", "examplechart-0.1.0.tgz"))
				assert.FileExists(t, filepath.Join(chartDir, "charts", "demo-0.1.0.tgz"))
			},
		},
		{
			name: "update with existing Chart.lock",
			setupHTTPRegistry: func(t *testing.T) string {
				root := t.TempDir()
				require.NoError(
					t,
					fs.CopyFile(
						"testdata/charts/examplechart-0.1.0.tgz",
						filepath.Join(root, "examplechart-0.1.0.tgz"),
					),
				)

				server := httptest.NewServer(http.FileServer(http.Dir(root)))
				t.Cleanup(server.Close)

				index, err := repo.IndexDirectory(root, server.URL)
				require.NoError(t, err)
				require.NoError(t, index.WriteFile(filepath.Join(root, "index.yaml"), 0o600))
				return server.URL
			},
			chartMetadata: &chart.Metadata{
				APIVersion: chart.APIVersionV2,
				Name:       "test-chart",
				Version:    "1.0.0",
				Dependencies: []*chart.Dependency{
					{
						Name:       "examplechart",
						Repository: testHTTPHostReplace,
						Version:    "0.1.*",
					},
				},
			},
			existingLock: func(serverURL string) string {
				return fmt.Sprintf(`# This is an existing Chart.lock file
dependencies:
- name: examplechart
  repository: %s
  version: 0.0.9
digest: sha256:old-digest-should-be-replaced
generated: "2023-01-01T00:00:00Z"
`, serverURL)
			},
			assertions: func(t *testing.T, chartDir string, updates map[string]string, err error) {
				assert.NoError(t, err)
				assert.Len(t, updates, 1)

				// Verify the version change was reported correctly
				expectedUpdates := map[string]string{
					"examplechart": "0.0.9 -> 0.1.0",
				}
				assert.Equal(t, expectedUpdates, updates)

				// Verify Chart.lock contains the NEW content
				lockFile := filepath.Join(chartDir, "Chart.lock")
				assert.FileExists(t, lockFile)

				content, err := os.ReadFile(lockFile)
				assert.NoError(t, err)

				// Should contain the NEW version (0.1.0), not the old version (0.0.9)
				assert.Contains(t, string(content), "version: 0.1.0")
				assert.NotContains(t, string(content), "version: 0.0.9")

				// Should NOT contain the old digest
				assert.NotContains(t, string(content), "old-digest-should-be-replaced")
				// Should still contain a digest
				assert.Contains(t, string(content), "digest: sha256:")

				// Verify the charts were downloaded
				assert.FileExists(t, filepath.Join(chartDir, "charts", "examplechart-0.1.0.tgz"))

				// No backup files should remain
				entries, readDirErr := os.ReadDir(chartDir)
				assert.NoError(t, readDirErr)
				for _, entry := range entries {
					assert.False(t, strings.HasSuffix(entry.Name(), ".bak"),
						"backup file should be cleaned up: %s", entry.Name())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			em := &EphemeralDependencyManager{
				credsDB:    &credentials.FakeDB{},
				authorizer: NewEphemeralAuthorizer(),
				workDir:    absoluteTempDir(t),
				helmHome:   absoluteTempDir(t),
			}

			chartDir := filepath.Join(em.workDir, "test-chart")
			require.NoError(t, os.Mkdir(chartDir, 0o700))

			if tt.setupHTTPRegistry != nil {
				httpURL := tt.setupHTTPRegistry(t)
				for i, dep := range tt.chartMetadata.Dependencies {
					if strings.Contains(dep.Repository, testHTTPHostReplace) {
						tt.chartMetadata.Dependencies[i].Repository = strings.Replace(
							dep.Repository, testHTTPHostReplace, httpURL, 1,
						)
					}
				}

				if tt.existingLock != nil {
					lockContent := tt.existingLock(httpURL)
					lockFile := filepath.Join(chartDir, "Chart.lock")
					require.NoError(t, os.WriteFile(lockFile, []byte(lockContent), 0o600))
				}
			}

			if tt.setupOCIRegistry != nil {
				ociURL := tt.setupOCIRegistry(t)

				for i, dep := range tt.chartMetadata.Dependencies {
					if strings.Contains(dep.Repository, testOCIHostReplace) {
						tt.chartMetadata.Dependencies[i].Repository = strings.Replace(
							dep.Repository,
							testOCIHostReplace,
							strings.TrimPrefix(ociURL, "http://"),
							1,
						)
					}
				}
			}

			if tt.chartMetadata != nil {
				chartFile := filepath.Join(chartDir, "Chart.yaml")
				b, err := yaml.Marshal(tt.chartMetadata)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(chartFile, b, 0o600))
			}

			if tt.chartMetadata != nil && tt.chartMetadata.Dependencies != nil {
				dependencies := make([]ChartDependency, len(tt.chartMetadata.Dependencies))
				for i, dep := range tt.chartMetadata.Dependencies {
					dependencies[i] = ChartDependency{
						Name:       dep.Name,
						Repository: dep.Repository,
						Version:    dep.Version,
					}
				}
				require.NoError(t, em.setupRepositories(context.Background(), dependencies))
			}

			updates, err := em.update(chartDir)
			tt.assertions(t, chartDir, updates, err)
		})
	}
}

func TestEphemeralDependencyManager_validateDependencies(t *testing.T) {
	tests := []struct {
		name         string
		dependencies []ChartDependency
		setup        func(t *testing.T, workDir, chartDir string)
		assertions   func(t *testing.T, err error)
	}{
		{
			name:         "no dependencies",
			dependencies: []ChartDependency{},
			assertions: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "valid local dependency",
			dependencies: []ChartDependency{
				{Name: "local-chart", Repository: "file://../local-chart"},
			},
			setup: func(t *testing.T, workDir, _ string) {
				localDepDir := filepath.Join(workDir, "local-chart")
				require.NoError(t, os.MkdirAll(localDepDir, 0o700))
			},
			assertions: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "invalid local dependency outside work directory",
			dependencies: []ChartDependency{
				{Name: "outside-chart", Repository: "file://../../outside-chart"},
			},
			assertions: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "invalid dependency")
			},
		},
		{
			name: "remote dependencies are valid",
			dependencies: []ChartDependency{
				{Name: "nginx", Repository: "https://charts.bitnami.com/bitnami"},
				{Name: "redis", Repository: "oci://registry.com/charts"},
			},
			assertions: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := absoluteTempDir(t)
			chartDir := filepath.Join(workDir, "test-chart")
			require.NoError(t, os.MkdirAll(chartDir, 0o700))

			em := &EphemeralDependencyManager{
				workDir: workDir,
			}

			if tt.setup != nil {
				tt.setup(t, workDir, chartDir)
			}

			err := em.validateDependencies(chartDir, tt.dependencies)
			tt.assertions(t, err)
		})
	}
}

func TestEphemeralDependencyManager_validateLocalDependency(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) (workDir, chartPath, dependencyPath string)
		assertions func(t *testing.T, err error)
	}{
		{
			name: "valid local dependency",
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
				assert.ErrorContains(t, err, "dependency path is outside the work directory")
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
				assert.ErrorContains(t, err, "dependency path is outside the work directory")
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
				assert.ErrorContains(t, err, "resolve dependency path")
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
				assert.ErrorContains(t, err, `dependency path "/absolute-dep" must be relative`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir, chartPath, dependencyPath := tt.setup(t)
			em := &EphemeralDependencyManager{
				workDir: workDir,
			}
			err := em.validateLocalDependency(chartPath, dependencyPath)
			tt.assertions(t, err)
		})
	}
}

func TestEphemeralDependencyManager_processDependencyUpdates(t *testing.T) {
	tests := []struct {
		name       string
		chartFile  string
		updates    []ChartDependency
		assertions func(t *testing.T, chartPath string, err error)
	}{
		{
			name: "successful update with version change",
			chartFile: `name: test-chart
version: 1.0.0
dependencies:
  - name: redis
    repository: https://charts.bitnami.com/bitnami
    version: 17.0.0
  - name: postgresql
    repository: https://charts.bitnami.com/bitnami
    version: 12.0.0
`,
			updates: []ChartDependency{
				{Name: "redis", Repository: "https://charts.bitnami.com/bitnami", Version: "18.0.0"},
			},
			assertions: func(t *testing.T, chartPath string, err error) {
				assert.NoError(t, err)

				dependencies, err := GetChartDependencies(chartPath)
				assert.NoError(t, err)
				assert.Equal(t, []ChartDependency{
					{Name: "redis", Repository: "https://charts.bitnami.com/bitnami", Version: "18.0.0"},
					{Name: "postgresql", Repository: "https://charts.bitnami.com/bitnami", Version: "12.0.0"},
				}, dependencies)
			},
		},
		{
			name: "no update when version equals",
			chartFile: `name: test-chart
version: 1.0.0
dependencies:
  - name: redis
    repository: https://charts.bitnami.com/bitnami
    version: 17.0.0
`,
			updates: []ChartDependency{
				{Name: "redis", Repository: "https://charts.bitnami.com/bitnami", Version: "17.0.0"},
			},
			assertions: func(t *testing.T, chartPath string, err error) {
				assert.NoError(t, err)

				dependencies, err := GetChartDependencies(chartPath)
				assert.NoError(t, err)
				assert.Equal(t, []ChartDependency{
					{Name: "redis", Repository: "https://charts.bitnami.com/bitnami", Version: "17.0.0"},
				}, dependencies)
			},
		},
		{
			name: "multiple updates with mixed changes",
			chartFile: `name: test-chart
version: 1.0.0
dependencies:
  - name: redis
    repository: https://charts.bitnami.com/bitnami
    version: 17.0.0
  - name: postgresql
    repository: https://charts.bitnami.com/bitnami
    version: 12.0.0
  - name: mysql
    repository: https://charts.bitnami.com/bitnami
    version: 9.0.0
`,
			updates: []ChartDependency{
				{Name: "redis", Repository: "https://charts.bitnami.com/bitnami", Version: "18.0.0"},
				{Name: "mysql", Repository: "https://charts.bitnami.com/bitnami", Version: "10.0.0"},
			},
			assertions: func(t *testing.T, chartPath string, err error) {
				assert.NoError(t, err)

				dependencies, err := GetChartDependencies(chartPath)
				assert.NoError(t, err)
				assert.Equal(t, []ChartDependency{
					{Name: "redis", Repository: "https://charts.bitnami.com/bitnami", Version: "18.0.0"},
					{Name: "postgresql", Repository: "https://charts.bitnami.com/bitnami", Version: "12.0.0"},
					{Name: "mysql", Repository: "https://charts.bitnami.com/bitnami", Version: "10.0.0"},
				}, dependencies)
			},
		},
		{
			name: "dependency name mismatch",
			chartFile: `name: test-chart
version: 1.0.0
dependencies:
  - name: redis
    repository: https://charts.bitnami.com/bitnami
    version: 17.0.0
`,
			updates: []ChartDependency{
				{Name: "postgresql", Repository: "https://charts.bitnami.com/bitnami", Version: "12.0.0"},
			},
			assertions: func(t *testing.T, _ string, err error) {
				assert.ErrorContains(t, err, `no dependency in Chart.yaml matches update`)
			},
		},
		{
			name: "dependency repository mismatch",
			chartFile: `name: test-chart
version: 1.0.0
dependencies:
  - name: redis
    repository: https://charts.bitnami.com/bitnami
    version: 17.0.0
`,
			updates: []ChartDependency{
				{Name: "redis", Repository: "https://different-repo.com", Version: "18.0.0"},
			},
			assertions: func(t *testing.T, _ string, err error) {
				assert.ErrorContains(t, err, `no dependency in Chart.yaml matches update`)
			},
		},
		{
			name: "empty updates list",
			chartFile: `name: test-chart
version: 1.0.0
dependencies:
  - name: redis
    repository: https://charts.bitnami.com/bitnami
    version: 17.0.0
`,
			updates: []ChartDependency{},
			assertions: func(t *testing.T, _ string, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "empty dependencies list",
			chartFile: `name: test-chart
version: 1.0.0
`,
			updates: []ChartDependency{
				{Name: "redis", Repository: "https://charts.bitnami.com/bitnami", Version: "17.0.0"},
			},
			assertions: func(t *testing.T, _ string, err error) {
				assert.ErrorContains(t, err, `no dependency in Chart.yaml matches update`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			chartPath := filepath.Join(tempDir, "Chart.yaml")
			require.NoError(t, os.WriteFile(chartPath, []byte(tt.chartFile), 0o600))

			dependencies, err := GetChartDependencies(chartPath)
			require.NoError(t, err)

			em := &EphemeralDependencyManager{}
			err = em.processDependencyUpdates(chartPath, dependencies, tt.updates)

			tt.assertions(t, chartPath, err)
		})
	}
}

func TestEphemeralDependencyManager_setupRepositories(t *testing.T) {
	const (
		testOCIUsername    = "oci-user"
		testOCIPassword    = "oci-pass"
		testOCIHostReplace = "<OCI_HOST>"

		testHTTPUsername = "http-user"
		testHTTPPassword = "http-pass"
	)

	tests := []struct {
		name             string
		dependencies     []ChartDependency
		credsDB          credentials.Database
		setupOCIRegistry func(*testing.T) string
		assertions       func(t *testing.T, registryURL string, em *EphemeralDependencyManager, err error)
	}{
		{
			name: "mixed dependency types with credentials",
			dependencies: []ChartDependency{
				{Name: "local-chart", Repository: "file://./charts/local"},
				{Name: "nginx", Repository: "https://charts.bitnami.com/bitnami"},
				{Name: "redis", Repository: "http://charts.example.com"},
				{Name: "postgres", Repository: fmt.Sprintf("oci://%s/charts", testOCIHostReplace)},
				{Name: "empty-repo", Repository: ""},
			},
			credsDB: &credentials.FakeDB{
				GetFn: func(_ context.Context, _ string, _ credentials.Type, repo string) (*credentials.Credentials, error) {
					switch {
					case strings.Contains(repo, "charts.bitnami.com"):
						return &credentials.Credentials{
							Username: testHTTPUsername,
							Password: testHTTPPassword,
						}, nil
					case strings.HasPrefix(repo, "oci://"):
						return &credentials.Credentials{
							Username: testOCIUsername,
							Password: testOCIPassword,
						}, nil
					default:
						return nil, nil
					}
				},
			},
			setupOCIRegistry: func(t *testing.T) string {
				server := newAuthRegistryServer(testOCIUsername, testOCIPassword)
				server.Start()
				t.Cleanup(server.Close)
				return server.URL
			},
			assertions: func(t *testing.T, ociServer string, em *EphemeralDependencyManager, err error) {
				assert.NoError(t, err)

				// Verify the repositories file was created
				assert.FileExists(t, em.repositoryConfig())

				// Verify the file data
				repoFile, err := repo.LoadFile(em.repositoryConfig())
				assert.NoError(t, err)

				// Should have two repositories
				assert.Len(t, repoFile.Repositories, 2)
				assert.Equal(t, []*repo.Entry{
					{
						Name:     "54d2620bbb6f1bb3f35d4c7f945bfa25077949488dcbb0a4d01c90f2c35baa59",
						URL:      "https://charts.bitnami.com/bitnami",
						Username: testHTTPUsername,
						Password: testHTTPPassword,
					},
					{
						Name: "541bf121d8d326bc43ed9cfa1b4e31a4861f9fd63670d67f9227ec225ecdb3fc",
						URL:  "http://charts.example.com",
					},
				}, repoFile.Repositories)

				// Verify an entry exists for the OCI host
				ociHost := hostForRepositoryURL(ociServer)
				entry, err := em.authorizer.Get(context.Background(), ociHost)
				assert.NoError(t, err)
				assert.NotNil(t, entry)
			},
		},
		{
			name: "HTTPS repository credential database error",
			dependencies: []ChartDependency{
				{Name: "nginx", Repository: "https://charts.bitnami.com/bitnami"},
			},
			credsDB: &credentials.FakeDB{
				GetFn: func(_ context.Context, _ string, _ credentials.Type, _ string) (*credentials.Credentials, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ string, _ *EphemeralDependencyManager, err error) {
				assert.ErrorContains(t, err, "obtain credentials for repository")
				assert.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "OCI credential database error",
			dependencies: []ChartDependency{
				{Name: "postgres", Repository: fmt.Sprintf("oci://%s/charts", testOCIHostReplace)},
			},
			credsDB: &credentials.FakeDB{
				GetFn: func(_ context.Context, _ string, _ credentials.Type, _ string) (*credentials.Credentials, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ string, _ *EphemeralDependencyManager, err error) {
				assert.ErrorContains(t, err, "obtain credentials for repository")
				assert.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "OCI authentication with invalid credentials",
			dependencies: []ChartDependency{
				{Name: "postgres", Repository: fmt.Sprintf("oci://%s/charts", testOCIHostReplace)},
			},
			credsDB: &credentials.FakeDB{
				GetFn: func(_ context.Context, _ string, _ credentials.Type, _ string) (*credentials.Credentials, error) {
					return &credentials.Credentials{
						Username: "invalid",
						Password: "invalid",
					}, nil
				},
			},
			setupOCIRegistry: func(t *testing.T) string {
				server := newAuthRegistryServer(testOCIUsername, testOCIPassword)
				server.Start()
				t.Cleanup(server.Close)
				return server.URL
			},
			assertions: func(t *testing.T, _ string, _ *EphemeralDependencyManager, err error) {
				assert.ErrorContains(t, err, "authenticate with chart repository")
			},
		},
		{
			name: "multiple OCI repositories same host",
			dependencies: []ChartDependency{
				{Name: "postgres", Repository: fmt.Sprintf("oci://%s/charts", testOCIHostReplace)},
				{Name: "mysql", Repository: fmt.Sprintf("oci://%s/databases", testOCIHostReplace)},
			},
			credsDB: &credentials.FakeDB{
				GetFn: func(_ context.Context, _ string, _ credentials.Type, repo string) (*credentials.Credentials, error) {
					if strings.Contains(repo, "postgres") {
						return &credentials.Credentials{
							Username: testOCIUsername,
							Password: testOCIPassword,
						}, nil
					}
					// Invalid credentials should not cause an error here because
					// Postgres successfully authenticated
					return &credentials.Credentials{
						Username: "invalid",
						Password: "invalid",
					}, nil
				},
			},
			setupOCIRegistry: func(t *testing.T) string {
				server := newAuthRegistryServer(testOCIUsername, testOCIPassword)
				server.Start()
				t.Cleanup(server.Close)
				return server.URL
			},
			assertions: func(t *testing.T, ociServer string, em *EphemeralDependencyManager, err error) {
				assert.NoError(t, err)

				// Verify an entry exists for the OCI host
				ociHost := hostForRepositoryURL(ociServer)
				entry, err := em.authorizer.Get(context.Background(), ociHost)
				assert.NoError(t, err)
				assert.NotNil(t, entry)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			em := &EphemeralDependencyManager{
				helmHome:   tempDir,
				authorizer: NewEphemeralAuthorizer(),
				credsDB:    tt.credsDB,
			}

			dependencies := slices.Clone(tt.dependencies)
			var registryURL string
			if tt.setupOCIRegistry != nil {
				registryURL = tt.setupOCIRegistry(t)

				for i, d := range dependencies {
					if strings.Contains(d.Repository, testOCIHostReplace) {
						dependencies[i].Repository = strings.Replace(
							d.Repository, testOCIHostReplace,
							strings.TrimPrefix(registryURL, "http://"),
							1,
						)
					}
				}
			}

			err := em.setupRepositories(context.Background(), dependencies)
			tt.assertions(t, registryURL, em, err)
		})
	}
}

func TestEphemeralDependencyManager_fetchRepositoryIndexes(t *testing.T) {
	const (
		mockIndexYAML = `
apiVersion: v1
entries:
  nginx:
  - name: nginx
    version: 1.0.0
    description: A basic nginx chart
    created: 2023-01-01T00:00:00Z
    urls:
    - charts/nginx-1.0.0.tgz
  redis:
  - name: redis
    version: 2.1.0
    description: A Redis chart
    created: 2023-01-01T00:00:00Z
    urls:
    - charts/redis-2.1.0.tgz
generated: 2023-01-01T00:00:00Z
`
		mockRepositoriesYAML = `
apiVersion: ""
generated: "0001-01-01T00:00:00Z"
repositories:
- name: test-repo
  url: %s
  username: ""
  password: ""
  certFile: ""
  keyFile: ""
  caFile: ""
  insecure_skip_tls_verify: false
- name: second-repo
  url: %s/alt
  username: ""
  password: ""
  certFile: ""
  keyFile: ""
  caFile: ""
  insecure_skip_tls_verify: false
`
	)

	tests := []struct {
		name          string
		setupServer   func() (*httptest.Server, func())
		setupRepoFile func(tempDir, serverURL string) error
		assertFunc    func(t *testing.T, err error, tempDir string)
	}{
		{
			name: "successful fetch with valid repositories",
			setupServer: func() (*httptest.Server, func()) {
				mux := http.NewServeMux()
				mux.HandleFunc("/index.yaml", func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/yaml")
					w.WriteHeader(http.StatusOK)
					_, _ = fmt.Fprint(w, mockIndexYAML)
				})
				mux.HandleFunc("/alt/index.yaml", func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/yaml")
					w.WriteHeader(http.StatusOK)
					_, _ = fmt.Fprint(w, mockIndexYAML)
				})

				server := httptest.NewServer(mux)
				return server, func() { server.Close() }
			},
			setupRepoFile: func(tempDir, serverURL string) error {
				repoContent := fmt.Sprintf(mockRepositoriesYAML, serverURL, serverURL)
				return os.WriteFile(filepath.Join(tempDir, "repositories.yaml"), []byte(repoContent), 0o600)
			},
			assertFunc: func(t *testing.T, err error, tempDir string) {
				assert.NoError(t, err)

				// Verify cache files were created
				cacheDir := filepath.Join(tempDir, "cache")
				entries, err := os.ReadDir(cacheDir)
				assert.NoError(t, err)
				assert.NotEmpty(t, entries, "cache directory should contain index files")

				// Verify specific repository cache files exist
				testRepoCache := filepath.Join(cacheDir, "test-repo-index.yaml")
				secondRepoCache := filepath.Join(cacheDir, "second-repo-index.yaml")

				assert.FileExists(t, testRepoCache)
				assert.FileExists(t, secondRepoCache)
			},
		},
		{
			name: "repository config file not found",
			setupServer: func() (*httptest.Server, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
				return server, func() { server.Close() }
			},
			setupRepoFile: func(_, _ string) error {
				// Don't create the repositories.yaml file
				return nil
			},
			assertFunc: func(t *testing.T, err error, _ string) {
				assert.ErrorContains(t, err, "load repository config file")
			},
		},
		{
			name: "server returns 404 for index file",
			setupServer: func() (*httptest.Server, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
				return server, func() { server.Close() }
			},
			setupRepoFile: func(tempDir, serverURL string) error {
				repoContent := fmt.Sprintf(mockRepositoriesYAML, serverURL, serverURL)
				return os.WriteFile(filepath.Join(tempDir, "repositories.yaml"), []byte(repoContent), 0o600)
			},
			assertFunc: func(t *testing.T, err error, _ string) {
				assert.ErrorContains(t, err, "download repository index")
			},
		},
		{
			name: "invalid repository URL",
			setupServer: func() (*httptest.Server, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
				return server, func() { server.Close() }
			},
			setupRepoFile: func(tempDir, _ string) error {
				// Use an invalid URL
				invalidRepoYAML := fmt.Sprintf(mockRepositoriesYAML, "invalid-url", "invalid-url")
				return os.WriteFile(filepath.Join(tempDir, "repositories.yaml"), []byte(invalidRepoYAML), 0o600)
			},
			assertFunc: func(t *testing.T, err error, _ string) {
				assert.ErrorContains(t, err, "create chart repository")
			},
		},
		{
			name: "empty repositories list",
			setupServer: func() (*httptest.Server, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
				return server, func() { server.Close() }
			},
			setupRepoFile: func(tempDir, _ string) error {
				emptyRepoYAML := `
apiVersion: ""
generated: "0001-01-01T00:00:00Z"
repositories: []
`
				return os.WriteFile(filepath.Join(tempDir, "repositories.yaml"), []byte(emptyRepoYAML), 0o600)
			},
			assertFunc: func(t *testing.T, err error, _ string) {
				assert.NoError(t, err, "empty repositories should not cause an error")
			},
		},
		{
			name: "malformed repositories.yaml",
			setupServer: func() (*httptest.Server, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
				return server, func() { server.Close() }
			},
			setupRepoFile: func(tempDir, _ string) error {
				malformedYAML := `invalid: yaml: content: [unclosed`
				return os.WriteFile(filepath.Join(tempDir, "repositories.yaml"), []byte(malformedYAML), 0o600)
			},
			assertFunc: func(t *testing.T, err error, _ string) {
				assert.ErrorContains(t, err, "load repository config file")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for Helm home and the cache directory
			tempDir := t.TempDir()
			cacheDir := filepath.Join(tempDir, "cache")
			assert.NoError(t, os.MkdirAll(cacheDir, 0o700))

			// Setup test server
			server, cleanup := tt.setupServer()
			t.Cleanup(cleanup)

			// Setup repository file
			assert.NoError(t, tt.setupRepoFile(tempDir, server.URL))

			em := &EphemeralDependencyManager{
				helmHome: tempDir,
			}

			err := em.fetchRepositoryIndexes()
			tt.assertFunc(t, err, tempDir)
		})
	}
}

func TestEphemeralDependencyManager_repositoryConfig(t *testing.T) {
	helmHome := t.TempDir()
	em := &EphemeralDependencyManager{
		helmHome: helmHome,
	}

	expected := filepath.Join(helmHome, "repositories.yaml")
	assert.Equal(t, expected, em.repositoryConfig())
}

func TestEphemeralDependencyManager_repositoryCache(t *testing.T) {
	helmHome := t.TempDir()
	em := &EphemeralDependencyManager{
		helmHome: helmHome,
	}

	expected := filepath.Join(helmHome, "cache")
	assert.Equal(t, expected, em.repositoryCache())
}

func Test_newFileBackup(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name       string
		setup      func(t *testing.T) string
		assertions func(t *testing.T, bf *backupFile, err error)
	}{
		{
			name: "file does not exist",
			setup: func(_ *testing.T) string {
				return filepath.Join(tempDir, "nonexistent.txt")
			},
			assertions: func(t *testing.T, bf *backupFile, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, bf)
				assert.Equal(t, filepath.Join(tempDir, "nonexistent.txt"), bf.originalPath)
				assert.Empty(t, bf.backupPath)
			},
		},
		{
			name: "regular file exists",
			setup: func(t *testing.T) string {
				filePath := filepath.Join(tempDir, "regular.txt")
				require.NoError(t, os.WriteFile(filePath, []byte("test content"), 0o600))
				return filePath
			},
			assertions: func(t *testing.T, bf *backupFile, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, bf)

				assert.Equal(t, filepath.Join(tempDir, "regular.txt"), bf.originalPath)
				assert.NotEmpty(t, bf.backupPath)
				assert.True(t, strings.HasSuffix(bf.backupPath, ".bak"))
				assert.True(t, strings.Contains(bf.backupPath, time.Now().Format("20060102")))

				// Verify backup file was created
				_, err = os.Stat(bf.backupPath)
				assert.NoError(t, err)

				// Verify content matches
				originalContent, err := os.ReadFile(bf.originalPath)
				require.NoError(t, err)
				backupContent, err := os.ReadFile(bf.backupPath)
				require.NoError(t, err)
				assert.Equal(t, originalContent, backupContent)
			},
		},
		{
			name: "symlink file",
			setup: func(t *testing.T) string {
				targetPath := filepath.Join(tempDir, "target.txt")
				err := os.WriteFile(targetPath, []byte("target content"), 0o600)
				require.NoError(t, err)

				symlinkPath := filepath.Join(tempDir, "symlink.txt")
				err = os.Symlink(targetPath, symlinkPath)
				require.NoError(t, err)
				return symlinkPath
			},
			assertions: func(t *testing.T, bf *backupFile, err error) {
				assert.Nil(t, bf)
				assert.ErrorContains(t, err, "cannot create backup of symlink")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			bf, err := newFileBackup(path)
			tt.assertions(t, bf, err)
		})
	}
}

func Test_backupFile_Restore(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name       string
		setup      func(t *testing.T) *backupFile
		assertions func(t *testing.T, bf *backupFile, err error)
	}{
		{
			name: "no backup path",
			setup: func(_ *testing.T) *backupFile {
				return &backupFile{
					originalPath: filepath.Join(tempDir, "original.txt"),
					backupPath:   "",
				}
			},
			assertions: func(t *testing.T, bf *backupFile, err error) {
				assert.NoError(t, err)
				assert.Empty(t, bf.backupPath)
			},
		},
		{
			name: "successful restore",
			setup: func(t *testing.T) *backupFile {
				originalPath := filepath.Join(tempDir, "original.txt")
				backupPath := filepath.Join(tempDir, "backup.txt")

				// Create backup file with test content
				backupContent := []byte("backup content")
				err := os.WriteFile(backupPath, backupContent, 0o600)
				require.NoError(t, err)

				return &backupFile{
					originalPath: originalPath,
					backupPath:   backupPath,
				}
			},
			assertions: func(t *testing.T, bf *backupFile, err error) {
				assert.NoError(t, err)
				assert.Empty(t, bf.backupPath)

				// Verify original file exists with correct content
				content, err := os.ReadFile(bf.originalPath)
				assert.NoError(t, err)
				assert.Equal(t, []byte("backup content"), content)

				// Verify backup file no longer exists
				_, err = os.Stat(filepath.Join(tempDir, "backup.txt"))
				assert.True(t, os.IsNotExist(err))
			},
		},
		{
			name: "backup file doesn't exist",
			setup: func(_ *testing.T) *backupFile {
				return &backupFile{
					originalPath: filepath.Join(tempDir, "original.txt"),
					backupPath:   filepath.Join(tempDir, "nonexistent_backup.txt"),
				}
			},
			assertions: func(t *testing.T, bf *backupFile, err error) {
				assert.ErrorContains(t, err, "failed to restore backup file")
				assert.NotEmpty(t, bf.backupPath)
			},
		},
		{
			name: "target exists as directory",
			setup: func(t *testing.T) *backupFile {
				// Create backup file
				backupPath := filepath.Join(tempDir, "dir_backup.txt")
				require.NoError(t, os.WriteFile(backupPath, []byte("content"), 0o600))

				// Create a directory where we want to restore the file
				originalPath := filepath.Join(tempDir, "target_dir")
				require.NoError(t, os.Mkdir(originalPath, 0o755))

				// Put a file inside to make sure it's not empty
				require.NoError(t, os.WriteFile(filepath.Join(originalPath, "dummy.txt"), []byte("content"), 0o600))

				return &backupFile{
					originalPath: originalPath,
					backupPath:   backupPath,
				}
			},
			assertions: func(t *testing.T, bf *backupFile, err error) {
				assert.ErrorContains(t, err, "failed to restore backup file")
				assert.NotEmpty(t, bf.backupPath)

				// Verify backup file still exists since restore failed
				_, err = os.Stat(bf.backupPath)
				assert.NoError(t, err)

				// Verify target directory still exists
				stat, statErr := os.Stat(bf.originalPath)
				assert.NoError(t, statErr)
				assert.True(t, stat.IsDir())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bf := tt.setup(t)
			err := bf.Restore()
			tt.assertions(t, bf, err)
		})
	}
}

func Test_backupFile_Remove(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name       string
		setup      func(t *testing.T) *backupFile
		assertions func(t *testing.T, bf *backupFile, err error)
	}{
		{
			name: "no backup path",
			setup: func(_ *testing.T) *backupFile {
				return &backupFile{
					originalPath: filepath.Join(tempDir, "original.txt"),
					backupPath:   "",
				}
			},
			assertions: func(t *testing.T, _ *backupFile, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "successful removal",
			setup: func(t *testing.T) *backupFile {
				backupPath := filepath.Join(tempDir, "backup.txt")
				require.NoError(t, os.WriteFile(backupPath, []byte("backup content"), 0o600))

				return &backupFile{
					originalPath: filepath.Join(tempDir, "original.txt"),
					backupPath:   backupPath,
				}
			},
			assertions: func(t *testing.T, bf *backupFile, err error) {
				assert.NoError(t, err)

				// Verify backup file was removed
				_, err = os.Stat(bf.backupPath)
				assert.True(t, os.IsNotExist(err))
			},
		},
		{
			name: "file does not exist",
			setup: func(_ *testing.T) *backupFile {
				return &backupFile{
					originalPath: filepath.Join(tempDir, "original.txt"),
					backupPath:   filepath.Join(tempDir, "nonexistent_backup.txt"),
				}
			},
			assertions: func(t *testing.T, _ *backupFile, err error) {
				assert.ErrorContains(t, err, "failed to remove backup file")
			},
		},
		{
			name: "try to remove directory instead of file",
			setup: func(t *testing.T) *backupFile {
				// Create a directory where we expect a file
				backupPath := filepath.Join(tempDir, "backup_dir")
				require.NoError(t, os.Mkdir(backupPath, 0o755))

				// Put a file inside to make sure it's not empty
				require.NoError(t, os.WriteFile(filepath.Join(backupPath, "dummy.txt"), []byte("content"), 0o600))

				return &backupFile{
					originalPath: filepath.Join(tempDir, "original.txt"),
					backupPath:   backupPath,
				}
			},
			assertions: func(t *testing.T, bf *backupFile, err error) {
				assert.ErrorContains(t, err, "failed to remove backup file")

				// Verify directory still exists
				stat, statErr := os.Stat(bf.backupPath)
				assert.NoError(t, statErr)
				assert.True(t, stat.IsDir())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bf := tt.setup(t)
			err := bf.Remove()
			tt.assertions(t, bf, err)
		})
	}
}

func Test_compareChartVersions(t *testing.T) {
	tests := []struct {
		name   string
		before []ChartDependency
		after  []ChartDependency
		want   map[string]string
	}{
		{
			name: "no changes",
			before: []ChartDependency{
				{Name: "chart1", Version: "1.0.0"},
				{Name: "chart2", Version: "2.0.0"},
			},
			after: []ChartDependency{
				{Name: "chart1", Version: "1.0.0"},
				{Name: "chart2", Version: "2.0.0"},
			},
			want: map[string]string{},
		},
		{
			name: "version update",
			before: []ChartDependency{
				{Name: "chart1", Version: "1.0.0"},
				{Name: "chart2", Version: "2.0.0"},
			},
			after: []ChartDependency{
				{Name: "chart1", Version: "1.1.0"},
				{Name: "chart2", Version: "2.0.0"},
			},
			want: map[string]string{"chart1": "1.0.0 -> 1.1.0"},
		},
		{
			name: "new chart added",
			before: []ChartDependency{
				{Name: "chart1", Version: "1.0.0"},
			},
			after: []ChartDependency{
				{Name: "chart1", Version: "1.0.0"},
				{Name: "chart2", Version: "2.0.0"},
			},
			want: map[string]string{"chart2": "2.0.0"},
		},
		{
			name: "chart removed",
			before: []ChartDependency{
				{Name: "chart1", Version: "1.0.0"},
				{Name: "chart2", Version: "2.0.0"},
			},
			after: []ChartDependency{
				{Name: "chart1", Version: "1.0.0"},
			},
			want: map[string]string{"chart2": ""},
		},
		{
			name: "multiple changes",
			before: []ChartDependency{
				{Name: "chart1", Version: "1.0.0"},
				{Name: "chart2", Version: "2.0.0"},
				{Name: "chart3", Version: "3.0.0"},
			},
			after: []ChartDependency{
				{Name: "chart1", Version: "1.1.0"},
				{Name: "chart2", Version: "2.0.0"},
				{Name: "chart4", Version: "4.0.0"},
			},
			want: map[string]string{"chart1": "1.0.0 -> 1.1.0", "chart3": "", "chart4": "4.0.0"},
		},
		{
			name:   "empty before",
			before: []ChartDependency{},
			after: []ChartDependency{
				{Name: "chart1", Version: "1.0.0"},
				{Name: "chart2", Version: "2.0.0"},
			},
			want: map[string]string{"chart1": "1.0.0", "chart2": "2.0.0"},
		},
		{
			name: "empty after",
			before: []ChartDependency{
				{Name: "chart1", Version: "1.0.0"},
				{Name: "chart2", Version: "2.0.0"},
			},
			after: []ChartDependency{},
			want:  map[string]string{"chart1": "", "chart2": ""},
		},
		{
			name:   "both empty",
			before: []ChartDependency{},
			after:  []ChartDependency{},
			want:   map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, compareChartVersions(tt.before, tt.after))
		})
	}
}

func Test_nameForRepositoryURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid HTTP URL",
			input:    "http://example.com/org/repo",
			expected: "http://example.com/org/repo",
		},
		{
			name:     "valid HTTPS URL",
			input:    "https://example.com/org",
			expected: "https://example.com/org",
		},
		{
			name:     "valid URL without path",
			input:    "https://example.com",
			expected: "https://example.com/",
		},
		{
			name:     "URL requiring path cleaning",
			input:    "https://example.com//org//repo",
			expected: "https://example.com/org/repo",
		},
		{
			name:     "URL requiring path cleaning with dot segments",
			input:    "https://example.com/org/./repo",
			expected: "https://example.com/org/repo",
		},
		{
			name:     "URL requiring path cleaning with parent segments",
			input:    "https://example.com/org/../repo",
			expected: "https://example.com/repo",
		},
		{
			name:     "URL treated as file path",
			input:    "git@example.com:org/repo.git",
			expected: "git@example.com:org/repo.git",
		},
		{
			name:     "file path with leading slash",
			input:    "/local/path/repo",
			expected: "/local/path/repo",
		},
		{
			name:     "file path with relative segments",
			input:    "./repo",
			expected: "repo",
		},
		{
			name:     "file path with multiple slashes",
			input:    "/path//with//slashes",
			expected: "/path/with/slashes",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "/",
		},
		{
			name:     "colon character",
			input:    ":",
			expected: ":",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nameForRepositoryURL(tt.input)
			expected := fmt.Sprintf("%x", sha256.Sum256([]byte(tt.expected)))

			assert.Equal(t, expected, result)
			assert.Len(t, result, 64)
		})
	}
}

func Test_hostForRepositoryURL(t *testing.T) {
	tests := []struct {
		name     string
		repoURL  string
		expected string
	}{
		{
			name:     "OCI URL with path",
			repoURL:  "oci://registry.com/charts/nginx",
			expected: "registry.com",
		},
		{
			name:     "OCI URL without path",
			repoURL:  "oci://registry.com",
			expected: "registry.com",
		},
		{
			name:     "HTTPS URL with path",
			repoURL:  "https://charts.bitnami.com/bitnami",
			expected: "charts.bitnami.com",
		},
		{
			name:     "HTTP URL with path",
			repoURL:  "http://localhost:8080/charts",
			expected: "localhost:8080",
		},
		{
			name:     "URL without scheme",
			repoURL:  "registry.com/charts",
			expected: "registry.com",
		},
		{
			name:     "hostname only",
			repoURL:  "registry.com",
			expected: "registry.com",
		},
		{
			name:     "hostname with port",
			repoURL:  "registry.com:5000",
			expected: "registry.com:5000",
		},
		{
			name:     "localhost with port and path",
			repoURL:  "localhost:8080/v2/charts",
			expected: "localhost:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hostForRepositoryURL(tt.repoURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func absoluteTempDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	absDir, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	return absDir
}
