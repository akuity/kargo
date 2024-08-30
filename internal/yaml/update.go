package yaml

import (
	"fmt"
	"strings"

	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

// UpdateField updates the value of a field in a YAML document. The field is
// specified using a dot-separated path. If the field does not exist, it is
// created. The YAML node is modified in place, preserving comments and style.
//
// The value parameter can be any Go value that can be marshaled to YAML using
// the yaml package. This includes basic types (string, int, bool, etc.), maps,
// slices, and structs.
func UpdateField(node *yaml.Node, key string, value any) error {
	parts := strings.Split(key, pathSeparator)
	return updateNodeRecursively(node, parts, value)
}

// updateNodeRecursively traverses the YAML node structure and updates or adds
// the specified field. It recursively descends into mapping nodes and sequence
// nodes to find the target field. If the field does not exist, it is created.
func updateNodeRecursively(node *yaml.Node, parts []string, newValue any) error {
	if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
		return fmt.Errorf("empty field path")
	}

	currentPart := parts[0]
	remainingParts := parts[1:]

	switch node.Kind {
	case yaml.DocumentNode:
		return updateNodeRecursively(node.Content[0], parts, newValue)
	case yaml.MappingNode:
		return updateMappingNode(node, currentPart, remainingParts, newValue)
	case yaml.SequenceNode:
		return updateSequenceNode(node, currentPart, remainingParts, newValue)
	case yaml.ScalarNode:
		// If we have more parts to process, we need to convert the ScalarNode
		// to a MappingNode
		if len(remainingParts) > 0 {
			newNode := &yaml.Node{Kind: yaml.MappingNode}
			*node = *newNode
			return updateMappingNode(node, currentPart, remainingParts, newValue)
		}
		return updateNodeInPlace(node, newValue)
	case yaml.AliasNode:
		// For alias nodes, we update the target node
		return updateNodeRecursively(node.Alias, parts, newValue)
	default:
		return fmt.Errorf("unexpected node kind: %v", node.Kind)
	}
}

// updateMappingNode handles updating or adding a field within a YAML mapping
// node. It searches for the specified key and either updates the value
// directly, continues recursion, or adds a new field.
func updateMappingNode(node *yaml.Node, currentPart string, remainingParts []string, newValue any) error {
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == currentPart {
			if len(remainingParts) == 0 {
				return updateNodeInPlace(node.Content[i+1], newValue)
			}
			return updateNodeRecursively(node.Content[i+1], remainingParts, newValue)
		}
	}
	// Key not found, add new field.
	return addNewField(node, currentPart, remainingParts, newValue)
}

// addNewField adds a new field to a mapping node, creating nested structures
// as needed.
func addNewField(node *yaml.Node, currentPart string, remainingParts []string, newValue any) error {
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: currentPart}
	var valueNode *yaml.Node

	if len(remainingParts) == 0 {
		valueNode = &yaml.Node{}
		if err := valueNode.Encode(newValue); err != nil {
			return fmt.Errorf("error encoding new value: %v", err)
		}
	} else {
		valueNode = &yaml.Node{Kind: yaml.MappingNode}
		if err := updateNodeRecursively(valueNode, remainingParts, newValue); err != nil {
			return err
		}
	}

	node.Content = append(node.Content, keyNode, valueNode)
	return nil
}

// updateSequenceNode handles updating or adding an item within a YAML sequence
// node. It parses the index from the current part and updates the corresponding
// sequence item or adds a new one.
func updateSequenceNode(node *yaml.Node, currentPart string, remainingParts []string, newValue any) error {
	index, err := parseIndex(currentPart)
	if err != nil {
		return err
	}

	if index < 0 {
		return fmt.Errorf("invalid negative index: %d", index)
	}

	if index >= len(node.Content) {
		// Add new node at the end of the sequence
		var newNode *yaml.Node
		if len(remainingParts) > 0 {
			newNode = &yaml.Node{Kind: yaml.MappingNode}
		} else {
			newNode = &yaml.Node{Kind: yaml.ScalarNode}
		}
		node.Content = append(node.Content, newNode)
		index = len(node.Content) - 1 // Set index to the newly added item
	}

	if len(remainingParts) == 0 {
		return updateNodeInPlace(node.Content[index], newValue)
	}
	return updateNodeRecursively(node.Content[index], remainingParts, newValue)
}

// updateNodeInPlace updates a YAML node with a new value while preserving
// comments and style.
func updateNodeInPlace(node *yaml.Node, newValue any) error {
	newNode := &yaml.Node{}
	if err := newNode.Encode(newValue); err != nil {
		return fmt.Errorf("error encoding new value: %v", err)
	}

	preserveComments(node, newNode)
	newNode.Style = node.Style
	*node = *newNode

	return nil
}

// preserveComments copies comments from the old node to the new node,
// recursively handling mapping and sequence nodes.
func preserveComments(oldNode, newNode *yaml.Node) {
	newNode.HeadComment = oldNode.HeadComment
	newNode.LineComment = oldNode.LineComment
	newNode.FootComment = oldNode.FootComment

	if oldNode.Kind == yaml.MappingNode && newNode.Kind == yaml.MappingNode {
		oldMap := make(map[string]*yaml.Node)
		for i := 0; i < len(oldNode.Content); i += 2 {
			oldMap[oldNode.Content[i].Value] = oldNode.Content[i+1]
		}

		for i := 0; i < len(newNode.Content); i += 2 {
			key := newNode.Content[i].Value
			if oldValue, exists := oldMap[key]; exists {
				preserveComments(oldValue, newNode.Content[i+1])
			}
		}
	} else if oldNode.Kind == yaml.SequenceNode && newNode.Kind == yaml.SequenceNode {
		for i := 0; i < len(newNode.Content) && i < len(oldNode.Content); i++ {
			preserveComments(oldNode.Content[i], newNode.Content[i])
		}
	}
}

// parseIndex extracts and returns the numeric index from a string in the
// format "[n]".
func parseIndex(s string) (int, error) {
	var index int
	if _, err := fmt.Sscanf(s, "[%d]", &index); err != nil {
		return 0, fmt.Errorf("invalid index format: %s", s)
	}
	return index, nil
}
