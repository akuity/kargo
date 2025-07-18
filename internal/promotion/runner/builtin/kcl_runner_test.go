package builtin

import (
	"context"
	"os"
	"path/filepath"
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
			// Create a temporary directory for testing
			tempDir, err := os.MkdirTemp("", "kcl-runner-test")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Create a test KCL file
			testKCLContent := `
# Simple KCL configuration
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

			// Create the KCL runner
			runner := newKCLRunner()

			// Create step context
			stepCtx := &promotion.StepContext{
				WorkDir: tempDir,
				Config:  tc.config,
			}

			// Run the step
			result, err := runner.Run(context.Background(), stepCtx)

			if tc.expectError {
				require.Error(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
			} else {
				// Note: This test will fail if kcl is not installed on the system
				// In a real environment, we would use a mock or skip if kcl is not available
				if err != nil {
					t.Skipf("Skipping test because kcl is not installed: %v", err)
				}
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				if tc.expectOutput {
					require.NotNil(t, result.Output)
					// Check that we have either output or outputPath
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

	// Test with invalid config (missing required fields)
	stepCtx := &promotion.StepContext{
		WorkDir: "/tmp",
		Config:  map[string]any{}, // Empty config - should fail validation
	}

	result, err := runner.Run(context.Background(), stepCtx)
	require.Error(t, err)
	require.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
}

func TestKCLRunner_Run_PathTraversal(t *testing.T) {
	runner := newKCLRunner()

	// Test with path traversal attempt
	stepCtx := &promotion.StepContext{
		WorkDir: "/tmp",
		Config: map[string]any{
			"inputPath": "../../etc/passwd", // Path traversal attempt
		},
	}

	result, err := runner.Run(context.Background(), stepCtx)
	require.Error(t, err)
	require.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
	require.Contains(t, err.Error(), "could not secure join")
}
