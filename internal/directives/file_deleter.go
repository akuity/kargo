package directives

import (
	"context"
	"fmt"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"
	"os"
)

func init() {
	builtins.RegisterPromotionStepRunner(newFileDeleter(), nil)
}

type fileDeleter struct {
	schemaLoader gojsonschema.JSONLoader
}

func newFileDeleter() PromotionStepRunner {
	r := &fileDeleter{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

func (f *fileDeleter) Name() string {
	return "delete"
}

func (f *fileDeleter) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	// Validate the configuration against the JSON Schema.
	if err := validate(f.schemaLoader, gojsonschema.NewGoLoader(stepCtx.Config), f.Name()); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}

	// Convert the configuration into a typed object.
	cfg, err := ConfigToStruct[DeleteConfig](stepCtx.Config)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("could not convert config into %s config: %w", f.Name(), err)
	}

	return f.runPromotionStep(ctx, stepCtx, cfg)
}

func (f *fileDeleter) runPromotionStep(
	_ context.Context,
	stepCtx *PromotionStepContext,
	cfg DeleteConfig,
) (PromotionStepResult, error) {
	pathToDelete, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("could not secure join path %q: %w", cfg.Path, err)
	}

	if err = removePath(pathToDelete); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("failed to delete %q: %w", cfg.Path, sanitizePathError(err, stepCtx.WorkDir))
	}

	return PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, nil
}

func removePath(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}

	if fi.IsDir() {
		return os.RemoveAll(path)
	}

	return os.Remove(path)
}
