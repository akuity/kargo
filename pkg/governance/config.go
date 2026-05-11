package governance

import (
	"fmt"

	"github.com/goccy/go-yaml"
)

// config represents top-level governance bot configuration.
type config struct {
	// MaintainerAssociations is a list of GitHub association types that are
	// considered maintainers. Maintainers can use slash commands and depending
	// upon configuration may be exempt from certain PR policies. Valid values
	// are: "OWNER", "MEMBER", "CONTRIBUTOR". If empty, no associations are
	// considered maintainers.
	MaintainerAssociations []string `json:"maintainerAssociations,omitempty"`
	// Issues defines the configuration applied to issues.
	Issues *issuesConfig `json:"issues,omitempty"`
	// PullRequests defines the configuration applied to pull requests, including
	// exemptions for maintainers and bots and actions to take for PRs without a
	// linked issue or with a linked issue that is blocked.
	PullRequests *pullRequestsConfig `json:"pullRequests,omitempty"`
}

// issuesConfig defines the configuration applied to issues.
type issuesConfig struct {
	// RequiredLabelPrefixes lists the label prefixes that every issue must
	// carry at least one label for. For any missing prefix, a "needs/<prefix>"
	// label is added automatically.
	RequiredLabelPrefixes []string `json:"requiredLabelPrefixes,omitempty"`
	// SlashCommands defines the slash commands available on issues, keyed by
	// command name.
	SlashCommands map[string]commandDef `json:"slashCommands,omitempty"`
}

// pullRequestsConfig defines the configuration applied to pull requests,
// including exemptions and actions for PRs without a linked issue or with a
// linked issue that is blocked.
type pullRequestsConfig struct {
	// Exemptions defines the criteria under which a PR is exempt from
	// automated policy enforcement. The criteria are OR'd: a PR matching any
	// configured exemption is exempt. Slash commands like /policy bypass
	// exemptions and always apply the configured policy.
	Exemptions *exemptionsConfig `json:"exemptions,omitempty"`
	// OnNoLinkedIssue defines the actions to take for PRs without a linked issue.
	OnNoLinkedIssue *onNoLinkedIssueConfig `json:"onNoLinkedIssue,omitempty"`
	// OnBlockedIssue defines the actions to take for PRs whose linked issue is
	// blocked by the presence of certain labels.
	OnBlockedIssue *onBlockedIssueConfig `json:"onBlockedIssue,omitempty"`
	// OnPass defines the actions to take when the PR passes policy — i.e.
	// when neither OnNoLinkedIssue nor OnBlockedIssue actions fire. Useful for
	// cleaning up state left by a prior failing evaluation (e.g. removing a
	// policy/* label that was added when the PR was previously drafted).
	OnPass *onPassConfig `json:"onPass,omitempty"`
	// InheritedLabelPrefixes lists the label prefixes that a pull request
	// inherits from its linked issue when opened.
	InheritedLabelPrefixes []string `json:"inheritedLabelPrefixes,omitempty"`
	// RequiredLabelPrefixes lists the label prefixes that every pull request
	// must carry at least one label for. For any missing prefix, a
	// "needs/<prefix>" label is added automatically.
	RequiredLabelPrefixes []string `json:"requiredLabelPrefixes,omitempty"`
	// SlashCommands defines the slash commands available on pull requests,
	// keyed by command name.
	SlashCommands map[string]commandDef `json:"slashCommands,omitempty"`
}

// exemptionsConfig defines the criteria under which a PR is exempt from
// automated policy enforcement. Each configured criterion is independently
// evaluated; a PR matching any one of them is exempt.
type exemptionsConfig struct {
	// Maintainers indicates whether maintainers are exempt from PR policy.
	Maintainers bool `json:"maintainers,omitempty"`
	// Bots indicates whether bots are exempt from PR policy.
	Bots bool `json:"bots,omitempty"`
	// MaxChangedLines exempts PRs whose total additions + deletions are less
	// than or equal to this value. A value <= 0 disables the check.
	MaxChangedLines uint `json:"maxChangedLines,omitempty"`
	// PathPatterns is a list of gitignore-style patterns. A PR is exempt
	// when every file it changes matches at least one pattern. An empty
	// list disables the check.
	PathPatterns []string `json:"pathPatterns,omitempty"`
}

// onNoLinkedIssueConfig defines the actions to take for issues without linked
// issues.
type onNoLinkedIssueConfig struct {
	// Actions defines the actions to take for issues without linked issues.
	Actions []action `json:"actions,omitempty"`
}

// onBlockedIssueConfig defines the actions to take for issues that are blocked by
// certain labels.
type onBlockedIssueConfig struct {
	// BlockingLabels defines the labels that indicate an issue is blocked.
	BlockingLabels []string `json:"blockingLabels,omitempty"`
	// Actions defines the actions to take for issues that are blocked by certain labels.
	Actions []action `json:"actions,omitempty"`
}

// onPassConfig defines the actions to take when a PR passes policy: it has
// a linked, unblocked issue (or the operator hasn't configured the
// corresponding check).
type onPassConfig struct {
	// Actions defines the actions to take when the PR passes policy.
	Actions []action `json:"actions,omitempty"`
}

// action is the parsed form of one entry in an action list. Its YAML shape
// is a mapping with exactly one key — the action's kind name (e.g.
// "addLabels", "comment") — and a value that's the kind-specific
// configuration. The value's bytes are captured raw and decoded by the
// runner registered for that kind.
//
// Examples:
//
//	addLabels: [needs/area, needs/kind]
//	→ action{kind: "addLabels", config: []byte("[needs/area, needs/kind]\n")}
//
//	close: true
//	→ action{kind: "close", config: []byte("true\n")}
type action struct {
	kind   string
	config []byte
}

// UnmarshalYAML implements goccy/go-yaml's BytesUnmarshaler. The expected
// form is a mapping with exactly one entry.
func (a *action) UnmarshalYAML(data []byte) error {
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("error parsing action: %w", err)
	}
	if len(m) != 1 {
		return fmt.Errorf(
			"action must have exactly one key, got %d", len(m),
		)
	}
	for k, v := range m {
		cfg, err := yaml.Marshal(v)
		if err != nil {
			return fmt.Errorf("error capturing %q config: %w", k, err)
		}
		a.kind = k
		a.config = cfg
	}
	return nil
}

// commandDef defines a single slash command, including its description, whether
// it requires an argument, and the actions to take when the command is
// executed.
type commandDef struct {
	// Description provides a brief description of the slash command, which can be
	// used in help messages or documentation to explain the purpose of the
	// command to users.
	Description string `json:"description"`
	// RequiresArg indicates whether the slash command requires an argument. If
	// true, the command will only be executed if an argument is provided. If
	// false, the command can be executed without an argument.
	RequiresArg bool `json:"requiresArg"`
	// Actions defines the actions to take when the slash command is executed.
	Actions []action `json:"actions"`
}
