package governance

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/require"
)

func Test_repoContext_isMaintainer(t *testing.T) {
	testCases := []struct {
		name        string
		cfg         config
		authorAssoc string
		login       string
		orgsClient  OrganizationsClient
		assert      func(*testing.T, bool, error)
	}{
		{
			name: "fast path: MEMBER matches",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER", "OWNER"},
			},
			authorAssoc: "MEMBER",
			login:       "alice",
			assert: func(t *testing.T, ok bool, err error) {
				require.NoError(t, err)
				require.True(t, ok)
			},
		},
		{
			name: "fast path: OWNER matches",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER", "OWNER"},
			},
			authorAssoc: "OWNER",
			login:       "alice",
			assert: func(t *testing.T, ok bool, err error) {
				require.NoError(t, err)
				require.True(t, ok)
			},
		},
		{
			name: "fast path: case insensitive",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER", "OWNER"},
			},
			authorAssoc: "member",
			login:       "alice",
			assert: func(t *testing.T, ok bool, err error) {
				require.NoError(t, err)
				require.True(t, ok)
			},
		},
		{
			name: "fast path miss + no orgsClient: no fallback",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER", "OWNER"},
			},
			authorAssoc: "CONTRIBUTOR",
			login:       "alice",
			orgsClient:  nil,
			assert: func(t *testing.T, ok bool, err error) {
				require.NoError(t, err)
				require.False(t, ok)
			},
		},
		{
			name: "fast path miss + empty login: no fallback",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER", "OWNER"},
			},
			authorAssoc: "CONTRIBUTOR",
			login:       "",
			orgsClient: &fakeOrganizationsClient{
				IsMemberFn: func(context.Context, string, string) (bool, *github.Response, error) {
					t.Fatal("IsMember should not be called without a login")
					return false, nil, nil
				},
			},
			assert: func(t *testing.T, ok bool, err error) {
				require.NoError(t, err)
				require.False(t, ok)
			},
		},
		{
			name: "fast path miss + MEMBER not configured: no fallback",
			cfg: config{
				MaintainerAssociations: []string{"OWNER"},
			},
			authorAssoc: "CONTRIBUTOR",
			login:       "alice",
			orgsClient: &fakeOrganizationsClient{
				IsMemberFn: func(context.Context, string, string) (bool, *github.Response, error) {
					t.Fatal("IsMember should not be called when MEMBER is not configured")
					return false, nil, nil
				},
			},
			assert: func(t *testing.T, ok bool, err error) {
				require.NoError(t, err)
				require.False(t, ok)
			},
		},
		{
			// Same exit point as the preceding cases — fast-path loop
			// finds nothing, !wantMember is true, no fallback.
			name: "no associations configured: nobody is a maintainer",
			cfg: config{
				MaintainerAssociations: nil,
			},
			authorAssoc: "MEMBER",
			login:       "alice",
			assert: func(t *testing.T, ok bool, err error) {
				require.NoError(t, err)
				require.False(t, ok)
			},
		},
		{
			name: "slow path: concealed member resolves as maintainer",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER", "OWNER"},
			},
			authorAssoc: "CONTRIBUTOR",
			login:       "alice",
			orgsClient: &fakeOrganizationsClient{
				IsMemberFn: func(_ context.Context, org, user string) (bool, *github.Response, error) {
					require.Equal(t, "akuity", org)
					require.Equal(t, "alice", user)
					return true, nil, nil
				},
			},
			assert: func(t *testing.T, ok bool, err error) {
				require.NoError(t, err)
				require.True(t, ok)
			},
		},
		{
			name: "slow path: non-member is not a maintainer",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER", "OWNER"},
			},
			authorAssoc: "CONTRIBUTOR",
			login:       "bob",
			orgsClient: &fakeOrganizationsClient{
				IsMemberFn: func(context.Context, string, string) (bool, *github.Response, error) {
					return false, nil, nil
				},
			},
			assert: func(t *testing.T, ok bool, err error) {
				require.NoError(t, err)
				require.False(t, ok)
			},
		},
		{
			name: "slow path: IsMember error propagates",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER", "OWNER"},
			},
			authorAssoc: "CONTRIBUTOR",
			login:       "alice",
			orgsClient: &fakeOrganizationsClient{
				IsMemberFn: func(context.Context, string, string) (bool, *github.Response, error) {
					return false, nil, errors.New("boom")
				},
			},
			assert: func(t *testing.T, ok bool, err error) {
				require.ErrorContains(t, err, "boom")
				require.False(t, ok)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rc := &repoContext{
				cfg:        testCase.cfg,
				owner:      "akuity",
				orgsClient: testCase.orgsClient,
			}
			ok, err := rc.isMaintainer(
				context.Background(),
				testCase.authorAssoc,
				testCase.login,
			)
			testCase.assert(t, ok, err)
		})
	}
}

