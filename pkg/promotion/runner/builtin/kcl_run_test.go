package builtin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	configbuiltin "github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_kclRunner_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name:   "path not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): path is required",
				"(root): outPath is required",
			},
		},
		{
			name: "path is empty string",
			config: promotion.Config{
				"path":    "",
				"outPath": "./out.yaml",
			},
			expectedProblems: []string{
				"path: String length must be greater than or equal to 1",
			},
		},
		{
			name: "outPath is empty string",
			config: promotion.Config{
				"path":    "./app/main.k",
				"outPath": "",
			},
			expectedProblems: []string{
				"outPath: String length must be greater than or equal to 1",
			},
		},
		{
			name: "argument name missing",
			config: promotion.Config{
				"path":    "./app/main.k",
				"outPath": "./out.yaml",
				"arguments": []promotion.Config{{
					"value": "demo",
				}},
			},
			expectedProblems: []string{
				"arguments.0: name is required",
			},
		},
		{
			name: "argument name empty",
			config: promotion.Config{
				"path":    "./app/main.k",
				"outPath": "./out.yaml",
				"arguments": []promotion.Config{{
					"name":  "",
					"value": "demo",
				}},
			},
			expectedProblems: []string{
				"arguments.0.name: String length must be greater than or equal to 1",
			},
		},
		{
			name: "invalid outputFormat",
			config: promotion.Config{
				"path":         "./app/main.k",
				"outPath":      "./out",
				"outputFormat": "invalid",
			},
			expectedProblems: []string{
				"outputFormat: outputFormat must be one of the following:",
			},
		},
		{
			name: "valid minimal config",
			config: promotion.Config{
				"path":    "./app/main.k",
				"outPath": "./out.yaml",
			},
		},
		{
			name: "valid config with arguments and output format",
			config: promotion.Config{
				"path":         "./app/main.k",
				"outPath":      "./out",
				"outputFormat": "kustomize",
				"arguments": []promotion.Config{{
					"name":  "appName",
					"value": "demo",
				}},
			},
		},
	}

	r := newKCLRunner(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*kclRunner)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_kclRunner_run(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles func(*testing.T, string)
		config     configbuiltin.KCLRunConfig
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "successful run to file",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "app"), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "app", "main.k"), []byte(`
app = {
    apiVersion = "apps/v1"
    kind = "Deployment"
    metadata = {
        name = "demo"
    }
}
`), 0o600))
			},
			config: configbuiltin.KCLRunConfig{
				Path:    "./app/main.k",
				OutPath: "./out.yaml",
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				b, readErr := os.ReadFile(filepath.Join(dir, "out.yaml"))
				require.NoError(t, readErr)
				assert.Contains(t, string(b), "kind: Deployment")
				assert.Contains(t, string(b), "name: demo")
			},
		},
		{
			name: "successful run to directory with kargo format",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "app"), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "app", "main.k"), []byte(`
items = [
    {
        apiVersion = "apps/v1"
        kind = "Deployment"
        metadata = {
            name = "demo"
        }
    },
    {
        apiVersion = "v1"
        kind = "Service"
        metadata = {
            name = "demo"
            namespace = "prod"
        }
    }
]
`), 0o600))
			},
			config: configbuiltin.KCLRunConfig{
				Path:         "./app/main.k",
				OutPath:      "./out",
				OutputFormat: ptr.To(configbuiltin.Kargo),
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				assert.FileExists(t, filepath.Join(dir, "out", "deployment-demo.yaml"))
				assert.FileExists(t, filepath.Join(dir, "out", "prod-service-demo.yaml"))
			},
		},
		{
			name: "successful run to directory with kustomize format",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "app"), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "app", "main.k"), []byte(`
app = {
    apiVersion = "apps/v1"
    kind = "Deployment"
    metadata = {
        name = option("appName")
        namespace = "prod"
    }
}
`), 0o600))
			},
			config: configbuiltin.KCLRunConfig{
				Path:         "./app/main.k",
				OutPath:      "./out",
				OutputFormat: ptr.To(configbuiltin.Kustomize),
				Arguments: []configbuiltin.Argument{{
					Name:  "appName",
					Value: "demo",
				}},
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				assert.FileExists(t, filepath.Join(dir, "out", "prod_apps_v1_deployment_demo.yaml"))
			},
		},
		{
			name: "successful run from kcl.yaml input path",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "app"), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "app", "main.k"), []byte(`
app = {
    apiVersion = "v1"
    kind = "ConfigMap"
    metadata = {
        name = option("app-name")
        namespace = option("namespace")
    }
    data = {
        version = option("version")
    }
}
`), 0o600))
				require.NoError(
					t,
					os.WriteFile(
						filepath.Join(dir, "app", "kcl.yaml"),
						[]byte(`kcl_cli_configs:
	file:
		- main.k

kcl_options:
	- key: app-name
		value: demo
	- key: namespace
		value: prod
	- key: version
		value: "v1.2.3"
`),
						0o600,
					),
				)
			},
			config: configbuiltin.KCLRunConfig{
				Path:    "./app/kcl.yaml",
				OutPath: "./out.yaml",
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				b, readErr := os.ReadFile(filepath.Join(dir, "out.yaml"))
				require.NoError(t, readErr)
				assert.Contains(t, string(b), "kind: ConfigMap")
				assert.Contains(t, string(b), "name: demo")
				assert.Contains(t, string(b), "namespace: prod")
				assert.Contains(t, string(b), "version: v1.2.3")
			},
		},
		{
			name: "successful run with public remote dependency",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "app"), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "app", "kcl.mod"), []byte(`[package]
name = "remote_dep"
edition = "0.0.1"
version = "0.0.1"

[dependencies]
helloworld = { oci = "oci://ghcr.io/kcl-lang/helloworld", tag = "0.1.0" }
`), 0o600))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "app", "main.k"), []byte(`
import helloworld

message = helloworld.The_first_kcl_program
`), 0o600))
			},
			config: configbuiltin.KCLRunConfig{
				Path:    "./app",
				OutPath: "./out.yaml",
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				b, readErr := os.ReadFile(filepath.Join(dir, "out.yaml"))
				require.NoError(t, readErr)
				assert.Contains(t, string(b), "Hello World")
			},
		},
		{
			name:       "missing input file",
			setupFiles: func(*testing.T, string) {},
			config: configbuiltin.KCLRunConfig{
				Path:    "./missing/main.k",
				OutPath: "./out.yaml",
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				assert.NoFileExists(t, filepath.Join(dir, "out.yaml"))
			},
		},
	}

	runner := &kclRunner{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tt.setupFiles(t, tempDir)

			stepCtx := &promotion.StepContext{WorkDir: tempDir}
			result, err := runner.run(t.Context(), stepCtx, tt.config)
			tt.assertions(t, tempDir, result, err)
		})
	}
}
