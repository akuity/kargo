package fs

import (
	"os"
	"path/filepath"
	"strings"
)

// IsSubPath checks if the child path is a subpath of the parent path.
func IsSubPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != ".."
}

// ContainedRelPath returns a path relative to the base path, or the base if the
// path cannot be made relative.
func ContainedRelPath(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil || strings.Contains(rel, "..") {
		// If we can't make it relative, just use the filename.
		return filepath.Base(path)
	}
	return rel
}
