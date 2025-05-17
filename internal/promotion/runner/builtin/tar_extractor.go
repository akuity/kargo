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
	// 100 MB maximum total size for all extracted files
	MaxDecompressedTarSize int64 = 100 * 1024 * 1024
	// 50 MB maximum size for a single file
	MaxDecompressedFileSize int64 = 50 * 1024 * 1024

	// gzipID1 is the first byte of the gzip magic number
	gzipID1 = 0x1F
	// gzipID2 is the second byte of the gzip magic number
	gzipID2 = 0x8B

	// defaultPermissions is the default permissions for created directories and files
	defaultPermissions = 0750
)

// tarExtractor is an implementation of the promotion.StepRunner interface that
// extracts a tar file to a specified directory.
type tarExtractor struct {
	schemaLoader gojsonschema.JSONLoader
}

// newTarExtractor returns an implementation of the promotion.StepRunner interface
// that extracts a tar file.
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

	// Ensure the output directory exists
	if err := os.MkdirAll(outPath, defaultPermissions); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to create output directory %q: %w", cfg.OutPath, err)
	}

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
	_, err = file.Read(header)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to read file header: %w", err)
	}

	// Reset to beginning of file
	_, err = file.Seek(0, 0)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to seek in tar file: %w", err)
	}

	// Check for gzip magic numbers (0x1F 0x8B)
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

	// Track total size extracted
	var totalExtractedSize int64 = 0

	// Track directories created to avoid duplicates
	madeDir := map[string]bool{}

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
			logger.Debug("skipping non-regular file", "path", header.Name, "type", header.Typeflag)
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
				logger.Debug("skipping file with insufficient path components", "path", header.Name)
				continue
			}
		}

		// Skip any empty targetName which can happen if we're processing a directory entry
		if targetName == "" || targetName == "/" {
			continue
		}

		// Validate the target path to prevent directory traversal
		if !t.validRelPath(targetName) {
			logger.Debug("skipping file with unsafe path", "path", targetName)
			continue
		}

		// Simple check for exact filename match for ignore patterns
		if cfg.Ignore != "" && filepath.Base(targetName) == strings.TrimSpace(cfg.Ignore) {
			logger.Debug("ignoring exact match path", "path", targetName)
			continue
		}

		// Check more complex patterns using gitignore matcher
		isDir := header.Typeflag == tar.TypeDir
		pathParts := strings.Split(targetName, "/")
		if matcher.Match(pathParts, isDir) {
			logger.Debug("ignoring path based on pattern", "path", targetName)
			continue
		}

		// Create destination directory for files and links
		targetPath := filepath.Join(outPath, targetName)

		// Double check the target path is within the output directory
		relPath, err := filepath.Rel(outPath, targetPath)
		if err != nil || strings.HasPrefix(relPath, "..") || strings.HasPrefix(relPath, "/") {
			logger.Debug("skipping path escaping target directory", "path", targetName)
			continue
		}

		destDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(destDir, defaultPermissions); err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("failed to create directory %s: %w", destDir, err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(targetPath, defaultPermissions); err != nil {
				return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
					fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
			madeDir[targetPath] = true

		case tar.TypeSymlink:
			// Validate the symlink target to prevent symlink attacks
			if !t.validRelPath(header.Linkname) {
				logger.Debug("skipping symlink with unsafe target", "path", targetName, "target", header.Linkname)
				continue
			}
			// Create symlink
			if err := os.Symlink(header.Linkname, targetPath); err != nil && !os.IsExist(err) {
				logger.Debug("failed to create symlink", "path", targetPath, "target", header.Linkname, "error", err)
			}

		case tar.TypeReg:
			// Check if single file exceeds max size limit
			logger.Trace("checking file size", "path", targetName, "size", header.Size)
			if header.Size > MaxDecompressedFileSize {
				logger.Debug("aborting extraction due to exceeding file size limit",
					"path", targetName,
					"size", header.Size,
					"limit", MaxDecompressedFileSize)
				return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
					fmt.Errorf("extraction aborted: file %s exceeds size limit of %d bytes", targetName, MaxDecompressedFileSize)
			}

			// Check if total extracted size would exceed limit
			if totalExtractedSize+header.Size > MaxDecompressedTarSize {
				logger.Debug("aborting extraction due to exceeding total size limit",
					"total_size", totalExtractedSize,
					"file_size", header.Size,
					"limit", MaxDecompressedTarSize)
				return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
					fmt.Errorf("extraction aborted: total size would exceed limit of %d bytes", MaxDecompressedTarSize)
			}

			dir := filepath.Dir(targetPath)
			// Create the directory if it doesn't exist
			if !madeDir[dir] {
				if err := os.MkdirAll(dir, defaultPermissions); err != nil {
					return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
						fmt.Errorf("failed to create directory %s: %w", dir, err)
				}
				madeDir[dir] = true
			}

			// Create file
			outFile, err := os.OpenFile(targetPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fs.FileMode(header.Mode))
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

	// If no patterns were provided, add a default pattern to ignore .git directory
	if len(ps) == 0 {
		ps = append(ps, gitignore.ParsePattern(".git", domain))
	}

	return gitignore.NewMatcher(ps), nil
}
