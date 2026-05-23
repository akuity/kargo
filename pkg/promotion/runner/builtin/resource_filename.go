package builtin

import (
	"fmt"
	"path/filepath"
	"strings"
)

// validateGeneratedResourceFilename rejects filenames derived from rendered
// Kubernetes resource metadata that contain path separators, parent-directory
// segments, or other components that could cause a write to escape the
// configured output directory when joined to it.
//
// The flat layout for helm-template and the per-resource directory output for
// kustomize-build both contract to write each resource as a single file at the
// top level of the configured output directory. Any generated name that is not
// a plain filename violates that contract -- whether or not it would actually
// escape the directory after joining.
func validateGeneratedResourceFilename(fileName string) error {
	if fileName == "" {
		return fmt.Errorf("generated resource filename is empty")
	}
	// "." and ".." are their own clean forms, so the Clean check below does not
	// reject them. They must be excluded explicitly: "." would target outPath
	// itself, and ".." would escape it.
	if fileName == "." || fileName == ".." ||
		filepath.IsAbs(fileName) ||
		strings.ContainsAny(fileName, `/\`) ||
		filepath.Clean(fileName) != fileName {
		return fmt.Errorf("unsafe generated resource filename %q", fileName)
	}
	return nil
}
