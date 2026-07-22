package builtin

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_tarConfig_convert(t *testing.T) {
	t.Parallel()
	tests := []validationTestCase{
		{
			name:   "inPath not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): inPath is required",
			},
		},
		{
			name: "inPath is empty string",
			config: promotion.Config{
				"inPath": "",
			},
			expectedProblems: []string{
				"inPath: String length must be greater than or equal to 1",
			},
		},
		{
			name: "outPath not specified",
			config: promotion.Config{
				"inPath": "source_dir/",
			},
			expectedProblems: []string{
				"(root): outPath is required",
			},
		},
		{
			name: "outPath is empty string",
			config: promotion.Config{
				"inPath":  "source_dir/",
				"outPath": "",
			},
			expectedProblems: []string{
				"outPath: String length must be greater than or equal to 1",
			},
		},
		{
			name: "ignore is empty string",
			config: promotion.Config{
				"inPath":  "source_dir/",
				"outPath": "archive.tar.gz",
				"ignore":  "",
			},
			expectedProblems: []string{
				"ignore: String length must be greater than or equal to 1",
			},
		},
		{
			name: "valid minimal config",
			config: promotion.Config{
				"inPath":  "source_dir/",
				"outPath": "archive.tar.gz",
			},
		},
		{
			name: "valid config with gzip set to false",
			config: promotion.Config{
				"inPath":  "source_dir/",
				"outPath": "archive.tar",
				"gzip":    false,
			},
		},
		{
			name: "valid config with ignore patterns",
			config: promotion.Config{
				"inPath":  "source_dir/",
				"outPath": "archive.tar.gz",
				"ignore":  "*.log\n*.tmp\n__pycache__/",
			},
		},
	}

	r := newTarCreator(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*tarCreator)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_tarCreator_run(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		setupFiles func(*testing.T) string
		cfg        builtin.TarConfig
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "succeeds with basic tar creation and atomic behavior",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				// Create input directory and file.
				inDir := filepath.Join(tmpDir, "source")
				require.NoError(t, os.Mkdir(inDir, 0o750))
				require.NoError(t, os.WriteFile(filepath.Join(inDir, "file.txt"), []byte("test content"), 0o600))

				return tmpDir
			},
			cfg: builtin.TarConfig{
				InPath:  "source",
				OutPath: "archive.tar",
				Gzip:    false,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				outPath := filepath.Join(workDir, "archive.tar")
				assert.FileExists(t, outPath)

				verifyNoTempFiles(t, workDir)
			},
		},
		{
			name: "creates parent directories for nested output path",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				inDir := filepath.Join(tmpDir, "source")
				require.NoError(t, os.Mkdir(inDir, 0o750))
				require.NoError(t, os.WriteFile(filepath.Join(inDir, "file.txt"), []byte("test content"), 0o600))

				return tmpDir
			},
			cfg: builtin.TarConfig{
				InPath:  "source",
				OutPath: filepath.Join("nested", "archive.tar"),
				Gzip:    false,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				outPath := filepath.Join(workDir, "nested", "archive.tar")
				assert.FileExists(t, outPath)
				verifyNoTempFiles(t, filepath.Join(workDir, "nested"))
			},
		},
		{
			name: "atomically replaces existing destination",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				inDir := filepath.Join(tmpDir, "source")
				require.NoError(t, os.Mkdir(inDir, 0o750))
				require.NoError(t, os.WriteFile(filepath.Join(inDir, "new_file.txt"), []byte("new content"), 0o600))

				// Create an existing "old" archive.
				oldPath := filepath.Join(tmpDir, "archive.tar")
				require.NoError(t, os.WriteFile(oldPath, []byte("old archive content"), 0o600))

				return tmpDir
			},
			cfg: builtin.TarConfig{
				InPath:  "source",
				OutPath: "archive.tar",
				Gzip:    false,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				outPath := filepath.Join(workDir, "archive.tar")
				assert.FileExists(t, outPath)

				// The file should be a valid tar now, not the old fake content.
				f, err := os.Open(outPath)
				require.NoError(t, err)
				defer f.Close()

				tr := tar.NewReader(f)
				hdr, err := tr.Next()
				assert.NoError(t, err, "Should be a valid tar archive replacing the old file")
				assert.Equal(t, "new_file.txt", hdr.Name)
			},
		},
		{
			name: "fails with invalid input path",
			setupFiles: func(t *testing.T) string {
				return t.TempDir()
			},
			cfg: builtin.TarConfig{
				InPath:  "nonexistent_source",
				OutPath: "archive.tar",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
				assert.ErrorContains(t, err, "does not exist")

				outPath := filepath.Join(workDir, "archive.tar")
				_, statErr := os.Stat(outPath)
				assert.True(t, os.IsNotExist(statErr))
			},
		},
		{
			name: "fails to create output directory when blocked by a file",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				inDir := filepath.Join(tmpDir, "source")
				require.NoError(t, os.Mkdir(inDir, 0o750))

				require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "nested"), []byte("blocking file"), 0o600))

				return tmpDir
			},
			cfg: builtin.TarConfig{
				InPath:  "source",
				OutPath: filepath.Join("nested", "archive.tar"),
				Gzip:    false,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
				assert.ErrorContains(t, err, "failed to create output directory")
			},
		},
	}

	runner := &tarCreator{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			workDir := tt.setupFiles(t)
			result, err := runner.run(
				context.Background(),
				&promotion.StepContext{WorkDir: workDir},
				tt.cfg,
			)
			tt.assertions(t, workDir, result, err)
		})
	}
}

