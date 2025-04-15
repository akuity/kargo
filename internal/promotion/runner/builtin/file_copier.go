package builtin

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/otiai10/copy"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// fileCopier is an implementation of the promotion.StepRunner interface that
// copies a file or directory.
//
// The copy is recursive, merging directories if the destination directory
// already exists. If the destination is an existing file, it will be
// overwritten. Symlinks are ignored.
type fileCopier struct {
	schemaLoader gojsonschema.JSONLoader
}

// newFileCopier returns an implementation of the promotion.StepRunner interface
// that copies a file or directory.
func newFileCopier() promotion.StepRunner {
	r := &fileCopier{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (f *fileCopier) Name() string {
	return "copy"
}

// Run implements the promotion.StepRunner interface.
func (f *fileCopier) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	// Validate the configuration against the JSON Schema.
	if err := validate(f.schemaLoader, gojsonschema.NewGoLoader(stepCtx.Config), f.Name()); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepPhaseErrored}, err
	}

	// Convert the configuration into a typed object.
	cfg, err := promotion.ConfigToStruct[builtin.CopyConfig](stepCtx.Config)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepPhaseErrored},
			fmt.Errorf("could not convert config into %s config: %w", f.Name(), err)
	}

	return f.run(ctx, stepCtx, cfg)
}

func (f *fileCopier) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.CopyConfig,
) (promotion.StepResult, error) {
	// Secure join the paths to prevent path traversal attacks.
	inPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.InPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepPhaseErrored},
			fmt.Errorf("could not secure join inPath %q: %w", cfg.InPath, err)
	}
	outPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepPhaseErrored},
			fmt.Errorf("could not secure join outPath %q: %w", cfg.OutPath, err)
	}

	// Load the ignore rules.
	matcher, err := f.loadIgnoreRules(inPath, cfg.Ignore)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepPhaseErrored},
			fmt.Errorf("failed to load ignore rules: %w", err)
	}

	// Perform the copy operation.
	opts := copy.Options{
		OnSymlink: func(src string) copy.SymlinkAction {
			logging.LoggerFromContext(ctx).Trace("ignoring symlink", "src", src)
			return copy.Skip
		},
		Skip: func(f os.FileInfo, src, _ string) (bool, error) {
			return matcher.Match(strings.Split(src, string(filepath.Separator)), f.IsDir()), nil
		},
		OnError: func(_, _ string, err error) error {
			return sanitizePathError(err, stepCtx.WorkDir)
		},
	}
	if err = copy.Copy(inPath, outPath, opts); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepPhaseErrored},
			fmt.Errorf("failed to copy %q to %q: %w", cfg.InPath, cfg.OutPath, err)
	}
	return promotion.StepResult{Status: kargoapi.PromotionStepPhaseSucceeded}, nil
}

// loadIgnoreRules loads the ignore rules from the given string. The rules are
// separated by newlines, and comments are allowed with the '#' character.
// It returns a gitignore.Matcher that can be used to match paths against the
// rules.
func (f *fileCopier) loadIgnoreRules(inPath, rules string) (gitignore.Matcher, error) {
	// Determine the domain for the ignore rules. For directories, the domain is
	// the directory itself. For files, the domain is the parent directory.
	fi, err := os.Lstat(inPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Let the error slide if the path does not exist, to allow
			// the copy operation to fail later. This provides a more
			// predictable user experience.
			return gitignore.NewMatcher(nil), nil
		}
		return nil, fmt.Errorf("failed to determine domain: %w", err)
	}
	var domain []string
	switch {
	case fi.IsDir():
		domain = strings.Split(inPath, string(filepath.Separator))
	default:
		domain = strings.Split(filepath.Dir(inPath), string(filepath.Separator))
	}

	// Default patterns to ignore the .git directory.
	ps := []gitignore.Pattern{
		gitignore.ParsePattern(".git", domain),
	}

	// Parse additional user-provided rules.
	scanner := bufio.NewScanner(strings.NewReader(rules))
	for scanner.Scan() {
		s := scanner.Text()
		if !strings.HasPrefix(s, "#") && len(strings.TrimSpace(s)) > 0 {
			ps = append(ps, gitignore.ParsePattern(s, domain))
		}
	}

	return gitignore.NewMatcher(ps), nil
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
