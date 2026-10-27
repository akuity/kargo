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
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindDelete = "delete"

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
		relPaths []string
		err      error
	)
	if cfg.PathsAreGlobs {
		relPaths, err = f.resolveGlobs(stepCtx.Root, paths, cfg.Strict)
	} else {
		relPaths, err = f.resolveLiteralPaths(stepCtx.Root, paths, cfg.Strict)
	}
	if err != nil {
		if errors.Is(err, doublestar.ErrBadPattern) {
			// An invalid pattern can never succeed on retry.
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				&promotion.TerminalError{Err: err}
		}
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	for _, relPath := range relPaths {
		// A path may have already been removed by an earlier entry (e.g. a
		// parent directory), so a not-found error here is not fatal.
		if err = removePath(stepCtx.Root, relPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("failed to delete: %w", err)
		}
	}
	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
}

// resolveLiteralPaths resolves each entry as a literal (non-glob) path
// relative to root. It returns an error when strict is true and a path does
// not exist.
func (f *fileDeleter) resolveLiteralPaths(
	root *os.Root,
	paths []string,
	strict bool,
) ([]string, error) {
	var relPaths []string
	seen := make(map[string]struct{})
	for _, path := range paths {
		relPath, err := promotion.WorkspacePath(path)
		if err != nil {
			return nil, err
		}
		exists, err := f.exists(root, relPath)
		if err != nil {
			return nil, fmt.Errorf("could not resolve path %q: %w", path, err)
		}
		if !exists {
			if strict {
				return nil, fmt.Errorf("path %q does not exist", path)
			}
			continue
		}

		if _, ok := seen[relPath]; !ok {
			seen[relPath] = struct{}{}
			relPaths = append(relPaths, relPath)
		}
	}

	return relPaths, nil
}

// resolveGlobs expands the provided glob patterns relative to root and
// returns the deduplicated paths, relative to root, that should be deleted.
// Returns an error when strict is true and a glob pattern matches nothing.
func (f *fileDeleter) resolveGlobs(
	root *os.Root,
	patterns []string,
	strict bool,
) ([]string, error) {
	var relPaths []string
	seen := make(map[string]struct{})
	for _, pattern := range patterns {
		if _, err := promotion.WorkspacePath(pattern); err != nil {
			return nil, err
		}
		// Strip a leading "./" so that glob patterns are matched relative to
		// the working directory.
		globPattern := strings.TrimPrefix(pattern, "./")
		matches, err := f.expandGlob(root, globPattern)
		if err != nil {
			return nil, err
		}

		if len(matches) == 0 {
			if strict {
				return nil, fmt.Errorf("no paths matched pattern %q", pattern)
			}
			continue
		}

		for _, match := range matches {
			relPath := filepath.FromSlash(match)
			if _, ok := seen[relPath]; !ok {
				seen[relPath] = struct{}{}
				relPaths = append(relPaths, relPath)
			}
		}
	}
	return relPaths, nil
}

// expandGlob returns the paths, relative to root, matched by pattern. A
// pattern without metacharacters names one path, and a symlink there is
// matched as a link. The literal prefix of any other pattern must resolve
// inside root; entries the expansion merely walks past are skipped if they
// cannot be read, as a shell would skip them.
func (f *fileDeleter) expandGlob(root *os.Root, pattern string) ([]string, error) {
	if !strings.ContainsAny(pattern, `*?[{\`) {
		exists, err := f.exists(root, filepath.FromSlash(pattern))
		if err != nil {
			return nil, fmt.Errorf("could not resolve path %q: %w", pattern, err)
		}
		if !exists {
			return nil, nil
		}
		return []string{pattern}, nil
	}
	base, _ := doublestar.SplitPattern(pattern)
	if _, err := root.Stat(base); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("could not resolve path %q: %w", base, err)
	}
	matches, err := doublestar.Glob(root.FS(), pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
	}
	return matches, nil
}

// exists reports whether relPath exists within root. A symlink at the leaf
// counts as existing without being followed, so that it is later deleted as a
// link rather than followed through to its target.
func (f *fileDeleter) exists(root *os.Root, relPath string) (bool, error) {
	if _, err := root.Lstat(relPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// removePath deletes the file, directory, or symlink at relPath within root.
// A symlink is removed as a link rather than followed through to its target.
func removePath(root *os.Root, relPath string) error {
	fi, err := root.Lstat(relPath)
	if err != nil {
		return err
	}

	if fi.IsDir() {
		return root.RemoveAll(relPath)
	}

	return root.Remove(relPath)
}
