package governance

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/require"
)

func Test_prHandler_handleOpened(t *testing.T) {
	// Cases here focus on what the handler does that isn't already proven
	// by direct unit tests on repoContext: orchestrating the four steps
	// (assign → inherit → enforce → apply policy), and honoring the
	// applyPolicyOnly bypass on reopen / ready_for_review. Branch-level
	// coverage of policy outcomes lives in Test_repoContext_applyPRPolicy
	// and Test_repoContext_isPRExempt; coverage of inheritance and
	// required-label enforcement lives in their respective unit tests.
	cfg := config{
		MaintainerAssociations: []string{"OWNER", "MEMBER"},
		PullRequests: &pullRequestsConfig{
			Exemptions: &exemptionsConfig{Maintainers: true},
			OnNoLinkedIssue: &onNoLinkedIssueConfig{
				Actions: []action{
					mustAction("addLabels", []string{"policy/no-linked-issue"}),
					mustAction("comment", "No linked issue."),
					mustAction("close", true),
				},
			},
			OnPass: &onPassConfig{
				Actions: []action{
					mustAction("comment", "PR passes policy."),
				},
			},
			InheritedLabelPrefixes: []string{"area", "kind", "priority"},
			RequiredLabelPrefixes:  []string{"area", "kind", "priority"},
		},
	}

	testCases := []struct {
		name            string
		action          string
		body            string
		author          string
		sender          string
		linkedIssue     *github.Issue
		isMembers       map[string]bool
		addAssigneesErr error
		assert          func(
			t *testing.T,
			assignees []string,
			labels map[string]struct{},
			comments map[string]struct{},
			closed bool,
			err error,
		)
	}{
		{
			// Happy path on `opened`: all four steps must run end-to-end.
			// The PR is assigned to its author, inheritable labels from
			// the linked issue land on the PR, required-label enforcement
			// fills the remaining gap (needs/priority), and OnPass fires.
			name:   "opened, linked issue with no blockers: all steps run end-to-end",
			body:   "Closes #99",
			author: "external",
			linkedIssue: &github.Issue{Labels: []*github.Label{
				{Name: github.Ptr("area/cli")},
				{Name: github.Ptr("kind/chore")},
			}},
			assert: func(
				t *testing.T,
				assignees []string,
				labels, comments map[string]struct{},
				closed bool,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, []string{"external"}, assignees)
				require.Equal(t, map[string]struct{}{
					"area/cli":       {}, // inherited
					"kind/chore":     {}, // inherited
					"needs/priority": {}, // enforced (priority/* missing)
				}, labels)
				require.Equal(t, map[string]struct{}{"PR passes policy.": {}}, comments)
				require.False(t, closed)
			},
		},
		{
			// Step 3 (required-label enforcement) must run before step 4
			// (policy), even when step 4 is about to close the PR. Both
			// the needs/* labels and the policy/no-linked-issue label end
			// up on the PR.
			name:   "opened, no linked issue: enforce runs before policy closes",
			body:   "No issue link here.",
			author: "external",
			assert: func(
				t *testing.T,
				assignees []string,
				labels, comments map[string]struct{},
				closed bool,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, []string{"external"}, assignees)
				require.Equal(t, map[string]struct{}{
					"needs/area":             {},
					"needs/kind":             {},
					"needs/priority":         {},
					"policy/no-linked-issue": {},
				}, labels)
				require.Equal(t, map[string]struct{}{"No linked issue.": {}}, comments)
				require.True(t, closed)
			},
		},
		{
			// Error accumulation: when one step fails, subsequent steps
			// must still run. Here step 1 (assign) is forced to fail; the
			// returned error must wrap that failure, AND the
			// inherit/enforce/policy steps must still have produced their
			// expected side effects.
			name:            "step failure does not halt subsequent steps",
			body:            "Closes #99",
			author:          "external",
			addAssigneesErr: errors.New("assignees boom"),
			linkedIssue: &github.Issue{Labels: []*github.Label{
				{Name: github.Ptr("kind/chore")},
			}},
			assert: func(
				t *testing.T,
				assignees []string,
				labels, comments map[string]struct{},
				closed bool,
				err error,
			) {
				require.ErrorContains(t, err, "assign PR to author")
				require.ErrorContains(t, err, "assignees boom")
				// Step 1 was called even though it failed.
				require.Equal(t, []string{"external"}, assignees)
				// Step 2 (inherit) ran after step 1's failure.
				require.Contains(t, labels, "kind/chore")
				// Step 3 (enforce) ran — needs/area and needs/priority
				// added for the still-missing required prefixes.
				require.Contains(t, labels, "needs/area")
				require.Contains(t, labels, "needs/priority")
				// Step 4 (apply policy) ran — OnPass fired.
				require.Equal(t, map[string]struct{}{"PR passes policy.": {}}, comments)
				require.False(t, closed)
			},
		},
		{
			// applyPolicyOnly path: steps 1-3 (assign, inherit, enforce)
			// must be skipped. Only step 4 runs. A maintainer sender on a
			// non-maintainer author's PR is treated as exempt; OnPass
			// fires without policy violations or label churn.
			name:   "ready_for_review with maintainer sender: only policy runs",
			action: prActionReadyForReview,
			body:   "No issue link here.",
			author: "external",
			sender: "kent",
			isMembers: map[string]bool{
				"external": false,
				"kent":     true,
			},
			assert: func(
				t *testing.T,
				assignees []string,
				labels, comments map[string]struct{},
				closed bool,
				err error,
			) {
				require.NoError(t, err)
				require.Empty(t, assignees)
				require.Empty(t, labels)
				require.Equal(t, map[string]struct{}{"PR passes policy.": {}}, comments)
				require.False(t, closed)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var assignees []string
			labels := map[string]struct{}{}
			comments := map[string]struct{}{}
			closed := false

			issuesClient := &fakeIssuesClient{
				AddAssigneesFn: func(
					_ context.Context,
					_, _ string,
					_ int,
					a []string,
				) (*github.Issue, *github.Response, error) {
					assignees = append(assignees, a...)
					return nil, nil, testCase.addAssigneesErr
				},
				GetFn: func(
					context.Context, string, string, int,
				) (*github.Issue, *github.Response, error) {
					if testCase.linkedIssue != nil {
						return testCase.linkedIssue, nil, nil
					}
					return &github.Issue{}, nil, nil
				},
				AddLabelsToIssueFn: func(
					_ context.Context,
					_, _ string,
					_ int,
					added []string,
				) ([]*github.Label, *github.Response, error) {
					for _, l := range added {
						labels[l] = struct{}{}
					}
					return nil, nil, nil
				},
				CreateCommentFn: func(
					_ context.Context,
					_, _ string,
					_ int,
					comment *github.IssueComment,
				) (*github.IssueComment, *github.Response, error) {
					comments[comment.GetBody()] = struct{}{}
					return comment, nil, nil
				},
			}
			prsClient := &fakePullRequestsClient{
				EditFn: func(
					_ context.Context,
					_, _ string,
					_ int,
					pr *github.PullRequest,
				) (*github.PullRequest, *github.Response, error) {
					if pr.GetState() == prStateClosed {
						closed = true
					}
					return pr, nil, nil
				},
			}
			orgsClient := &fakeOrganizationsClient{
				IsMemberFn: func(
					_ context.Context,
					_ string,
					user string,
				) (bool, *github.Response, error) {
					return testCase.isMembers[user], nil, nil
				},
			}

			sender := testCase.sender
			if sender == "" {
				sender = testCase.author
			}
			action := testCase.action
			if action == "" {
				action = prActionOpened
			}
			event := &github.PullRequestEvent{
				Action: github.Ptr(action),
				PullRequest: &github.PullRequest{
					Number:            github.Ptr(1),
					Body:              github.Ptr(testCase.body),
					User:              &github.User{Login: github.Ptr(testCase.author)},
					AuthorAssociation: github.Ptr("NONE"),
				},
				Repo: &github.Repository{
					Name:  github.Ptr("kargo"),
					Owner: &github.User{Login: github.Ptr("akuity")},
				},
				Sender:       &github.User{Login: github.Ptr(sender)},
				Installation: &github.Installation{ID: github.Ptr(int64(1))},
			}

			h := &prHandler{
				repoContext: repoContext{
					cfg:          cfg,
					owner:        "akuity",
					repo:         "kargo",
					issuesClient: issuesClient,
					prsClient:    prsClient,
					orgsClient:   orgsClient,
				},
			}
			var opts *handlePROpenedOpts
			if action == prActionReopened || action == prActionReadyForReview {
				opts = &handlePROpenedOpts{applyPolicyOnly: true}
			}
			err := h.handleOpened(t.Context(), event, opts)
			testCase.assert(t, assignees, labels, comments, closed, err)
		})
	}
}