func Test_tarCreator_createTarball(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		setupFiles func(*testing.T) (string, string, string) // workDir, inPath, outPath
		gzip       bool
		ignore     string
		assertions func(*testing.T, string, promotion.StepResult, error) // outPath, result, err
	}{
		{
			name: "succeeds creating simple uncompressed tar file",
			setupFiles: func(t *testing.T) (string, string, string) {
				workDir := t.TempDir()

				inPath := filepath.Join(workDir, "source")
				require.NoError(t, os.Mkdir(inPath, 0o750))
				require.NoError(t, os.WriteFile(filepath.Join(inPath, "file1.txt"), []byte("test content"), 0o600))

				subDir := filepath.Join(inPath, "testdir")
				require.NoError(t, os.Mkdir(subDir, 0o750))
				require.NoError(t, os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("nested content"), 0o600))

				outPath := filepath.Join(workDir, "archive.tar")
				return workDir, inPath, outPath
			},
			gzip:   false,
			ignore: "",
			assertions: func(t *testing.T, outPath string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				expectedFiles := map[string]string{
					"file1.txt":         "test content",
					"testdir/file2.txt": "nested content",
					"testdir/":          "",
				}
				verifyTarContents(t, outPath, false, expectedFiles)
			},
		},
		{
			name: "succeeds creating gzipped tar file",
			setupFiles: func(t *testing.T) (string, string, string) {
				workDir := t.TempDir()

				inPath := filepath.Join(workDir, "source")
				require.NoError(t, os.Mkdir(inPath, 0o750))
				require.NoError(t, os.WriteFile(filepath.Join(inPath, "compressed.txt"), []byte("compressed content"), 0o600))

				outPath := filepath.Join(workDir, "archive.tar.gz")
				return workDir, inPath, outPath
			},
			gzip:   true,
			ignore: "",
			assertions: func(t *testing.T, outPath string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				expectedFiles := map[string]string{
					"compressed.txt": "compressed content",
				}
				verifyTarContents(t, outPath, true, expectedFiles)
			},
		},
		{
			name: "handles ignore rules",
			setupFiles: func(t *testing.T) (string, string, string) {
				workDir := t.TempDir()

				inPath := filepath.Join(workDir, "source")
				require.NoError(t, os.Mkdir(inPath, 0o750))
				require.NoError(t, os.WriteFile(filepath.Join(inPath, "include.txt"), []byte("include me"), 0o600))
				require.NoError(t, os.WriteFile(filepath.Join(inPath, "ignore.txt"), []byte("ignore me"), 0o600))

				outPath := filepath.Join(workDir, "archive.tar")
				return workDir, inPath, outPath
			},
			gzip:   false,
			ignore: "ignore.txt",
			assertions: func(t *testing.T, outPath string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				expectedFiles := map[string]string{
					"include.txt": "include me",
				}
				verifyTarContents(t, outPath, false, expectedFiles)
			},
		},
		{
			name: "handles symbolic links",
			setupFiles: func(t *testing.T) (string, string, string) {
				workDir := t.TempDir()

				inPath := filepath.Join(workDir, "source")
				require.NoError(t, os.Mkdir(inPath, 0o750))
				require.NoError(t, os.WriteFile(filepath.Join(inPath, "target.txt"), []byte("target content"), 0o600))
				require.NoError(t, os.Symlink("target.txt", filepath.Join(inPath, "link.txt")))

				outPath := filepath.Join(workDir, "archive.tar")
				return workDir, inPath, outPath
			},
			gzip:   false,
			ignore: "",
			assertions: func(t *testing.T, outPath string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				expectedFiles := map[string]string{
					"target.txt": "target content",
					"link.txt":   "target.txt", // Symlink target.
				}
				verifyTarContents(t, outPath, false, expectedFiles)
			},
		},
		{
			name: "fails with file larger than max size",
			setupFiles: func(t *testing.T) (string, string, string) {
				workDir := t.TempDir()

				inPath := filepath.Join(workDir, "source")
				require.NoError(t, os.Mkdir(inPath, 0o750))

				largeFilePath := filepath.Join(inPath, "huge.bin")
				f, err := os.Create(largeFilePath)
				require.NoError(t, err)
				require.NoError(t, f.Truncate(maxUncompressedFileSize+1))
				require.NoError(t, f.Close())

				outPath := filepath.Join(workDir, "archive.tar")
				return workDir, inPath, outPath
			},
			gzip:   false,
			ignore: "",
			assertions: func(t *testing.T, outPath string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
				assert.ErrorContains(t, err, "exceeds the maximum allowed single file size")

				// Check that no partial file exists.
				_, statErr := os.Stat(outPath)
				assert.True(t, os.IsNotExist(statErr))
			},
		},
		{
			name: "fails with total archive size larger than limit",
			setupFiles: func(t *testing.T) (string, string, string) {
				workDir := t.TempDir()

				inPath := filepath.Join(workDir, "source")
				require.NoError(t, os.Mkdir(inPath, 0o750))

				// Create 3 files that together exceed the 100MB limit.
				for i := range 3 {
					f, err := os.Create(filepath.Join(inPath, fmt.Sprintf("chunk%d.bin", i)))
					require.NoError(t, err)
					require.NoError(t, f.Truncate(40*1024*1024))
					require.NoError(t, f.Close())
				}

				outPath := filepath.Join(workDir, "archive.tar")
				return workDir, inPath, outPath
			},
			gzip:   false,
			ignore: "",
			assertions: func(t *testing.T, outPath string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
				assert.ErrorContains(t, err, "total tar archive source size exceeds the maximum allowed size")

				// Check that no partial file exists.
				_, statErr := os.Stat(outPath)
				assert.True(t, os.IsNotExist(statErr))
			},
		},
		{
			name: "succeeds archiving a single file directly",
			setupFiles: func(t *testing.T) (string, string, string) {
				workDir := t.TempDir()

				inPath := filepath.Join(workDir, "single_file.txt")
				require.NoError(t, os.WriteFile(inPath, []byte("single file content"), 0o600))

				outPath := filepath.Join(workDir, "archive.tar")
				return workDir, inPath, outPath
			},
			gzip:   false,
			ignore: "",
			assertions: func(t *testing.T, outPath string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				expectedFiles := map[string]string{
					"single_file.txt": "single file content",
				}
				verifyTarContents(t, outPath, false, expectedFiles)
			},
		},
		{
			name: "succeeds archiving an empty directory",
			setupFiles: func(t *testing.T) (string, string, string) {
				workDir := t.TempDir()

				inPath := filepath.Join(workDir, "empty_dir")
				require.NoError(t, os.Mkdir(inPath, 0o750))

				outPath := filepath.Join(workDir, "archive.tar")
				return workDir, inPath, outPath
			},
			gzip:   false,
			ignore: "",
			assertions: func(t *testing.T, outPath string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				// Tar should exist but have no file entries.
				expectedFiles := map[string]string{}
				verifyTarContents(t, outPath, false, expectedFiles)
			},
		},
		{
			name: "skips ignored directories",
			setupFiles: func(t *testing.T) (string, string, string) {
				workDir := t.TempDir()

				inPath := filepath.Join(workDir, "source")
				require.NoError(t, os.Mkdir(inPath, 0o750))

				require.NoError(t, os.WriteFile(filepath.Join(inPath, "valid.txt"), []byte("valid"), 0o600))

				ignoreDir := filepath.Join(inPath, "node_modules")
				require.NoError(t, os.Mkdir(ignoreDir, 0o750))
				require.NoError(t, os.WriteFile(filepath.Join(ignoreDir, "secret.txt"), []byte("secret"), 0o600))

				outPath := filepath.Join(workDir, "archive.tar")
				return workDir, inPath, outPath
			},
			gzip:   false,
			ignore: "node_modules/",
			assertions: func(t *testing.T, outPath string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				expectedFiles := map[string]string{
					"valid.txt": "valid",
				}
				verifyTarContents(t, outPath, false, expectedFiles)
			},
		},
		{
			name: "fails to create temp file",
			setupFiles: func(t *testing.T) (string, string, string) {
				workDir := t.TempDir()
				inPath := filepath.Join(workDir, "source")
				require.NoError(t, os.Mkdir(inPath, 0o750))

				outPath := filepath.Join(workDir, "non_existent_dir", "archive.tar")

				return workDir, inPath, outPath
			},
			gzip:   false,
			ignore: "",
			assertions: func(t *testing.T, outPath string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
				assert.ErrorContains(t, err, "failed to create temporary file")
			},
		},
	}

	runner := &tarCreator{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, inPath, outPath := tt.setupFiles(t)

			result, err := runner.createTarball(
				inPath,
				outPath,
				tt.gzip,
				tt.ignore,
			)

			tt.assertions(t, outPath, result, err)
		})
	}
}

