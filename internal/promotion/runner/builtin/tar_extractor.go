package builtin

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
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
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// Size limits to prevent decompression bombs
const (
	// MaxDecompressedTarSize is the maximum size of all files extracted from a
	// tar archive.
	MaxDecompressedTarSize int64 = 100 * 1024 * 1024
	// MaxDecompressedFileSize is the maximum size of a single file extracted
	// from a tar archive.
	MaxDecompressedFileSize int64 = 50 * 1024 * 1024

	// gzipID1 is the first byte of the gzip magic number
	gzipID1 = 0x1F
	// gzipID2 is the second byte of the gzip magic number
	gzipID2 = 0x8B

	// defaultDirPermissions is the default permissions for directories created
	// by the tar extractor.
	defaultDirPermissions = 0o750
)

// tarExtractor is an implementation of the promotion.StepRunner interface that
// extracts a tar file to a specified directory.
type tarExtractor struct {
	schemaLoader gojsonschema.JSONLoader
}

// newTarExtractor returns an implementation of the promotion.StepRunner
// interface that extracts a tar file.
func newTarExtractor() promotion.StepRunner {
	r := &tarExtractor{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (t *tarExtractor) Name() string {
	return "untar"
}

// Run implements the promotion.StepRunner interface.
func (t *tarExtractor) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	// Validate the configuration against the JSON Schema.
	if err := validate(t.schemaLoader, gojsonschema.NewGoLoader(stepCtx.Config), t.Name()); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	// Convert the configuration into a typed object.
	cfg, err := promotion.ConfigToStruct[builtin.UntarConfig](stepCtx.Config)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("could not convert config into %s config: %w", t.Name(), err)
	}

	return t.run(ctx, stepCtx, cfg)
}

func (t *tarExtractor) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.UntarConfig,
) (promotion.StepResult, error) {
	logger := logging.LoggerFromContext(ctx)

	// Secure join the paths to prevent path traversal attacks
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

	// Create a temporary directory to atomically extract the tar file
	tempDir, err := os.MkdirTemp(stepCtx.WorkDir, "." + t.Name() + "-*")
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to create temporary directory for extraction: %w", err)
	}

	// Ensure the temporary directory is cleaned up after extraction
	defer func() {
		if err = os.RemoveAll(tempDir); err != nil {
			logger.Error(err, "failed to remove temporary directory after extraction")
		}
	}()

	// Extract the tar file to the temporary directory
	result, err := t.extractToDir(ctx, cfg, inPath, tempDir)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to extract tar file %q: %w", inPath, err)
	}

	// Move the extracted files from the temporary directory to the final output
	// path atomically
	if err = t.simpleAtomicMove(tempDir, outPath); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	// Return the result of the extraction
	return result, nil
}

