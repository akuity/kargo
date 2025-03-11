package directives

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/x/directive/builtin"
)

// fileDeleter is an implementation of the PromotionStepRunner interface that
// deletes a file or directory.
type fileDeleter struct {
	schemaLoader gojsonschema.JSONLoader
}

// newFileDeleter returns an implementation of the PromotionStepRunner interface
// that deletes a file or directory.
func newFileDeleter() PromotionStepRunner {
	r := &fileDeleter{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the PromotionStepRunner interface
func (f *fileDeleter) Name() string {
	return "delete"
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (f *fileDeleter) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	// Validate the configuration against the JSON Schema.
	if err := validate(f.schemaLoader, gojsonschema.NewGoLoader(stepCtx.Config), f.Name()); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}

	// Convert the configuration into a typed object.
	cfg, err := ConfigToStruct[builtin.DeleteConfig](stepCtx.Config)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("could not convert config into %s config: %w", f.Name(), err)
	}

	return f.runPromotionStep(ctx, stepCtx, cfg)
}

func (f *fileDeleter) runPromotionStep(
	_ context.Context,
	stepCtx *PromotionStepContext,
	cfg builtin.DeleteConfig,
) (PromotionStepResult, error) {
	absPath, err := f.resolveAbsPath(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("could not secure join path %q: %w", cfg.Path, err)
	}

	symlink, err := f.isSymlink(absPath)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}

	if symlink {
		if f.ignoreNotExist(cfg.Strict, os.Remove(absPath)) != nil {
			return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, err
		}
	} else {
		// Secure join the paths to prevent path traversal.
		pathToDelete, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Path)
		if err != nil {
			return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
				fmt.Errorf("could not secure join path %q: %w", cfg.Path, err)
		}

		if err = f.ignoreNotExist(
			cfg.Strict,
			removePath(pathToDelete),
		); err != nil {
			return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
				fmt.Errorf("failed to delete %q: %w", cfg.Path, sanitizePathError(err, stepCtx.WorkDir))
		}
	}

	return PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, nil
}

// isSymlink checks if a path is a symlink.
func (f *fileDeleter) isSymlink(path string) (bool, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		// If file doesn't exist, it's not a symlink
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return fi.Mode()&os.ModeSymlink != 0, nil
}

// resolveAbsPath resolves the absolute path from the workDir base path.
func (f *fileDeleter) resolveAbsPath(workDir string, path string) (string, error) {
	absBase, err := filepath.Abs(workDir)
	if err != nil {
		return "", err
	}

	fullPath := filepath.Join(workDir, path)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}

	// Get the relative path from base to the requested path
	// If the requested path tries to escape, this will return
	// an error or a path starting with "../"
	relPath, err := filepath.Rel(absBase, absPath)
	if err != nil {
		return "", err
	}

	// Check if path attempts to escape
	if strings.HasPrefix(relPath, "..") {
		return "", errors.New("path attempts to traverse outside the working directory")
	}

	return absPath, nil
}

// ignoreNotExist ignores os.IsNotExist errors depending on the strict
// flag. If strict is false and the error is os.IsNotExist, it returns
// nil.
func (f *fileDeleter) ignoreNotExist(strict bool, err error) error {
	if !strict && os.IsNotExist(err) {
		return nil
	}
	return err
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