func Test_tarCreator_prepareInputPath(t *testing.T) {
	t.Parallel()
	runner := &tarCreator{}
	workDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(workDir, "real.txt"), nil, 0o600))
	require.NoError(t, os.Mkdir(filepath.Join(workDir, "real_dir"), 0o750))

	tests := []struct {
		name          string
		inPath        string
		expectedError string
	}{
		{"valid file path", "real.txt", ""},
		{"valid dir path", "real_dir", ""},
		{"non-existent path", "fake.txt", "does not exist"},
		{"path traversal attempt", "../outside.txt", "does not exist"},
		{"fails to secure join", string([]byte{0}), "failed to secure join input path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			absPath, err := runner.prepareInputPath(workDir, tt.inPath)
			if tt.expectedError != "" {
				assert.ErrorContains(t, err, tt.expectedError)
				assert.Empty(t, absPath)
			} else {
				assert.NoError(t, err)
				assert.True(t, strings.HasSuffix(absPath, tt.inPath))
			}
		})
	}
}

func Test_tarCreator_prepareOutputPath(t *testing.T) {
	t.Parallel()
	runner := &tarCreator{}

	t.Run("creates parent directories correctly", func(t *testing.T) {
		t.Parallel()
		workDir := t.TempDir()
		nestedOutPath := "deeply/nested/archive.tar"

		absPath, err := runner.prepareOutputPath(workDir, nestedOutPath)
		assert.NoError(t, err)
		assert.True(t, strings.HasSuffix(absPath, nestedOutPath))

		// Check the directory exists.
		dirInfo, statErr := os.Stat(filepath.Dir(absPath))
		assert.NoError(t, statErr)
		assert.True(t, dirInfo.IsDir())
	})

	t.Run("fails to secure join", func(t *testing.T) {
		t.Parallel()
		workDir := t.TempDir()

		_, err := runner.prepareOutputPath(workDir, string([]byte{0}))
		assert.ErrorContains(t, err, "failed to secure join output path")
	})
}

