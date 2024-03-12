package yaml

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var sep = []byte("\n---\n")

// Read reads all files represented by the given paths having a .yaml or .yml
// extension, joins them with "\n---\n", and returns the result. For any path
// that is a directory, it recursively reads all files in the directory. This
// function performs no validation on the contents of the files.
func Read(paths []string) ([]byte, error) {
	var err error
	for i, path := range paths {
		// Note that filepath.Abs will also drop any trailing slash
		if paths[i], err = filepath.Abs(path); err != nil {
			return nil, err
		}
		if paths[i], err = filepath.EvalSymlinks(paths[i]); err != nil {
			return nil, err
		}
	}
	var allBytes [][]byte
	pathsRead := make(map[string]struct{})
	for _, path := range paths {
		readBytes, err := read(path, paths, pathsRead)
		if err != nil {
			return nil, err
		}
		// Note this is only resizing a slice of references to byte slices and not
		// resizing the byte slices themselves
		allBytes = append(allBytes, readBytes...)
	}
	return bytes.Join(allBytes, sep), nil
}

// read recursively reads all files represented by the given path, provided they
// have a .yaml or .yml extension. entryPaths represents a broader selection of
// files that is being read and is used in logic to avoid infinite recursion due
// to symlinks. pathsRead tracks file and directory paths that have been read in
// order to prevent duplicate reads due to symlinks. It is assumed that path and
// all entryPaths are absolute paths with no trailing slash. It is further
// assumed that no entryPaths are themselves symlinks.
func read(
	path string,
	entryPaths []string,
	pathsRead map[string]struct{},
) ([][]byte, error) {
	evaledPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	if _, alreadyRead := pathsRead[evaledPath]; alreadyRead {
		return nil, nil
	}
	// If evaledPath is different from path, then path is a symlink. This is fine
	// as long as the symlink target is not already contained within the
	// entryPaths. Otherwise, we ignore the symlink because in the best case, what
	// it points to is already being read, and in the worst case, it would cause
	// infinite recursion.
	//
	// Note: We do not worry about the case where the symlink target is a
	// directory that contains directories or files that are already being read.
	// We allow that, but use the pathsRead map to prevent duplicate reads.
	if evaledPath != path && isPathInPaths(evaledPath, entryPaths) {
		return nil, nil
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !fileInfo.IsDir() {
		// path points to a file. Just read the file.
		if filepath.Ext(path) != ".yaml" && filepath.Ext(path) != ".yml" {
			return nil, nil
		}
		var readBytes []byte
		if readBytes, err = os.ReadFile(path); err != nil {
			return nil, err
		}
		pathsRead[evaledPath] = struct{}{}
		return [][]byte{readBytes}, nil
	}
	// path points to a directory. Read all files in the directory.
	var allBytes [][]byte
	var dirEntries []os.DirEntry
	if dirEntries, err = os.ReadDir(path); err != nil {
		return nil, err
	}
	for _, dirEntry := range dirEntries {
		readBytes, err :=
			read(filepath.Join(path, dirEntry.Name()), entryPaths, pathsRead)
		if err != nil {
			return nil, err
		}
		// Note this is only resizing a slice of references to byte slices and not
		// resizing the byte slices themselves.
		allBytes = append(allBytes, readBytes...)
	}
	pathsRead[evaledPath] = struct{}{}
	return allBytes, nil
}

// isPathInPaths returns true if any of the paths in the paths slice is a prefix
// of the path string. It is assumed that all paths are absolute and do not
// contain any trailing slashes.
func isPathInPaths(path string, paths []string) bool {
	for _, p := range paths {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}

// SetStringsInFile overwrites the specified file with the changes specified by
// the changes map applied. The changes map maps keys to new values. Keys are of
// the form <key 0>.<key 1>...<key n>. Integers may be used as keys in cases
// where a specific node needs to be selected from a sequence. Individual
// changes are ignored without error if their key is not found or if their key
// is found not to address a scalar node. Importantly, all comments and style
// choices in the input bytes are preserved in the output.
func SetStringsInFile(file string, changes map[string]string) error {
	inBytes, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf(
			"error reading file %q: %w",
			file,
			err,
		)
	}
	outBytes, err := SetStringsInBytes(inBytes, changes)
	if err != nil {
		return fmt.Errorf("error mutating bytes: %w", err)
	}

	// This file should always exist already, so the permissions we choose here
	// don't really matter. (They only matter when this function call creates
	// the file, which it never will.) We went with 0600 just to appease the
	// gosec linter.
	if err = os.WriteFile(file, outBytes, 0600); err != nil {
		return fmt.Errorf(
			"error writing mutated bytes to file %q: %w",
			file,
			err,
		)
	}
	return nil
}

// SetStringsInBytes returns a copy of the provided bytes with the changes
// specified by the changes map applied. The changes map maps keys to new
// values. Keys are of the form <key 0>.<key 1>...<key n>. Integers may be used
// as keys in cases where a specific node needs to be selected from a sequence.
// Individual changes are ignored without error if their key is not found or
// if their key is found not to address a scalar node. Importantly, all comments
// and style choices in the input bytes are preserved in the output.
func SetStringsInBytes(
	inBytes []byte,
	changes map[string]string,
) ([]byte, error) {
	doc := &yaml.Node{}
	if err := yaml.Unmarshal(inBytes, doc); err != nil {
		return nil, fmt.Errorf("error unmarshaling input: %w", err)
	}

	type change struct {
		col   int
		value string
	}
	changesByLine := map[int]change{}
	for k, v := range changes {
		keyPath := strings.Split(k, ".")
		if found, line, col := findScalarNode(doc, keyPath); found {
			changesByLine[line] = change{
				col:   col,
				value: v,
			}
		}
	}

	outBuf := &bytes.Buffer{}

	scanner := bufio.NewScanner(bytes.NewBuffer(inBytes))
	scanner.Split(bufio.ScanLines)
	var line int
	for scanner.Scan() {
		const errMsg = "error writing to byte buffer"
		change, found := changesByLine[line]
		if !found {
			if _, err := outBuf.WriteString(scanner.Text()); err != nil {
				return nil, fmt.Errorf("%s: %w", errMsg, err)
			}
			if _, err := outBuf.WriteString("\n"); err != nil {
				return nil, fmt.Errorf("%s: %w", errMsg, err)
			}
		} else {
			unchanged := scanner.Text()[0:change.col]
			if _, err := outBuf.WriteString(unchanged); err != nil {
				return nil, fmt.Errorf("%s: %w", errMsg, err)
			}
			if !strings.HasSuffix(unchanged, " ") {
				if _, err := outBuf.WriteString(" "); err != nil {
					return nil, fmt.Errorf("%s: %w", errMsg, err)
				}
			}
			if _, err := outBuf.WriteString(change.value); err != nil {
				return nil, fmt.Errorf("%s: %w", errMsg, err)
			}
			if _, err := outBuf.WriteString("\n"); err != nil {
				return nil, fmt.Errorf("%s: %w", errMsg, err)
			}
		}
		line++
	}

	return outBuf.Bytes(), nil
}

func findScalarNode(node *yaml.Node, keyPath []string) (bool, int, int) {
	if len(keyPath) == 0 {
		if node.Kind == yaml.ScalarNode {
			return true, node.Line - 1, node.Column - 1
		}
		return false, 0, 0
	}
	switch node.Kind {
	case yaml.DocumentNode:
		return findScalarNode(node.Content[0], keyPath)
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			if node.Content[i].Value == keyPath[0] {
				return findScalarNode(node.Content[i+1], keyPath[1:])
			}
		}
	case yaml.SequenceNode:
		index, err := strconv.Atoi(keyPath[0])
		if err != nil {
			return false, 0, 0
		}
		return findScalarNode(node.Content[index], keyPath[1:])
	}
	return false, 0, 0
}
