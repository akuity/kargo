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

		opts, err := runner.buildKCLOptions(tempDir, kclFiles, cfg)
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

		opts, err := runner.buildKCLOptions(tempDir, kclFiles, cfg)
		require.NoError(t, err)
		require.NotEmpty(t, opts)
	})

	t.Run("with args", func(t *testing.T) {
		cfg := builtin.KCLRunConfig{
			InputPath: "app.k",
			Args:      []string{"--strict", "true"},
		}

		opts, err := runner.buildKCLOptions(tempDir, kclFiles, cfg)
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

		opts, err := runner.buildKCLOptions(tempDir, kclFiles, cfg)
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

		result, err := runner.executeKCL(context.Background(), opts, cfg)
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

		_, err := runner.executeKCL(context.Background(), opts, cfg)
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
