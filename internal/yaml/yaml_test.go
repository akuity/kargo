package yaml

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

func TestSetStringsInBytes(t *testing.T) {
	testCases := []struct {
		name       string
		inBytes    []byte
		changes    map[string]string
		assertions func(*testing.T, []byte, error)
	}{
		{
			name: "invalid YAML",
			// Note: This YAML is invalid because one line is indented with a tab
			inBytes: []byte(`
characters:
- name: Anakin
	affiliation: Light side
`),
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.ErrorContains(t, err, "error unmarshaling input")
				require.Nil(t, bytes)
			},
		},
		{
			name: "success",
			inBytes: []byte(`
characters:
- name: Anakin
  affiliation: Light side
`),
			changes: map[string]string{
				"characters.0.affiliation": "Dark side",
			},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]byte(`
characters:
- name: Anakin
  affiliation: Dark side
`),
					bytes,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			b, err := SetStringsInBytes(testCase.inBytes, testCase.changes)
			testCase.assertions(t, b, err)
		})
	}
}

func TestFindScalarNode(t *testing.T) {
	yamlBytes := []byte(`
characters:
  rebels:
  - name: Skywalker
`)
	testCases := []struct {
		name       string
		keyPath    string
		assertions func(t *testing.T, line, col int, err error)
	}{
		{
			name:    "node not found",
			keyPath: "characters.imperials",
			assertions: func(t *testing.T, _, _ int, err error) {
				require.ErrorContains(t, err, "key path not found")
			},
		},
		{
			name: "node not found due to error parsing int",
			// Really, this is a special case of a key that doesn't address a node,
			// because there is alpha input where numeric input would be expected.
			keyPath: "characters.rebels.first.name",
			assertions: func(t *testing.T, _, _ int, err error) {
				require.ErrorContains(t, err, "strconv.Atoi")
			},
		},
		{
			name:    "node found, but isn't a scalar node",
			keyPath: "characters.rebels",
			assertions: func(t *testing.T, _, _ int, err error) {
				require.ErrorContains(t, err, "key path does not address a scalar node")
			},
		},
		{
			name:    "success",
			keyPath: "characters.rebels.0.name",
			assertions: func(t *testing.T, line, col int, err error) {
				require.NoError(t, err)
				require.Equal(t, 3, line)
				require.Equal(t, 10, col)
			},
		},
	}
	doc := &yaml.Node{}
	err := yaml.Unmarshal(yamlBytes, doc)
	require.NoError(t, err)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			line, col, err := findScalarNode(doc, strings.Split(testCase.keyPath, "."))
			testCase.assertions(t, line, col, err)
		})
	}
}
