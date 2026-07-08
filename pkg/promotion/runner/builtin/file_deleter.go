package builtin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/io/fs"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindDelete = "delete"

// errPathTraversal is returned when a configured path would result in the
// deletion of something outside the working directory.
var errPathTraversal = errors.New("path attempts to traverse outside the working directory")

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindDelete,
			Value: newFileDeleter,
		},
	)
}

// fileDeleter is an implementation of the promotion.StepRunner interface that
// deletes one or more files or directories.
type fileDeleter struct {
	schemaLoader gojsonschema.JSONLoader
}

// newFileDeleter returns an implementation of the promotion.StepRunner interface
// that deletes one or more files or directories.
func newFileDeleter(promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &fileDeleter{schemaLoader: getConfigSchemaLoader(stepKindDelete)}
}

// Run implements the promotion.StepRunner interface.
func (f *fileDeleter) Run(
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

// convert validates fileDeleter configuration against a JSON schema and
// converts it into a builtin.DeleteConfig struct.
func (f *fileDeleter) convert(cfg promotion.Config) (builtin.DeleteConfig, error) {
	return validateAndConvert[builtin.DeleteConfig](f.schemaLoader, cfg, stepKindDelete)
}

func (f *fileDeleter) run(
	_ context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.DeleteConfig,
) (promotion.StepResult, error) {
	paths := cfg.Paths
	if cfg.Path != "" {
		paths = append([]string{cfg.Path}, paths...)
	}

	var (
		absPaths []string
		err      error
	)
	if cfg.PathsAreGlobs {
		absPaths, err = f.resolveGlobs(stepCtx.WorkDir, paths, cfg.Strict)
	} else {
		absPaths, err = f.resolveLiteralPaths(stepCtx.WorkDir, paths, cfg.Strict)
	}
	if err != nil {
		if errors.Is(err, doublestar.ErrBadPattern) {
			// An invalid pattern can never succeed on retry.
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				&promotion.TerminalError{Err: err}
		}
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	for _, absPath := range absPaths {
		// A path may have already been removed by an earlier entry (e.g. a
		// parent directory), so a not-found error here is not fatal.
		if err = removePath(absPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("failed to delete: %w", fs.SanitizePathError(err, stepCtx.WorkDir))
		}
	}
	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
}

// resolveLiteralPaths resolves each entry as a literal (non-glob) path. It
// returns an error when strict is true and a path does not exist.
func (f *fileDeleter) resolveLiteralPaths(
	workDir string,
	paths []string,
	strict bool,
) ([]string, error) {
	var absPaths []string
	seen := make(map[string]struct{})
	for _, path := range paths {
		absPath, err := f.secureResolve(workDir, path)
		if err != nil {
			return nil, fmt.Errorf("could not resolve path %q: %w", path, err)
		}
		if absPath == "" {
			// The path does not exist.
			if strict {
				return nil, fmt.Errorf("path %q does not exist", path)
			}
			continue
		}

		if _, ok := seen[absPath]; !ok {
			seen[absPath] = struct{}{}
			absPaths = append(absPaths, absPath)
		}
	}

	return absPaths, nil
}

// resolveGlobs expands the provided glob patterns relative to workDir and
// returns the deduplicated absolute paths that should be deleted. Returns
// an error when strict is true and a glob pattern matches nothing.
func (f *fileDeleter) resolveGlobs(
	workDir string,
	patterns []string,
	strict bool,
) ([]string, error) {
	root, err := filepath.EvalSymlinks(workDir)
	if err != nil {
		return nil, fmt.Errorf("could not resolve working directory: %w", err)
	}
	workDirFS := os.DirFS(root)

	var absPaths []string
	seen := make(map[string]struct{})
	for _, pattern := range patterns {
		// Strip a leading "./" so that glob patterns are matched relative to
		// the working directory.
		globPattern := strings.TrimPrefix(pattern, "./")
		matches, err := doublestar.Glob(workDirFS, globPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}

		if len(matches) == 0 {
			if strict {
				return nil, fmt.Errorf("no paths matched pattern %q", pattern)
			}
			continue
		}

		for _, match := range matches {
			absPath, err := f.secureResolve(root, filepath.FromSlash(match))
			if err != nil {
				return nil, err
			}
			if absPath == "" {
				// Matched but no longer exists; nothing to delete.
				continue
			}
			if _, ok := seen[absPath]; !ok {
				seen[absPath] = struct{}{}
				absPaths = append(absPaths, absPath)
			}
		}
	}
	return absPaths, nil
}

// secureResolve resolves path against workDir and returns its absolute
// location, guaranteeing the result stays within workDir. A symlink at the leaf
// is returned as-is so that it is deleted as a link rather than followed through
// to its target.
func (f *fileDeleter) secureResolve(workDir, path string) (string, error) {
	// workDir itself might contain symlinks, so we need to resolve it first.
	root, err := filepath.EvalSymlinks(workDir)
	if err != nil {
		return "", fmt.Errorf("could not resolve working directory: %w", err)
	}

	// Reject any path that lexically escapes the working directory.
	absPath := filepath.Join(root, path)
	if !fs.IsSubPath(root, absPath) {
		return "", errPathTraversal
	}

	resolvedParent, err := filepath.EvalSymlinks(filepath.Dir(absPath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// The parent does not exist, so neither does the target.
			return "", nil
		}
		return "", err
	}
	if !fs.IsSubPath(root, resolvedParent) {
		return "", errPathTraversal
	}

	// Use Lstat so that a symlink counts as existing and is removed as a link
	// rather than followed.
	target := filepath.Join(resolvedParent, filepath.Base(absPath))
	if _, err = os.Lstat(target); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fs.SanitizePathError(err, root)
	}

	return target, nil
}

// removePath deletes the file, directory, or symlink at path. A symlink is
// removed as a link rather than followed through to its target.
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
