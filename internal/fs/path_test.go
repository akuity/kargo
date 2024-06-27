package fs

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithinBasePath(t *testing.T) {
	const sep = string(filepath.Separator)

	tests := []struct {
		name     string
		base     string
		path     string
		expected bool
	}{
		{
			name:     "path is within base path",
			base:     filepath.Join(sep, "a"),
			path:     filepath.Join(sep, "a", "b"),
			expected: true,
		},
		{
			name:     "path is exactly the base path",
			base:     filepath.Join(sep, "a"),
			path:     filepath.Join(sep, "a"),
			expected: true,
		},
		{
			name:     "path is outside the base path",
			base:     filepath.Join(sep, "a"),
			path:     filepath.Join(sep, "b"),
			expected: false,
		},
		{
			name:     "relative path outside the base path",
			base:     filepath.Join(sep, "a"),
			path:     filepath.Join("..", "b"),
			expected: false,
		},
		{
			name:     "path with .. within the base path",
			base:     filepath.Join(sep, "a"),
			path:     filepath.Join(sep, "c", "..", "a", "b"),
			expected: true,
		},
		{
			name:     "path navigating up and back to the base path",
			base:     filepath.Join(sep, "a"),
			path:     filepath.Join(sep, "a", "..", "..", "a", "b"),
			expected: true,
		},
		{
			name:     "path navigating up and out of the base path",
			base:     filepath.Join(sep, "a"),
			path:     filepath.Join(sep, "a", "..", "..", "b"),
			expected: false,
		},
		{
			name:     "path navigating out of the base path from within",
			base:     filepath.Join(sep, "a"),
			path:     filepath.Join(sep, "a", "b", "..", "..", "c"),
			expected: false,
		},
		{
			name:     "base path is empty",
			base:     "",
			path:     filepath.Join(sep, "a"),
			expected: false,
		},
		{
			name:     "path is empty",
			base:     filepath.Join(sep, "a"),
			path:     "",
			expected: false,
		},
		{
			name:     "base path is root",
			base:     sep,
			path:     filepath.Join(sep, "a"),
			expected: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WithinBasePath(tt.base, tt.path)
			require.Equal(
				t,
				result,
				tt.expected,
				"WithinBasePath(%q, %q) = %v != %v", tt.base, tt.path, result, tt.expected,
			)
		})
	}
}
