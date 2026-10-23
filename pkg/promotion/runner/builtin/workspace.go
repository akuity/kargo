package builtin

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// maxTempNameAttempts bounds the number of random names tried when creating a
// temporary file or directory.
const maxTempNameAttempts = 10000

// maxStatTreeDepth bounds how deep statTree descends, which also stops it
// from looping forever through a cycle of symlinks.
const maxStatTreeDepth = 64

// createTempFile creates a new file in dir, a directory relative to root, and
// returns it open for reading and writing along with its path relative to
// root. The file is named by tempName.
func createTempFile(root *os.Root, dir, pattern string) (*os.File, string, error) {
	for range maxTempNameAttempts {
		name := filepath.Join(dir, tempName(pattern))
		f, err := root.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			return f, name, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, "", err
		}
	}
	return nil, "", fmt.Errorf("failed to find an unused name for a temporary file in %q", dir)
}

// mkdirTemp creates a new directory in dir, a directory relative to root, and
// returns its path relative to root. The directory is named by tempName.
func mkdirTemp(root *os.Root, dir, pattern string) (string, error) {
	for range maxTempNameAttempts {
		name := filepath.Join(dir, tempName(pattern))
		err := root.Mkdir(name, 0o700)
		if err == nil {
			return name, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", err
		}
	}
	return "", fmt.Errorf("failed to find an unused name for a temporary directory in %q", dir)
}

// tempName returns pattern with its last "*" replaced by a random string, or
// pattern followed by a random string if it contains none.
func tempName(pattern string) string {
	random := rand.Text()
	if i := strings.LastIndex(pattern, "*"); i >= 0 {
		return pattern[:i] + random + pattern[i+1:]
	}
	return pattern + random
}

// replace renames src to dst, both relative to root. A symlink at dst is
// resolved first, so that what it points to is replaced rather than the link
// itself. Anything already at dst that the rename cannot replace in place is
// removed first.
func replace(root *os.Root, src, dst string) error {
	dst, err := resolvePath(root, dst)
	if err != nil {
		return err
	}
	if err = root.Rename(src, dst); err != nil {
		if !errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("failed to move %s to %s: %w", src, dst, err)
		}
		if err = root.RemoveAll(dst); err != nil {
			return fmt.Errorf("failed to remove existing destination %s: %w", dst, err)
		}
		if err = root.Rename(src, dst); err != nil {
			return fmt.Errorf("failed to move %s to %s after removing existing: %w", src, dst, err)
		}
	}
	return nil
}

// statTree stats p, a path relative to root, and if it is a directory every
// entry beneath it, following symlinks and descending into directories they
// lead to. It returns an error if anything cannot be resolved, including an
// entry that resolves outside root or a dangling symlink. Only p itself being
// absent is reported as fs.ErrNotExist. A clean return means a library that
// reads or writes the tree by path will, at this moment, touch nothing outside
// root.
func statTree(root *os.Root, p string) error {
	info, err := root.Stat(p)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}
	return statTreeAtDepth(root, p, 0)
}

func statTreeAtDepth(root *os.Root, dir string, depth int) error {
	if depth > maxStatTreeDepth {
		return fmt.Errorf("directory %q is nested too deeply", dir)
	}
	entries, err := fs.ReadDir(root.FS(), dir)
	if err != nil {
		return fmt.Errorf("failed to read directory %q: %w", dir, hideNotExist(err))
	}
	for _, entry := range entries {
		name := path.Join(dir, entry.Name())
		info, err := root.Stat(name)
		if err != nil {
			return fmt.Errorf("failed to inspect %q: %w", name, hideNotExist(err))
		}
		if info.IsDir() {
			if err = statTreeAtDepth(root, name, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

// hideNotExist returns err, or an error with the same message that does not
// match fs.ErrNotExist if err does.
func hideNotExist(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return errors.New(err.Error())
	}
	return err
}

// resolvePath resolves the symlinks in rel, a path relative to root, and
// returns the resulting path, also relative to root. A path that does not
// exist is returned unchanged. It returns an error if the path resolves
// outside root.
func resolvePath(root *os.Root, rel string) (string, error) {
	resolved, err := filepath.EvalSymlinks(filepath.Join(root.Name(), rel))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return rel, nil
		}
		return "", fmt.Errorf("failed to resolve path %q: %w", rel, err)
	}
	workDir, err := filepath.EvalSymlinks(root.Name())
	if err != nil {
		return "", fmt.Errorf("failed to resolve working directory: %w", err)
	}
	resolvedRel, err := filepath.Rel(workDir, resolved)
	if err != nil || !filepath.IsLocal(resolvedRel) {
		return "", fmt.Errorf("path %q resolves outside the working directory", rel)
	}
	return resolvedRel, nil
}
