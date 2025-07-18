package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"
	kcl "kcl-lang.io/kcl-go"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// kclRunner is an implementation of the promotion.StepRunner interface that
// executes KCL (Kushion Configuration Language) programs.
type kclRunner struct {
	schemaLoader gojsonschema.JSONLoader
}

// newKCLRunner returns an implementation of the promotion.StepRunner interface
// that executes KCL programs.
func newKCLRunner() promotion.StepRunner {
	r := &kclRunner{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (k *kclRunner) Name() string {
	return "kcl-run"
}

// Run implements the promotion.StepRunner interface.
func (k *kclRunner) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	failure := promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}

	// Validate the configuration against the JSON Schema
	if err := validate(
		k.schemaLoader,
		gojsonschema.NewGoLoader(stepCtx.Config),
		k.Name(),
	); err != nil {
		return failure, err
	}

	// Convert the configuration into a typed struct
	cfg, err := promotion.ConfigToStruct[builtin.KCLRunConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", k.Name(), err)
	}

	return k.run(ctx, stepCtx, cfg)
}

func (k *kclRunner) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.KCLRunConfig,
) (promotion.StepResult, error) {
	failure := promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}
	logger := logging.LoggerFromContext(ctx)

	// Secure join the input path to prevent path traversal attacks
	inputPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.InputPath)
	if err != nil {
		return failure, fmt.Errorf("could not secure join inputPath %q: %w", cfg.InputPath, err)
	}

	// Check if the input path exists
	if _, err := os.Stat(inputPath); err != nil {
		return failure, fmt.Errorf("input path %q does not exist: %w", cfg.InputPath, err)
	}

	// Prepare KCL run options
	var opts []kcl.Option

	// Add the input path/file
	opts = append(opts, kcl.WithKFilenames(inputPath))

	// Add working directory
	opts = append(opts, kcl.WithWorkDir(stepCtx.WorkDir))

	// Add settings as key-value pairs
	if len(cfg.Settings) > 0 {
		var keyValuePairs []string
		for key, value := range cfg.Settings {
			keyValuePairs = append(keyValuePairs, fmt.Sprintf("%s=%s", key, value))
		}
		opts = append(opts, kcl.WithOptions(keyValuePairs...))
	}

	// Add any additional arguments as options
	if len(cfg.Args) > 0 {
		opts = append(opts, kcl.WithOptions(cfg.Args...))
	}

	logger.Debug("executing kcl with options", "inputPath", cfg.InputPath, "args", cfg.Args, "settings", cfg.Settings)

	// Execute KCL
	result, err := kcl.Run(inputPath, opts...)
	if err != nil {
		return failure, fmt.Errorf("error executing kcl: %w", err)
	}

	// Get the YAML result
	yamlResult := result.GetRawYamlResult()

	logger.Debug("kcl executed successfully", "outputLength", len(yamlResult))

	// Handle output
	stepResult := promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}

	if cfg.OutputPath != "" {
		// If output path is specified, write the result to file
		outputPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutputPath)
		if err != nil {
			return failure, fmt.Errorf("could not secure join outputPath %q: %w", cfg.OutputPath, err)
		}

		// Create the output directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return failure, fmt.Errorf("could not create output directory: %w", err)
		}

		// Write the result to file
		if err := os.WriteFile(outputPath, []byte(yamlResult), 0644); err != nil {
			return failure, fmt.Errorf("could not write output to file %q: %w", cfg.OutputPath, err)
		}

		stepResult.Output = map[string]any{
			"outputPath": cfg.OutputPath,
		}
	} else {
		// If no output path, return the YAML result
		stepResult.Output = map[string]any{
			"output": yamlResult,
		}
	}

	return stepResult, nil
}
