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

func (d *copyDirective) Name() string {
	return "copy"
}

func (d *copyDirective) Run(ctx context.Context, stepCtx *StepContext) (Result, error) {
	failure := Result{Status: StatusFailure}
	// Validate the configuration against the JSON Schema.
	if err := validate(d.schemaLoader, gojsonschema.NewGoLoader(stepCtx.Config), d.Name()); err != nil {
		return failure, err
	}

	// Convert the configuration into a typed object.
	cfg, err := configToStruct[CopyConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", d.Name(), err)
	}

	return d.run(ctx, stepCtx, cfg)
}

func (d *copyDirective) run(ctx context.Context, stepCtx *StepContext, cfg CopyConfig) (Result, error) {
	// Secure join the paths to prevent path traversal attacks.
	inPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.InPath)
	if err != nil {
		return Result{Status: StatusFailure}, fmt.Errorf("could not secure join inPath %q: %w", cfg.InPath, err)
	}
	outPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return Result{Status: StatusFailure}, fmt.Errorf("could not secure join outPath %q: %w", cfg.OutPath, err)
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
		return Result{Status: StatusFailure}, fmt.Errorf("failed to copy %q to %q: %w", cfg.InPath, cfg.OutPath, err)
	}
	return Result{Status: StatusSuccess}, nil
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
		sanitizedPath, relErr := filepath.Rel(workDir, pathErr.Path)
		if relErr != nil || strings.Contains(sanitizedPath, "..") {
			// If we can't make it relative, just use the filename.
			sanitizedPath = filepath.Base(pathErr.Path)
		}

		// Reconstruct the error with the sanitized path.
		return &fs.PathError{
			Op:   pathErr.Op,
			Path: sanitizedPath,
			Err:  pathErr.Err,
		}
	}
	// Return the original error if it's not a path error.
	return err
}
