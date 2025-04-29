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
			name: "succeeds extracting simple tar file",
			setupFiles: func(t *testing.T) string {
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
					Mode: 0600,
					Size: int64(len(file1Data)),
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write(file1Data)
				require.NoError(t, err)

				// Add a directory to the tar
				hdr = &tar.Header{
					Name:     "testdir/",
					Mode:     0755,
					Typeflag: tar.TypeDir,
				}
				require.NoError(t, tw.WriteHeader(hdr))

				// Add a file in the directory
				file2Data := []byte("nested content")
				hdr = &tar.Header{
					Name: "testdir/file2.txt",
					Mode: 0600,
					Size: int64(len(file2Data)),
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write(file2Data)
				require.NoError(t, err)

				return tmpDir
			},
			cfg: builtin.UntarConfig{
				InPath:  "archive.tar",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

				extractDir := filepath.Join(workDir, "extracted")

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
			setupFiles: func(t *testing.T) string {
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
					Mode: 0600,
					Size: int64(len(file1Data)),
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write(file1Data)
				require.NoError(t, err)

				return tmpDir
			},
			cfg: builtin.UntarConfig{
				InPath:  "archive.tar.gz",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

				extractDir := filepath.Join(workDir, "extracted")
				filePath := filepath.Join(extractDir, "compressed.txt")
				b, err := os.ReadFile(filePath)
				assert.NoError(t, err)
				assert.Equal(t, "compressed content", string(b))
			},
		},
		{
			name: "strip components",
			setupFiles: func(t *testing.T) string {
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
					Mode: 0600,
					Size: int64(len(data)),
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write(data)
				require.NoError(t, err)

				return tmpDir
			},
			cfg: builtin.UntarConfig{
				InPath:          "archive.tar",
				OutPath:         "extracted/",
				StripComponents: intPtr(2), // Use helper function to create int64 pointer
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

				extractDir := filepath.Join(workDir, "extracted")
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
			setupFiles: func(t *testing.T) string {
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
					Mode: 0600,
					Size: int64(len(file1Data)),
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write(file1Data)
				require.NoError(t, err)

				file2Data := []byte("ignore me")
				hdr = &tar.Header{
					Name: "ignore.txt",
					Mode: 0600,
					Size: int64(len(file2Data)),
				}
				require.NoError(t, tw.WriteHeader(hdr))
				_, err = tw.Write(file2Data)
				require.NoError(t, err)

				return tmpDir
			},
			cfg: builtin.UntarConfig{
				InPath:  "archive.tar",
				OutPath: "extracted/",
				Ignore:  "ignore.txt",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

				extractDir := filepath.Join(workDir, "extracted")

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
			setupFiles: func(t *testing.T) string {
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
					Mode: 0600,
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
					Mode:     0777,
				}
				require.NoError(t, tw.WriteHeader(hdr))

				return tmpDir
			},
			cfg: builtin.UntarConfig{
				InPath:  "archive.tar",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

				extractDir := filepath.Join(workDir, "extracted")

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
			name: "fails with invalid input file",
			setupFiles: func(t *testing.T) string {
				return t.TempDir()
			},
			cfg: builtin.UntarConfig{
				InPath:  "nonexistent.tar",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionPhaseErrored}, result)
				assert.ErrorContains(t, err, "failed to open tar file")
			},
		},
		{
			name: "fails with non-tar file",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()
				badPath := filepath.Join(tmpDir, "notatar.txt")
				require.NoError(t, os.WriteFile(badPath, []byte("not a tar file"), 0600))
				return tmpDir
			},
			cfg: builtin.UntarConfig{
				InPath:  "notatar.txt",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionPhaseErrored}, result)
				assert.ErrorContains(t, err, "error reading tar")
			},
		},
		{
			name: "skips unsafe path traversal attempts",
			setupFiles: func(t *testing.T) string {
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
						Mode: 0600,
						Size: int64(len(data)),
					}
					require.NoError(t, tw.WriteHeader(hdr))
					_, err = tw.Write(data)
					require.NoError(t, err)
				}

				return tmpDir
			},
			cfg: builtin.UntarConfig{
				InPath:  "unsafe.tar",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

				extractDir := filepath.Join(workDir, "extracted")

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
			setupFiles: func(t *testing.T) string {
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
					Mode: 0600,
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
						Mode:     0777,
					}
					require.NoError(t, tw.WriteHeader(hdr))
				}

				return tmpDir
			},
			cfg: builtin.UntarConfig{
				InPath:  "unsafe_symlinks.tar",
				OutPath: "extracted/",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

				extractDir := filepath.Join(workDir, "extracted")

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

// Helper function to create an int64 pointer
func intPtr(i int64) *int64 {
	return &i
}
