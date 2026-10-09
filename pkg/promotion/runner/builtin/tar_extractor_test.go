package builtin

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_untarConfig_convert(t *testing.T) {
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
				"inPath": "/path/to/archive.tar.gz",
			},
			expectedProblems: []string{
				"(root): outPath is required",
			},
		},
		{
			name: "outPath is empty string",
			config: promotion.Config{
				"inPath":  "/path/to/archive.tar.gz",
				"outPath": "",
			},
			expectedProblems: []string{
				"outPath: String length must be greater than or equal to 1",
			},
		},
		{
			name: "both required fields missing",
			config: promotion.Config{
				"stripComponents": 1,
			},
			expectedProblems: []string{
				"(root): inPath is required",
				"(root): outPath is required",
			},
		},
		{
			name: "stripComponents is negative",
			config: promotion.Config{
				"inPath":          "/path/to/archive.tar.gz",
				"outPath":         "/destination/path",
				"stripComponents": -1,
			},
			expectedProblems: []string{
				"stripComponents: Must be greater than or equal to 0",
			},
		},
		{
			name: "ignore is empty string",
			config: promotion.Config{
				"inPath":  "/path/to/archive.tar.gz",
				"outPath": "/destination/path",
				"ignore":  "",
			},
			expectedProblems: []string{
				"ignore: String length must be greater than or equal to 1",
			},
		},
		{
			name: "valid minimal config",
			config: promotion.Config{
				"inPath":  "/path/to/archive.tar.gz",
				"outPath": "/destination/path",
			},
		},
		{
			name: "valid config with stripComponents=0",
			config: promotion.Config{
				"inPath":          "/path/to/archive.tar.gz",
				"outPath":         "/destination/path",
				"stripComponents": 0,
			},
		},
		{
			name: "valid config with stripComponents=1",
			config: promotion.Config{
				"inPath":          "/path/to/archive.tar.gz",
				"outPath":         "/destination/path",
				"stripComponents": 1,
			},
		},
		{
			name: "valid config with stripComponents=3",
			config: promotion.Config{
				"inPath":          "/path/to/archive.tar.gz",
				"outPath":         "/destination/path",
				"stripComponents": 3,
			},
		},
		{
			name: "valid config with ignore patterns",
			config: promotion.Config{
				"inPath":  "/path/to/archive.tar.gz",
				"outPath": "/destination/path",
				"ignore":  "*.log\n*.tmp\n__pycache__/",
			},
		},
		{
			name: "valid config with stripComponents and ignore",
			config: promotion.Config{
				"inPath":          "/path/to/archive.tar.gz",
				"outPath":         "/destination/path",
				"stripComponents": 2,
				"ignore":          "node_modules/\n.git/\n*.log",
			},
		},
		{
			name: "valid kitchen sink",
			config: promotion.Config{
				"inPath":          "/path/to/source-code.tar.gz",
				"outPath":         "/workspace/extracted",
				"stripComponents": 1,
				"ignore": `# Ignore build artifacts
build/
dist/
target/

# Ignore logs
*.log
*.tmp

# Ignore version control
.git/
.svn/

# Ignore dependencies
node_modules/
vendor/`,
			},
		},
	}

	r := newTarExtractor(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*tarExtractor)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_tarExtractor_run(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles func(*testing.T) string
		cfg        builtin.UntarConfig
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "succeeds with basic extraction and atomic behavior",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				// Create a tar file
				tarPath := filepath.Join(tmpDir, "archive.tar")
				tarFile, err := os.Create(tarPath)
				require.NoError(t, err)
				defer tarFile.Close()

				tw := tar.NewWriter(tarFile)
				defer tw.Close()

				// Add a simple file
				data := []byte("test content")
				hdr := &tar.Header{
					Name: "file.txt",
					Mode: 0o600,
					Size: int64(len(data)),
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write(data)
				require.NoError(t, err)

				return tmpDir
			},
			cfg: builtin.UntarConfig{
				InPath:  "archive.tar",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				// Verify extraction worked
				extractDir := filepath.Join(workDir, "extracted")
				filePath := filepath.Join(extractDir, "file.txt")
				content, err := os.ReadFile(filePath)
				assert.NoError(t, err)
				assert.Equal(t, "test content", string(content))

				// Verify no temp directories remain
				entries, err := os.ReadDir(workDir)
				assert.NoError(t, err)
				for _, entry := range entries {
					assert.False(t, entry.IsDir() && entry.Name()[0] == '.' &&
						len(entry.Name()) > 6 && entry.Name()[1:6] == "untar",
						"No temporary directories should remain: %s", entry.Name())
				}
			},
		},
		{
			name: "creates parent directories for nested output path",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				// Create a tar file
				tarPath := filepath.Join(tmpDir, "archive.tar")
				tarFile, err := os.Create(tarPath)
				require.NoError(t, err)
				defer tarFile.Close()

				tw := tar.NewWriter(tarFile)
				defer tw.Close()

				// Add a simple file
				data := []byte("test content")
				hdr := &tar.Header{
					Name: "file.txt",
					Mode: 0o600,
					Size: int64(len(data)),
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write(data)
				require.NoError(t, err)

				return tmpDir
			},
			cfg: builtin.UntarConfig{
				InPath:  "archive.tar",
				OutPath: filepath.Join("nested", "extracted"),
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				// Verify extraction worked and parent directories were created
				extractDir := filepath.Join(workDir, "nested", "extracted")
				filePath := filepath.Join(extractDir, "file.txt")
				content, err := os.ReadFile(filePath)
				assert.NoError(t, err)
				assert.Equal(t, "test content", string(content))

				// Verify no temp directories remain
				entries, err := os.ReadDir(workDir)
				assert.NoError(t, err)
				for _, entry := range entries {
					assert.False(t, entry.IsDir() && entry.Name()[0] == '.' &&
						len(entry.Name()) > 6 && entry.Name()[1:6] == "untar",
						"No temporary directories should remain: %s", entry.Name())
				}
			},
		},
		{
			name: "atomically replaces existing destination",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				// Create existing destination with old content
				extractDir := filepath.Join(tmpDir, "extracted")
				require.NoError(t, os.MkdirAll(extractDir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(extractDir, "old_file.txt"), []byte("old content"), 0o600))

				// Create a tar file
				tarPath := filepath.Join(tmpDir, "archive.tar")
				tarFile, err := os.Create(tarPath)
				require.NoError(t, err)
				defer tarFile.Close()

				tw := tar.NewWriter(tarFile)
				defer tw.Close()

				// Add a new file to the tar
				newData := []byte("new content")
				hdr := &tar.Header{
					Name: "new_file.txt",
					Mode: 0o600,
					Size: int64(len(newData)),
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write(newData)
				require.NoError(t, err)

				return tmpDir
			},
			cfg: builtin.UntarConfig{
				InPath:  "archive.tar",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				extractDir := filepath.Join(workDir, "extracted")

				// New file should exist
				newFilePath := filepath.Join(extractDir, "new_file.txt")
				b, err := os.ReadFile(newFilePath)
				assert.NoError(t, err)
				assert.Equal(t, "new content", string(b))

				// Old file should be gone (atomically replaced)
				oldFilePath := filepath.Join(extractDir, "old_file.txt")
				_, err = os.Stat(oldFilePath)
				assert.True(t, os.IsNotExist(err), "Old file should be replaced")
			},
		},
		{
			name: "fails with invalid input file path and maintains atomicity",
			setupFiles: func(t *testing.T) string {
				return t.TempDir()
			},
			cfg: builtin.UntarConfig{
				InPath:  "nonexistent.tar",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
				assert.ErrorContains(t, err, "failed to open tar file")

				// Verify no extraction directory was created due to atomic behavior
				extractDir := filepath.Join(workDir, "extracted")
				_, err = os.Stat(extractDir)
				assert.True(t, os.IsNotExist(err), "No extraction directory should exist after failure")

				// Verify no temp directories remain
				entries, err := os.ReadDir(workDir)
				assert.NoError(t, err)
				for _, entry := range entries {
					assert.False(t, entry.IsDir() && entry.Name()[0] == '.' &&
						len(entry.Name()) > 6 && entry.Name()[1:6] == "untar",
						"No temporary directories should remain: %s", entry.Name())
				}
			},
		},
		{
			name: "fails during extraction and maintains atomicity",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				// Create a tar file with a file that exceeds size limits
				tarPath := filepath.Join(tmpDir, "oversized.tar")
				tarFile, err := os.Create(tarPath)
				require.NoError(t, err)
				defer tarFile.Close()

				tw := tar.NewWriter(tarFile)
				defer tw.Close()

				// Add a file that exceeds the size limit
				largeFileSize := maxDecompressedFileSize + 1
				hdr := &tar.Header{
					Name: "large_file.bin",
					Mode: 0o600,
					Size: largeFileSize,
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write(make([]byte, largeFileSize))
				require.NoError(t, err)

				return tmpDir
			},
			cfg: builtin.UntarConfig{
				InPath:  "oversized.tar",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
				assert.ErrorContains(t, err, "exceeds size limit")

				// Ensure no partial files remain due to atomic extraction
				extractDir := filepath.Join(workDir, "extracted")
				_, err = os.Stat(extractDir)
				assert.True(t, os.IsNotExist(err))

				// Verify no temp directories remain
				entries, err := os.ReadDir(workDir)
				assert.NoError(t, err)
				for _, entry := range entries {
					assert.False(t, entry.IsDir() && entry.Name()[0] == '.' &&
						len(entry.Name()) > 6 && entry.Name()[1:6] == "untar",
						"no temporary directories should remain: %s", entry.Name())
				}
			},
		},
	}

	runner := &tarExtractor{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := tt.setupFiles(t)
			result, err := runner.run(
				t.Context(),
				&promotion.StepContext{WorkDir: workDir},
				tt.cfg,
			)
			tt.assertions(t, workDir, result, err)
		})
	}
}

