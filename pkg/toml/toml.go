package toml

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	tomlv2 "github.com/pelletier/go-toml/v2"
	// Note: This is the only go-toml API that exposes byte offsets, which we need
	// for in-place edits. It is explicitly not a stable API, so go-toml version
	// bumps may require changes here. Test coverage in toml_test.go exercises
	// every node type and should catch most unanticipated breaking changes.
	"github.com/pelletier/go-toml/v2/unstable"

	"github.com/akuity/kargo/pkg/sjson"
)

// Update represents a discrete update to be made to a TOML document.
type Update struct {
	// Key is the dot-separated path to the field to update.
	Key string
	// Value is the new value to set for the field.
	Value any
}

type nodeRef struct {
	kind unstable.Kind
	raw  unstable.Range
}

type nodeIndex struct {
	nodes   map[string]nodeRef
	scalars map[string]nodeRef
}

type change struct {
	offset      int
	length      int
	replacement []byte
}

// SetValuesInFile overwrites the specified file with the changes specified by
// the list of Updates. Keys are of the form <key 0>.<key 1>...<key n>.
// Integers may be used as keys where a specific array element needs to be
// selected. An error is returned for any attempted update to a key that does
// not exist or does not address a scalar node. Untouched bytes are preserved.
func SetValuesInFile(file string, updates []Update) error {
	inBytes, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("error reading file %q: %w", file, err)
	}
	outBytes, err := SetValuesInBytes(inBytes, updates)
	if err != nil {
		return fmt.Errorf("error mutating bytes: %w", err)
	}
	if err = os.WriteFile(file, outBytes, 0o600); err != nil {
		return fmt.Errorf("error writing mutated bytes to file %q: %w", file, err)
	}
	return nil
}

// SetValuesInBytes returns a copy of the provided bytes with the changes
// specified by Updates applied. Keys are of the form <key 0>.<key 1>...<key n>.
// Integers may be used as keys where a specific array element needs to be
// selected. An error is returned for any attempted update to a key that does
// not exist or does not address a scalar node. Untouched bytes are preserved.
func SetValuesInBytes(inBytes []byte, updates []Update) ([]byte, error) {
	index, err := indexNodes(inBytes)
	if err != nil {
		return nil, err
	}

	changesByPath := map[string]change{}
	for _, update := range updates {
		keyPath, err := sjson.SplitKey(update.Key)
		if err != nil {
			return nil, fmt.Errorf("error splitting key %s: %w", update.Key, err)
		}
		encodedPath := encodePath(keyPath)

		node, ok := index.scalars[encodedPath]
		if !ok {
			if ref, found := index.nodes[encodedPath]; found {
				return nil, fmt.Errorf(
					"error finding key %s: key path addresses %s instead of a scalar node",
					update.Key,
					ref.kind,
				)
			}
			return nil, fmt.Errorf("error finding key %s: key path not found", update.Key)
		}

		replacement, err := FormatValue(update.Value)
		if err != nil {
			return nil, fmt.Errorf("error formatting value for key %s: %w", update.Key, err)
		}

		changesByPath[encodedPath] = change{
			offset:      int(node.raw.Offset),
			length:      int(node.raw.Length),
			replacement: replacement,
		}
	}

	outBytes := bytes.Clone(inBytes)
	changes := make([]change, 0, len(changesByPath))
	for _, ch := range changesByPath {
		changes = append(changes, ch)
	}
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].offset > changes[j].offset
	})

	for _, ch := range changes {
		end := ch.offset + ch.length
		if ch.offset < 0 || end > len(outBytes) {
			return nil, fmt.Errorf("invalid replacement range [%d:%d]", ch.offset, end)
		}

		updatedBytes := make([]byte, 0, len(outBytes)-ch.length+len(ch.replacement))
		updatedBytes = append(updatedBytes, outBytes[:ch.offset]...)
		updatedBytes = append(updatedBytes, ch.replacement...)
		updatedBytes = append(updatedBytes, outBytes[end:]...)
		outBytes = updatedBytes
	}

	return outBytes, nil
}

