package builtin

import (
	"context"
	"fmt"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"
	"os"
	"path/filepath"
)

// fileWriter is an implementation of the promotion.StepRunner interface that
// writes to a file.
//
// If the destination is an existing file, it will be overwritten.
// If the file does not exist, it will be created.
// If the destination is in a directory that does not exist, the directory will be created.
// If the destination is a directory, an error will be returned.
type fileWriter struct {
	schemaLoader gojsonschema.JSONLoader
}

// newFileWriter returns an implementation of the promotion.StepRunner interface
// that writes to a file.
func newFileWriter() promotion.StepRunner {
	r := &fileWriter{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (f *fileWriter) Name() string {
	return "write"
}

// Run implements the promotion.StepRunner interface.
func (f *fileWriter) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	// Validate the configuration against the JSON Schema.
	if err := validate(f.schemaLoader, gojsonschema.NewGoLoader(stepCtx.Config), f.Name()); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	// Convert the configuration into a typed object.
	cfg, err := promotion.ConfigToStruct[builtin.WriteConfig](stepCtx.Config)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("could not convert config into %s config: %w", f.Name(), err)
	}

	return f.run(ctx, stepCtx, cfg)
}

func (f *fileWriter) run(
	_ context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.WriteConfig,
) (promotion.StepResult, error) {
	// Secure join the paths to prevent path traversal attacks.
	outFile, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutFile)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("could not secure join outFile %q: %w", cfg.OutFile, err)
	}

	outFileDir := filepath.Dir(outFile)

	if _, statErr := os.Stat(outFileDir); os.IsNotExist(statErr) {
		if mkdirErr := os.MkdirAll(outFileDir, 0755); mkdirErr != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("failed to write to %q: %w", outFile, err)
		}
	} else if statErr != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to write to %q: %w", outFile, err)
	}

	err = os.WriteFile(outFile, []byte(cfg.Contents), 0644)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to write to %q: %w", outFile, err)
	}

	result := promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}
	result.Output = map[string]any{
		"commitMessage": f.generateCommitMessage(cfg.OutFile),
		"filePath":      outFile,
	}

	return result, nil
}

func (*fileWriter) generateCommitMessage(path string) string {
	return fmt.Sprintf("Updated %s\n- ", path)
}