func Test_repoContext_enforceRequiredLabels(t *testing.T) {
	testCases := []struct {
		name           string
		existingLabels map[string]struct{}
		prefixes       []string
		addLabelsErr   error
		assert         func(t *testing.T, addedLabels []string, err error)
	}{
		{
			name:           "all prefixes already present: no calls",
			existingLabels: map[string]struct{}{"kind/bug": {}, "priority/high": {}},
			prefixes:       []string{"kind", "priority"},
			assert: func(t *testing.T, addedLabels []string, err error) {
				require.NoError(t, err)
				require.Empty(t, addedLabels)
			},
		},
		{
			name:           "one prefix missing: adds the needs/ label",
			existingLabels: map[string]struct{}{"kind/bug": {}},
			prefixes:       []string{"kind", "priority"},
			assert: func(t *testing.T, addedLabels []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"needs/priority"}, addedLabels)
			},
		},
		{
			name:           "all prefixes missing: adds all needs/ labels in order",
			existingLabels: map[string]struct{}{},
			prefixes:       []string{"area", "kind", "priority"},
			assert: func(t *testing.T, addedLabels []string, err error) {
				require.NoError(t, err)
				require.Equal(t,
					[]string{"needs/area", "needs/kind", "needs/priority"},
					addedLabels,
				)
			},
		},
		{
			name:           "no prefixes configured: no calls",
			existingLabels: map[string]struct{}{},
			prefixes:       nil,
			assert: func(t *testing.T, addedLabels []string, err error) {
				require.NoError(t, err)
				require.Empty(t, addedLabels)
			},
		},
		{
			// AddLabelsToIssue fails on the first missing prefix; the loop
			// short-circuits and the error propagates without attempting
			// subsequent additions.
			name:           "AddLabelsToIssue error short-circuits and propagates",
			existingLabels: map[string]struct{}{},
			prefixes:       []string{"area", "kind"},
			addLabelsErr:   errors.New("network error"),
			assert: func(t *testing.T, addedLabels []string, err error) {
				require.ErrorContains(t, err, "error adding label")
				require.Equal(t, []string{"needs/area"}, addedLabels)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var added []string
			issuesClient := &fakeIssuesClient{
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
			rc := &repoContext{
				owner:        "akuity",
				repo:         "kargo",
				issuesClient: issuesClient,
			}
			err := rc.enforceRequiredLabels(
				t.Context(), 1, testCase.existingLabels, testCase.prefixes,
			)
			testCase.assert(t, added, err)
		})
	}
}

func Test_needsLabel(t *testing.T) {
	testCases := []struct {
		name           string
		prefix         string
		existingLabels map[string]struct{}
		expected       bool
	}{
		{
			name:           "label present",
			prefix:         "kind",
			existingLabels: map[string]struct{}{"kind/bug": {}},
			expected:       false,
		},
		{
			name:           "label missing",
			prefix:         "kind",
			existingLabels: map[string]struct{}{"priority/high": {}},
			expected:       true,
		},
		{
			name:           "no labels at all",
			prefix:         "kind",
			existingLabels: map[string]struct{}{},
			expected:       true,
		},
		{
			name:           "prefix without slash does not match",
			prefix:         "kind",
			existingLabels: map[string]struct{}{"kinder": {}},
			expected:       true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := needsLabel(testCase.prefix, testCase.existingLabels)
			require.Equal(t, testCase.expected, result)
		})
	}
}

