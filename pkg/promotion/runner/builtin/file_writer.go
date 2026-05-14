package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindFileWrite = "file-write"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindFileWrite,
			Value: newFileWriter,
		},
	)
}

// fileWriter is an implementation of the promotion.StepRunner interface that
// writes content to a file.
type fileWriter struct {
	schemaLoader gojsonschema.JSONLoader
}

// newFileWriter returns an implementation of the promotion.StepRunner interface
// that writes content to a file.
func newFileWriter(promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &fileWriter{schemaLoader: getConfigSchemaLoader(stepKindFileWrite)}
}

// Run implements the promotion.StepRunner interface.
func (f *fileWriter) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := f.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return f.run(ctx, stepCtx, cfg)
}

// convert validates fileWriter configuration against a JSON schema and converts
// it into a builtin.FileWriteConfig struct.
func (f *fileWriter) convert(cfg promotion.Config) (builtin.FileWriteConfig, error) {
	return validateAndConvert[builtin.FileWriteConfig](f.schemaLoader, cfg, stepKindFileWrite)
}

func (f *fileWriter) run(
	_ context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.FileWriteConfig,
) (promotion.StepResult, error) {
	if filepath.IsAbs(cfg.Path) {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("path %q must be relative", cfg.Path)
	}
	cleanPath := filepath.Clean(cfg.Path)
	if cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("path %q attempts to traverse outside the working directory", cfg.Path)
	}

	absPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("could not secure join path %q: %w", cfg.Path, err)
	}

	if _, err = os.Stat(absPath); err == nil && !cfg.Overwrite {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("file %q already exists", cfg.Path)
	}

	if err = os.MkdirAll(filepath.Dir(absPath), 0o700); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error creating parent directories for %q: %w", cfg.Path, err)
	}

	if err = os.WriteFile(absPath, []byte(cfg.Contents), 0o600); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error writing file %q: %w", cfg.Path, err)
	}

	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
}
