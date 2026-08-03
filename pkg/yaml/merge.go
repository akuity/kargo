package yaml

import (
	"bytes"
	"fmt"
	"os"
	"path"

	yaml "go.yaml.in/yaml/v3"
)

// MergeFiles merges the specified list of YAML files into an output file at
// the specified path. All specified input files must exist. If a file already
// exists at the specified path for the output, it will be overwritten.
//
// Files are merged in order: the first file is the base and each subsequent
// file is layered over it. Mappings are merged recursively; sequences and
// scalars (including explicit nulls) from a later file fully replace the
// corresponding value from an earlier file; and keys present in only one
// file are carried through unchanged. Anchors, aliases, and merge keys are
// never resolved or expanded. They are copied through as-is.
func MergeFiles(inputPaths []string, outputPath string) error {
	if outputPath == "" {
		return fmt.Errorf("output path must not be empty")
	}

	var merged *yaml.Node
	for _, inputPath := range inputPaths {
		fileBytes, err := os.ReadFile(inputPath)
		if err != nil {
			return fmt.Errorf("error reading input file %s: %w", inputPath, err)
		}
		var doc yaml.Node
		if err = yaml.Unmarshal(fileBytes, &doc); err != nil {
			return fmt.Errorf("error reading input file %s: %w", inputPath, err)
		}
		if len(doc.Content) == 0 {
			continue // Empty file
		}
		merged = mergeNodes(merged, doc.Content[0])
	}

	if err := os.MkdirAll(path.Dir(outputPath), 0o700); err != nil {
		return fmt.Errorf("error writing merged YAML to %s: %w", outputPath, err)
	}
	if merged == nil {
		if err := os.WriteFile(outputPath, []byte{}, 0600); err != nil {
			return fmt.Errorf("error writing empty file to %s: %w", outputPath, err)
		}
		return nil
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(merged); err != nil {
		return fmt.Errorf("error marshaling merged YAML: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("error marshaling merged YAML: %w", err)
	}
	if err := os.WriteFile(outputPath, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("error writing merged YAML to %s: %w", outputPath, err)
	}

	return nil
}

// mergeNodes merges src over dst and returns the result. When both nodes are
// mappings, they are merged recursively, key by key. In every other case
// (including a kind mismatch, e.g. a sequence overriding a mapping) src is
// returned as-is: scalars and sequences fully replace the value from dst,
// and any node kind this function doesn't specifically interpret (notably
// yaml.AliasNode, which backs both aliases and the `<<` merge key) is
// preserved rather than silently dropped.
func mergeNodes(dst, src *yaml.Node) *yaml.Node {
	if dst == nil || dst.Kind != yaml.MappingNode || src.Kind != yaml.MappingNode {
		return src
	}

	merged := &yaml.Node{
		Kind:        yaml.MappingNode,
		Style:       dst.Style,
		Tag:         dst.Tag,
		HeadComment: dst.HeadComment,
		LineComment: dst.LineComment,
		FootComment: dst.FootComment,
	}

	srcValueByKey := make(map[string]*yaml.Node, len(src.Content)/2)
	for i := 0; i < len(src.Content); i += 2 {
		srcValueByKey[src.Content[i].Value] = src.Content[i+1]
	}

	mergedKeys := make(map[string]bool, len(dst.Content)/2)
	for i := 0; i < len(dst.Content); i += 2 {
		key, value := dst.Content[i], dst.Content[i+1]
		mergedKeys[key.Value] = true
		if srcValue, ok := srcValueByKey[key.Value]; ok {
			value = mergeNodes(value, srcValue)
		}
		merged.Content = append(merged.Content, key, value)
	}

	for i := 0; i < len(src.Content); i += 2 {
		key := src.Content[i]
		if mergedKeys[key.Value] {
			continue
		}
		merged.Content = append(merged.Content, key, src.Content[i+1])
	}

	return merged
}