func Test_tarExtractor_extractToDir(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles func(*testing.T) (string, string) // returns workDir, tarPath
		cfg        builtin.UntarConfig
		assertions func(*testing.T, string, string, promotion.StepResult, error) // workDir, extractDir, result, err
	}{
		{
			name: "succeeds extracting simple tar file",
			setupFiles: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()

				// Create a tar file
				tarPath := filepath.Join(tmpDir, "archive.tar")
				tarFile, err := os.Create(tarPath)
				require.NoError(t, err)
				defer tarFile.Close()

				tw := tar.NewWriter(tarFile)
				defer tw.Close()

				// Add a file to the tar
				file1Data := []byte("test content")
				hdr := &tar.Header{
					Name: "file1.txt",
					Mode: 0o600,
					Size: int64(len(file1Data)),
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write(file1Data)
				require.NoError(t, err)

				// Add a directory to the tar
				hdr = &tar.Header{
					Name:     "testdir/",
					Mode:     0o755,
					Typeflag: tar.TypeDir,
				}
				require.NoError(t, tw.WriteHeader(hdr))

				// Add a file in the directory
				file2Data := []byte("nested content")
				hdr = &tar.Header{
					Name: "testdir/file2.txt",
					Mode: 0o600,
					Size: int64(len(file2Data)),
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write(file2Data)
				require.NoError(t, err)

				return tmpDir, tarPath
			},
			cfg: builtin.UntarConfig{
				InPath:  "archive.tar",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, _, extractDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				// Check the root file
				file1Path := filepath.Join(extractDir, "file1.txt")
				b, err := os.ReadFile(file1Path)
				assert.NoError(t, err)
				assert.Equal(t, "test content", string(b))

				// Check the directory structure
				testDirPath := filepath.Join(extractDir, "testdir")
				assert.DirExists(t, testDirPath)

				// Check the nested file
				file2Path := filepath.Join(testDirPath, "file2.txt")
				b, err = os.ReadFile(file2Path)
				assert.NoError(t, err)
				assert.Equal(t, "nested content", string(b))
			},
		},
		{
			name: "succeeds extracting gzipped tar file",
			setupFiles: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()

				// Create a tar.gz file
				tarPath := filepath.Join(tmpDir, "archive.tar.gz")
				tarFile, err := os.Create(tarPath)
				require.NoError(t, err)
				defer tarFile.Close()

				gzw := gzip.NewWriter(tarFile)
				defer gzw.Close()

				tw := tar.NewWriter(gzw)
				defer tw.Close()

				// Add a file to the tar
				file1Data := []byte("compressed content")
				hdr := &tar.Header{
					Name: "compressed.txt",
					Mode: 0o600,
					Size: int64(len(file1Data)),
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write(file1Data)
				require.NoError(t, err)

				return tmpDir, tarPath
			},
			cfg: builtin.UntarConfig{
				InPath:  "archive.tar.gz",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, _, extractDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				filePath := filepath.Join(extractDir, "compressed.txt")
				b, err := os.ReadFile(filePath)
				assert.NoError(t, err)
				assert.Equal(t, "compressed content", string(b))
			},
		},
		{
			name: "strip components",
			setupFiles: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()

				// Create a tar file
				tarPath := filepath.Join(tmpDir, "archive.tar")
				tarFile, err := os.Create(tarPath)
				require.NoError(t, err)
				defer tarFile.Close()

				tw := tar.NewWriter(tarFile)
				defer tw.Close()

				// Add files with nested paths
				data := []byte("test content")
				hdr := &tar.Header{
					Name: "prefix1/prefix2/file.txt",
					Mode: 0o600,
					Size: int64(len(data)),
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write(data)
				require.NoError(t, err)

				return tmpDir, tarPath
			},
			cfg: builtin.UntarConfig{
				InPath:          "archive.tar",
				OutPath:         "extracted/",
				StripComponents: ptr.To(int64(2)),
			},
			assertions: func(t *testing.T, _, extractDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				filePath := filepath.Join(extractDir, "file.txt")
				b, err := os.ReadFile(filePath)
				assert.NoError(t, err)
				assert.Equal(t, "test content", string(b))

				// Ensure prefix directories were stripped
				assert.NoDirExists(t, filepath.Join(extractDir, "prefix1"))
			},
		},
		{
			name: "ignore rules",
			setupFiles: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()

				// Create a tar file
				tarPath := filepath.Join(tmpDir, "archive.tar")
				tarFile, err := os.Create(tarPath)
				require.NoError(t, err)
				defer tarFile.Close()

				tw := tar.NewWriter(tarFile)
				defer tw.Close()

				// Add files
				file1Data := []byte("include me")
				hdr := &tar.Header{
					Name: "include.txt",
					Mode: 0o600,
					Size: int64(len(file1Data)),
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write(file1Data)
				require.NoError(t, err)

				file2Data := []byte("ignore me")
				hdr = &tar.Header{
					Name: "ignore.txt",
					Mode: 0o600,
					Size: int64(len(file2Data)),
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write(file2Data)
				require.NoError(t, err)

				return tmpDir, tarPath
			},
			cfg: builtin.UntarConfig{
				InPath:  "archive.tar",
				OutPath: "extracted/",
				Ignore:  "ignore.txt",
			},
			assertions: func(t *testing.T, _, extractDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				// Check included file
				includePath := filepath.Join(extractDir, "include.txt")
				b, err := os.ReadFile(includePath)
				assert.NoError(t, err)
				assert.Equal(t, "include me", string(b))

				// Check ignored file
				ignorePath := filepath.Join(extractDir, "ignore.txt")
				_, err = os.Stat(ignorePath)
				assert.True(t, os.IsNotExist(err))
			},
		},
		{
			name: "handles symbolic links",
			setupFiles: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()

				// Create a tar file
				tarPath := filepath.Join(tmpDir, "archive.tar")
				tarFile, err := os.Create(tarPath)
				require.NoError(t, err)
				defer tarFile.Close()

				tw := tar.NewWriter(tarFile)
				defer tw.Close()

				// Add a file
				fileData := []byte("target content")
				hdr := &tar.Header{
					Name: "target.txt",
					Mode: 0o600,
					Size: int64(len(fileData)),
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write(fileData)
				require.NoError(t, err)

				// Add a symlink
				hdr = &tar.Header{
					Name:     "link.txt",
					Linkname: "target.txt",
					Typeflag: tar.TypeSymlink,
					Mode:     0o777,
				}
				require.NoError(t, tw.WriteHeader(hdr))

				return tmpDir, tarPath
			},
			cfg: builtin.UntarConfig{
				InPath:  "archive.tar",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, _, extractDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				// Check the regular file
				targetPath := filepath.Join(extractDir, "target.txt")
				b, err := os.ReadFile(targetPath)
				assert.NoError(t, err)
				assert.Equal(t, "target content", string(b))

				// Check the symlink
				linkPath := filepath.Join(extractDir, "link.txt")
				linkTarget, err := os.Readlink(linkPath)
				assert.NoError(t, err)
				assert.Equal(t, "target.txt", linkTarget)

				// Verify symlink works
				b, err = os.ReadFile(linkPath)
				assert.NoError(t, err)
				assert.Equal(t, "target content", string(b))
			},
		},
		{
			name: "fails with non-tar file",
			setupFiles: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				badPath := filepath.Join(tmpDir, "notatar.txt")
				require.NoError(t, os.WriteFile(badPath, []byte("not a tar file"), 0o600))
				return tmpDir, badPath
			},
			cfg: builtin.UntarConfig{
				InPath:  "notatar.txt",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, _, _ string, result promotion.StepResult, err error) {
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
				assert.ErrorContains(t, err, "error reading tar")
			},
		},
		{
			name: "skips unsafe path traversal attempts",
			setupFiles: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()

				// Create a tar file with unsafe paths
				tarPath := filepath.Join(tmpDir, "unsafe.tar")
				tarFile, err := os.Create(tarPath)
				require.NoError(t, err)
				defer tarFile.Close()

				tw := tar.NewWriter(tarFile)
				defer tw.Close()

				// Add files with unsafe paths
				unsafePaths := []string{
					"../outside.txt",           // Path traversal
					"/etc/passwd",              // Absolute path
					"subdir/../../outside.txt", // Path traversal
					"./config.txt",             // Relative current directory
					"safe.txt",                 // Safe path (should be extracted)
				}

				for _, path := range unsafePaths {
					data := []byte("test content for " + path)
					hdr := &tar.Header{
						Name: path,
						Mode: 0o600,
						Size: int64(len(data)),
					}
					require.NoError(t, tw.WriteHeader(hdr))
					_, err = tw.Write(data)
					require.NoError(t, err)
				}

				return tmpDir, tarPath
			},
			cfg: builtin.UntarConfig{
				InPath:  "unsafe.tar",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, workDir, extractDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				// Only the safe file should exist
				safePath := filepath.Join(extractDir, "safe.txt")
				_, err = os.Stat(safePath)
				assert.NoError(t, err, "Safe file should be extracted")

				// These unsafe paths should not exist relative to the extract dir
				unsafePaths := []string{
					filepath.Join(workDir, "outside.txt"),      // Path traversal
					filepath.Join(extractDir, "etc", "passwd"), // Absolute path
					filepath.Join(workDir, "outside.txt"),      // Path traversal
					filepath.Join(extractDir, "config.txt"),    // Relative current directory
				}

				for _, path := range unsafePaths {
					_, err = os.Stat(path)
					assert.True(t, os.IsNotExist(err), fmt.Sprintf("Unsafe path should not be extracted: %s", path))
				}
			},
		},
		{
			name: "skips unsafe symlinks",
			setupFiles: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()

				// Create a tar file with unsafe symlinks
				tarPath := filepath.Join(tmpDir, "unsafe_symlinks.tar")
				tarFile, err := os.Create(tarPath)
				require.NoError(t, err)
				defer tarFile.Close()

				tw := tar.NewWriter(tarFile)
				defer tw.Close()

				// Add a valid file
				fileData := []byte("target content")
				hdr := &tar.Header{
					Name: "safe.txt",
					Mode: 0o600,
					Size: int64(len(fileData)),
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write(fileData)
				require.NoError(t, err)

				// Add symlinks with unsafe targets
				symlinks := map[string]string{
					"safe_link.txt":    "safe.txt",            // Safe symlink
					"unsafe_link1.txt": "../outside.txt",      // Path traversal
					"unsafe_link2.txt": "/etc/passwd",         // Absolute path
					"unsafe_link3.txt": "./unsafe_target.txt", // Starts with ./
				}

				for name, target := range symlinks {
					hdr = &tar.Header{
						Name:     name,
						Linkname: target,
						Typeflag: tar.TypeSymlink,
						Mode:     0o777,
					}
					require.NoError(t, tw.WriteHeader(hdr))
				}

				return tmpDir, tarPath
			},
			cfg: builtin.UntarConfig{
				InPath:  "unsafe_symlinks.tar",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, _, extractDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				// The safe file and symlink should exist
				safePath := filepath.Join(extractDir, "safe.txt")
				_, err = os.Stat(safePath)
				assert.NoError(t, err, "Safe file should be extracted")

				safeLinkPath := filepath.Join(extractDir, "safe_link.txt")
				_, err = os.Stat(safeLinkPath)
				assert.NoError(t, err, "Safe symlink should be extracted")

				// Unsafe symlinks should not be created
				unsafeLinks := []string{
					filepath.Join(extractDir, "unsafe_link1.txt"),
					filepath.Join(extractDir, "unsafe_link2.txt"),
					filepath.Join(extractDir, "unsafe_link3.txt"),
				}

				for _, path := range unsafeLinks {
					_, err = os.Stat(path)
					assert.True(t, os.IsNotExist(err), fmt.Sprintf("Unsafe symlink should not be extracted: %s", path))
				}
			},
		},
		{
			name: "fails with file larger than size limit",
			setupFiles: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()

				// Create a tar file
				tarPath := filepath.Join(tmpDir, "oversized.tar")
				tarFile, err := os.Create(tarPath)
				require.NoError(t, err)
				defer tarFile.Close()

				tw := tar.NewWriter(tarFile)
				defer tw.Close()

				// Mock large file that exceeds the size limit
				largeFileSize := maxDecompressedFileSize + 1
				hdr := &tar.Header{
					Name: "large_file.bin",
					Mode: 0o600,
					Size: largeFileSize,
				}
				require.NoError(t, tw.WriteHeader(hdr))

				// Write zeros to the file
				_, err = tw.Write(make([]byte, largeFileSize))
				require.NoError(t, err)

				return tmpDir, tarPath
			},
			cfg: builtin.UntarConfig{
				InPath:  "oversized.tar",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, _, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
				assert.ErrorContains(t, err, "exceeds size limit")
				assert.ErrorContains(t, err, "large_file.bin")
			},
		},
		{
			name: "fails with total archive size larger than limit",
			setupFiles: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()

				// Create a tar file
				tarPath := filepath.Join(tmpDir, "oversized.tar")
				tarFile, err := os.Create(tarPath)
				require.NoError(t, err)
				defer tarFile.Close()

				tw := tar.NewWriter(tarFile)
				defer tw.Close()

				nbFiles := (maxDecompressedTarSize / maxDecompressedFileSize) + 1
				for i := range int(nbFiles) {
					hdr := &tar.Header{
						Name: fmt.Sprintf("file%d.bin", i),
						Mode: 0o600,
						Size: maxDecompressedFileSize - 1,
					}
					require.NoError(t, tw.WriteHeader(hdr))

					// Write zeros to the file
					_, err = tw.Write(make([]byte, maxDecompressedFileSize-1))
					require.NoError(t, err)
				}

				return tmpDir, tarPath
			},
			cfg: builtin.UntarConfig{
				InPath:  "oversized.tar",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, _, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
				assert.ErrorContains(t, err, "extraction aborted: total size would exceed limit")
			},
		},
		{
			name: "handles file permissions safely (masks setuid/setgid)",
			setupFiles: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()

				// Create a tar file
				tarPath := filepath.Join(tmpDir, "permissions.tar")
				tarFile, err := os.Create(tarPath)
				require.NoError(t, err)
				defer tarFile.Close()

				tw := tar.NewWriter(tarFile)
				defer tw.Close()

				// Add a file with setuid bit set (should be masked)
				data := []byte("test content")
				hdr := &tar.Header{
					Name: "setuid_file.txt",
					Mode: 0o4755, // setuid + rwxr-xr-x
					Size: int64(len(data)),
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write(data)
				require.NoError(t, err)

				return tmpDir, tarPath
			},
			cfg: builtin.UntarConfig{
				InPath:  "permissions.tar",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, _, extractDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				// Check that file was created with safe permissions (setuid bit removed)
				filePath := filepath.Join(extractDir, "setuid_file.txt")
				info, err := os.Stat(filePath)
				assert.NoError(t, err)

				// Should be 0755 (setuid bit removed by & 0o777 mask)
				assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())

				// Verify content is correct
				content, err := os.ReadFile(filePath)
				assert.NoError(t, err)
				assert.Equal(t, "test content", string(content))
			},
		},
	}

	runner := &tarExtractor{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir, tarPath := tt.setupFiles(t)
			extractDir := filepath.Join(workDir, "temp_extract")

			result, err := runner.extractToDir(
				t.Context(),
				tt.cfg,
				tarPath,
				extractDir,
			)

			tt.assertions(t, workDir, extractDir, result, err)
		})
	}
}

