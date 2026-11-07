package chart

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// maxIndexLineSize bounds the length of a single line in a YAML repository
// index.
const maxIndexLineSize = 16 << 20

var errNotAnIndex = errors.New("response is not a repository index")

type indexEntry struct {
	Version string `json:"version" yaml:"version"`
}

// versionsFromIndex returns the version strings that a classic Helm
// repository index lists for the named chart, or nil if the index does not
// list the chart.
//
// Repository indices for popular repositories are tens of megabytes covering
// hundreds of charts, and unmarshaling one whole costs an order of magnitude
// more memory than its size. Only the named chart's entries are decoded here.
// The rest of the index is scanned without being parsed, so a syntax error
// outside those entries goes undetected.
func versionsFromIndex(r io.Reader, chartName string) ([]string, error) {
	br := bufio.NewReaderSize(r, 64<<10)
	first, err := skipLeadingWhitespace(br)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, err
	}
	var entries []indexEntry
	if first == '{' {
		entries, err = entriesFromJSONIndex(br, chartName)
	} else {
		entries, err = entriesFromYAMLIndex(br, chartName)
	}
	if err != nil || entries == nil {
		return nil, err
	}
	versions := make([]string, len(entries))
	for i, entry := range entries {
		versions[i] = entry.Version
	}
	return versions, nil
}

// skipLeadingWhitespace discards whitespace from the start of br and returns
// the first byte that follows it without consuming that byte.
func skipLeadingWhitespace(br *bufio.Reader) (byte, error) {
	for {
		b, err := br.ReadByte()
		if err != nil {
			return 0, err
		}
		switch b {
		case ' ', '\t', '\r', '\n':
			continue
		}
		return b, br.UnreadByte()
	}
}

// entriesFromJSONIndex handles indices written as JSON, as `helm repo index
// --json` does.
func entriesFromJSONIndex(r io.Reader, chartName string) ([]indexEntry, error) {
	dec := json.NewDecoder(r)
	if err := expectJSONDelim(dec, '{'); err != nil {
		return nil, err
	}
	for dec.More() {
		key, err := jsonKey(dec)
		if err != nil {
			return nil, err
		}
		if key != "entries" {
			if err = skipJSONValue(dec); err != nil {
				return nil, err
			}
			continue
		}
		if err = expectJSONDelim(dec, '{'); err != nil {
			return nil, err
		}
		for dec.More() {
			if key, err = jsonKey(dec); err != nil {
				return nil, err
			}
			if key != chartName {
				if err = skipJSONValue(dec); err != nil {
					return nil, err
				}
				continue
			}
			var entries []indexEntry
			if err = dec.Decode(&entries); err != nil {
				return nil, fmt.Errorf("error decoding entries for chart %q: %w", chartName, err)
			}
			return entries, nil
		}
		return nil, nil
	}
	return nil, nil
}

func expectJSONDelim(dec *json.Decoder, want json.Delim) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if d, ok := tok.(json.Delim); !ok || d != want {
		return fmt.Errorf("expected %q, found %v", want, tok)
	}
	return nil
}

func jsonKey(dec *json.Decoder) (string, error) {
	tok, err := dec.Token()
	if err != nil {
		return "", err
	}
	key, ok := tok.(string)
	if !ok {
		return "", fmt.Errorf("expected object key, found %v", tok)
	}
	return key, nil
}

func skipJSONValue(dec *json.Decoder) error {
	var skipped json.RawMessage
	return dec.Decode(&skipped)
}

