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

// The maximum size of the input buffer for processing YAML files. Given that the max size of a
// Kubernetes manifest (the most common thing we're dealing with) is 1 MB, this number is chosen to
// assume that the max value of any given key would be just shy of that number (imagine a config map
// with a single large value which is a config file for something in the container). Please note
// that we do not allocate a buffer of this size up front; this is just the maximum size that we
// will allow when processing input.
const maxBufferSize = 1024 * 1024 // 1 MB

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
		col   int
		value any
	}
	changesByLine := map[int]change{}
	for _, update := range updates {
		keyPath := strings.Split(update.Key, ".")
		line, col, err := findScalarNode(doc, keyPath)
		if err != nil {
			return nil, fmt.Errorf("error finding key %s: %w", update.Key, err)
		}
		changesByLine[line] = change{
			col:   col,
			value: update.Value,
		}
	}

	outBuf := &bytes.Buffer{}

	scanner := bufio.NewScanner(bytes.NewBuffer(inBytes))
	scanner.Split(bufio.ScanLines)
	// Create an initial buffer of 100B which should be plenty for most lines. This means we will
	// likely avoid an allocation on the first scan. If we encounter a line that exceeds this size,
	// the buffer will grow up to maxBufferSize.
	var buf = make([]byte, 0, 100)
	scanner.Buffer(buf, maxBufferSize)
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
	if scanner.Err() != nil {
		return nil, fmt.Errorf("error scanning input bytes: %w", scanner.Err())
	}

	return outBuf.Bytes(), nil
}

func findScalarNode(node *yaml.Node, keyPath []string) (int, int, error) {
	if len(keyPath) == 0 {
		if node.Kind == yaml.ScalarNode {
			return node.Line - 1, node.Column - 1, nil
		}
		return 0, 0, fmt.Errorf("key path does not address a scalar node")
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
			return 0, 0, err
		}
		return findScalarNode(node.Content[index], keyPath[1:])
	}
	return 0, 0, fmt.Errorf("key path not found")
}
