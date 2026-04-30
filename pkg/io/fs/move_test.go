package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleAtomicMove(t *testing.T) {
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
				require.NoError(t, os.WriteFile(filepath.Join(src, "file.txt"), []byte("content"), 0o600))

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
				require.NoError(t, os.WriteFile(filepath.Join(src, "new.txt"), []byte("new content"), 0o600))

				// Create existing destination with different content
				require.NoError(t, os.MkdirAll(dst, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(dst, "old.txt"), []byte("old content"), 0o600))

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
			src, dst := tt.setupFunc(t)

			err := SimpleAtomicMove(src, dst)
			tt.assertions(t, src, dst, err)
		})
	}
}