// entriesFromYAMLIndex handles indices in the block-style YAML that Helm
// writes. It finds the top-level entries mapping, copies out the lines of the
// named chart's key, and unmarshals only those.
func entriesFromYAMLIndex(r io.Reader, chartName string) ([]indexEntry, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64<<10), maxIndexLineSize)

	sawTopLevelKey := false
	inEntries := false
	sawChartKey := false
	childIndent := -1
	var captured []byte
	for scanner.Scan() {
		line := bytes.TrimSuffix(scanner.Bytes(), []byte("\r"))
		content := bytes.TrimLeft(line, " ")
		indent := len(line) - len(content)
		if len(content) == 0 || content[0] == '#' {
			if captured != nil {
				captured = appendDedented(captured, line, childIndent)
			}
			continue
		}

		if captured != nil {
			if indent > childIndent || (indent == childIndent && isSequenceItem(content)) {
				captured = appendDedented(captured, line, childIndent)
				continue
			}
			break
		}

		if !inEntries {
			if indent != 0 {
				continue
			}
			key, rest, ok := yamlMappingKey(content)
			if !ok {
				continue
			}
			sawTopLevelKey = true
			if key != "entries" {
				continue
			}
			if isEmptyFlowValue(rest) {
				// Helm writes "entries: {}" for a repository with no charts.
				return nil, nil
			}
			if len(rest) != 0 && rest[0] != '#' {
				return nil, errors.New(
					"repository index entries are not in block style; this is not supported",
				)
			}
			inEntries = true
			continue
		}

		if indent == 0 {
			// The entries mapping has ended without listing the chart.
			return nil, nil
		}
		if childIndent == -1 {
			childIndent = indent
		}
		if indent > childIndent {
			continue
		}
		if indent < childIndent {
			return nil, errors.New("repository index entries are malformed")
		}
		if isSequenceItem(content) {
			// Helm does not indent a chart's sequence of entries beneath its key.
			if !sawChartKey {
				return nil, errors.New("repository index entries are malformed")
			}
			continue
		}
		sawChartKey = true
		if key, _, ok := yamlMappingKey(content); ok && key == chartName {
			captured = appendDedented(make([]byte, 0, 64<<10), line, childIndent)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if !sawTopLevelKey {
		return nil, errNotAnIndex
	}
	if captured == nil {
		return nil, nil
	}

	chart := map[string][]indexEntry{}
	if err := yaml.Unmarshal(captured, &chart); err != nil {
		return nil, fmt.Errorf("error unmarshaling entries for chart %q: %w", chartName, err)
	}
	return chart[chartName], nil
}

func isSequenceItem(content []byte) bool {
	return len(content) == 1 && content[0] == '-' ||
		len(content) > 1 && content[0] == '-' && (content[1] == ' ' || content[1] == '\t')
}

// appendDedented appends line to buf, without its first indent spaces, and a
// newline.
func appendDedented(buf, line []byte, indent int) []byte {
	if len(line) >= indent && len(bytes.TrimLeft(line[:indent], " ")) == 0 {
		line = line[indent:]
	}
	buf = append(buf, line...)
	return append(buf, '\n')
}

// yamlMappingKey extracts the key from a line holding a simple, plain or
// quoted, YAML mapping key, and returns what follows the colon with
// surrounding spaces removed.
func yamlMappingKey(content []byte) (string, []byte, bool) {
	var key []byte
	var rest []byte
	switch content[0] {
	case '"', '\'':
		end := bytes.IndexByte(content[1:], content[0])
		if end == -1 {
			return "", nil, false
		}
		key = content[1 : end+1]
		rest = bytes.TrimLeft(content[end+2:], " ")
		if len(rest) == 0 || rest[0] != ':' {
			return "", nil, false
		}
		rest = rest[1:]
	default:
		colon := -1
		for i, b := range content {
			if b == ':' && (i == len(content)-1 || content[i+1] == ' ' || content[i+1] == '\t') {
				colon = i
				break
			}
		}
		if colon == -1 {
			return "", nil, false
		}
		key = bytes.TrimRight(content[:colon], " ")
		rest = content[colon+1:]
	}
	return string(key), bytes.TrimSpace(rest), true
}

// isEmptyFlowValue reports whether value, a mapping value with surrounding
// spaces removed, is an empty mapping or null.
func isEmptyFlowValue(value []byte) bool {
	if i := bytes.IndexByte(value, '#'); i != -1 {
		value = bytes.TrimSpace(value[:i])
	}
	switch string(value) {
	case "{}", "~", "null", "Null", "NULL":
		return true
	}
	return false
}
