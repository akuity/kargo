package builtin

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	intfs "github.com/akuity/kargo/pkg/io/fs"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const (
	stepKindTar = "tar"

	// Size limits to prevent creation of massive tar archives.

	// maxUncompressedTarSize is the maximum size of all files to be archived into a tar archive.
	maxUncompressedTarSize int64 = 100 * 1024 * 1024
	// maxUncompressedFileSize is the maximum size of a single file to be archived into a tar archive.
	maxUncompressedFileSize int64 = 50 * 1024 * 1024
)

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindTar,
			Value: newTarCreator,
		},
	)
}

// tarCreator is an implementation of the promotion.StepRunner interface that
// archives a file or directory into a tar archive.
type tarCreator struct {
	schemaLoader gojsonschema.JSONLoader
}

// newTarCreator returns an implementation of the promotion.StepRunner
// interface that archives a file or directory into a tar archive.
func newTarCreator(promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &tarCreator{schemaLoader: getConfigSchemaLoader(stepKindTar)}
}

// Run implements the promotion.StepRunner interface.
func (t *tarCreator) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := t.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return t.run(ctx, stepCtx, cfg)
}

// convert validates the configuration against a JSON schema and converts it
// into a builtin.TarConfig struct.
func (t *tarCreator) convert(cfg promotion.Config) (builtin.TarConfig, error) {
	return validateAndConvert[builtin.TarConfig](t.schemaLoader, cfg, stepKindTar)
}

func (t *tarCreator) run(
	_ context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.TarConfig,
) (promotion.StepResult, error) {
	absInPath, err := t.prepareInputPath(stepCtx.WorkDir, cfg.InPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}
	absOutPath, err := t.prepareOutputPath(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	result, err := t.createTarball(absInPath, absOutPath, cfg.Gzip, cfg.Ignore)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to create the tar archive %q: %w", cfg.OutPath, err)
	}

	return result, nil
}

// prepareInputPath validates the input path.
func (t *tarCreator) prepareInputPath(
	workDir, inPath string,
) (string, error) {
	absInPath, err := securejoin.SecureJoin(workDir, inPath)
	if err != nil {
		return "", fmt.Errorf("failed to secure join input path %q: %w", inPath, err)
	}

	_, err = os.Stat(absInPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("input path %q does not exist: %w", inPath, err)
		}
		return "", fmt.Errorf("failed to stat input path %q: %w", inPath, err)
	}

	return absInPath, nil
}

// prepareOutputPath validates and prepares the output path.
func (t *tarCreator) prepareOutputPath(
	workDir, outPath string,
) (string, error) {
	absOutPath, err := securejoin.SecureJoin(workDir, outPath)
	if err != nil {
		return "", fmt.Errorf("failed to secure join output path %q: %w", outPath, err)
	}

	absOutPathDir := filepath.Dir(absOutPath)
	err = os.MkdirAll(absOutPathDir, 0o750)
	if err != nil {
		return "", fmt.Errorf("failed to create output directory for %q: %w", outPath, err)
	}

	return absOutPath, nil
}

// createTempFile creates a temporary file.
func (t *tarCreator) createTempFile(
	absOutPath string,
) (*os.File, string, error) {
	destDir := filepath.Dir(absOutPath)
	baseFile := filepath.Base(absOutPath)

	tempFile, err := os.CreateTemp(destDir, baseFile+".tmp")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temporary file in %q: %w", destDir, err)
	}

	return tempFile, tempFile.Name(), nil
}

// createTarball walks the input path and writes its contents to a tar archive.
func (t *tarCreator) createTarball(
	absInPath, absOutPath string,
	shouldGzip bool,
	ignore string,
) (promotion.StepResult, error) {
	matcher := t.buildIgnoreMatcher(ignore)

	tempFile, tempFilePath, err := t.createTempFile(absOutPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFilePath)
	}()

	var (
		tw        *tar.Writer
		gw        *gzip.Writer
		totalSize int64
	)

	if shouldGzip {
		gw = gzip.NewWriter(tempFile)
		tw = tar.NewWriter(gw)
	} else {
		tw = tar.NewWriter(tempFile)
	}

	err = filepath.WalkDir(absInPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("failed to get file info for %q: %w", path, err)
		}

		// Tar header's relative path.
		var relPath string
		if path == absInPath {
			if info.IsDir() {
				// Skip root dir to avoid a "." entry.
				return nil
			}
			relPath = filepath.Base(path)
		} else {
			var relErr error
			relPath, relErr = filepath.Rel(absInPath, path)
			if relErr != nil {
				return fmt.Errorf("failed to calculate relative path for %q: %w", path, relErr)
			}
		}

		if matcher.Match(strings.Split(relPath, "/"), info.IsDir()) {
			if info.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if info.Mode().IsRegular() {
			if info.Size() > maxUncompressedFileSize {
				return fmt.Errorf("file %q size (%d bytes) exceeds the maximum allowed single file size (%d bytes)",
					path, info.Size(), maxUncompressedFileSize)
			}

			totalSize += info.Size()
			if totalSize > maxUncompressedTarSize {
				return fmt.Errorf("total tar archive source size exceeds the maximum allowed size (%d bytes)",
					maxUncompressedTarSize)
			}
		}

		relPath = filepath.ToSlash(relPath)

		// Read its target if it's a symlink.
		var linkTarget string
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err = os.Readlink(path)
			if err != nil {
				return fmt.Errorf("failed to read symlink target for %q: %w", path, err)
			}
			linkTarget = filepath.ToSlash(linkTarget)
		}

		header, err := tar.FileInfoHeader(info, linkTarget)
		if err != nil {
			return fmt.Errorf("failed to create tar header for %q: %w", path, err)
		}
		// Directories need a trailing slash.
		if info.IsDir() && !strings.HasSuffix(relPath, "/") {
			relPath += "/"
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header for %q: %w", path, err)
		}

		if info.Mode().IsRegular() {
			srcFile, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open source file %q: %w", path, err)
			}

			_, copyErr := io.Copy(tw, srcFile)
			_ = srcFile.Close()

			if copyErr != nil {
				return fmt.Errorf("failed to copy file contents for %q: %w", path, copyErr)
			}
		}

		return nil
	})

	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed during directory walk: %w", err)
	}

	if err = tw.Close(); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to close tar writer: %w", err)
	}
	if shouldGzip {
		if err = gw.Close(); err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("failed to close gzip writer: %w", err)
		}
	}

	if err = tempFile.Sync(); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to sync tar archive to disk: %w", err)
	}
	if err = tempFile.Close(); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to close temporary file: %w", err)
	}

	if err = intfs.SimpleAtomicMove(tempFilePath, absOutPath); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to move temporary file to output path: %w", err)
	}

	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
}

// buildIgnoreMatcher parses a multiline string of gitignore-style patterns
// into a gitignore.Matcher.
func (t *tarCreator) buildIgnoreMatcher(ignore string) gitignore.Matcher {
	ignore = strings.TrimSpace(ignore)
	if ignore == "" {
		return gitignore.NewMatcher([]gitignore.Pattern{
			gitignore.ParsePattern(".git", nil),
		})
	}

	var patterns []gitignore.Pattern

	for ignore != "" {
		var line string
		line, ignore, _ = strings.Cut(ignore, "\n")

		line = strings.TrimSpace(line)

		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, gitignore.ParsePattern(line, nil))
		}
	}

	return gitignore.NewMatcher(patterns)
}
