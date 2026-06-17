package governance

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/require"
)

func Test_commentHandler_handleCreated(t *testing.T) {
	cfg := config{
		MaintainerAssociations: []string{"MEMBER"},
		Issues: &issuesConfig{
			SlashCommands: map[string]commandDef{
				"label-only": {
					Description: "Adds a label",
					Actions: []action{
						mustAction("addLabels", []string{"test-label"}),
						mustAction("comment", "Label added."),
					},
				},
				"close-issue": {
					Description: "Closes the issue",
					Actions: []action{
						mustAction("comment", "Closing."),
						mustAction("close", true),
					},
				},
				"with-arg": {
					Description: "Needs an arg",
					RequiresArg: true,
					Actions: []action{
						mustAction("comment", "Arg was #{{.Arg}}."),
					},
				},
			},
		},
		PullRequests: &pullRequestsConfig{
			SlashCommands: map[string]commandDef{
				"label-pr": {
					Description: "Adds a label to the PR",
					Actions: []action{
						mustAction("addLabels", []string{"pr-label"}),
						mustAction("comment", "PR label added."),
					},
				},
				"close-pr": {
					Description: "Closes the PR",
					Actions: []action{
						mustAction("comment", "Closing PR."),
						mustAction("close", true),
					},
				},
				"with-arg-pr": {
					Description: "Needs an arg",
					RequiresArg: true,
					Actions: []action{
						mustAction("comment", "PR arg was #{{.Arg}}."),
					},
				},
			},
		},
	}

	testCases := []struct {
		name                  string
		body                  string
		authorLogin           string
		authorAssoc           string
		isMembers             map[string]bool
		isPR                  bool
		expectedLabelsAdded   map[string]struct{}
		expectedCommentsAdded map[string]struct{}
		expectClosedAsIssue   bool
		expectClosedAsPR      bool
	}{
		// Issue cases
		{
			name:        "non-slash issue comment",
			body:        "just a regular comment",
			authorAssoc: "MEMBER",
		},
		{
			name:        "non-maintainer issue comment",
			body:        "/label-only",
			authorAssoc: "NONE",
			isMembers:   map[string]bool{"maintainer": false},
		},
		{
			// Concealed (private) org member: GitHub reports
			// author_association as CONTRIBUTOR in webhook payloads even
			// when the App has Organization Members: Read. The
			// org-membership fallback restores their maintainer status.
			name:                  "concealed maintainer's slash command is honored",
			body:                  "/label-only",
			authorLogin:           "frankenstein",
			authorAssoc:           "CONTRIBUTOR",
			isMembers:             map[string]bool{"frankenstein": true},
			expectedLabelsAdded:   map[string]struct{}{"test-label": {}},
			expectedCommentsAdded: map[string]struct{}{"Label added.": {}},
		},
		{
			name:        "issue comment with unknown command",
			body:        "/nonexistent",
			authorAssoc: "MEMBER",
		},
		{
			name:        "issue comment with PR-only command",
			body:        "/label-pr",
			authorAssoc: "MEMBER",
		},
		{
			name:                  "issue comment with valid command",
			body:                  "/label-only",
			authorAssoc:           "MEMBER",
			expectedLabelsAdded:   map[string]struct{}{"test-label": {}},
			expectedCommentsAdded: map[string]struct{}{"Label added.": {}},
		},
		{
			name:        "issue comment with valid command missing required arg",
			body:        "/with-arg",
			authorAssoc: "MEMBER",
			expectedCommentsAdded: map[string]struct{}{
				"The `/with-arg` command requires an argument. See `/help` for usage.": {},
			},
		},
		{
			name:                  "issue comment with command indented and mid-body",
			body:                  "Some context here.\n\n  /label-only\n\nThanks!",
			authorAssoc:           "MEMBER",
			expectedLabelsAdded:   map[string]struct{}{"test-label": {}},
			expectedCommentsAdded: map[string]struct{}{"Label added.": {}},
		},
		{
			name:                "issue comment with multiple commands",
			body:                "/label-only\n/close-issue",
			authorAssoc:         "MEMBER",
			expectedLabelsAdded: map[string]struct{}{"test-label": {}},
			expectedCommentsAdded: map[string]struct{}{
				"Label added.": {},
				"Closing.":     {},
			},
			expectClosedAsIssue: true,
		},
		{
			name:        "issue comment with help command",
			body:        "/help",
			authorAssoc: "MEMBER",
			expectedCommentsAdded: map[string]struct{}{
				buildHelpComment(cfg.Issues.SlashCommands): {},
			},
		},
		// PR cases — minimal coverage: just enough to prove context
		// routing works both ways. The dispatch / parse / gate behavior
		// is identical to the issue path and is fully exercised above.
		{
			// Routing: in PR context, an issue-only slash command falls
			// through as if unknown.
			name:        "PR comment with issue-only command",
			body:        "/label-only",
			authorAssoc: "MEMBER",
			isPR:        true,
		},
		{
			// Routing: in PR context, a PR slash command dispatches as
			// expected.
			name:                  "PR comment with valid command",
			body:                  "/label-pr",
			authorAssoc:           "MEMBER",
			isPR:                  true,
			expectedLabelsAdded:   map[string]struct{}{"pr-label": {}},
			expectedCommentsAdded: map[string]struct{}{"PR label added.": {}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			labelsAdded := map[string]struct{}{}
			commentsAdded := map[string]struct{}{}
			closedAsIssue := false
			closedAsPR := false

			issuesClient := &fakeIssuesClient{
				AddLabelsToIssueFn: func(
					_ context.Context,
					_ string,
					_ string,
					_ int,
					labels []string,
				) ([]*github.Label, *github.Response, error) {
					for _, label := range labels {
						labelsAdded[label] = struct{}{}
					}
					return nil, nil, nil
				},
				CreateCommentFn: func(
					_ context.Context,
					_ string,
					_ string,
					_ int,
					comment *github.IssueComment,
				) (*github.IssueComment, *github.Response, error) {
					commentsAdded[comment.GetBody()] = struct{}{}
					return comment, nil, nil
				},
				EditFn: func(
					_ context.Context,
					_ string,
					_ string,
					_ int,
					req *github.IssueRequest,
				) (*github.Issue, *github.Response, error) {
					if req.GetState() == issueStateClosed {
						closedAsIssue = true
					}
					return &github.Issue{}, nil, nil
				},
			}
			prsClient := &fakePullRequestsClient{
				EditFn: func(
					_ context.Context,
					_ string,
					_ string,
					_ int,
					pr *github.PullRequest,
				) (*github.PullRequest, *github.Response, error) {
					if pr.GetState() == prStateClosed {
						closedAsPR = true
					}
					return pr, nil, nil
				},
			}

			issue := &github.Issue{Number: github.Ptr(42)}
			if testCase.isPR {
				issue.PullRequestLinks = &github.PullRequestLinks{
					URL: github.Ptr("https://api.github.com/repos/test/repo/pulls/42"),
				}
			}

			authorLogin := testCase.authorLogin
			if authorLogin == "" {
				authorLogin = "maintainer"
			}
			event := &github.IssueCommentEvent{
				Action: github.Ptr("created"),
				Issue:  issue,
				Comment: &github.IssueComment{
					Body:              github.Ptr(testCase.body),
					User:              &github.User{Login: github.Ptr(authorLogin)},
					AuthorAssociation: github.Ptr(testCase.authorAssoc),
				},
				Repo: &github.Repository{
					Name:  github.Ptr("kargo"),
					Owner: &github.User{Login: github.Ptr("akuity")},
				},
				Sender:       &github.User{Login: github.Ptr(authorLogin)},
				Installation: &github.Installation{ID: github.Ptr(int64(1))},
			}

			orgsClient := &fakeOrganizationsClient{
				IsMemberFn: func(
					_ context.Context,
					_ string,
					user string,
				) (bool, *github.Response, error) {
					member, ok := testCase.isMembers[user]
					if !ok {
						return false, nil, nil
					}
					return member, nil, nil
				},
			}

			h := &commentHandler{
				repoContext: repoContext{
					cfg:          cfg,
					owner:        "akuity",
					repo:         "kargo",
					issuesClient: issuesClient,
					prsClient:    prsClient,
					orgsClient:   orgsClient,
				},
			}
			err := h.handleCreated(t.Context(), event)
			require.NoError(t, err)

			if testCase.expectedLabelsAdded == nil {
				testCase.expectedLabelsAdded = map[string]struct{}{}
			}
			if testCase.expectedCommentsAdded == nil {
				testCase.expectedCommentsAdded = map[string]struct{}{}
			}
			require.Equal(t, testCase.expectedLabelsAdded, labelsAdded)
			require.Equal(t, testCase.expectedCommentsAdded, commentsAdded)
			require.Equal(t, testCase.expectClosedAsIssue, closedAsIssue)
			require.Equal(t, testCase.expectClosedAsPR, closedAsPR)
		})
	}
}

