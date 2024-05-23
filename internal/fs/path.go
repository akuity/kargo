package fs

import (
	"path/filepath"
	"strings"
)

const dotDotSlash = ".." + string(filepath.Separator)

// WithinBasePath returns true if the path is within the base path.
func WithinBasePath(base, path string) bool {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	return rel != dotDotSlash && !strings.HasPrefix(rel, dotDotSlash)
}
