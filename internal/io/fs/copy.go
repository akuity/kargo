package fs

import (
	"io"
	"os"
)

// CopyFile copies a file from src to dst, preserving the file's permissions.
// It returns an error if the source file cannot be opened, if the destination
// file cannot be created (e.g., if it already exists), or if the copy operation
// fails.
func CopyFile(src, dst string) (err error) {
	// Open the source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := srcFile.Close()
		if err == nil {
			err = closeErr
		}
	}()

	// Get file info to retrieve permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create the destination file with the same permissions
	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer func() {
		closeErr := dstFile.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(dst)
		}
	}()

	// Copy the contents
	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return nil
}
