package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
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