func Test_tarCreator_createTempFile(t *testing.T) {
	t.Parallel()
	runner := &tarCreator{}

	t.Run("succeeds to create temp file", func(t *testing.T) {
		t.Parallel()
		workDir := t.TempDir()

		f, path, err := runner.createTempFile(filepath.Join(workDir, "out.tar"))
		assert.NoError(t, err)
		assert.NotNil(t, f)
		assert.NotEmpty(t, path)

		f.Close()
	})

	t.Run("fails to create temp file", func(t *testing.T) {
		t.Parallel()
		workDir := t.TempDir()

		f, path, err := runner.createTempFile(filepath.Join(workDir, "non_existent_dir", "out.tar"))
		assert.ErrorContains(t, err, "failed to create temporary file")
		assert.Nil(t, f)
		assert.Empty(t, path)
	})
}

func Test_tarCreator_buildIgnoreMatcher(t *testing.T) {
	t.Parallel()
	runner := &tarCreator{}

	tests := []struct {
		name     string
		ignore   string
		paths    []string
		isDir    []bool
		expected []bool // True means the path should be ignored.
	}{
		{
			name:     "empty string defaults to ignoring .git",
			ignore:   "",
			paths:    []string{".git", "file.txt", "src/.git"},
			isDir:    []bool{true, false, true},
			expected: []bool{true, false, true},
		},
		{
			name:     "spaces default to ignoring .git",
			ignore:   "   \n  ",
			paths:    []string{".git", "file.txt"},
			isDir:    []bool{true, false},
			expected: []bool{true, false},
		},
		{
			name: "custom patterns",
			ignore: `
# This is a comment
*.log
node_modules/
`,
			paths:    []string{"error.log", "node_modules", "src/main.go", ".git"},
			isDir:    []bool{false, true, false, true},
			expected: []bool{true, true, false, false}, // Custom overrides the default.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			matcher := runner.buildIgnoreMatcher(tt.ignore)

			for i, path := range tt.paths {
				pathParts := strings.Split(path, "/")
				result := matcher.Match(pathParts, tt.isDir[i])
				assert.Equal(t, tt.expected[i], result, "Path: %s", path)
			}
		})
	}
}