func Test_prHandler_inheritLabels(t *testing.T) {
	testCases := []struct {
		name           string
		cfg            config
		issueNumber    int
		linkedIssue    *github.Issue
		linkedIssueErr error
		addLabelsErr   error
		assert         func(t *testing.T, inherited, addedLabels []string, err error)
	}{
		{
			// issueNumber == 0 means parseLinkedIssue didn't find a
			// reference — short-circuit before any API call.
			name:        "no linked issue: no-op",
			cfg:         config{PullRequests: &pullRequestsConfig{InheritedLabelPrefixes: []string{"kind"}}},
			issueNumber: 0,
			assert: func(t *testing.T, inherited, addedLabels []string, err error) {
				require.NoError(t, err)
				require.Empty(t, inherited)
				require.Empty(t, addedLabels)
			},
		},
		{
			name:        "PullRequests config nil: no-op",
			cfg:         config{},
			issueNumber: 99,
			assert: func(t *testing.T, inherited, addedLabels []string, err error) {
				require.NoError(t, err)
				require.Empty(t, inherited)
				require.Empty(t, addedLabels)
			},
		},
		{
			name:        "no InheritedLabelPrefixes configured: no-op",
			cfg:         config{PullRequests: &pullRequestsConfig{}},
			issueNumber: 99,
			assert: func(t *testing.T, inherited, addedLabels []string, err error) {
				require.NoError(t, err)
				require.Empty(t, inherited)
				require.Empty(t, addedLabels)
			},
		},
		{
			name: "issuesClient.Get error propagates",
			cfg: config{PullRequests: &pullRequestsConfig{
				InheritedLabelPrefixes: []string{"kind"},
			}},
			issueNumber:    99,
			linkedIssueErr: errors.New("network error"),
			assert: func(t *testing.T, inherited, addedLabels []string, err error) {
				require.ErrorContains(t, err, "error fetching linked issue")
				require.Empty(t, inherited)
				require.Empty(t, addedLabels)
			},
		},
		{
			name: "some issue labels match prefixes: inherited and added",
			cfg: config{PullRequests: &pullRequestsConfig{
				InheritedLabelPrefixes: []string{"area", "kind"},
			}},
			issueNumber: 99,
			linkedIssue: &github.Issue{Labels: []*github.Label{
				{Name: github.Ptr("area/cli")},
				{Name: github.Ptr("kind/bug")},
				{Name: github.Ptr("priority/high")}, // not in prefixes
				{Name: github.Ptr("random")},        // no slash
			}},
			assert: func(t *testing.T, inherited, addedLabels []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"area/cli", "kind/bug"}, inherited)
				require.Equal(t, []string{"area/cli", "kind/bug"}, addedLabels)
			},
		},
		{
			name: "no issue labels match prefixes: AddLabelsToIssue is NOT called",
			cfg: config{PullRequests: &pullRequestsConfig{
				InheritedLabelPrefixes: []string{"area"},
			}},
			issueNumber: 99,
			linkedIssue: &github.Issue{Labels: []*github.Label{
				{Name: github.Ptr("kind/bug")},
				{Name: github.Ptr("priority/high")},
			}},
			assert: func(t *testing.T, inherited, addedLabels []string, err error) {
				require.NoError(t, err)
				require.Empty(t, inherited)
				require.Empty(t, addedLabels)
			},
		},
		{
			// Multiple labels under the same prefix all carry over —
			// inheritance isn't one-per-prefix, it's all-matching-labels.
			name: "multiple labels with same prefix: all inherited",
			cfg: config{PullRequests: &pullRequestsConfig{
				InheritedLabelPrefixes: []string{"kind"},
			}},
			issueNumber: 99,
			linkedIssue: &github.Issue{Labels: []*github.Label{
				{Name: github.Ptr("kind/bug")},
				{Name: github.Ptr("kind/documentation")},
			}},
			assert: func(t *testing.T, inherited, addedLabels []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"kind/bug", "kind/documentation"}, inherited)
				require.Equal(t, []string{"kind/bug", "kind/documentation"}, addedLabels)
			},
		},
		{
			name: "AddLabelsToIssue error propagates",
			cfg: config{PullRequests: &pullRequestsConfig{
				InheritedLabelPrefixes: []string{"kind"},
			}},
			issueNumber: 99,
			linkedIssue: &github.Issue{Labels: []*github.Label{
				{Name: github.Ptr("kind/bug")},
			}},
			addLabelsErr: errors.New("network error"),
			assert: func(t *testing.T, inherited, addedLabels []string, err error) {
				require.ErrorContains(t, err, "error adding inherited labels")
				require.Empty(t, inherited)
				require.Equal(t, []string{"kind/bug"}, addedLabels)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var added []string
			issuesClient := &fakeIssuesClient{
				GetFn: func(
					context.Context, string, string, int,
				) (*github.Issue, *github.Response, error) {
					if testCase.linkedIssueErr != nil {
						return nil, nil, testCase.linkedIssueErr
					}
					if testCase.linkedIssue != nil {
						return testCase.linkedIssue, nil, nil
					}
					return &github.Issue{}, nil, nil
				},
				AddLabelsToIssueFn: func(
					_ context.Context,
					_, _ string,
					_ int,
					labels []string,
				) ([]*github.Label, *github.Response, error) {
					added = append(added, labels...)
					return nil, nil, testCase.addLabelsErr
				},
			}
			h := &prHandler{
				repoContext: repoContext{
					cfg:          testCase.cfg,
					owner:        "akuity",
					repo:         "kargo",
					issuesClient: issuesClient,
				},
			}
			inherited, err := h.inheritLabels(
				t.Context(), 1, testCase.issueNumber,
			)
			testCase.assert(t, inherited, added, err)
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
