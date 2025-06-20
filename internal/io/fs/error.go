package fs

import (
	"errors"
	"io/fs"
)

// SanitizePathError sanitizes the path in a path error to be relative to the
// work directory. If the path cannot be made relative, the filename is used
// instead.
//
// This is useful for making error messages more user-friendly, as the work
// directory is typically a temporary directory that the user does not care
// about.
func SanitizePathError(err error, workDir string) error {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		// Reconstruct the error with the sanitized path.
		return &fs.PathError{
			Op:   pathErr.Op,
			Path: ContainedRelPath(workDir, pathErr.Path),
			Err:  pathErr.Err,
		}
	}
	// Return the original error if it's not a path error.
	return err
}
