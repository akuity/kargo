package directives

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

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/x/directive/builtin"
)

func Test_kustomizeBuilder_runPromotionStep(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles func(*testing.T, string)
		config     builtin.KustomizeBuildConfig
		assertions func(*testing.T, string, PromotionStepResult, error)
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
			assertions: func(t *testing.T, dir string, result PromotionStepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

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
				require.NoError(t, copyFile(
					"testdata/helm/charts/demo-0.1.0.tgz",
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
			assertions: func(t *testing.T, dir string, result PromotionStepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

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
			assertions: func(t *testing.T, dir string, result PromotionStepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

				assert.DirExists(t, filepath.Join(dir, "output"))
				b, err := os.ReadFile(filepath.Join(dir, "output", "deployment-test-deployment.yaml"))
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
			assertions: func(t *testing.T, dir string, result PromotionStepResult, err error) {
				require.ErrorContains(t, err, "no such file or directory")
				assert.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, result)

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
			assertions: func(t *testing.T, dir string, result PromotionStepResult, err error) {
				require.ErrorContains(t, err, "invalid Kustomization")
				assert.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, result)

				assert.NoFileExists(t, filepath.Join(dir, "output.yaml"))
			},
		},
	}

	runner := &kustomizeBuilder{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			tt.setupFiles(t, tempDir)

			stepCtx := &PromotionStepContext{
				WorkDir: tempDir,
			}

			result, err := runner.runPromotionStep(stepCtx, tt.config)
			tt.assertions(t, tempDir, result, err)
		})
	}
}
