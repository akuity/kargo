package yaml

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Update represents a discrete update to be made to a YAML document.
type Update struct {
	// Key is the dot-separated path to the field to update.
	Key string
	// Value is the new value to set for the field.
	Value any
}

// SetValuesInFile overwrites the specified file with the changes specified by
// the the list of Updates. Keys are of the form <key 0>.<key 1>...<key n>.
// Integers may be used as keys in cases where a specific node needs to be
// selected from a sequence. An error is returned for any attempted update to a
// key that does not exist or does not address a scalar node. Importantly, all
// comments and style choices in the input bytes are preserved in the output.
func SetValuesInFile(file string, updates []Update) error {
	inBytes, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf(
			"error reading file %q: %w",
			file,
			err,
		)
	}
	outBytes, err := SetValuesInBytes(inBytes, updates)
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

// SetValuesInBytes returns a copy of the provided bytes with the changes
// specified by Updates applied. Keys are of the form <key 0>.<key 1>...<key n>.
// Integers may be used as keys in cases where a specific node needs to be
// selected from a sequence. An error is returned for any attempted update to a
// key that does not exist or does not address a scalar node. Importantly, all
// comments and style choices in the input bytes are preserved in the output.
func SetValuesInBytes(inBytes []byte, updates []Update) ([]byte, error) {
	doc := &yaml.Node{}
	if err := yaml.Unmarshal(inBytes, doc); err != nil {
		return nil, fmt.Errorf("error unmarshaling input: %w", err)
	}

	type change struct {
		col     int
		value   any
		key     string
		newitem bool
	}
	changesByLine := map[int]change{}
	for _, update := range updates {
		keyPath := strings.Split(update.Key, ".")
		line, col, newitem, err := findScalarNode(doc, keyPath)
		if err != nil {
			return nil, fmt.Errorf("error finding key %s: %w", update.Key, err)
		}

		changesByLine[line] = change{
			col:     col,
			key:     keyPath[len(keyPath)-1],
			value:   update.Value,
			newitem: newitem,
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
			if err := writeLineToBuffer(outBuf, scanner.Text(), errMsg); err != nil {
				return nil, err
			}
		} else {
			line := scanner.Text()

			if change.newitem {
				if err := writeLineToBuffer(outBuf, scanner.Text(), errMsg); err != nil {
					return nil, err
				}

				// generate new line with prefilled key
				line = line[0:change.col] + change.key + ": "
				change.col = len(line)
			}

			unchanged := line[0:change.col]
			if _, err := outBuf.WriteString(unchanged); err != nil {
				return nil, fmt.Errorf("%s: %w", errMsg, err)
			}
			if !strings.HasSuffix(unchanged, " ") {
				if _, err := outBuf.WriteString(" "); err != nil {
					return nil, fmt.Errorf("%s: %w", errMsg, err)
				}
			}
			if _, err := fmt.Fprintf(outBuf,
				"%v", QuoteIfNecessary(change.value)); err != nil {
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

func writeLineToBuffer(buf *bytes.Buffer, text, errMsg string) error {
	if _, err := buf.WriteString(text); err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	if _, err := buf.WriteString("\n"); err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	return nil
}

func findScalarNode(node *yaml.Node, keyPath []string) (int, int, bool, error) {
	if len(keyPath) == 0 {
		if node.Kind == yaml.ScalarNode {
			return node.Line - 1, node.Column - 1, false, nil
		}
		return 0, 0, false, fmt.Errorf("key path does not address a scalar node")
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
		// key not found. If this is a leaf mapping node, return last node position.
		if len(keyPath) == 1 && len(node.Content) >= 2 {
			last_node := node.Content[len(node.Content)-2]
			return last_node.Line - 1, last_node.Column - 1, true, nil
		}
	case yaml.SequenceNode:
		index, err := strconv.Atoi(keyPath[0])
		if err != nil {
			return 0, 0, false, err
		}
		return findScalarNode(node.Content[index], keyPath[1:])
	}
	return 0, 0, false, fmt.Errorf("key path not found")
}
