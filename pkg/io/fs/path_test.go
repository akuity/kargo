package fs

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSubPath(t *testing.T) {
	tests := []struct {
		name   string
		parent string
		child  string
		want   bool
	}{
		{
			name:   "child is direct subdirectory",
			parent: "a/b",
			child:  "a/b/c",
			want:   true,
		},
		{
			name:   "child is nested subdirectory",
			parent: "a/b",
			child:  "a/b/c/d",
			want:   true,
		},
		{
			name:   "child is parent directory",
			parent: "a/b/c",
			child:  "a",
			want:   false,
		},
		{
			name:   "child is same as parent",
			parent: "a/b",
			child:  "a/b",
			want:   true,
		},
		{
			name:   "child is sibling directory",
			parent: "a/b1",
			child:  "a/b2",
			want:   false,
		},
		{
			name:   "parent is root",
			parent: ".",
			child:  "a/b",
			want:   true,
		},
		{
			name:   "child contains parent as prefix",
			parent: "a/b",
			child:  "a/bc/d",
			want:   false,
		},
		{
			name:   "parent and child on different roots",
			parent: "x/y",
			child:  "a/b",
			want:   false,
		},
		{
			name:   "child is parent with trailing separator",
			parent: "a/b",
			child:  "a/b/",
			want:   true,
		},
		{
			name:   "parent and child with relative paths",
			parent: "a",
			child:  "a/b",
			want:   true,
		},
		{
			name:   "complex nested structure",
			parent: "a/b/c",
			child:  "a/b/c/d/e/f",
			want:   true,
		},
		{
			name:   "parent is empty string",
			parent: "",
			child:  "a",
			want:   true,
		},
		{
			name:   "child is empty string",
			parent: "a",
			child:  "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert slashes to OS-specific separators
			parent := filepath.FromSlash(tt.parent)
			child := filepath.FromSlash(tt.child)

			got := IsSubPath(parent, child)
			assert.Equal(t, tt.want, got)
		})
	}
}
