package builtin

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/otiai10/copy"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/io/fs"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindCopy = "copy"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindCopy,
			Value: newFileCopier,
		},
	)
}

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
func newFileCopier(promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &fileCopier{schemaLoader: getConfigSchemaLoader(stepKindCopy)}
}

// Run implements the promotion.StepRunner interface.
func (f *fileCopier) Run(
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

// convert validates fileCopier configuration against a JSON schema and
// converts it into a builtin.CopyConfig struct.
func (f *fileCopier) convert(cfg promotion.Config) (builtin.CopyConfig, error) {
	return validateAndConvert[builtin.CopyConfig](f.schemaLoader, cfg, stepKindCopy)
}

func (f *fileCopier) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.CopyConfig,
) (promotion.StepResult, error) {
	// Secure join the paths to prevent path traversal attacks.
	inPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.InPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("could not secure join inPath %q: %w", cfg.InPath, err)
	}
	outPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("could not secure join outPath %q: %w", cfg.OutPath, err)
	}

	// Load the ignore rules.
	matcher, err := f.loadIgnoreRules(inPath, cfg.Ignore)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
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
			return fs.SanitizePathError(err, stepCtx.WorkDir)
		},
	}
	if err = copy.Copy(inPath, outPath, opts); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to copy %q to %q: %w", cfg.InPath, cfg.OutPath, err)
	}
	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
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
