package governance

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/require"
)

func Test_prHandler_handleOpened(t *testing.T) {
	cfg := config{
		MaintainerAssociations: []string{"OWNER", "MEMBER"},
		PullRequests: &pullRequestsConfig{
			Exemptions: &exemptionsConfig{
				Maintainers: true,
				Bots:        true,
			},
			OnNoLinkedIssue: &onNoLinkedIssueConfig{
				Actions: []action{
					{AddLabels: []string{"policy/no-linked-issue"}},
					{Comment: "No linked issue."},
					{Close: true},
				},
			},
			OnBlockedIssue: &onBlockedIssueConfig{
				BlockingLabels: []string{"kind/proposal"},
				Actions: []action{
					{AddLabels: []string{"policy/blocked-issue"}},
					{Comment: "Issue #{{.IssueNumber}} blocked by {{.BlockingLabels}}"},
					{Close: true},
				},
			},
			OnPass: &onPassConfig{
				Actions: []action{
					{Comment: "PR passes policy."},
				},
			},
			InheritedLabelPrefixes: []string{"area", "kind", "priority"},
			RequiredLabelPrefixes:  []string{"area", "kind", "priority"},
		},
	}

	testCases := []struct {
		name                  string
		body                  string
		authorAssoc           string
		sender                string
		existingIssue         *github.Issue
		expectedLabelsAdded   map[string]struct{}
		expectedCommentsAdded map[string]struct{}
		expectClosed          bool
	}{
		{
			name:        "maintainer is exempt from policy check",
			body:        "No issue reference here.", // Would ordinarily close the PR
			authorAssoc: "MEMBER",
			expectedLabelsAdded: map[string]struct{}{
				"needs/area":     {},
				"needs/kind":     {},
				"needs/priority": {},
			},
			// OnPass fires regardless of exemption (cleanup-style actions).
			expectedCommentsAdded: map[string]struct{}{"PR passes policy.": {}},
		},
		{
			name:        "bot is exempt from policy check",
			body:        "No issue reference.", // Would ordinarily close the PR
			authorAssoc: "NONE",
			sender:      "dependabot[bot]",
			expectedLabelsAdded: map[string]struct{}{
				"needs/area":     {},
				"needs/kind":     {},
				"needs/priority": {},
			},
			// OnPass fires regardless of exemption.
			expectedCommentsAdded: map[string]struct{}{"PR passes policy.": {}},
		},
		{
			name: "no linked issue",
			body: "This PR has no issue link.",
			expectedLabelsAdded: map[string]struct{}{
				// Required-label enforcement runs before policy, so needs/*
				// labels are applied even when the PR is about to be closed.
				"needs/area":             {},
				"needs/kind":             {},
				"needs/priority":         {},
				"policy/no-linked-issue": {},
			},
			expectedCommentsAdded: map[string]struct{}{"No linked issue.": {}},
			expectClosed:          true,
		},
		{
			name: "linked issue without blocking labels",
			body: "Closes #99",
			existingIssue: &github.Issue{Labels: []*github.Label{
				{Name: github.Ptr("area/cli")}, // Inheritable
				// Explicitly testing inheritance of multiple labels from the same group
				{Name: github.Ptr("kind/chore")},         // Inheritable
				{Name: github.Ptr("kind/documentation")}, // Inheritable
				{Name: github.Ptr("random")},             // Not inheritable

			}},
			expectedLabelsAdded: map[string]struct{}{
				"area/cli":           {}, // Inherited from issue
				"kind/chore":         {}, // Inherited from issue
				"kind/documentation": {}, // Inherited from issue
				"needs/priority":     {}, // Added due to label governance
			},
			// OnPass fires because neither OnNoLinkedIssue nor OnBlockedIssue
			// acted.
			expectedCommentsAdded: map[string]struct{}{"PR passes policy.": {}},
		},
		{
			name: "linked issue with blocking label",
			body: "Closes #99",
			existingIssue: &github.Issue{Labels: []*github.Label{
				{Name: github.Ptr("area/cli")},
				{Name: github.Ptr("kind/enhancement")},
				{Name: github.Ptr("kind/proposal")},
				{Name: github.Ptr("priority/high")},
			}},
			expectedLabelsAdded: map[string]struct{}{
				// Labels inherit before policy runs, and inheriting a
				// blocking label onto the PR is intentional — it's an
				// additional signal to the author.
				"area/cli":             {},
				"kind/enhancement":     {},
				"kind/proposal":        {},
				"priority/high":        {},
				"policy/blocked-issue": {},
			},
			expectedCommentsAdded: map[string]struct{}{"Issue #99 blocked by `kind/proposal`": {}},
			expectClosed:          true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			labelsAdded := map[string]struct{}{}
			commentsAdded := map[string]struct{}{}
			closed := false

			issuesClient := &fakeIssuesClient{
				GetFn: func(
					context.Context,
					string,
					string,
					int,
				) (*github.Issue, *github.Response, error) {
					if testCase.existingIssue != nil {
						return testCase.existingIssue, nil, nil
					}
					return &github.Issue{}, nil, nil
				},
				AddLabelsToIssueFn: func(
					_ context.Context,
					_ string,
					_ string,
					_ int,
					labels []string,
				) ([]*github.Label, *github.Response, error) {
					for _, l := range labels {
						labelsAdded[l] = struct{}{}
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
						closed = true
					}
					return pr, nil, nil
				},
			}

			authorAssoc := testCase.authorAssoc
			if authorAssoc == "" {
				authorAssoc = "NONE"
			}
			sender := testCase.sender
			if sender == "" {
				sender = "some-user"
			}
			event := &github.PullRequestEvent{
				Action: github.Ptr("opened"),
				PullRequest: &github.PullRequest{
					Number:            github.Ptr(1),
					Body:              github.Ptr(testCase.body),
					AuthorAssociation: github.Ptr(authorAssoc),
				},
				Repo: &github.Repository{
					Name:  github.Ptr("kargo"),
					Owner: &github.User{Login: github.Ptr("akuity")},
				},
				Sender:       &github.User{Login: github.Ptr(sender)},
				Installation: &github.Installation{ID: github.Ptr(int64(1))},
			}

			h := &prHandler{
				cfg:          cfg,
				owner:        "akuity",
				repo:         "kargo",
				issuesClient: issuesClient,
				prsClient:    prsClient,
			}
			err := h.handleOpened(t.Context(), event, nil)
			require.NoError(t, err)

			if testCase.expectedLabelsAdded == nil {
				testCase.expectedLabelsAdded = map[string]struct{}{}
			}
			if testCase.expectedCommentsAdded == nil {
				testCase.expectedCommentsAdded = map[string]struct{}{}
			}

			require.Equal(t, testCase.expectedLabelsAdded, labelsAdded)
			require.Equal(t, testCase.expectedCommentsAdded, commentsAdded)
			require.Equal(t, testCase.expectClosed, closed)
		})
	}
}

