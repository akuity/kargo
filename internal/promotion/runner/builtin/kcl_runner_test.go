package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	kcl "kcl-lang.io/kcl-go"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func TestKCLRunner_Name(t *testing.T) {
	r := newKCLRunner()
	require.Equal(t, "kcl-run", r.Name())
}

func TestKCLRunner_Run(t *testing.T) {
	testCases := []struct {
		name         string
		config       map[string]any
		expectError  bool
		expectOutput bool
	}{
		{
			name: "valid config with input path",
			config: map[string]any{
				"inputPath": "test.k",
			},
			expectError:  false,
			expectOutput: true,
		},
		{
			name: "valid config with input and output paths",
			config: map[string]any{
				"inputPath":  "test.k",
				"outputPath": "output.yaml",
			},
			expectError:  false,
			expectOutput: true,
		},
		{
			name: "valid config with additional args",
			config: map[string]any{
				"inputPath": "test.k",
				"args":      []string{"--strict", "true", "--verbose", "true"},
			},
			expectError:  false,
			expectOutput: true,
		},
		{
			name: "valid config with settings",
			config: map[string]any{
				"inputPath": "test.k",
				"settings": map[string]string{
					"debug":   "true",
					"verbose": "true",
				},
			},
			expectError:  false,
			expectOutput: true,
		},
		{
			name: "valid config with OCI settings",
			config: map[string]any{
				"inputPath": "test.k",
				"oci": map[string]string{
					"registry": "ghcr.io",
					"repo":     "kcl-lang",
				},
			},
			expectError:  false,
			expectOutput: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "kcl-runner-test")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			testKCLContent := `
name = "test-app"
version = "1.0.0"
replicas = 3

config = {
    name = name
    version = version
    spec = {
        replicas = replicas
    }
}
`
			kclFile := filepath.Join(tempDir, "test.k")
			err = os.WriteFile(kclFile, []byte(testKCLContent), 0644)
			require.NoError(t, err)

			runner := newKCLRunner()

			stepCtx := &promotion.StepContext{
				WorkDir: tempDir,
				Config:  tc.config,
			}

			result, err := runner.Run(context.Background(), stepCtx)

			if tc.expectError {
				require.Error(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
			} else {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				if tc.expectOutput {
					require.NotNil(t, result.Output)
					_, hasOutput := result.Output["output"]
					_, hasOutputPath := result.Output["outputPath"]
					require.True(t, hasOutput || hasOutputPath)
				}
			}
		})
	}
}

func TestKCLRunner_Run_InvalidConfig(t *testing.T) {
	runner := newKCLRunner()

	stepCtx := &promotion.StepContext{
		WorkDir: "/tmp",
		Config:  map[string]any{},
	}

	result, err := runner.Run(context.Background(), stepCtx)
	require.Error(t, err)
	require.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
}

func TestKCLRunner_Run_PathTraversal(t *testing.T) {
	runner := newKCLRunner()

	stepCtx := &promotion.StepContext{
		WorkDir: "/tmp",
		Config: map[string]any{
			"inputPath": "../../etc/passwd",
		},
	}

	result, err := runner.Run(context.Background(), stepCtx)
	require.Error(t, err)
	require.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
	require.True(t,
		strings.Contains(err.Error(), "could not secure join") ||
			strings.Contains(err.Error(), "does not exist"),
	)
}

func TestKCLRunner_Run_FileCreation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kcl-runner-file-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testKCLContent := `
name = "test-app"
replicas = 2

config = {
    apiVersion = "apps/v1"
    kind = "Deployment"
    metadata = {
        name = name
    }
    spec = {
        replicas = replicas
    }
}
`
	kclFile := filepath.Join(tempDir, "app.k")
	err = os.WriteFile(kclFile, []byte(testKCLContent), 0644)
	require.NoError(t, err)

	runner := newKCLRunner()

	outputFile := filepath.Join(tempDir, "output", "app.yaml")
	stepCtx := &promotion.StepContext{
		WorkDir: tempDir,
		Config: map[string]any{
			"inputPath":  "app.k",
			"outputPath": "output/app.yaml",
		},
	}

	result, err := runner.Run(context.Background(), stepCtx)
	require.NoError(t, err)
	require.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

	require.NotNil(t, result.Output)
	outputPath, hasOutputPath := result.Output["outputPath"]
	require.True(t, hasOutputPath)
	require.Equal(t, "output/app.yaml", outputPath)

	require.FileExists(t, outputFile)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	contentStr := string(content)
	require.Contains(t, contentStr, "apiVersion: apps/v1")
	require.Contains(t, contentStr, "kind: Deployment")
	require.Contains(t, contentStr, "name: test-app")
	require.Contains(t, contentStr, "replicas: 2")
}

