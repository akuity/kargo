package fs

import (
	"fmt"
	"os"
)

// SimpleAtomicMove performs an atomic move operation from src to dst.
// If the destination already exists, it removes it before attempting the move.
// It returns an error if the move operation fails for any reason, including
// if the source file does not exist or if the destination cannot be created.
func SimpleAtomicMove(src, dst string) error {
	if err := os.Rename(src, dst); err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("failed to move %s to %s: %w", src, dst, err)
		}

		// If the destination already exists, remove it and try again
		if err = os.RemoveAll(dst); err != nil {
			return fmt.Errorf("failed to remove existing destination %s: %w", dst, err)
		}

		if err = os.Rename(src, dst); err != nil {
			return fmt.Errorf("failed to move %s to %s after removing existing: %w", src, dst, err)
		}
	}
	return nil
}