func Test_repoContext_applyPRPolicy(t *testing.T) {
	testCases := []struct {
		name           string
		cfg            config
		prBody         string
		association    string
		senderLogin    string
		linkedIssue    *github.Issue // used when Get is called
		linkedIssueErr error
		assert         func(t *testing.T, addedLabels []string, err error)
	}{
		{
			// Defensive: handler is robust against a config where the PR
			// section is entirely absent — no-op, no error.
			name:   "nil PullRequests is a no-op",
			cfg:    config{},
			prBody: "no issue link",
			assert: func(t *testing.T, addedLabels []string, err error) {
				require.NoError(t, err)
				require.Empty(t, addedLabels)
			},
		},
		{
			// Exempt + OnPass configured: blocking outcomes are skipped,
			// but OnPass still fires (cleanup-style actions remain useful
			// for exempt PRs).
			name: "exempt PR with no linked issue: skips blocking, runs OnPass",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER"},
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{Maintainers: true},
					OnNoLinkedIssue: &onNoLinkedIssueConfig{
						Actions: []action{
							mustAction("addLabels", []string{"on-no-linked-issue"}),
						},
					},
					OnPass: &onPassConfig{
						Actions: []action{
							mustAction("addLabels", []string{"on-pass"}),
						},
					},
				},
			},
			prBody:      "no issue link",
			association: "MEMBER",
			assert: func(t *testing.T, addedLabels []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"on-pass"}, addedLabels)
			},
		},
		{
			name: "exempt PR, OnPass not configured: no-op",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER"},
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{Maintainers: true},
					OnNoLinkedIssue: &onNoLinkedIssueConfig{
						Actions: []action{
							mustAction("addLabels", []string{"on-no-linked-issue"}),
						},
					},
				},
			},
			prBody:      "no issue link",
			association: "MEMBER",
			assert: func(t *testing.T, addedLabels []string, err error) {
				require.NoError(t, err)
				require.Empty(t, addedLabels)
			},
		},
		{
			name: "no linked issue, not exempt: OnNoLinkedIssue fires",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					OnNoLinkedIssue: &onNoLinkedIssueConfig{
						Actions: []action{
							mustAction("addLabels", []string{"on-no-linked-issue"}),
						},
					},
				},
			},
			prBody:      "no issue link here",
			association: "NONE",
			assert: func(t *testing.T, addedLabels []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"on-no-linked-issue"}, addedLabels)
			},
		},
		{
			// OnNoLinkedIssue is the *blocking* outcome here; when it's
			// not configured, the policy is permissive — fall through to
			// OnPass instead of failing closed.
			name: "no linked issue, OnNoLinkedIssue not configured: falls through to OnPass",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					OnPass: &onPassConfig{
						Actions: []action{
							mustAction("addLabels", []string{"on-pass"}),
						},
					},
				},
			},
			prBody:      "no issue link",
			association: "NONE",
			assert: func(t *testing.T, addedLabels []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"on-pass"}, addedLabels)
			},
		},
		{
			name: "linked issue Get returns an error: propagates",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					OnBlockedIssue: &onBlockedIssueConfig{
						BlockingLabels: []string{"kind/proposal"},
					},
				},
			},
			prBody:         "Closes #99",
			association:    "NONE",
			linkedIssueErr: errors.New("network error"),
			assert: func(t *testing.T, addedLabels []string, err error) {
				require.ErrorContains(t, err, "error fetching linked issue")
				require.Empty(t, addedLabels)
			},
		},
		{
			name: "linked issue with blocking label: OnBlockedIssue fires",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					OnBlockedIssue: &onBlockedIssueConfig{
						BlockingLabels: []string{"kind/proposal"},
						Actions: []action{
							mustAction("addLabels", []string{"on-blocked-issue"}),
						},
					},
				},
			},
			prBody:      "Closes #99",
			association: "NONE",
			linkedIssue: &github.Issue{Labels: []*github.Label{{Name: github.Ptr("kind/proposal")}}},
			assert: func(t *testing.T, addedLabels []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"on-blocked-issue"}, addedLabels)
			},
		},
		{
			name: "linked issue without blocking label: OnPass fires",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					OnBlockedIssue: &onBlockedIssueConfig{
						BlockingLabels: []string{"kind/proposal"},
					},
					OnPass: &onPassConfig{
						Actions: []action{
							mustAction("addLabels", []string{"on-pass"}),
						},
					},
				},
			},
			prBody:      "Closes #99",
			association: "NONE",
			linkedIssue: &github.Issue{Labels: []*github.Label{{Name: github.Ptr("kind/bug")}}},
			assert: func(t *testing.T, addedLabels []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"on-pass"}, addedLabels)
			},
		},
		{
			// OnBlockedIssue not configured: linked-issue path is a no-op
			// for the blocking check; OnPass runs.
			name: "linked issue, OnBlockedIssue not configured: OnPass fires",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					OnPass: &onPassConfig{
						Actions: []action{
							mustAction("addLabels", []string{"on-pass"}),
						},
					},
				},
			},
			prBody:      "Closes #99",
			association: "NONE",
			linkedIssue: &github.Issue{Labels: []*github.Label{{Name: github.Ptr("kind/proposal")}}},
			assert: func(t *testing.T, addedLabels []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"on-pass"}, addedLabels)
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
					return nil, nil, nil
				},
			}
			rc := &repoContext{
				cfg:          testCase.cfg,
				owner:        "akuity",
				repo:         "kargo",
				issuesClient: issuesClient,
			}
			pr := &github.PullRequest{
				Number:            github.Ptr(1),
				Body:              github.Ptr(testCase.prBody),
				User:              &github.User{Login: github.Ptr("author")},
				AuthorAssociation: github.Ptr(testCase.association),
			}
			err := rc.applyPRPolicy(t.Context(), pr, testCase.senderLogin)
			testCase.assert(t, added, err)
		})
	}
}

