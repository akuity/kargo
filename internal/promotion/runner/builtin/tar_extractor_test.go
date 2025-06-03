package builtin

import (
	"archive/tar"
	"compress/gzip"
	"context"
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
			name: "atomically replaces existing destination",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				// Create existing destination with old content
				extractDir := filepath.Join(tmpDir, "extracted")
				require.NoError(t, os.MkdirAll(extractDir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(extractDir, "old_file.txt"), []byte("old content"), 0o644))

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
				largeFileSize := MaxDecompressedFileSize + 1
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
				context.Background(),
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
			assertions: func(t *testing.T, workDir, extractDir string, result promotion.StepResult, err error) {
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
			assertions: func(t *testing.T, workDir, extractDir string, result promotion.StepResult, err error) {
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
			assertions: func(t *testing.T, workDir, extractDir string, result promotion.StepResult, err error) {
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
			assertions: func(t *testing.T, workDir, extractDir string, result promotion.StepResult, err error) {
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
			assertions: func(t *testing.T, workDir, extractDir string, result promotion.StepResult, err error) {
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
			assertions: func(t *testing.T, workDir, extractDir string, result promotion.StepResult, err error) {
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
			assertions: func(t *testing.T, workDir, extractDir string, result promotion.StepResult, err error) {
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
				largeFileSize := MaxDecompressedFileSize + 1
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
			assertions: func(t *testing.T, workDir, extractDir string, result promotion.StepResult, err error) {
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

				nbFiles := (MaxDecompressedTarSize / MaxDecompressedFileSize) + 1
				for i := range int(nbFiles) {
					hdr := &tar.Header{
						Name: fmt.Sprintf("file%d.bin", i),
						Mode: 0o600,
						Size: MaxDecompressedFileSize - 1,
					}
					require.NoError(t, tw.WriteHeader(hdr))

					// Write zeros to the file
					_, err = tw.Write(make([]byte, MaxDecompressedFileSize-1))
					require.NoError(t, err)
				}

				return tmpDir, tarPath
			},
			cfg: builtin.UntarConfig{
				InPath:  "oversized.tar",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, workDir, extractDir string, result promotion.StepResult, err error) {
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
			assertions: func(t *testing.T, workDir, extractDir string, result promotion.StepResult, err error) {
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
				context.Background(),
				tt.cfg,
				tarPath,
				extractDir,
			)

			tt.assertions(t, workDir, extractDir, result, err)
		})
	}
}

func Test_tarExtractor_simpleAtomicMove(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func(*testing.T) (string, string)
		assertions func(*testing.T, string, string, error)
	}{
		{
			name: "successful move to non-existent destination",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				src := filepath.Join(tmpDir, "src")
				dst := filepath.Join(tmpDir, "dst")

				// Create source directory with content
				require.NoError(t, os.MkdirAll(src, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(src, "file.txt"), []byte("content"), 0o644))

				return src, dst
			},
			assertions: func(t *testing.T, src, dst string, err error) {
				assert.NoError(t, err)

				// Source should no longer exist
				_, err = os.Stat(src)
				assert.True(t, os.IsNotExist(err))

				// Destination should exist with content
				content, err := os.ReadFile(filepath.Join(dst, "file.txt"))
				assert.NoError(t, err)
				assert.Equal(t, "content", string(content))
			},
		},
		{
			name: "successful move overwriting existing destination",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				src := filepath.Join(tmpDir, "src")
				dst := filepath.Join(tmpDir, "dst")

				// Create source directory with content
				require.NoError(t, os.MkdirAll(src, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(src, "new.txt"), []byte("new content"), 0o644))

				// Create existing destination with different content
				require.NoError(t, os.MkdirAll(dst, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(dst, "old.txt"), []byte("old content"), 0o644))

				return src, dst
			},
			assertions: func(t *testing.T, src, dst string, err error) {
				assert.NoError(t, err)

				// Source should no longer exist
				_, err = os.Stat(src)
				assert.True(t, os.IsNotExist(err))

				// Destination should have new content, not old
				content, err := os.ReadFile(filepath.Join(dst, "new.txt"))
				assert.NoError(t, err)
				assert.Equal(t, "new content", string(content))

				// Old file should not exist
				_, err = os.Stat(filepath.Join(dst, "old.txt"))
				assert.True(t, os.IsNotExist(err))
			},
		},
		{
			name: "fails when source doesn't exist",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				src := filepath.Join(tmpDir, "nonexistent")
				dst := filepath.Join(tmpDir, "dst")
				return src, dst
			},
			assertions: func(t *testing.T, src, dst string, err error) {
				assert.Error(t, err)

				// Neither should exist
				_, err = os.Stat(src)
				assert.True(t, os.IsNotExist(err))
				_, err = os.Stat(dst)
				assert.True(t, os.IsNotExist(err))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := &tarExtractor{}
			src, dst := tt.setupFunc(t)

			err := extractor.simpleAtomicMove(src, dst)
			tt.assertions(t, src, dst, err)
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
