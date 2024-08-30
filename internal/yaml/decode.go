package yaml

import (
	"fmt"
	"strings"

	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

const pathSeparator = "."

// DecodeField retrieves the value at the specified path in the YAML document
// and decodes it into the provided value. The path is specified using a
// dot-separated string, similar to the UpdateField function.
func DecodeField(node *yaml.Node, path string, out any) error {
	parts := strings.Split(path, pathSeparator)
	targetNode, err := findNode(node, parts)
	if err != nil {
		return err
	}
	return targetNode.Decode(out)
}

// findNode traverses the YAML structure to find the node at the specified path.
func findNode(node *yaml.Node, parts []string) (*yaml.Node, error) {
	if len(parts) == 0 {
		return node, nil
	}

	currentPart := parts[0]
	remainingParts := parts[1:]

	switch node.Kind {
	case yaml.DocumentNode:
		return findNode(node.Content[0], parts)
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			if node.Content[i].Value == currentPart {
				return findNode(node.Content[i+1], remainingParts)
			}
		}
		return nil, fmt.Errorf("field '%s' not found", currentPart)
	case yaml.SequenceNode:
		index, err := parseIndex(currentPart)
		if err != nil {
			return nil, err
		}
		if index < 0 || index >= len(node.Content) {
			return nil, fmt.Errorf("index out of range: %d", index)
		}
		return findNode(node.Content[index], remainingParts)
	default:
		if len(parts) > 0 {
			return nil, fmt.Errorf("cannot access nested field on scalar node")
		}
		return node, nil
	}
}
