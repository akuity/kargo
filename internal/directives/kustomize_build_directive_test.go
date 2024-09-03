package directives

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_kustomizeBuildDirective_run(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles func(*testing.T, string)
		config     KustomizeBuildConfig
		assertions func(*testing.T, string, Result, error)
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
			config: KustomizeBuildConfig{
				Path:    ".",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, dir string, result Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, Result{Status: StatusSuccess}, result)

				assert.FileExists(t, filepath.Join(dir, "output.yaml"))
				b, err := os.ReadFile(filepath.Join(dir, "output.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(b), "test-deployment")
			},
		},
		{
			name:       "kustomization file not found",
			setupFiles: func(*testing.T, string) {},
			config: KustomizeBuildConfig{
				Path:    "invalid/",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, dir string, result Result, err error) {
				require.ErrorContains(t, err, "no such file or directory")
				assert.Equal(t, Result{Status: StatusFailure}, result)

				assert.NoFileExists(t, filepath.Join(dir, "output.yaml"))
			},
		},
		{
			name: "invalid kustomization",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte(`invalid`), 0o600))
			},
			config: KustomizeBuildConfig{
				Path:    ".",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, dir string, result Result, err error) {
				require.ErrorContains(t, err, "invalid Kustomization")
				assert.Equal(t, Result{Status: StatusFailure}, result)

				assert.NoFileExists(t, filepath.Join(dir, "output.yaml"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			tt.setupFiles(t, tempDir)

			stepCtx := &StepContext{
				WorkDir: tempDir,
			}

			d := &kustomizeBuildDirective{}
			result, err := d.run(stepCtx, tt.config)
			tt.assertions(t, tempDir, result, err)
		})
	}
}
