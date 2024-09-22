package directives

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/otiai10/copy"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

func init() {
	// Register the copy directive with the builtins registry.
	builtins.RegisterDirective(&copyDirective{}, nil)
}

// copyDirective is a directive that copies a file or directory.
//
// The copy is recursive, merging directories if the destination directory
// already exists. If the destination is an existing file, it will be
// overwritten. Symlinks are ignored.
type copyDirective struct {
	schemaLoader gojsonschema.JSONLoader
}

// Name implements the Directive interface.
func (d *copyDirective) Name() string {
	return "copy"
}

// RunPromotionStep implements the Directive interface.
func (d *copyDirective) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	// Validate the configuration against the JSON Schema.
	if err := validate(d.schemaLoader, gojsonschema.NewGoLoader(stepCtx.Config), d.Name()); err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure}, err
	}

	// Convert the configuration into a typed object.
	cfg, err := configToStruct[CopyConfig](stepCtx.Config)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("could not convert config into %s config: %w", d.Name(), err)
	}

	return d.runPromotionStep(ctx, stepCtx, cfg)
}

// RunHealthCheckStep implements the Directive interface.
func (d *copyDirective) RunHealthCheckStep(
	context.Context,
	*HealthCheckStepContext,
) HealthCheckStepResult {
	return HealthCheckStepResult{Status: kargoapi.HealthStateNotApplicable}
}

func (d *copyDirective) runPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	cfg CopyConfig,
) (PromotionStepResult, error) {
	// Secure join the paths to prevent path traversal attacks.
	inPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.InPath)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("could not secure join inPath %q: %w", cfg.InPath, err)
	}
	outPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("could not secure join outPath %q: %w", cfg.OutPath, err)
	}

	// Perform the copy operation.
	opts := copy.Options{
		OnSymlink: func(src string) copy.SymlinkAction {
			logging.LoggerFromContext(ctx).Trace("ignoring symlink", "src", src)
			return copy.Skip
		},
		OnError: func(_, _ string, err error) error {
			return sanitizePathError(err, stepCtx.WorkDir)
		},
	}
	if err = copy.Copy(inPath, outPath, opts); err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("failed to copy %q to %q: %w", cfg.InPath, cfg.OutPath, err)
	}
	return PromotionStepResult{Status: PromotionStatusSuccess}, nil
}

// sanitizePathError sanitizes the path in a path error to be relative to the
// work directory. If the path cannot be made relative, the filename is used
// instead.
//
// This is useful for making error messages more user-friendly, as the work
// directory is typically a temporary directory that the user does not care
// about.
func sanitizePathError(err error, workDir string) error {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		// Reconstruct the error with the sanitized path.
		return &fs.PathError{
			Op:   pathErr.Op,
			Path: relativePath(workDir, pathErr.Path),
			Err:  pathErr.Err,
		}
	}
	// Return the original error if it's not a path error.
	return err
}

// relativePath returns a path relative to the base path, or the base if
// the path cannot be made relative.
func relativePath(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil || strings.Contains(rel, "..") {
		// If we can't make it relative, just use the filename.
		return filepath.Base(path)
	}
	return rel
}
