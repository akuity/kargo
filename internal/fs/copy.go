package fs

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// CopyDir recursively copies a directory tree from the source path to the
// destination path. It preserves file permissions and directory structures,
// but does not copy symbolic links.
func CopyDir(src, dst string) error {
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	absDst, err := filepath.Abs(dst)
	if err != nil {
		return err
	}

	return filepath.WalkDir(absSrc, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the source directory
		if path == absSrc {
			return nil
		}

		// Do not recurse into the destination directory
		if path == absDst {
			return fs.SkipDir
		}

		// Determine the relative path from the source directory
		relPath, err := filepath.Rel(absSrc, path)
		if err != nil {
			return err
		}

		// Determine the destination path
		destPath := filepath.Join(absDst, relPath)

		switch {
		case d.IsDir():
			// Get the directory information from the source to create the
			// directory at the destination with the same permissions
			info, err := d.Info()
			if err != nil {
				return err
			}
			// Create the directory at the destination
			return os.MkdirAll(destPath, info.Mode())
		case d.Type().IsRegular():
			// Copy the file from the source to the destination
			return CopyFile(path, destPath)
		default:
			// Ignore any other file types
			return nil
		}
	})
}

// CopyFile copies a file from the source to the destination. The file is
// copied with the same permissions as the source.
// If the destination file exists, it will be overwritten.
// If the source file is a symbolic link, the destination file will be a
// regular file with the same contents as the target of the symbolic link.
func CopyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := out.Close()
		if err == nil {
			err = closeErr
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}

	info, err := in.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(dst, info.Mode())
}
