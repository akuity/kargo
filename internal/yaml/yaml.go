package yaml

import (
	"bufio"
	"bytes"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

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
		return errors.Wrapf(
			err,
			"error reading file %q",
			file,
		)
	}
	outBytes, err := SetStringsInBytes(inBytes, changes)
	if err != nil {
		return errors.Wrap(err, "error mutating bytes")
	}
	return errors.Wrapf(
		// This file should always exist already, so the permissions we choose here
		// don't really matter. (They only matter when this function call creates
		// the file, which it never will.) We went with 0600 just to appease the
		// gosec linter.
		os.WriteFile(file, outBytes, 0600),
		"error writing mutated bytes to file %q",
		file,
	)
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
		return nil, errors.Wrap(err, "error unmarshaling input")
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
				return nil, errors.Wrap(err, errMsg)
			}
			if _, err := outBuf.WriteString("\n"); err != nil {
				return nil, errors.Wrap(err, errMsg)
			}
		} else {
			unchanged := scanner.Text()[0:change.col]
			if _, err := outBuf.WriteString(unchanged); err != nil {
				return nil, errors.Wrap(err, errMsg)
			}
			if !strings.HasSuffix(unchanged, " ") {
				if _, err := outBuf.WriteString(" "); err != nil {
					return nil, errors.Wrap(err, errMsg)
				}
			}
			if _, err := outBuf.WriteString(change.value); err != nil {
				return nil, errors.Wrap(err, errMsg)
			}
			if _, err := outBuf.WriteString("\n"); err != nil {
				return nil, errors.Wrap(err, errMsg)
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
