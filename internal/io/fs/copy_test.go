package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyFile(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*testing.T) (string, string)
		assertions func(*testing.T, string, string, error)
	}{
		{
			name: "successful backup",
			setup: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				src := filepath.Join(tmpDir, "test_src.txt")
				dst := filepath.Join(tmpDir, "test_dst.txt")
				require.NoError(t, os.WriteFile(src, []byte("test content"), 0o600))
				return src, dst
			},
			assertions: func(t *testing.T, src, dst string, err error) {
				require.NoError(t, err)
				require.FileExists(t, dst)

				// Compare contents
				srcContent, err := os.ReadFile(src)
				require.NoError(t, err)
				dstContent, err := os.ReadFile(dst)
				require.NoError(t, err)
				assert.Equal(t, srcContent, dstContent)

				// Compare permissions
				srcInfo, err := os.Stat(src)
				require.NoError(t, err)
				dstInfo, err := os.Stat(dst)
				require.NoError(t, err)
				assert.Equal(t, srcInfo.Mode(), dstInfo.Mode())
			},
		},
		{
			name: "source file does not exist",
			setup: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				src := filepath.Join(tmpDir, "nonexistent.txt")
				dst := filepath.Join(tmpDir, "test_dst.txt")
				return src, dst
			},
			assertions: func(t *testing.T, _, _ string, err error) {
				assert.ErrorIs(t, err, os.ErrNotExist)
			},
		},
		{
			name: "destination file already exists",
			setup: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()

				src := filepath.Join(tmpDir, "test_src.txt")
				dst := filepath.Join(tmpDir, "test_dst.txt")

				require.NoError(t, os.WriteFile(src, []byte("test content"), 0o600))
				require.NoError(t, os.WriteFile(dst, []byte("existing content"), 0o600))
				return src, dst
			},
			assertions: func(t *testing.T, _, _ string, err error) {
				assert.ErrorIs(t, err, os.ErrExist)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst := tt.setup(t)
			tt.assertions(t, src, dst, CopyFile(src, dst))
		})
	}
}