// Helpers

// verifyTarContents reads a tar file and checks if the contents match.
func verifyTarContents(t *testing.T, tarPath string, isGzip bool, expectedFiles map[string]string) {
	f, err := os.Open(tarPath)
	require.NoError(t, err)
	defer f.Close()

	var tr *tar.Reader
	if isGzip {
		gzr, err := gzip.NewReader(f)
		require.NoError(t, err)
		defer gzr.Close()
		tr = tar.NewReader(gzr)
	} else {
		tr = tar.NewReader(f)
	}

	foundFiles := make(map[string]bool)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		if expectedContent, ok := expectedFiles[header.Name]; ok {
			foundFiles[header.Name] = true
			switch header.Typeflag {
			case tar.TypeReg:
				b, err := io.ReadAll(tr)
				require.NoError(t, err)
				assert.Equal(t, expectedContent, string(b))
			case tar.TypeSymlink:
				assert.Equal(t, expectedContent, header.Linkname)
			case tar.TypeDir:
				assert.Empty(t, expectedContent)
			}
		} else {
			t.Errorf("unexpected entry in tar: %s", header.Name)
		}
	}

	for expected := range expectedFiles {
		assert.True(t, foundFiles[expected], "missing expected entry in tar: %s", expected)
	}
}

// verifyNoTempFiles ensures no temp files remain in the target directory.
func verifyNoTempFiles(t *testing.T, dir string) {
	entries, err := os.ReadDir(dir)
	assert.NoError(t, err)
	for _, entry := range entries {
		assert.False(t, !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tmp"),
			"No temporary files should remain: %s", entry.Name())
	}
}