func TestKCLRunner_resolveKCLFiles(t *testing.T) {
	runner := newKCLRunner().(*kclRunner)

	tempDir, err := os.MkdirTemp("", "kcl-resolve-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	kclFile1 := filepath.Join(tempDir, "app.k")
	kclFile2 := filepath.Join(tempDir, "config.k")
	txtFile := filepath.Join(tempDir, "readme.txt")

	testContent := `name = "test"`
	require.NoError(t, os.WriteFile(kclFile1, []byte(testContent), 0644))
	require.NoError(t, os.WriteFile(kclFile2, []byte(testContent), 0644))
	require.NoError(t, os.WriteFile(txtFile, []byte("readme"), 0644))

	t.Run("single file", func(t *testing.T) {
		files, err := runner.resolveKCLFiles(tempDir, "app.k")
		require.NoError(t, err)
		require.Len(t, files, 1)
		require.Equal(t, kclFile1, files[0])
	})

	t.Run("directory with KCL files", func(t *testing.T) {
		files, err := runner.resolveKCLFiles(tempDir, ".")
		require.NoError(t, err)
		require.Len(t, files, 2)
		require.Contains(t, files, kclFile1)
		require.Contains(t, files, kclFile2)
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := runner.resolveKCLFiles(tempDir, "nonexistent.k")
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("directory with no KCL files", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		require.NoError(t, os.Mkdir(emptyDir, 0755))

		_, err := runner.resolveKCLFiles(tempDir, "empty")
		require.Error(t, err)
		require.Contains(t, err.Error(), "no KCL files (*.k) found")
	})

	t.Run("path traversal protection", func(t *testing.T) {
		_, err := runner.resolveKCLFiles(tempDir, "../../../etc/passwd")
		require.Error(t, err)
		require.True(t,
			strings.Contains(err.Error(), "could not secure join") ||
				strings.Contains(err.Error(), "does not exist"),
		)
	})
}

func TestKCLRunner_buildKCLOptions(t *testing.T) {
	runner := newKCLRunner().(*kclRunner)

	tempDir, err := os.MkdirTemp("", "kcl-options-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	kclFiles := []string{filepath.Join(tempDir, "app.k")}

	t.Run("basic options", func(t *testing.T) {
		cfg := builtin.KCLRunConfig{
			InputPath: "app.k",
		}

		opts, err := runner.buildKCLOptions(tempDir, kclFiles, cfg, nil)
		require.NoError(t, err)
		require.NotEmpty(t, opts)
	})

	t.Run("with settings", func(t *testing.T) {
		cfg := builtin.KCLRunConfig{
			InputPath: "app.k",
			Settings: map[string]string{
				"debug":   "true",
				"verbose": "1",
			},
		}

		opts, err := runner.buildKCLOptions(tempDir, kclFiles, cfg, nil)
		require.NoError(t, err)
		require.NotEmpty(t, opts)
	})

	t.Run("with args", func(t *testing.T) {
		cfg := builtin.KCLRunConfig{
			InputPath: "app.k",
			Args:      []string{"--strict", "true"},
		}

		opts, err := runner.buildKCLOptions(tempDir, kclFiles, cfg, nil)
		require.NoError(t, err)
		require.NotEmpty(t, opts)
	})

	t.Run("with both settings and args", func(t *testing.T) {
		cfg := builtin.KCLRunConfig{
			InputPath: "app.k",
			Settings: map[string]string{
				"debug": "true",
			},
			Args: []string{"--verbose", "true"},
		}

		opts, err := runner.buildKCLOptions(tempDir, kclFiles, cfg, nil)
		require.NoError(t, err)
		require.NotEmpty(t, opts)
	})

	t.Run("with OCI config", func(t *testing.T) {
		cfg := builtin.KCLRunConfig{
			InputPath: "app.k",
			OCI: &builtin.KCLOCIConfig{
				Registry: "custom-registry.io",
				Repo:     "my-org",
			},
		}

		opts, err := runner.buildKCLOptions(tempDir, kclFiles, cfg, nil)
		require.NoError(t, err)
		require.NotEmpty(t, opts)
	})

	t.Run("with OCI config using defaults", func(t *testing.T) {
		cfg := builtin.KCLRunConfig{
			InputPath: "app.k",
			OCI:       &builtin.KCLOCIConfig{},
		}

		opts, err := runner.buildKCLOptions(tempDir, kclFiles, cfg, nil)
		require.NoError(t, err)
		require.NotEmpty(t, opts)
	})

	t.Run("with partial OCI config", func(t *testing.T) {
		cfg := builtin.KCLRunConfig{
			InputPath: "app.k",
			OCI: &builtin.KCLOCIConfig{
				Registry: "custom-registry.io",
			},
		}

		opts, err := runner.buildKCLOptions(tempDir, kclFiles, cfg, nil)
		require.NoError(t, err)
		require.NotEmpty(t, opts)
	})
}

func TestKCLRunner_executeKCL(t *testing.T) {
	runner := newKCLRunner().(*kclRunner)

	tempDir, err := os.MkdirTemp("", "kcl-execute-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testKCLContent := `
name = "test-app"
version = "1.0.0"

config = {
    name = name
    version = version
}
`
	kclFile := filepath.Join(tempDir, "test.k")
	require.NoError(t, os.WriteFile(kclFile, []byte(testKCLContent), 0644))

	t.Run("successful execution", func(t *testing.T) {
		opts := []kcl.Option{
			kcl.WithKFilenames(kclFile),
			kcl.WithWorkDir(tempDir),
		}

		cfg := builtin.KCLRunConfig{InputPath: "test.k"}

		result, err := runner.executeKCL(context.Background(), opts, cfg, nil, []string{kclFile}, tempDir)
		require.NoError(t, err)
		require.NotEmpty(t, result)
		require.Contains(t, result, "name: test-app")
		require.Contains(t, result, "'1.0.0'")
	})

	t.Run("execution with invalid KCL", func(t *testing.T) {
		invalidKCLContent := `invalid KCL syntax {{{`
		invalidFile := filepath.Join(tempDir, "invalid.k")
		require.NoError(t, os.WriteFile(invalidFile, []byte(invalidKCLContent), 0644))

		opts := []kcl.Option{
			kcl.WithKFilenames(invalidFile),
			kcl.WithWorkDir(tempDir),
		}

		cfg := builtin.KCLRunConfig{InputPath: "invalid.k"}

		_, err := runner.executeKCL(context.Background(), opts, cfg, nil, []string{invalidFile}, tempDir)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error executing kcl")
	})
}

func TestKCLRunner_handleOutput(t *testing.T) {
	runner := newKCLRunner().(*kclRunner)

	tempDir, err := os.MkdirTemp("", "kcl-output-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testYAMLOutput := `config:
  name: test-app
  version: 1.0.0
`

	t.Run("output to result only", func(t *testing.T) {
		result, err := runner.handleOutput(tempDir, "", testYAMLOutput)
		require.NoError(t, err)
		require.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)
		require.NotNil(t, result.Output)

		output, hasOutput := result.Output["output"]
		require.True(t, hasOutput)
		require.Equal(t, testYAMLOutput, output)

		_, hasOutputPath := result.Output["outputPath"]
		require.False(t, hasOutputPath)
	})

	t.Run("output to file", func(t *testing.T) {
		outputPath := "output/result.yaml"

		result, err := runner.handleOutput(tempDir, outputPath, testYAMLOutput)
		require.NoError(t, err)
		require.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)
		require.NotNil(t, result.Output)

		actualOutputPath, hasOutputPath := result.Output["outputPath"]
		require.True(t, hasOutputPath)
		require.Equal(t, outputPath, actualOutputPath)

		_, hasOutput := result.Output["output"]
		require.False(t, hasOutput)

		fullOutputPath := filepath.Join(tempDir, outputPath)
		require.FileExists(t, fullOutputPath)

		content, err := os.ReadFile(fullOutputPath)
		require.NoError(t, err)
		require.Equal(t, testYAMLOutput, string(content))
	})

	t.Run("output to nested directory", func(t *testing.T) {
		outputPath := "nested/deep/result.yaml"

		result, err := runner.handleOutput(tempDir, outputPath, testYAMLOutput)
		require.NoError(t, err)
		require.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

		fullOutputPath := filepath.Join(tempDir, outputPath)
		require.FileExists(t, fullOutputPath)
	})

	t.Run("path traversal protection", func(t *testing.T) {
		outputPath := "safe/output.yaml"

		result, err := runner.handleOutput(tempDir, outputPath, testYAMLOutput)
		require.NoError(t, err)
		require.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

		fullOutputPath := filepath.Join(tempDir, outputPath)
		require.FileExists(t, fullOutputPath)
	})
}

func TestKCLRunner_Run_WithExternalDependencies(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kcl-runner-k8s-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	kclContent := `# Import and use the contents of the external dependency 'k8s'.
import k8s.api.apps.v1 as apps

apps.Deployment {
    metadata.name = "nginx-deployment"
    metadata.labels.app = "nginx"
    spec: {
        replicas = 3
        selector.matchLabels = metadata.labels
        template: {
            metadata.labels = metadata.labels
            spec.containers = [{
                name = metadata.labels.app
                image = "nginx:1.14.2"
                ports: [{
                    containerPort = 80
                }]
            }]
        }
    }
}`

	kclModContent := `[package]
name = "my-module"
edition = "v0.11.2"
version = "0.0.1"

[dependencies]
k8s = { oci = "oci://ghcr.io/kcl-lang/k8s", tag = "1.32.4" }`

	kclModLockContent := `[dependencies]
  [dependencies.k8s]
    name = "k8s"
    full_name = "k8s_1.32.4"
    version = "1.32.4"
    sum = "WrltC/mTXtdzmhBZxlvM71wJL5C/UZ/vW+bF3nFvNbM="
    reg = "ghcr.io"
    repo = "kcl-lang/k8s"
    oci_tag = "1.32.4"`

	kclFile := filepath.Join(tempDir, "deployment.k")
	kclMod := filepath.Join(tempDir, "kcl.mod")
	kclModLock := filepath.Join(tempDir, "kcl.mod.lock")

	require.NoError(t, os.WriteFile(kclFile, []byte(kclContent), 0644))
	require.NoError(t, os.WriteFile(kclMod, []byte(kclModContent), 0644))
	require.NoError(t, os.WriteFile(kclModLock, []byte(kclModLockContent), 0644))

	runner := newKCLRunner()

	t.Run("with k8s dependencies using default OCI", func(t *testing.T) {
		stepCtx := &promotion.StepContext{
			WorkDir: tempDir,
			Config: map[string]any{
				"inputPath": "deployment.k",
				"oci": map[string]string{
					"registry": "ghcr.io",
					"repo":     "kcl-lang",
				},
			},
		}

		result, err := runner.Run(context.Background(), stepCtx)
		require.NoError(t, err)
		require.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

		require.NotNil(t, result.Output)
		output, hasOutput := result.Output["output"]
		require.True(t, hasOutput)

		yamlOutput := output.(string)
		require.Contains(t, yamlOutput, "apiVersion: apps/v1")
		require.Contains(t, yamlOutput, "kind: Deployment")
		require.Contains(t, yamlOutput, "name: nginx-deployment")
		require.Contains(t, yamlOutput, "app: nginx")
		require.Contains(t, yamlOutput, "replicas: 3")
		require.Contains(t, yamlOutput, "image: nginx:1.14.2")
		require.Contains(t, yamlOutput, "containerPort: 80")
	})

	t.Run("with k8s dependencies using custom OCI registry", func(t *testing.T) {
		stepCtx := &promotion.StepContext{
			WorkDir: tempDir,
			Config: map[string]any{
				"inputPath": "deployment.k",
				"oci": map[string]string{
					"registry": "artifacts.rbi.tech/ghcr-io-docker-proxy",
					"repo":     "kcl-lang",
				},
			},
		}

		result, err := runner.Run(context.Background(), stepCtx)
		require.NoError(t, err)
		require.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

		require.NotNil(t, result.Output)
		output, hasOutput := result.Output["output"]
		require.True(t, hasOutput)

		yamlOutput := output.(string)
		require.Contains(t, yamlOutput, "apiVersion: apps/v1")
		require.Contains(t, yamlOutput, "kind: Deployment")
		require.Contains(t, yamlOutput, "name: nginx-deployment")
	})

	t.Run("with k8s dependencies and file output", func(t *testing.T) {
		outputPath := "k8s-manifests/nginx.yaml"
		stepCtx := &promotion.StepContext{
			WorkDir: tempDir,
			Config: map[string]any{
				"inputPath":  "deployment.k",
				"outputPath": outputPath,
				"oci": map[string]string{
					"registry": "ghcr.io",
					"repo":     "kcl-lang",
				},
			},
		}

		result, err := runner.Run(context.Background(), stepCtx)
		require.NoError(t, err)
		require.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

		require.NotNil(t, result.Output)
		actualOutputPath, hasOutputPath := result.Output["outputPath"]
		require.True(t, hasOutputPath)
		require.Equal(t, outputPath, actualOutputPath)

		fullOutputPath := filepath.Join(tempDir, outputPath)
		require.FileExists(t, fullOutputPath)

		content, err := os.ReadFile(fullOutputPath)
		require.NoError(t, err)
		yamlContent := string(content)
		require.Contains(t, yamlContent, "apiVersion: apps/v1")
		require.Contains(t, yamlContent, "kind: Deployment")
		require.Contains(t, yamlContent, "name: nginx-deployment")
		require.Contains(t, yamlContent, "app: nginx")
		require.Contains(t, yamlContent, "replicas: 3")
		require.Contains(t, yamlContent, "image: nginx:1.14.2")
		require.Contains(t, yamlContent, "containerPort: 80")
	})
}