func Test_commentHandler_parseSlashCommands(t *testing.T) {
	testCases := []struct {
		name     string
		body     string
		expected []parsedCommand
	}{
		{
			name:     "empty body",
			body:     "",
			expected: nil,
		},
		{
			name:     "no slash commands",
			body:     "just a regular comment",
			expected: nil,
		},
		{
			name:     "single command",
			body:     "/close-issue",
			expected: []parsedCommand{{name: "close-issue"}},
		},
		{
			name:     "command with arg",
			body:     "/duplicate #42",
			expected: []parsedCommand{{name: "duplicate", arg: "42"}},
		},
		{
			name:     "command with leading whitespace",
			body:     "  /close-issue",
			expected: []parsedCommand{{name: "close-issue"}},
		},
		{
			name: "command mid-body",
			body: "Some context.\n\n/close-issue\n\nThanks!",
			expected: []parsedCommand{
				{name: "close-issue"},
			},
		},
		{
			name: "multiple commands in order",
			body: "/label-only\n/close-issue",
			expected: []parsedCommand{
				{name: "label-only"},
				{name: "close-issue"},
			},
		},
		{
			name:     "lone slash is ignored",
			body:     "/",
			expected: nil,
		},
	}
	h := &commentHandler{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t, testCase.expected, h.parseSlashCommands(testCase.body),
			)
		})
	}
}

func Test_buildHelpComment(t *testing.T) {
	commands := map[string]commandDef{
		"beta": {Description: "Second"},
		"alpha": {
			Description: "First",
			RequiresArg: true,
		},
	}
	result := buildHelpComment(commands)

	require.Contains(t, result, "Available Slash Commands")
	require.Contains(t, result, "| `/alpha #N` | First |")
	require.Contains(t, result, "| `/beta` | Second |")
	require.Contains(t, result, "| `/help` | Show this list |")

	alphaIdx := strings.Index(result, "/alpha")
	betaIdx := strings.Index(result, "/beta")
	helpIdx := strings.Index(result, "/help")
	require.Less(t, alphaIdx, betaIdx)
	require.Less(t, betaIdx, helpIdx)
}
