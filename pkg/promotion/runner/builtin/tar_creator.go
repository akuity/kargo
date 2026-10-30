package builtin

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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
	inPath, err := t.prepareInputPath(stepCtx.Root, cfg.InPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}
	outPath, err := t.prepareOutputPath(stepCtx.Root, cfg.OutPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	var mtime *time.Time
	if cfg.Mtime != "" {
		parsed, parseErr := time.Parse(time.RFC3339, cfg.Mtime)
		if parseErr != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("invalid mtime %q: %w", cfg.Mtime, parseErr)
		}
		mtime = &parsed
	}

	result, err := t.createTarball(stepCtx.Root, inPath, outPath, cfg.Gzip, cfg.Ignore, mtime)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to create the tar archive %q: %w", cfg.OutPath, err)
	}

	return result, nil
}

// prepareInputPath validates the input path and returns it relative to root.
func (t *tarCreator) prepareInputPath(root *os.Root, inPath string) (string, error) {
	relInPath, err := promotion.WorkspacePath(inPath)
	if err != nil {
		return "", err
	}

	if _, err = root.Stat(relInPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("input path %q does not exist: %w", inPath, err)
		}
		return "", fmt.Errorf("failed to stat input path %q: %w", inPath, err)
	}

	return relInPath, nil
}

// prepareOutputPath validates and prepares the output path and returns it
// relative to root.
func (t *tarCreator) prepareOutputPath(root *os.Root, outPath string) (string, error) {
	relOutPath, err := promotion.WorkspacePath(outPath)
	if err != nil {
		return "", err
	}

	if err = root.MkdirAll(filepath.Dir(relOutPath), 0o750); err != nil {
		return "", fmt.Errorf("failed to create output directory for %q: %w", outPath, err)
	}

	return relOutPath, nil
}

// createTarball walks the input path, which is relative to root, and writes its
// contents to a tar archive at the output path, which is also relative to root.
func (t *tarCreator) createTarball(
	root *os.Root,
	inPath, outPath string,
	shouldGzip bool,
	ignore string,
	mtime *time.Time,
) (promotion.StepResult, error) {
	matcher := t.buildIgnoreMatcher(ignore)

	tempFile, tempFilePath, err := createTempFile(root, filepath.Dir(outPath), filepath.Base(outPath)+".tmp")
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to create temporary file in %q: %w", filepath.Dir(outPath), err)
	}

	defer func() {
		_ = tempFile.Close()
		_ = root.Remove(tempFilePath)
	}()

	var (
		tw        *tar.Writer
		gw        *gzip.Writer
		totalSize int64
	)

	// digest hashes the archive as written; diffID hashes the uncompressed tar
	// stream.
	digest := sha256.New()
	diffID := digest
	out := io.MultiWriter(tempFile, digest)
	if shouldGzip {
		diffID = sha256.New()
		gw = gzip.NewWriter(out)
		tw = tar.NewWriter(io.MultiWriter(gw, diffID))
	} else {
		tw = tar.NewWriter(out)
	}

	err = fs.WalkDir(root.FS(), inPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("failed to get file info for %q: %w", path, err)
		}

		// Tar header's relative path.
		var relPath string
		if path == inPath {
			if info.IsDir() {
				// Skip root dir to avoid a "." entry.
				return nil
			}
			relPath = filepath.Base(path)
		} else {
			var relErr error
			relPath, relErr = filepath.Rel(inPath, path)
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

		// The tar header name uses forward slashes.
		relPath = filepath.ToSlash(relPath)

		// Read its target if it's a symlink.
		var linkTarget string
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err = root.Readlink(path)
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
		// Who owned the files on the controller is meaningless to anyone
		// extracting the archive, so it is never recorded.
		header.Uid, header.Gid = 0, 0
		header.Uname, header.Gname = "", ""
		if mtime != nil {
			header.ModTime = *mtime
			header.AccessTime, header.ChangeTime = time.Time{}, time.Time{}
		}

		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header for %q: %w", path, err)
		}

		if info.Mode().IsRegular() {
			srcFile, err := root.Open(path)
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

	if err = replace(root, tempFilePath, outPath); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to move temporary file to output path: %w", err)
	}

	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: map[string]any{
			"digest": v1.Hash{Algorithm: "sha256", Hex: hex.EncodeToString(digest.Sum(nil))}.String(),
			"diffID": v1.Hash{Algorithm: "sha256", Hex: hex.EncodeToString(diffID.Sum(nil))}.String(),
		},
	}, nil
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
