package yaml

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestSetValuesInBytes(t *testing.T) {
	testCases := []struct {
		name       string
		inBytes    []byte
		updates    []Update
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
			name: "no extra quotes around true number",
			inBytes: []byte(`
characters:
- name: Arthur Dent
  answer: idk
`),
			updates: []Update{
				{
					Key:   "characters.0.answer",
					Value: 42,
				},
			},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]byte(`
characters:
- name: Arthur Dent
  answer: 42
`),
					bytes,
				)
			},
		},
		{
			name: "extra quotes around string containing a valid number",
			inBytes: []byte(`
characters:
- name: Arthur Dent
  answer: idk
`),
			updates: []Update{
				{
					Key:   "characters.0.answer",
					Value: "42",
				},
			},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]byte(`
characters:
- name: Arthur Dent
  answer: "42"
`),
					bytes,
				)
			},
		},
		{
			name: "success",
			inBytes: []byte(`
characters:
- name: Anakin
  affiliation: Light side
`),
			updates: []Update{
				{
					Key:   "characters.0.affiliation",
					Value: "Dark side",
				},
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
		{
			name: "really long lines still work",
			// nolint:lll
			inBytes: []byte(`
characters:
- name: Anakin
  affiliation: Light side
  temptation: "` + strings.Repeat("Did you ever hear the tragedy of Darth Plagueis The Wise? I thought not. It's not a story the Jedi would tell you. It's a Sith legend. Darth Plagueis was a Dark Lord of the Sith, so powerful and so wise he could use the Force to influence the midichlorians to create life...He had such a knowledge of the dark side that he could even keep the ones he cared about from dying. The dark side of the Force is a pathway to many abilities some consider to be unnatural. He became so powerful...the only thing he was afraid of was losing his power, which eventually, of course, he did. Unfortunately, he taught his apprentice everything he knew, then his apprentice killed him in his sleep. Ironic. He could save others from death, but not himself.", 1000) + `"
`),
			updates: []Update{
				{
					Key:   "characters.0.affiliation",
					Value: "Dark side",
				},
			},
			assertions: func(t *testing.T, bytes []byte, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					// nolint:lll
					[]byte(`
characters:
- name: Anakin
  affiliation: Dark side
  temptation: "`+strings.Repeat("Did you ever hear the tragedy of Darth Plagueis The Wise? I thought not. It's not a story the Jedi would tell you. It's a Sith legend. Darth Plagueis was a Dark Lord of the Sith, so powerful and so wise he could use the Force to influence the midichlorians to create life...He had such a knowledge of the dark side that he could even keep the ones he cared about from dying. The dark side of the Force is a pathway to many abilities some consider to be unnatural. He became so powerful...the only thing he was afraid of was losing his power, which eventually, of course, he did. Unfortunately, he taught his apprentice everything he knew, then his apprentice killed him in his sleep. Ironic. He could save others from death, but not himself.", 1000)+`"
`),
					bytes,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			b, err := SetValuesInBytes(testCase.inBytes, testCase.updates)
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