func Test_tarExtractor_validRelPath(t *testing.T) {
	extractor := &tarExtractor{}

	tests := []struct {
		path     string
		expected bool
	}{
		{"file.txt", true},
		{"dir/file.txt", true},
		{"dir/subdir/file.txt", true},
		{"", false},                      // Empty path
		{"/absolute/path", false},        // Absolute path
		{"../traversal", false},          // Path traversal
		{"..", false},                    // Bare parent reference
		{"dir/../file", false},           // Path traversal
		{"./current", false},             // Current directory
		{"file\\with\\backslash", false}, // Backslash (Windows path separator)
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("path_%s", tt.path), func(t *testing.T) {
			result := extractor.validRelPath(tt.path)
			assert.Equal(t, tt.expected, result, "Path: %s", tt.path)
		})
	}
}

func Test_symlinkTargetIsLocal(t *testing.T) {
	tests := []struct {
		target   string
		expected bool
	}{
		{"file.txt", true},
		{".", true},
		{"dir/file.txt", true},
		{"..file", true},
		{"", false},
		{"..", false},
		{"../file.txt", false},
		{"a/..", false},
		{"a/../b", false},
		{"/etc/passwd", false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("target_%s", tt.target), func(t *testing.T) {
			require.Equal(t, tt.expected, symlinkTargetIsLocal(tt.target))
		})
	}
}