// FormatValue returns a TOML-encoded scalar value.
func FormatValue(value any) ([]byte, error) {
	if !isValidScalar(value) {
		return nil, fmt.Errorf("value is not a TOML scalar type")
	}

	encodedValue, err := tomlv2.Marshal(map[string]any{"value": value})
	if err != nil {
		return nil, fmt.Errorf("error marshaling TOML value: %w", err)
	}

	const prefix = "value = "
	encodedValue = bytes.TrimSuffix(encodedValue, []byte("\n"))
	if !bytes.HasPrefix(encodedValue, []byte(prefix)) {
		return nil, fmt.Errorf("unexpected encoded TOML value %q", encodedValue)
	}

	return bytes.Clone(encodedValue[len(prefix):]), nil
}

// FormatValueString returns a TOML-encoded scalar value as a string.
func FormatValueString(value any) string {
	formatted, err := FormatValue(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(formatted)
}

func indexNodes(inBytes []byte) (*nodeIndex, error) {
	idx := &nodeIndex{
		nodes:   map[string]nodeRef{},
		scalars: map[string]nodeRef{},
	}

	parser := unstable.Parser{KeepComments: true}
	parser.Reset(inBytes)

	currentPath := []string{}
	arrayTableCounts := map[string]int{}

	for parser.NextExpression() {
		expr := parser.Expression()
		switch expr.Kind {
		case unstable.Comment:
			continue
		case unstable.Table:
			currentPath = iteratorParts(expr.Key())
			idx.register(currentPath, expr)
		case unstable.ArrayTable:
			basePath := iteratorParts(expr.Key())
			idx.register(basePath, expr)
			countKey := encodePath(basePath)
			currentIndex := arrayTableCounts[countKey]
			arrayTableCounts[countKey] = currentIndex + 1
			currentPath = append(copyPath(basePath), strconv.Itoa(currentIndex))
			idx.register(currentPath, expr)
		case unstable.KeyValue:
			fullPath := append(copyPath(currentPath), iteratorParts(expr.Key())...)
			idx.indexValue(&parser, expr.Value(), fullPath)
		default:
			return nil, fmt.Errorf(
				"error parsing input: unsupported expression kind %s",
				expr.Kind,
			)
		}
	}

	if err := parser.Error(); err != nil {
		return nil, fmt.Errorf("error parsing input: %w", err)
	}

	return idx, nil
}

func (n *nodeIndex) indexValue(
	parser *unstable.Parser,
	node *unstable.Node,
	path []string,
) {
	if node == nil || !node.Valid() {
		return
	}

	n.register(path, node)

	switch node.Kind {
	case unstable.Array:
		it := node.Children()
		index := 0
		for it.Next() {
			n.indexValue(parser, it.Node(), append(copyPath(path), strconv.Itoa(index)))
			index++
		}
	case unstable.InlineTable:
		it := node.Children()
		for it.Next() {
			child := it.Node()
			if child.Kind != unstable.KeyValue {
				continue
			}
			childPath := append(copyPath(path), iteratorParts(child.Key())...)
			n.indexValue(parser, child.Value(), childPath)
		}
	case unstable.String,
		unstable.Bool,
		unstable.Float,
		unstable.Integer,
		unstable.LocalDate,
		unstable.LocalTime,
		unstable.LocalDateTime,
		unstable.DateTime:
		raw := node.Raw
		if raw.Length == 0 && len(node.Data) > 0 {
			raw = parser.Range(node.Data)
		}
		n.scalars[encodePath(path)] = nodeRef{kind: node.Kind, raw: raw}
	}
}

func (n *nodeIndex) register(path []string, node *unstable.Node) {
	n.nodes[encodePath(path)] = nodeRef{kind: node.Kind, raw: node.Raw}
}

func encodePath(path []string) string {
	var builder strings.Builder
	for _, part := range path {
		_, _ = fmt.Fprintf(&builder, "%d:%s|", len(part), part)
	}
	return builder.String()
}

func copyPath(path []string) []string {
	return append([]string(nil), path...)
}

func iteratorParts(iterator unstable.Iterator) []string {
	parts := []string{}
	for iterator.Next() {
		parts = append(parts, string(iterator.Node().Data))
	}
	return parts
}

func isValidScalar(value any) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64,
		string, bool,
		time.Time,
		tomlv2.LocalDate,
		tomlv2.LocalTime,
		tomlv2.LocalDateTime:
		return true
	default:
		return false
	}
}
