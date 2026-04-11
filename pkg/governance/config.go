package governance

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
// including exemptions for maintainers and bots and actions for PRs without a
// linked issue or with a linked issue that is blocked.
type pullRequestsConfig struct {
	// ExemptMaintainers indicates whether maintainers are exempt from PR
	// policies.
	ExemptMaintainers bool `json:"exemptMaintainers,omitempty"`
	// ExemptBots indicates whether bots are exempt from PR policies.
	ExemptBots bool `json:"exemptBots,omitempty"`
	// NoLinkedIssue defines the actions to take for PRs without a linked issue.
	NoLinkedIssue *noLinkedIssueConfig `json:"noLinkedIssue,omitempty"`
	// BlockedIssue defines the actions to take for PRs whose linked issue is
	// blocked by the presence of certain labels.
	BlockedIssue *blockedIssueConfig `json:"blockedIssue,omitempty"`
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

// noLinkedIssueConfig defines the actions to take for issues without linked
// issues.
type noLinkedIssueConfig struct {
	// Actions defines the actions to take for issues without linked issues.
	Actions []action `json:"actions,omitempty"`
}

// blockedIssueConfig defines the actions to take for issues that are blocked by
// certain labels.
type blockedIssueConfig struct {
	// BlockingLabels defines the labels that indicate an issue is blocked.
	BlockingLabels []string `json:"blockingLabels,omitempty"`
	// Actions defines the actions to take for issues that are blocked by certain labels.
	Actions []action `json:"actions,omitempty"`
}

// action defines a single action to take, which may include adding or removing
// labels, posting a comment, or closing the issue or pull request.
type action struct {
	// AddLabels defines the labels to add when the action is executed.
	AddLabels []string `json:"addLabels,omitempty"`
	// RemoveLabels defines the labels to remove when the action is executed.
	RemoveLabels []string `json:"removeLabels,omitempty"`
	// Comment defines the comment to post when the action is executed.
	Comment string `json:"comment,omitempty"`
	// Close indicates whether to close the issue or pull request when the action
	// is executed.

	Close bool `json:"close,omitempty"`
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
