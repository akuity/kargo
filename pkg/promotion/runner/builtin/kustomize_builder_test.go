package builtin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/io/fs"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_kustomizeBuilder_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name:   "path not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): path is required",
			},
		},
		{
			name: "path is empty string",
			config: promotion.Config{
				"path": "",
			},
			expectedProblems: []string{
				"path: String length must be greater than or equal to 1",
			},
		},
		{
			name: "outPath not specified",
			config: promotion.Config{
				"path": "/kustomization/path",
			},
			expectedProblems: []string{
				"(root): outPath is required",
			},
		},
		{
			name: "outPath is empty string",
			config: promotion.Config{
				"path":    "/kustomization/path",
				"outPath": "",
			},
			expectedProblems: []string{
				"outPath: String length must be greater than or equal to 1",
			},
		},
		{
			name: "both required fields missing",
			config: promotion.Config{
				"plugin": promotion.Config{},
			},
			expectedProblems: []string{
				"(root): path is required",
				"(root): outPath is required",
			},
		},
		{
			name: "plugin.helm.apiVersions contains empty string",
			config: promotion.Config{
				"path":    "/kustomization/path",
				"outPath": "/output/manifests.yaml",
				"plugin": promotion.Config{
					"helm": promotion.Config{
						"apiVersions": []string{"v1", ""},
					},
				},
			},
			expectedProblems: []string{
				"plugin.helm.apiVersions.1: String length must be greater than or equal to 1",
			},
		},
		{
			name: "valid minimal config",
			config: promotion.Config{
				"path":    "/kustomization/path",
				"outPath": "/output/manifests.yaml",
			},
			expectedProblems: nil,
		},
		{
			name: "valid config with empty plugin",
			config: promotion.Config{
				"path":    "/kustomization/path",
				"outPath": "/output/manifests.yaml",
				"plugin":  promotion.Config{},
			},
			expectedProblems: nil,
		},
		{
			name: "valid config with empty helm plugin",
			config: promotion.Config{
				"path":    "/kustomization/path",
				"outPath": "/output/manifests.yaml",
				"plugin": promotion.Config{
					"helm": promotion.Config{},
				},
			},
			expectedProblems: nil,
		},
		{
			name: "valid config with helm kubeVersion only",
			config: promotion.Config{
				"path":    "/kustomization/path",
				"outPath": "/output/manifests.yaml",
				"plugin": promotion.Config{
					"helm": promotion.Config{
						"kubeVersion": "1.28.0",
					},
				},
			},
			expectedProblems: nil,
		},
		{
			name: "valid config with helm apiVersions only",
			config: promotion.Config{
				"path":    "/kustomization/path",
				"outPath": "/output/manifests.yaml",
				"plugin": promotion.Config{
					"helm": promotion.Config{
						"apiVersions": []string{"v1", "apps/v1", "networking.k8s.io/v1"},
					},
				},
			},
			expectedProblems: nil,
		},
		{
			name: "valid config with both helm fields",
			config: promotion.Config{
				"path":    "/kustomization/path",
				"outPath": "/output/manifests.yaml",
				"plugin": promotion.Config{
					"helm": promotion.Config{
						"kubeVersion": "1.29.0",
						"apiVersions": []string{"v1", "apps/v1"},
					},
				},
			},
			expectedProblems: nil,
		},
		{
			name: "valid kitchen sink",
			config: promotion.Config{
				"path":    "/path/to/kustomization/directory",
				"outPath": "/output/built-manifests.yaml",
				"plugin": promotion.Config{
					"helm": promotion.Config{
						"kubeVersion": "1.29.2",
						"apiVersions": []string{
							"v1",
							"apps/v1",
							"networking.k8s.io/v1",
							"cert-manager.io/v1",
							"argoproj.io/v1alpha1",
						},
					},
				},
			},
			expectedProblems: nil,
		},
		{
			name: "invalid outputFormat",
			config: promotion.Config{
				"path":         "/kustomization/path",
				"outPath":      "/output/",
				"outputFormat": "invalid",
			},
			expectedProblems: []string{
				"outputFormat: outputFormat must be one of the following:",
			},
		},
		{
			name: "valid config with kargo outputFormat",
			config: promotion.Config{
				"path":         "/kustomization/path",
				"outPath":      "/output/",
				"outputFormat": "kargo",
			},
			expectedProblems: nil,
		},
		{
			name: "valid config with kustomize outputFormat",
			config: promotion.Config{
				"path":         "/kustomization/path",
				"outPath":      "/output/",
				"outputFormat": "kustomize",
			},
			expectedProblems: nil,
		},
	}

	r := newKustomizeBuilder(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*kustomizeBuilder)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_kustomizeBuilder_run(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles func(*testing.T, string)
		config     builtin.KustomizeBuildConfig
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "successful build",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte(`
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- deployment.yaml
`), 0o600))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "deployment.yaml"), []byte(`---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
`), 0o600))
			},
			config: builtin.KustomizeBuildConfig{
				Path:    ".",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				assert.FileExists(t, filepath.Join(dir, "output.yaml"))
				b, err := os.ReadFile(filepath.Join(dir, "output.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(b), "test-deployment")
			},
		},
		{
			name: "successful build with HelmChartInflationGenerator",
			setupFiles: func(t *testing.T, dir string) {
				// Create a temporary HTTP server to serve the Helm chart.
				httpRepositoryRoot := t.TempDir()
				require.NoError(t, fs.CopyFile(
					"../../../helm/testdata/charts/demo-0.1.0.tgz",
					filepath.Join(httpRepositoryRoot, "demo-0.1.0.tgz"),
				))
				httpRepository := httptest.NewServer(http.FileServer(http.Dir(httpRepositoryRoot)))

				repoIndex, err := repo.IndexDirectory(httpRepositoryRoot, httpRepository.URL)
				require.NoError(t, err)
				require.NoError(t, repoIndex.WriteFile(filepath.Join(httpRepositoryRoot, "index.yaml"), 0o600))
				t.Cleanup(httpRepository.Close)

				// Mock the further files.
				require.NoError(t, os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte(`
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
generators:
- chart.yaml
namespace: demo
`), 0o600))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "chart.yaml"), []byte(fmt.Sprintf(`---
apiVersion: builtin
kind: HelmChartInflationGenerator
metadata:
  name: demo
name: demo
releaseName: demo
repo: %s
valuesFile: values.yaml
`, httpRepository.URL)), 0o600))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "values.yaml"), []byte(`---
replicaCount: 3`), 0o600))
			},
			config: builtin.KustomizeBuildConfig{
				Path:    ".",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				assert.FileExists(t, filepath.Join(dir, "output.yaml"))
				b, err := os.ReadFile(filepath.Join(dir, "output.yaml"))
				require.NoError(t, err)

				// Should be inflated with (part of) the config set.
				assert.Contains(t, string(b), "namespace: demo")
				assert.Contains(t, string(b), "helm.sh/chart: demo-0.1.0")

				// The value from the values file should be in the output.
				assert.Contains(t, string(b), "replicas: 3")
			},
		},
		{
			name: "successful build with output directory",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte(`
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- deployment.yaml
`), 0o600))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "deployment.yaml"), []byte(`---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
`), 0o600))
			},
			config: builtin.KustomizeBuildConfig{
				Path:    ".",
				OutPath: "output/",
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				assert.DirExists(t, filepath.Join(dir, "output"))
				b, err := os.ReadFile(filepath.Join(dir, "output", "deployment-test-deployment.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(b), "test-deployment")
			},
		},
		{
			name: "successful build with kargo output format",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte(`
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- deployment.yaml
`), 0o600))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "deployment.yaml"), []byte(`---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
`), 0o600))
			},
			config: builtin.KustomizeBuildConfig{
				Path:         ".",
				OutPath:      "output/",
				OutputFormat: ptr.To(builtin.Kargo),
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				assert.DirExists(t, filepath.Join(dir, "output"))
				// Kargo format: kind-name.yaml (lowercase)
				b, err := os.ReadFile(filepath.Join(dir, "output", "deployment-test-deployment.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(b), "test-deployment")
			},
		},
		{
			name: "successful build with kustomize output format",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte(`
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- deployment.yaml
`), 0o600))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "deployment.yaml"), []byte(`---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
`), 0o600))
			},
			config: builtin.KustomizeBuildConfig{
				Path:         ".",
				OutPath:      "output/",
				OutputFormat: ptr.To(builtin.Kustomize),
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				assert.DirExists(t, filepath.Join(dir, "output"))
				// Kustomize format: group_version_kind_name.yaml
				b, err := os.ReadFile(filepath.Join(dir, "output", "apps_v1_deployment_test-deployment.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(b), "test-deployment")
			},
		},
		{
			name:       "kustomization file not found",
			setupFiles: func(*testing.T, string) {},
			config: builtin.KustomizeBuildConfig{
				Path:    "invalid/",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.ErrorContains(t, err, "no such file or directory")
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)

				assert.NoFileExists(t, filepath.Join(dir, "output.yaml"))
			},
		},
		{
			name: "invalid kustomization",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte(`invalid`), 0o600))
			},
			config: builtin.KustomizeBuildConfig{
				Path:    ".",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.ErrorContains(t, err, "invalid Kustomization")
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)

				assert.NoFileExists(t, filepath.Join(dir, "output.yaml"))
			},
		},
	}

	runner := &kustomizeBuilder{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			tt.setupFiles(t, tempDir)

			stepCtx := &promotion.StepContext{
				WorkDir: tempDir,
			}

			result, err := runner.run(stepCtx, tt.config)
			tt.assertions(t, tempDir, result, err)
		})
	}
}