func (t *tarExtractor) extractToDir(
	ctx context.Context,
	cfg builtin.UntarConfig,
	inPath, outPath string,
) (promotion.StepResult, error) {
	// Load the ignore rules.
	matcher, err := t.loadIgnoreRules(outPath, cfg.Ignore)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to load ignore rules: %w", err)
	}

	// Open the tar file
	file, err := os.Open(inPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to open tar file %q: %w", cfg.InPath, err)
	}
	defer file.Close()

	logger := logging.LoggerFromContext(ctx)
	var tarReader *tar.Reader

	// Read the first few bytes to check magic numbers
	header := make([]byte, 2)
	if _, err = file.Read(header); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to read file header: %w", err)
	}

	// Reset to the beginning of the file
	if _, err = file.Seek(0, 0); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to seek in tar file: %w", err)
	}

	// Check for gzip magic numbers
	if header[0] == gzipID1 && header[1] == gzipID2 {
		// File is gzipped
		gzr, err := gzip.NewReader(file)
		if err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzr.Close()
		tarReader = tar.NewReader(gzr)
		logger.Trace("treating file as gzipped tar based on magic numbers")
	} else {
		// File is not gzipped
		tarReader = tar.NewReader(file)
		logger.Trace("treating file as regular tar")
	}

	// Extract the tar file
	stripComponents := int64(0)
	if cfg.StripComponents != nil {
		stripComponents = *cfg.StripComponents
		if stripComponents > 0 {
			logger.Trace("stripping path components", "count", stripComponents)
		}
	}

	// Track the total size extracted
	var totalExtractedSize int64

	// Track directories created to avoid duplicates
	madeDir := make(map[string]bool)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("error reading tar: %w", err)
		}

		// Skip if this is not a regular file, symlink, or directory
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir && header.Typeflag != tar.TypeSymlink {
			logger.Trace("skipping non-regular file", "path", header.Name, "type", header.Typeflag)
			continue
		}

		// Handle stripping components if specified
		targetName := header.Name
		if stripComponents > 0 {
			// Get parts of the path
			parts := strings.Split(header.Name, "/")
			// Only strip if we have enough components
			if len(parts) > int(stripComponents) {
				targetName = strings.Join(parts[stripComponents:], "/")
			} else {
				// Skip this file if we don't have enough components
				logger.Trace("skipping file with insufficient path components", "path", header.Name)
				continue
			}
		}

		// Skip any empty targetName which can happen if we're processing a directory entry
		if targetName == "" || targetName == "/" {
			continue
		}

		// Validate the target path to prevent directory traversal
		if !t.validRelPath(targetName) {
			logger.Trace("skipping file with unsafe path", "path", targetName)
			continue
		}

		// Simple check for exact filename match for ignore patterns
		if cfg.Ignore != "" && filepath.Base(targetName) == strings.TrimSpace(cfg.Ignore) {
			logger.Trace("ignoring exact match path", "path", targetName)
			continue
		}

		// Check more complex patterns using gitignore matcher
		isDir := header.Typeflag == tar.TypeDir
		pathParts := strings.Split(targetName, "/")
		if matcher.Match(pathParts, isDir) {
			logger.Trace("ignoring path based on pattern", "path", targetName)
			continue
		}

		// Create the destination directory for files and links
		targetPath := filepath.Join(outPath, targetName)

		// Double-check the target path is within the output directory
		relPath, err := filepath.Rel(outPath, targetPath)
		if err != nil || strings.HasPrefix(relPath, "..") || strings.HasPrefix(relPath, "/") {
			logger.Trace("skipping path escaping target directory", "path", targetName)
			continue
		}

		destDir := filepath.Dir(targetPath)
		if err = os.MkdirAll(destDir, defaultDirPermissions); err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("failed to create directory %s: %w", destDir, err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err = os.MkdirAll(targetPath, defaultDirPermissions); err != nil {
				return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
					fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
			madeDir[targetPath] = true

		case tar.TypeSymlink:
			// Validate the symlink target to prevent symlink attacks
			if !t.validRelPath(header.Linkname) {
				logger.Trace("skipping symlink with unsafe target", "path", targetName, "target", header.Linkname)
				continue
			}
			// Create symlink
			if err := os.Symlink(header.Linkname, targetPath); err != nil && !os.IsExist(err) {
				logger.Error(err, "failed to create symlink", "path", targetPath, "target", header.Linkname)
			}
		case tar.TypeReg:
			// Check if single file exceeds max size limit
			logger.Trace("checking file size", "path", targetName, "size", header.Size)
			if header.Size > MaxDecompressedFileSize {
				logger.Trace("aborting extraction due to exceeding file size limit",
					"path", targetName,
					"size", header.Size,
					"limit", MaxDecompressedFileSize)
				return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
					fmt.Errorf("extraction aborted: file %s exceeds size limit of %d bytes", targetName, MaxDecompressedFileSize)
			}

			// Check if total extracted size would exceed limit
			if totalExtractedSize+header.Size > MaxDecompressedTarSize {
				logger.Trace("aborting extraction due to exceeding total size limit",
					"totalSize", totalExtractedSize,
					"fileSize", header.Size,
					"limit", MaxDecompressedTarSize)
				return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
					fmt.Errorf("extraction aborted: total size would exceed limit of %d bytes", MaxDecompressedTarSize)
			}

			dir := filepath.Dir(targetPath)
			// Create the directory if it doesn't exist
			if !madeDir[dir] {
				if err = os.MkdirAll(dir, defaultDirPermissions); err != nil {
					return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
						fmt.Errorf("failed to create directory %s: %w", dir, err)
				}
				madeDir[dir] = true
			}

			// Create a file
			safeMode := fs.FileMode(header.Mode) & 0o777
			outFile, err := os.OpenFile(targetPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, safeMode)
			if err != nil {
				return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
					fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			// Limit copying to the declared size
			written, err := io.CopyN(outFile, tarReader, header.Size)
			outFile.Close()
			if err != nil && err != io.EOF {
				return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
					fmt.Errorf("failed to write to file %s: %w", targetPath, err)
			}

			// Update total size counter
			totalExtractedSize += written

			// Change mod time
			if !header.ModTime.IsZero() {
				if err := os.Chtimes(targetPath, header.ModTime, header.ModTime); err != nil {
					return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
						fmt.Errorf("failed to change mod time for %s: %w", targetPath, err)
				}
			}
		}
	}

	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
}

// validRelPath checks if the path is safe to extract (no path traversal, etc.)
func (t *tarExtractor) validRelPath(p string) bool {
	if p == "" ||
		strings.Contains(p, `\`) ||
		strings.HasPrefix(p, "/") ||
		strings.Contains(p, "../") ||
		strings.HasPrefix(p, "./") {
		return false
	}
	return true
}

// loadIgnoreRules loads the ignore rules from the given string. The rules are
// separated by newlines, and comments are allowed with the '#' character.
// It returns a gitignore.Matcher that can be used to match paths against the
// rules.
func (t *tarExtractor) loadIgnoreRules(outPath, rules string) (gitignore.Matcher, error) {
	// Determine the domain for the ignore rules
	domain := strings.Split(outPath, string(filepath.Separator))

	// Create patterns from the provided rules
	var ps []gitignore.Pattern
	if rules != "" {
		scanner := bufio.NewScanner(strings.NewReader(rules))
		for scanner.Scan() {
			s := scanner.Text()
			if !strings.HasPrefix(s, "#") && len(strings.TrimSpace(s)) > 0 {
				ps = append(ps, gitignore.ParsePattern(s, domain))
			}
		}
	}

	// If no patterns were provided, add a default pattern to ignore any .git/
	// directory
	if len(ps) == 0 {
		ps = append(ps, gitignore.ParsePattern(".git", domain))
	}

	return gitignore.NewMatcher(ps), nil
}

func (t *tarExtractor) simpleAtomicMove(src, dst string) error {
	if err := os.Rename(src, dst); err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("failed to move %s to %s: %w", src, dst, err)
		}

		// If the destination already exists, remove it and try again
		if err := os.RemoveAll(dst); err != nil {
			return fmt.Errorf("failed to remove existing destination %s: %w", dst, err)
		}

		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("failed to move %s to %s after removing existing: %w", src, dst, err)
		}
	}
	return nil
}
