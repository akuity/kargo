package promotion

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// WorkspacePath converts a path from a step's configuration into a path
// relative to the step's working directory, suitable for use with the methods
// of an os.Root. A leading "/" denotes the working directory itself, as does an
// empty path. It returns an error if the path lexically climbs out of the
// working directory.
func WorkspacePath(p string) (string, error) {
	rel := filepath.Clean(strings.TrimLeft(p, "/"))
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q is outside the working directory", p)
	}
	return rel, nil
}

// ExternalPath returns the absolute path for p, for handing to a subprocess or
// a library that takes a path. It returns an error if p, or the nearest
// existing ancestor of p, resolves outside the step's working directory,
// symlinks included.
func ExternalPath(stepCtx *StepContext, p string) (string, error) {
	rel, err := WorkspacePath(p)
	if err != nil {
		return "", err
	}
	for check := rel; ; check = filepath.Dir(check) {
		if _, err = stepCtx.Root.Stat(check); err == nil {
			break
		}
		if !errors.Is(err, fs.ErrNotExist) || check == "." {
			return "", fmt.Errorf(
				"error resolving path %q in the working directory: %w", p, err,
			)
		}
	}
	return filepath.Join(stepCtx.WorkDir, rel), nil
}
