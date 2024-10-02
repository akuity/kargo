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

	"github.com/akuity/kargo/internal/logging"
)

func init() {
	builtins.RegisterPromotionStepRunner(newFileCopier(), nil)
}

// fileCopier is an implementation of the PromotionStepRunner interface that
// copies a file or directory.
//
// The copy is recursive, merging directories if the destination directory
// already exists. If the destination is an existing file, it will be
// overwritten. Symlinks are ignored.
type fileCopier struct {
	schemaLoader gojsonschema.JSONLoader
}

// newFileCopier returns an implementation of the PromotionStepRunner interface
// that copies a file or directory.
func newFileCopier() PromotionStepRunner {
	r := &fileCopier{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the PromotionStepRunner interface.
func (f *fileCopier) Name() string {
	return "copy"
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (f *fileCopier) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	// Validate the configuration against the JSON Schema.
	if err := validate(f.schemaLoader, gojsonschema.NewGoLoader(stepCtx.Config), f.Name()); err != nil {
		return PromotionStepResult{Status: PromotionStatusFailed}, err
	}

	// Convert the configuration into a typed object.
	cfg, err := configToStruct[CopyConfig](stepCtx.Config)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailed},
			fmt.Errorf("could not convert config into %s config: %w", f.Name(), err)
	}

	return f.runPromotionStep(ctx, stepCtx, cfg)
}

func (f *fileCopier) runPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	cfg CopyConfig,
) (PromotionStepResult, error) {
	// Secure join the paths to prevent path traversal attacks.
	inPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.InPath)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailed},
			fmt.Errorf("could not secure join inPath %q: %w", cfg.InPath, err)
	}
	outPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailed},
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
		return PromotionStepResult{Status: PromotionStatusFailed},
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
