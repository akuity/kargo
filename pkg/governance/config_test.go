package governance

import (
	"os"
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

func Test_config_onOpened(t *testing.T) {
	doc := `
issues:
  onOpened:
    actions:
    - comment: hi
pullRequests:
  onOpened:
    actions:
    - comment: hi
`
	var cfg config
	require.NoError(t, yaml.Unmarshal([]byte(doc), &cfg))

	require.NotNil(t, cfg.Issues)
	require.NotNil(t, cfg.Issues.OnOpened)
	require.Len(t, cfg.Issues.OnOpened.Actions, 1)
	require.Equal(t, "comment", cfg.Issues.OnOpened.Actions[0].kind)

	require.NotNil(t, cfg.PullRequests)
	require.NotNil(t, cfg.PullRequests.OnOpened)
	require.Len(t, cfg.PullRequests.OnOpened.Actions, 1)
	require.Equal(t, "comment", cfg.PullRequests.OnOpened.Actions[0].kind)
}

// Test_repoGovernanceConfig guards this repository's own
// .github/governance.yaml. The bot parses config leniently, so a mistyped
// key or action kind would otherwise go unnoticed until it misbehaved in
// production.
func Test_repoGovernanceConfig(t *testing.T) {
	data, err := os.ReadFile("../../.github/governance.yaml")
	require.NoError(t, err)

	var cfg config
	require.NoError(
		t,
		yaml.UnmarshalWithOptions(data, &cfg, yaml.DisallowUnknownField()),
	)

	lists := map[string][]action{}
	addCommands := func(prefix string, cmds map[string]commandDef) {
		for name, cmd := range cmds {
			lists[prefix+"/"+name] = cmd.Actions
		}
	}
	if i := cfg.Issues; i != nil {
		if i.OnOpened != nil {
			lists["issues.onOpened"] = i.OnOpened.Actions
		}
		addCommands("issues.slashCommands", i.SlashCommands)
	}
	if p := cfg.PullRequests; p != nil {
		if p.OnOpened != nil {
			lists["pullRequests.onOpened"] = p.OnOpened.Actions
		}
		if p.OnNoLinkedIssue != nil {
			lists["pullRequests.onNoLinkedIssue"] = p.OnNoLinkedIssue.Actions
		}
		if p.OnBlockedIssue != nil {
			lists["pullRequests.onBlockedIssue"] = p.OnBlockedIssue.Actions
		}
		if p.OnPass != nil {
			lists["pullRequests.onPass"] = p.OnPass.Actions
		}
		addCommands("pullRequests.slashCommands", p.SlashCommands)
	}
	require.NotEmpty(t, lists)

	for where, actions := range lists {
		for _, a := range actions {
			_, err = defaultActionRunnerRegistry.Get(a.kind)
			require.NoError(t, err, "unknown action %q in %s", a.kind, where)
		}
	}
}