func Test_repoContext_isPRExempt(t *testing.T) {
	testCases := []struct {
		name        string
		cfg         config
		authorLogin string
		association string
		senderLogin string
		additions   int
		deletions   int
		// listFiles, when set, populates fakePullRequestsClient.ListFilesFn.
		// nil means ListFiles returns no files (or isn't expected to be
		// called).
		listFiles []string
		// listFilesErr, when set, makes the fake return this error.
		listFilesErr error
		// isMembers maps username → membership state returned by the fake
		// OrganizationsClient. nil means IsMember should not be called.
		isMembers map[string]bool
		// isMemberErr, when set, makes the fake return this error.
		isMemberErr error
		expected    bool
		expectErr   bool
	}{
		{
			name:        "nil PullRequests not exempt",
			cfg:         config{},
			authorLogin: "kent",
			association: "MEMBER",
			senderLogin: "kent",
			expected:    false,
		},
		{
			name: "nil Exemptions not exempt",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER"},
				PullRequests:           &pullRequestsConfig{},
			},
			authorLogin: "kent",
			association: "MEMBER",
			senderLogin: "kent",
			expected:    false,
		},
		{
			name: "maintainer author exempt via fast path",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER"},
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{Maintainers: true},
				},
			},
			authorLogin: "kent",
			association: "MEMBER",
			senderLogin: "kent",
			expected:    true,
		},
		{
			name: "concealed-member author exempt via IsMember fallback",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER"},
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{Maintainers: true},
				},
			},
			authorLogin: "frankenstein",
			// Concealed members appear as CONTRIBUTOR in webhook payloads
			// even when the App holds Organization Members: Read.
			association: "CONTRIBUTOR",
			senderLogin: "frankenstein",
			isMembers:   map[string]bool{"frankenstein": true},
			expected:    true,
		},
		{
			name: "concealed non-member author not exempt",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER"},
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{Maintainers: true},
				},
			},
			authorLogin: "external",
			association: "CONTRIBUTOR",
			senderLogin: "external",
			isMembers:   map[string]bool{"external": false},
			expected:    false,
		},
		{
			name: "IsMember error on author propagates",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER"},
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{Maintainers: true},
				},
			},
			authorLogin: "frankenstein",
			association: "CONTRIBUTOR",
			senderLogin: "frankenstein",
			isMemberErr: errors.New("network error"),
			expected:    false,
			expectErr:   true,
		},
		{
			name: "sender-as-maintainer escape hatch: non-maintainer author, maintainer sender",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER"},
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{Maintainers: true},
				},
			},
			authorLogin: "external",
			association: "CONTRIBUTOR",
			senderLogin: "kent",
			isMembers: map[string]bool{
				"external": false,
				"kent":     true,
			},
			expected: true,
		},
		{
			name: "sender check skipped when sender == author",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER"},
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{Maintainers: true},
				},
			},
			authorLogin: "external",
			association: "CONTRIBUTOR",
			senderLogin: "external",
			isMembers:   map[string]bool{"external": false},
			expected:    false,
		},
		{
			name: "non-maintainer author + non-maintainer sender not exempt",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER"},
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{Maintainers: true},
				},
			},
			authorLogin: "external",
			association: "CONTRIBUTOR",
			senderLogin: "another-external",
			isMembers: map[string]bool{
				"external":         false,
				"another-external": false,
			},
			expected: false,
		},
		{
			// The `if ex.Maintainers` guard's false branch — the entire
			// block is skipped. With no other criteria enabled, we fall
			// all the way through to the final `return false`.
			name: "maintainer exemption disabled: maintainer author not exempt",
			cfg: config{
				MaintainerAssociations: []string{"MEMBER"},
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{Maintainers: false},
				},
			},
			authorLogin: "kent",
			association: "MEMBER",
			senderLogin: "kent",
			expected:    false,
		},
		{
			name: "bot exempt when enabled",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{Bots: true},
				},
			},
			authorLogin: "dependabot[bot]",
			association: "NONE",
			senderLogin: "dependabot[bot]",
			expected:    true,
		},
		{
			name: "bot not exempt when disabled",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{Bots: false},
				},
			},
			authorLogin: "dependabot[bot]",
			association: "NONE",
			senderLogin: "dependabot[bot]",
			expected:    false,
		},
		{
			name: "size exempt — total at limit",
			cfg: config{
				PullRequests: &pullRequestsConfig{
					Exemptions: &exemptionsConfig{MaxChangedLines: 5},
				},
			},
			authorLogin: "random-user",
			association: "NONE",
			senderLogin: "random-user",
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
			authorLogin: "random-user",
			association: "NONE",
			senderLogin: "random-user",
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
			authorLogin: "random-user",
			association: "NONE",
			senderLogin: "random-user",
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
			authorLogin: "random-user",
			association: "NONE",
			senderLogin: "random-user",
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
			authorLogin: "random-user",
			association: "NONE",
			senderLogin: "random-user",
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
			authorLogin: "random-user",
			association: "NONE",
			senderLogin: "random-user",
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
			authorLogin: "random-user",
			association: "NONE",
			senderLogin: "random-user",
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
			authorLogin:  "random-user",
			association:  "NONE",
			senderLogin:  "random-user",
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
			authorLogin: "random-user",
			association: "NONE",
			senderLogin: "random-user",
			isMembers:   map[string]bool{"random-user": false},
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
			orgsClient := &fakeOrganizationsClient{
				IsMemberFn: func(
					_ context.Context,
					_ string,
					user string,
				) (bool, *github.Response, error) {
					if testCase.isMemberErr != nil {
						return false, nil, testCase.isMemberErr
					}
					member, ok := testCase.isMembers[user]
					if !ok {
						t.Fatalf("unexpected IsMember call for %q", user)
					}
					return member, nil, nil
				},
			}
			pr := &github.PullRequest{
				Number:            github.Ptr(1),
				User:              &github.User{Login: github.Ptr(testCase.authorLogin)},
				AuthorAssociation: github.Ptr(testCase.association),
				Additions:         github.Ptr(testCase.additions),
				Deletions:         github.Ptr(testCase.deletions),
			}
			rc := &repoContext{
				cfg:        testCase.cfg,
				owner:      "akuity",
				repo:       "kargo",
				prsClient:  prsClient,
				orgsClient: orgsClient,
			}
			result, err := rc.isPRExempt(t.Context(), pr, testCase.senderLogin)
			if testCase.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, testCase.expected, result)
		})
	}
}