func Test_prHandler_isExemptFromPRPolicy(t *testing.T) {
	testCases := []struct {
		name        string
		cfg         config
		association string
		login       string
		additions   int
		deletions   int
		// listFiles, when set, populates fakePullRequestsClient.ListFilesFn.
		// nil means ListFiles returns no files (or isn't expected to be
		// called).
		listFiles []string
		// listFilesErr, when set, makes the fake return this error.
		listFilesErr error
		expected     bool
		expectErr    bool
	}{
		{
			name:        "nil PullRequests not exempt",
			cfg:         config{},
			association: "MEMBER",
			login:       "kent",
			expected:    false,
		},
		{
			name: "nil Exemptions not exempt",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER"},
				PullRequests:           &pullRequestsConfig{},
			},
			association: "MEMBER",
			login:       "kent",
			expected:    false,
		},
		{
			name: "maintainer exempt when enabled",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER"},
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{Maintainers: true},
				},
			},
			association: "MEMBER",
			login:       "kent",
			expected:    true,
		},
		{
			name: "maintainer not exempt when disabled",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER"},
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{Maintainers: false},
				},
			},
			association: "MEMBER",
			login:       "kent",
			expected:    false,
		},
		{
			name: "bot exempt when enabled",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{Bots: true},
				},
			},
			association: "NONE",
			login:       "dependabot[bot]",
			expected:    true,
		},
		{
			name: "bot not exempt when disabled",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{Bots: false},
				},
			},
			association: "NONE",
			login:       "dependabot[bot]",
			expected:    false,
		},
		{
			name: "size exempt — total at limit",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{MaxChangedLines: 5},
				},
			},
			association: "NONE",
			login:       "random-user",
			additions:   3,
			deletions:   2,
			expected:    true,
		},
		{
			name: "size exempt — total below limit",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{MaxChangedLines: 5},
				},
			},
			association: "NONE",
			login:       "random-user",
			additions:   1,
			deletions:   0,
			expected:    true,
		},
		{
			name: "size NOT exempt — total over limit",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{MaxChangedLines: 5},
				},
			},
			association: "NONE",
			login:       "random-user",
			additions:   4,
			deletions:   2,
			expected:    false,
		},
		{
			name: "size NOT exempt when MaxChangedLines is 0",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{MaxChangedLines: 0},
				},
			},
			association: "NONE",
			login:       "random-user",
			additions:   1,
			deletions:   0,
			expected:    false,
		},
		{
			name: "path exempt — all files match",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{
						PathPatterns: []string{"**/*.md", "docs/"},
					},
				},
			},
			association: "NONE",
			login:       "random-user",
			listFiles:   []string{"README.md", "docs/foo.md", "docs/sub/bar.md"},
			expected:    true,
		},
		{
			name: "path NOT exempt — one file doesn't match",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{
						PathPatterns: []string{"**/*.md"},
					},
				},
			},
			association: "NONE",
			login:       "random-user",
			listFiles:   []string{"README.md", "main.go"},
			expected:    false,
		},
		{
			name: "path NOT exempt — empty patterns short-circuits",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{},
				},
			},
			association: "NONE",
			login:       "random-user",
			expected:    false,
		},
		{
			name: "path check propagates ListFiles error",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{
						PathPatterns: []string{"**/*.md"},
					},
				},
			},
			association:  "NONE",
			login:        "random-user",
			listFilesErr: errors.New("network error"),
			expected:     false,
			expectErr:    true,
		},
		{
			name: "regular user not exempt",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER"},
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{
						Maintainers: true,
						Bots:        true,
					},
				},
			},
			association: "NONE",
			login:       "random-user",
			expected:    false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			prsClient := &fakePullRequestsClient{
				ListFilesFn: func(
					_ context.Context,
					_, _ string,
					_ int,
					_ *github.ListOptions,
				) ([]*github.CommitFile, *github.Response, error) {
					if testCase.listFilesErr != nil {
						return nil, nil, testCase.listFilesErr
					}
					files := make(
						[]*github.CommitFile, 0, len(testCase.listFiles),
					)
					for _, name := range testCase.listFiles {
						files = append(files, &github.CommitFile{
							Filename: github.Ptr(name),
						})
					}
					return files, nil, nil
				},
			}
			pr := &github.PullRequest{
				Number:            github.Ptr(1),
				AuthorAssociation: github.Ptr(testCase.association),
				Additions:         github.Ptr(testCase.additions),
				Deletions:         github.Ptr(testCase.deletions),
			}
			result, err := isExemptFromPRPolicy(
				t.Context(), testCase.cfg, prsClient,
				"akuity", "kargo", pr, testCase.login,
			)
			if testCase.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, testCase.expected, result)
		})
	}
}

