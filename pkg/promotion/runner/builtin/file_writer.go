package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindFileWrite = "file-write"

const defaultFileWritePermissions os.FileMode = 0o600

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
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{Err: fmt.Errorf("path %q must be relative", cfg.Path)}
	}
	cleanPath := filepath.Clean(cfg.Path)
	if cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{
				Err: fmt.Errorf("path %q attempts to traverse outside the working directory", cfg.Path),
			}
	}

	if cleanPath == ".git" || strings.HasPrefix(cleanPath, ".git"+string(filepath.Separator)) {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{
				Err: fmt.Errorf("writing to the .git directory is forbidden: %q", cfg.Path),
			}
	}

	permissions, err := parseFileWritePermissions(cfg.Permissions)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{Err: err}
	}

	absPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{
				Err: fmt.Errorf("could not secure join path %q: %w", cfg.Path, err),
			}
	}

	relPath, err := filepath.Rel(stepCtx.WorkDir, absPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{
				Err: fmt.Errorf("could not get relative path for %q: %w", absPath, err),
			}
	}
	if relPath != cleanPath {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{
				Err: fmt.Errorf("path %q resolves to a different path %q (symlinks are forbidden)", cfg.Path, relPath),
			}
	}

	if _, err = os.Stat(absPath); err == nil && !cfg.Overwrite {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{Err: fmt.Errorf(
				"file %q already exists; set overwrite to true to replace it", cfg.Path,
			)}
	}

	if err = os.MkdirAll(filepath.Dir(absPath), 0o700); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error creating parent directories for %q: %w", cfg.Path, err)
	}

	if err = os.WriteFile(absPath, []byte(cfg.Contents), permissions); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error writing file %q: %w", cfg.Path, err)
	}

	if err = os.Chmod(absPath, permissions); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error setting permissions on file %q: %w", cfg.Path, err)
	}

	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
}

func parseFileWritePermissions(permissions string) (os.FileMode, error) {
	if permissions == "" {
		return defaultFileWritePermissions, nil
	}

	parsed, err := strconv.ParseUint(permissions, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid permissions %q: must be an octal file mode", permissions)
	}

	mode := os.FileMode(parsed)
	// Only ordinary permission bits are allowed through. os.FileMode.Perm()
	// strips away special bits such as setuid, setgid, and sticky; if stripping
	// would change the requested mode, reject it instead of applying a surprising
	// permission set.
	if mode != mode.Perm() {
		return 0, fmt.Errorf("permissions %q must not include special mode bits", permissions)
	}
	// Do not allow file-write to create executable files. Promotion steps can
	// still write scripts as text, but another explicit step should be required
	// before they become executable.
	if mode&0o111 != 0 {
		return 0, fmt.Errorf("permissions %q must not include executable bits", permissions)
	}
	// World-writable files are unnecessarily broad for generated promotion
	// artifacts, so keep the writable surface limited to owner and group.
	if mode&0o002 != 0 {
		return 0, fmt.Errorf("permissions %q must not be world-writable", permissions)
	}

	return mode, nil
}
