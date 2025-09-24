package fs

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizePathError(t *testing.T) {
	tests := []struct {
		name       string
		workDir    string
		err        error
		assertions func(*testing.T, error)
	}{
		{
			name:    "PathError with relative path",
			workDir: "/tmp/work/dir",
			err:     &os.PathError{Op: "open", Path: "/tmp/work/dir/file.txt", Err: os.ErrNotExist},
			assertions: func(t *testing.T, result error) {
				var pathErr *os.PathError
				assert.True(t, errors.As(result, &pathErr))
				assert.Equal(t, "open", pathErr.Op)
				assert.Equal(t, "file.txt", pathErr.Path)
				assert.Equal(t, os.ErrNotExist, pathErr.Err)
			},
		},
		{
			name:    "PathError with path outside workDir",
			workDir: "/tmp/work/dir",
			err:     &os.PathError{Op: "read", Path: "/etc/config.ini", Err: os.ErrPermission},
			assertions: func(t *testing.T, result error) {
				var pathErr *os.PathError
				assert.True(t, errors.As(result, &pathErr))
				assert.Equal(t, "read", pathErr.Op)
				assert.Equal(t, "config.ini", pathErr.Path)
				assert.Equal(t, os.ErrPermission, pathErr.Err)
			},
		},
		{
			name:    "non-PathError",
			workDir: "/tmp/work/dir",
			err:     errors.New("generic error"),
			assertions: func(t *testing.T, result error) {
				assert.Equal(t, "generic error", result.Error())
			},
		},
		{
			name:    "PathError with workDir",
			workDir: "/tmp/work/dir",
			err:     &os.PathError{Op: "stat", Path: "/tmp/work/dir", Err: os.ErrNotExist},
			assertions: func(t *testing.T, result error) {
				var pathErr *os.PathError
				errors.As(result, &pathErr)
				assert.Equal(t, "stat", pathErr.Op)
				assert.Equal(t, ".", pathErr.Path)
				assert.Equal(t, os.ErrNotExist, pathErr.Err)
			},
		},
		{
			name: "nil error",
			err:  nil,
			assertions: func(t *testing.T, result error) {
				assert.Nil(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePathError(tt.err, tt.workDir)
			tt.assertions(t, result)
		})
	}
}