func Test_formatBlockers(t *testing.T) {
	require.Equal(
		t,
		"`kind/proposal`",
		formatBlockers([]string{"kind/proposal"}),
	)
	require.Equal(
		t,
		"`kind/proposal`, `needs discussion`",
		formatBlockers([]string{"kind/proposal", "needs discussion"}),
	)
}

func Test_parseLinkedIssue(t *testing.T) {
	testCases := []struct {
		name     string
		body     string
		expected int
	}{
		{
			name:     "Closes #123",
			body:     "Some description.\n\nCloses #123",
			expected: 123,
		},
		{
			name:     "Fixes #456",
			body:     "Fixes #456",
			expected: 456,
		},
		{
			name:     "Resolves #789",
			body:     "Resolves #789\n\nMore text here.",
			expected: 789,
		},
		{
			name:     "close #1 (singular)",
			body:     "close #1",
			expected: 1,
		},
		{
			name:     "closed #2 (past tense)",
			body:     "closed #2",
			expected: 2,
		},
		{
			name:     "fix #3 (singular)",
			body:     "fix #3",
			expected: 3,
		},
		{
			name:     "fixed #4 (past tense)",
			body:     "fixed #4",
			expected: 4,
		},
		{
			name:     "resolve #5 (singular)",
			body:     "resolve #5",
			expected: 5,
		},
		{
			name:     "resolved #6 (past tense)",
			body:     "resolved #6",
			expected: 6,
		},
		{
			name:     "case insensitive",
			body:     "CLOSES #99",
			expected: 99,
		},
		{
			name:     "full URL",
			body:     "Closes https://github.com/akuity/kargo/issues/42",
			expected: 42,
		},
		{
			name:     "full URL with hash",
			body:     "Fixes https://github.com/akuity/kargo/issues/#42",
			expected: 42,
		},
		{
			name:     "first match wins",
			body:     "Closes #10\nAlso fixes #20",
			expected: 10,
		},
		{
			name:     "no match returns 0",
			body:     "This PR does some stuff.",
			expected: 0,
		},
		{
			name:     "empty body returns 0",
			body:     "",
			expected: 0,
		},
		{
			name:     "hash without keyword returns 0",
			body:     "Related to #123",
			expected: 0,
		},
		{
			name:     "keyword without number returns 0",
			body:     "Closes the loop",
			expected: 0,
		},
		{
			name:     "embedded in PR template",
			body:     "**Policy statement**\n\nCloses #55\n\n## Description\n\nSome work.",
			expected: 55,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := parseLinkedIssue(testCase.body)
			require.Equal(t, testCase.expected, result)
		})
	}
}
