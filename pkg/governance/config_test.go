package governance

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"
)

func Test_action_UnmarshalYAML(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		expectKind string
		// expectConfigDecodes is the value the runner would decode the
		// captured config into. nil disables this check.
		expectConfigDecodes any
		expectErrContains   string
	}{
		{
			name:                "list-of-strings value",
			input:               `addLabels: [foo, bar]`,
			expectKind:          "addLabels",
			expectConfigDecodes: []string{"foo", "bar"},
		},
		{
			name:                "string value",
			input:               `comment: "hello"`,
			expectKind:          "comment",
			expectConfigDecodes: "hello",
		},
		{
			name:                "bool value",
			input:               `close: true`,
			expectKind:          "close",
			expectConfigDecodes: true,
		},
		{
			name:              "empty mapping is rejected",
			input:             `{}`,
			expectErrContains: "exactly one key, got 0",
		},
		{
			name:              "two-key mapping is rejected",
			input:             "addLabels: [foo]\nclose: true",
			expectErrContains: "exactly one key, got 2",
		},
		{
			name:              "non-mapping yaml is rejected",
			input:             `"not a map"`,
			expectErrContains: "error parsing action",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var a action
			err := yaml.Unmarshal([]byte(testCase.input), &a)
			if testCase.expectErrContains != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), testCase.expectErrContains)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.expectKind, a.kind)

			if testCase.expectConfigDecodes != nil {
				// The runner would decode action.config back into a typed
				// value. Verify the round-trip preserved the data.
				switch want := testCase.expectConfigDecodes.(type) {
				case []string:
					var got []string
					require.NoError(t, yaml.Unmarshal(a.config, &got))
					require.Equal(t, want, got)
				case string:
					var got string
					require.NoError(t, yaml.Unmarshal(a.config, &got))
					require.Equal(t, want, got)
				case bool:
					var got bool
					require.NoError(t, yaml.Unmarshal(a.config, &got))
					require.Equal(t, want, got)
				default:
					t.Fatalf("unhandled expectConfigDecodes type %T", want)
				}
			}
		})
	}
}

// Test_action_UnmarshalYAML_listFromConfig confirms that a real-world
// nested document round-trips correctly: parsing then re-marshaling the
// captured config bytes matches the original value structure.
func Test_action_UnmarshalYAML_listFromConfig(t *testing.T) {
	doc := `
- addLabels: [needs/area]
- comment: |
    Multi-line
    body here
- close: true
`
	var actions []action
	require.NoError(t, yaml.Unmarshal([]byte(doc), &actions))
	require.Len(t, actions, 3)

	require.Equal(t, "addLabels", actions[0].kind)
	require.Equal(t, "comment", actions[1].kind)
	require.Equal(t, "close", actions[2].kind)

	var labels []string
	require.NoError(t, yaml.Unmarshal(actions[0].config, &labels))
	require.Equal(t, []string{"needs/area"}, labels)

	var comment string
	require.NoError(t, yaml.Unmarshal(actions[1].config, &comment))
	require.True(
		t, strings.Contains(comment, "Multi-line"),
		"comment should preserve multi-line text, got %q", comment,
	)

	var closed bool
	require.NoError(t, yaml.Unmarshal(actions[2].config, &closed))
	require.True(t, closed)
}
