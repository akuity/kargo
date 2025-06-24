package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

// ValidateSymlinks checks for symlinks in the specified directory that point
// outside the specified root path. It avoids infinite recursion by limiting
// the depth of recursion. If a symlink is found that points outside the root
// path, or if the maximum recursion depth is exceeded, an error is returned.
//
// To allow for any depth of recursion, set maxDepth to -1.
func ValidateSymlinks(root, dir string, maxDepth int) error {
	// Validate the root path
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("root path %q does not exist: %w", root, err)
		}
		return fmt.Errorf("failed to access root path %q: %w", root, err)
	}

	// Create a map to track visited directories to avoid infinite recursion
	visited := make(map[string]struct{})

	// Start the recursive validation
	return validateSymlinks(root, dir, visited, 0, maxDepth)
}

// validateSymlinks recursively checks for symlinks that point outside the root
// path and avoids infinite recursion by using a single map of visited directories
// (absolute paths). The depth parameter is used to limit the recursion depth,
// with a value of -1 indicating no limit.
func validateSymlinks(root, dir string, visited map[string]struct{}, depth, maxDepth int) error {
	// Get the absolute path of the current directory
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for dir: %v", err)
	}

	// Check if we've already visited this directory
	if _, ok := visited[absDir]; ok {
		// Skip it to avoid infinite recursion or redundant visits
		return nil
	}

	// Mark this directory as visited only when starting to process it
	visited[absDir] = struct{}{}

	// Check if the recursion depth is within the limit
	if maxDepth >= 0 && depth >= maxDepth {
		return fmt.Errorf("maximum recursion depth exceeded")
	}

	// Open the directory
	dirEntries, err := os.ReadDir(absDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", SanitizePathError(err, root))
	}

	// Process each entry in the directory
	for _, entry := range dirEntries {
		entryPath := filepath.Join(dir, entry.Name())

		// If the entry is a symlink, resolve it
		if entry.Type()&os.ModeSymlink != 0 {
			// Resolve the symlink to its target
			target, pathErr := filepath.EvalSymlinks(entryPath)
			if pathErr != nil {
				return fmt.Errorf("failed to resolve symlink: %w", SanitizePathError(pathErr, root))
			}

			// Convert the target path to its absolute form
			absTarget, pathErr := filepath.Abs(target)
			if pathErr != nil {
				return pathErr
			}

			// Ensure the target is within the root directory
			if !IsSubPath(root, absTarget) {
				return fmt.Errorf("symlink at %s points outside the path boundary", ContainedRelPath(root, entryPath))
			}

			// Recursively check the symlinked directory or file if not visited yet
			if _, ok := visited[absTarget]; !ok {
				// Check if the symlink target is a directory
				targetInfo, pathErr := os.Stat(absTarget)
				if pathErr != nil {
					return fmt.Errorf(
						"failed to stat symlink target of %s: %w",
						ContainedRelPath(root, entryPath),
						SanitizePathError(pathErr, root),
					)
				}

				if targetInfo.IsDir() {
					// Recursively call the function for the symlinked directory
					if err = validateSymlinks(root, absTarget, visited, depth+1, maxDepth); err != nil {
						return err
					}
				}

				// It's a file, no further need for recursion here
				// We still add it to the visited map to avoid redundant checks
				visited[absTarget] = struct{}{}
			}
		} else if entry.IsDir() {
			// If it's a directory, manually recurse into it
			if err = validateSymlinks(root, entryPath, visited, depth+1, maxDepth); err != nil {
				return err
			}
		}
	}

	return nil
}