func Test_parentHasSymlink(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "real"), 0o750))
	require.NoError(t, os.Symlink("real", filepath.Join(dir, "link")))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file"), nil, 0o600))
	root, err := os.OpenRoot(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	tests := []struct {
		name   string
		assert func(*testing.T, bool, error)
	}{
		{
			name: "top-level",
			assert: func(t *testing.T, hasLink bool, err error) {
				require.NoError(t, err)
				require.False(t, hasLink)
			},
		},
		{
			name: "real/nested/file",
			assert: func(t *testing.T, hasLink bool, err error) {
				require.NoError(t, err)
				require.False(t, hasLink)
			},
		},
		{
			name: "missing/nested/file",
			assert: func(t *testing.T, hasLink bool, err error) {
				require.NoError(t, err)
				require.False(t, hasLink)
			},
		},
		{
			name: "link/file",
			assert: func(t *testing.T, hasLink bool, err error) {
				require.NoError(t, err)
				require.True(t, hasLink)
			},
		},
		{
			name: "real/link/file",
			assert: func(t *testing.T, hasLink bool, err error) {
				require.NoError(t, err)
				require.False(t, hasLink) // real/link does not exist
			},
		},
		{
			name: "file/nested/file",
			assert: func(t *testing.T, _ bool, err error) {
				require.Error(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasLink, err := parentHasSymlink(root, filepath.FromSlash(tt.name))
			tt.assert(t, hasLink, err)
		})
	}
}

func Test_tarExtractor_run_symlinkEscapes(t *testing.T) {
	type entry struct {
		name     string
		linkname string // non-empty for a symlink
		content  string // for a regular file
	}
	tests := []struct {
		name    string
		entries []entry
		assert  func(t *testing.T, outDir string)
	}{
		{
			name: "single symlink to parent",
			entries: []entry{
				{name: "a", linkname: ".."},
				{name: "a/pwned.txt", content: "pwned"},
			},
			assert: func(t *testing.T, outDir string) {
				// The unsafe symlink is skipped, so the file lands in a real directory.
				fi, err := os.Lstat(filepath.Join(outDir, "a"))
				require.NoError(t, err)
				require.True(t, fi.IsDir())
			},
		},
		{
			name: "chained symlinks to parent",
			entries: []entry{
				{name: "a", linkname: ".."},
				{name: "a/b", linkname: ".."},
				{name: "a/b/pwned.txt", content: "pwned"},
			},
			assert: func(t *testing.T, outDir string) {
				// The unsafe symlink is skipped, so the file lands in a real directory.
				fi, err := os.Lstat(filepath.Join(outDir, "a"))
				require.NoError(t, err)
				require.True(t, fi.IsDir())
			},
		},
		{
			name: "nested symlink escaping relative to its own directory",
			entries: []entry{
				{name: "sub/x", linkname: "../.."},
				{name: "sub/x/pwned.txt", content: "pwned"},
			},
			assert: func(t *testing.T, outDir string) {
				// The unsafe symlink is skipped, so the file lands in a real directory.
				fi, err := os.Lstat(filepath.Join(outDir, "sub", "x"))
				require.NoError(t, err)
				require.True(t, fi.IsDir())
			},
		},
		{
			name: "symlink targets climbing through an earlier symlink",
			entries: []entry{
				{name: "a", linkname: "."},
				{name: "y", linkname: "a/.."},
				{name: "w", linkname: "y/.."},
			},
			assert: func(t *testing.T, outDir string) {
				for _, name := range []string{"y", "w"} {
					_, err := os.Lstat(filepath.Join(outDir, name))
					require.ErrorIs(t, err, os.ErrNotExist)
				}
			},
		},
		{
			name: "entries written through a local symlink",
			entries: []entry{
				{name: "a", linkname: "."},
				{name: "a/b", linkname: ".."},
				{name: "a/pwned.txt", content: "pwned"},
			},
			assert: func(t *testing.T, outDir string) {
				// The local symlink itself is allowed...
				target, err := os.Readlink(filepath.Join(outDir, "a"))
				require.NoError(t, err)
				require.Equal(t, ".", target)
				// ...but nothing may be written through it.
				for _, name := range []string{"b", "pwned.txt"} {
					_, err = os.Lstat(filepath.Join(outDir, name))
					require.ErrorIs(t, err, os.ErrNotExist)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// sharedDir stands in for the work area shared by all promotions.
			sharedDir := t.TempDir()
			workDir := filepath.Join(sharedDir, "promotion")
			require.NoError(t, os.Mkdir(workDir, 0o750))

			tarFile, err := os.Create(filepath.Join(workDir, "evil.tar"))
			require.NoError(t, err)
			tw := tar.NewWriter(tarFile)
			for _, e := range tt.entries {
				hdr := &tar.Header{
					Name:     e.name,
					Typeflag: tar.TypeReg,
					Mode:     0o600,
					Size:     int64(len(e.content)),
				}
				if e.linkname != "" {
					hdr.Typeflag = tar.TypeSymlink
					hdr.Linkname = e.linkname
					hdr.Size = 0
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write([]byte(e.content))
				require.NoError(t, err)
			}
			require.NoError(t, tw.Close())
			require.NoError(t, tarFile.Close())

			result, err := (&tarExtractor{}).run(
				t.Context(),
				&promotion.StepContext{WorkDir: workDir},
				builtin.UntarConfig{InPath: "evil.tar", OutPath: "out"},
			)
			require.NoError(t, err)
			require.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

			// Nothing may be written outside the output directory.
			sharedEntries, err := os.ReadDir(sharedDir)
			require.NoError(t, err)
			require.Len(t, sharedEntries, 1)
			workEntries, err := os.ReadDir(workDir)
			require.NoError(t, err)
			require.Len(t, workEntries, 2) // evil.tar and out

			tt.assert(t, filepath.Join(workDir, "out"))
		})
	}
}
