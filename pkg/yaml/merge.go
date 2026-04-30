package yaml

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge2"
)

// MergeFiles merges the specified list of YAML files into an output file at
// the specified path. All specified input files must exist. If a file already
// exists at the specified path for the output, it will be overwritten.
func MergeFiles(inputPaths []string, outputPath string) error {
	if outputPath == "" {
		return fmt.Errorf("output path must not be empty")
	}

	var mergeTarget *kyaml.RNode
	for _, inputPath := range inputPaths {
		patchNode, err := kyaml.ReadFile(inputPath)
		if err != nil {
			if errors.Is(err, io.EOF) {
				continue
			}
			return fmt.Errorf("error reading input file %s: %w", inputPath, err)
		}
		if mergeTarget == nil {
			mergeTarget = patchNode
			continue
		}
		if mergeTarget, err = merge2.Merge(
			patchNode, mergeTarget, kyaml.MergeOptions{},
		); err != nil {
			return fmt.Errorf("error merging in file %s: %w", inputPath, err)
		}
	}

	if err := os.MkdirAll(path.Dir(outputPath), 0o700); err != nil {
		return fmt.Errorf("error writing merged YAML to %s: %w", outputPath, err)
	}
	if mergeTarget == nil {
		if err := os.WriteFile(outputPath, []byte{}, 0600); err != nil {
			return fmt.Errorf("error writing empty file to %s: %w", outputPath, err)
		}
		return nil
	}
	if err := kyaml.WriteFile(mergeTarget, outputPath); err != nil {
		return fmt.Errorf("error writing merged YAML to %s: %w", outputPath, err)
	}

	return nil
}
