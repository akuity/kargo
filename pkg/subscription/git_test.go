package subscription

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/controller/git/commit"
	"github.com/akuity/kargo/pkg/credentials"
)

// fakeSelector is a manual fake implementation of commit.Selector for testing
// the gitSubscriber's discovery orchestration.
type fakeSelector struct {
	listRefsFn func(context.Context) (*kargoapi.GitDiscoveryRefs, error)
	selectFn   func(context.Context) ([]kargoapi.DiscoveredCommit, error)
}

func (f *fakeSelector) MatchesRef(string) bool     { return false }
func (f *fakeSelector) MatchesPaths([]string) bool { return false }

func (f *fakeSelector) ListRefs(
	ctx context.Context,
) (*kargoapi.GitDiscoveryRefs, error) {
	return f.listRefsFn(ctx)
}

func (f *fakeSelector) Select(
	ctx context.Context,
) ([]kargoapi.DiscoveredCommit, error) {
	return f.selectFn(ctx)
}

func Test_gitSubscriber_DiscoverArtifacts(t *testing.T) {
	const (
		repoURL  = "https://example.com/repo"
		otherURL = "https://example.com/other"
	)
	prevCommits := []kargoapi.DiscoveredCommit{{ID: "old"}}
	freshCommits := []kargoapi.DiscoveredCommit{{ID: "new"}}
	observed := &kargoapi.GitDiscoveryRefs{BranchHead: "abc"}

	testCases := []struct {
		name        string
		last        any
		listRefs    *kargoapi.GitDiscoveryRefs
		listRefsErr error
		assert      func(t *testing.T, res kargoapi.GitDiscoveryResult, selectCalled bool)
	}{
		{
			name:     "refs unchanged skips clone and reuses prior commits",
			last:     kargoapi.GitDiscoveryResult{RepoURL: repoURL, Commits: prevCommits, ObservedRefs: observed},
			listRefs: &kargoapi.GitDiscoveryRefs{BranchHead: "abc"},
			assert: func(t *testing.T, res kargoapi.GitDiscoveryResult, selectCalled bool) {
				require.False(t, selectCalled, "Select must not run when refs are unchanged")
				require.Equal(t, prevCommits, res.Commits)
				require.Equal(t, observed, res.ObservedRefs)
			},
		},
		{
			name:     "refs changed falls through to selection",
			last:     kargoapi.GitDiscoveryResult{RepoURL: repoURL, Commits: prevCommits, ObservedRefs: observed},
			listRefs: &kargoapi.GitDiscoveryRefs{BranchHead: "def"},
			assert: func(t *testing.T, res kargoapi.GitDiscoveryResult, selectCalled bool) {
				require.True(t, selectCalled)
				require.Equal(t, freshCommits, res.Commits)
				require.Equal(t, &kargoapi.GitDiscoveryRefs{BranchHead: "def"}, res.ObservedRefs)
			},
		},
		{
			// A prior result for a different repo (e.g. were pairing ever wrong)
			// must never be reused, even if its observed refs happen to match.
			name:     "mismatched prior RepoURL falls through to selection",
			last:     kargoapi.GitDiscoveryResult{RepoURL: otherURL, Commits: prevCommits, ObservedRefs: observed},
			listRefs: &kargoapi.GitDiscoveryRefs{BranchHead: "abc"},
			assert: func(t *testing.T, res kargoapi.GitDiscoveryResult, selectCalled bool) {
				require.True(t, selectCalled)
				require.Equal(t, freshCommits, res.Commits)
			},
		},
		{
			name:     "no prior result falls through to selection",
			last:     nil,
			listRefs: observed,
			assert: func(t *testing.T, res kargoapi.GitDiscoveryResult, selectCalled bool) {
				require.True(t, selectCalled)
				require.Equal(t, freshCommits, res.Commits)
			},
		},
		{
			name:     "nil observation falls through even with prior result",
			last:     kargoapi.GitDiscoveryResult{RepoURL: repoURL, Commits: prevCommits, ObservedRefs: observed},
			listRefs: nil,
			assert: func(t *testing.T, res kargoapi.GitDiscoveryResult, selectCalled bool) {
				require.True(t, selectCalled)
				require.Equal(t, freshCommits, res.Commits)
				require.Nil(t, res.ObservedRefs)
			},
		},
		{
			name:        "ls-remote error falls through to selection",
			last:        kargoapi.GitDiscoveryResult{RepoURL: repoURL, Commits: prevCommits, ObservedRefs: observed},
			listRefsErr: errors.New("network down"),
			assert: func(t *testing.T, res kargoapi.GitDiscoveryResult, selectCalled bool) {
				require.True(t, selectCalled)
				require.Equal(t, freshCommits, res.Commits)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			selectCalled := false
			g := &gitSubscriber{
				credentialsDB: &credentials.FakeDB{},
				newSelectorFn: func(
					context.Context,
					kargoapi.GitSubscription,
					*git.RepoCredentials,
				) (commit.Selector, error) {
					return &fakeSelector{
						listRefsFn: func(context.Context) (*kargoapi.GitDiscoveryRefs, error) {
							return testCase.listRefs, testCase.listRefsErr
						},
						selectFn: func(context.Context) ([]kargoapi.DiscoveredCommit, error) {
							selectCalled = true
							return freshCommits, nil
						},
					}, nil
				},
			}
			res, err := g.DiscoverArtifacts(
				t.Context(),
				"fake-project",
				kargoapi.RepoSubscription{Git: &kargoapi.GitSubscription{RepoURL: repoURL}},
				testCase.last,
			)
			require.NoError(t, err)
			result, ok := res.(kargoapi.GitDiscoveryResult)
			require.True(t, ok)
			require.Equal(t, repoURL, result.RepoURL)
			testCase.assert(t, result, selectCalled)
		})
	}
}

func Test_gitRefsEqual(t *testing.T) {
	testCases := []struct {
		name  string
		a     *kargoapi.GitDiscoveryRefs
		b     *kargoapi.GitDiscoveryRefs
		equal bool
	}{
		{
			name:  "both nil",
			equal: false, // a nil observation never short-circuits
		},
		{
			name:  "one nil",
			a:     &kargoapi.GitDiscoveryRefs{BranchHead: "abc"},
			equal: false,
		},
		{
			name:  "equal branch heads",
			a:     &kargoapi.GitDiscoveryRefs{BranchHead: "abc"},
			b:     &kargoapi.GitDiscoveryRefs{BranchHead: "abc"},
			equal: true,
		},
		{
			name:  "different branch heads",
			a:     &kargoapi.GitDiscoveryRefs{BranchHead: "abc"},
			b:     &kargoapi.GitDiscoveryRefs{BranchHead: "def"},
			equal: false,
		},
		{
			name: "equal tag sets",
			a: &kargoapi.GitDiscoveryRefs{Tags: []kargoapi.DiscoveredRef{
				{Name: "v1.0.0", ID: "a"},
				{Name: "v1.1.0", ID: "b"},
			}},
			b: &kargoapi.GitDiscoveryRefs{Tags: []kargoapi.DiscoveredRef{
				{Name: "v1.0.0", ID: "a"},
				{Name: "v1.1.0", ID: "b"},
			}},
			equal: true,
		},
		{
			name: "tag repointed to a different commit",
			a: &kargoapi.GitDiscoveryRefs{Tags: []kargoapi.DiscoveredRef{
				{Name: "v1.0.0", ID: "a"},
			}},
			b: &kargoapi.GitDiscoveryRefs{Tags: []kargoapi.DiscoveredRef{
				{Name: "v1.0.0", ID: "z"},
			}},
			equal: false,
		},
		{
			name: "added tag",
			a: &kargoapi.GitDiscoveryRefs{Tags: []kargoapi.DiscoveredRef{
				{Name: "v1.0.0", ID: "a"},
			}},
			b: &kargoapi.GitDiscoveryRefs{Tags: []kargoapi.DiscoveredRef{
				{Name: "v1.0.0", ID: "a"},
				{Name: "v1.1.0", ID: "b"},
			}},
			equal: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.equal, gitRefsEqual(testCase.a, testCase.b))
		})
	}
}

func Test_gitSubscriber_ApplySubscriptionDefaults(t *testing.T) {
	s := &gitSubscriber{}

	t.Run("defaults empty fields", func(t *testing.T) {
		sub := &kargoapi.RepoSubscription{Git: &kargoapi.GitSubscription{}}
		err := s.ApplySubscriptionDefaults(t.Context(), sub)
		require.NoError(t, err)
		require.Equal(t, kargoapi.CommitSelectionStrategyNewestFromBranch, sub.Git.CommitSelectionStrategy)
		require.NotNil(t, sub.Git.StrictSemvers)
		require.True(t, *sub.Git.StrictSemvers)
		require.Equal(t, int64(20), sub.Git.DiscoveryLimit)
	})

	t.Run("preserves non-zero values", func(t *testing.T) {
		strict := false
		sub := &kargoapi.RepoSubscription{Git: &kargoapi.GitSubscription{
			CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
			StrictSemvers:           &strict,
			DiscoveryLimit:          7,
		}}
		err := s.ApplySubscriptionDefaults(t.Context(), sub)
		require.NoError(t, err)
		require.Equal(t, kargoapi.CommitSelectionStrategySemVer, sub.Git.CommitSelectionStrategy)
		require.NotNil(t, sub.Git.StrictSemvers)
		require.False(t, *sub.Git.StrictSemvers)
		require.Equal(t, int64(7), sub.Git.DiscoveryLimit)
	})

	t.Run("no-op on nil git", func(t *testing.T) {
		sub := &kargoapi.RepoSubscription{}
		// Should not panic
		err := s.ApplySubscriptionDefaults(t.Context(), sub)
		require.NoError(t, err)
		require.Nil(t, sub.Git)
	})
}

func Test_gitURLRegex(t *testing.T) {
	cases := map[string]bool{
		"":             false,
		":":            false,
		"/etc/passwd":  false,
		"//etc/passwd": false,
		"https:":       false,

		"https://not a url":                      false,
		"http://github.com/example/repo?foo=bar": true,
		"ssh://not a url":                        false,
		"ssh://github.com/example/repo?foo=bar":  true,
		"not even remotely a url":                false,
		// URLs of the form http[s]://[proxy-user:proxy-pass@]host.xz[:port][/path/to/repo[.git][/]]
		"https://github.com":          false,
		"https://github.com/":         true,
		"https://foo:bar@github.com":  false,
		"https://foo:bar@github.com/": true,
		// Variable features
		"https://github.com/example/repo":                  true,
		"https://github.com/example/repo/":                 true,
		"https://github.com/example/repo.git":              true,
		"https://github.com/example/repo.git/":             true,
		"https://localhost:8443/example/repo":              true,
		"https://localhost:8443/example/repo/":             true,
		"https://localhost:8443/example/repo.git":          true,
		"https://localhost:8443/example/repo.git/":         true,
		"https://foo:bar@github.com/example/repo":          true,
		"https://foo:bar@github.com/example/repo/":         true,
		"https://foo:bar@github.com/example/repo.git":      true,
		"https://foo:bar@github.com/example/repo.git/":     true,
		"https://foo:bar@localhost:8443/example/repo":      true,
		"https://foo:bar@localhost:8443/example/repo/":     true,
		"https://foo:bar@localhost:8443/example/repo.git":  true,
		"https://foo:bar@localhost:8443/example/repo.git/": true,
		// ssh forms
		"ssh://git.example.com":      false,
		"ssh://git.example.com/":     true,
		"ssh://git@git.example.com":  false,
		"ssh://git@git.example.com/": true,
		// Variable features
		"ssh://github.com/example/repo":              true,
		"ssh://github.com/example/repo/":             true,
		"ssh://github.com/example/repo.git":          true,
		"ssh://github.com/example/repo.git/":         true,
		"ssh://localhost:2222/example/repo":          true,
		"ssh://localhost:2222/example/repo/":         true,
		"ssh://localhost:2222/example/repo.git":      true,
		"ssh://localhost:2222/example/repo.git/":     true,
		"ssh://git@github.com/example/repo":          true,
		"ssh://git@github.com/example/repo/":         true,
		"ssh://git@github.com/example/repo.git":      true,
		"ssh://git@github.com/example/repo.git/":     true,
		"ssh://git@localhost:2222/example/repo":      true,
		"ssh://git@localhost:2222/example/repo/":     true,
		"ssh://git@localhost:2222/example/repo.git":  true,
		"ssh://git@localhost:2222/example/repo.git/": true,
		// SCP-style
		"git.example.com":     false,
		"git@git.example.com": false,
		// Variable features
		"github.com:example/repo":          false,
		"github.com:example/repo/":         false,
		"github.com:example/repo.git":      false,
		"github.com:example/repo.git/":     false,
		"git@github.com:example/repo":      true,
		"git@github.com:example/repo/":     true,
		"git@github.com:example/repo.git":  true,
		"git@github.com:example/repo.git/": true,
	}
	for input, expected := range cases {
		t.Run(input, func(t *testing.T) {
			require.Equal(t, expected, gitURLRegex.MatchString(input))
		})
	}
}

func Test_branchRegex(t *testing.T) {
	cases := map[string]bool{
		"":             false,
		"foo/bar":      true,
		"foo.bar":      true,
		"release-0.58": true,
		"/foo":         false,
		"foo/":         false,
		"foo//bar":     true,
		".foo":         false,
		"foo.":         false,
		"foo..bar":     true,
	}
	for input, expected := range cases {
		t.Run(input, func(t *testing.T) {
			require.Equal(t, expected, branchRegex.MatchString(input))
		})
	}
}

func Test_gitSubscriber_ValidateSubscription(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.GitSubscription
		assertions func(*testing.T, field.ErrorList)
	}{
		{
			name: "RepoURL empty",
			sub: kargoapi.GitSubscription{
				RepoURL: "",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "git.repoURL", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "RepoURL invalid format",
			sub: kargoapi.GitSubscription{
				RepoURL: "bogus",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "git.repoURL", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "Branch too long",
			sub: kargoapi.GitSubscription{
				RepoURL: "https://github.com/akuity/kargo.git",
				Branch:  "a" + strings.Repeat("b", 255),
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "git.branch", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "Branch invalid pattern",
			sub: kargoapi.GitSubscription{
				RepoURL: "https://github.com/akuity/kargo.git",
				Branch:  "-invalid",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "git.branch", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "CommitSelectionStrategy invalid",
			sub: kargoapi.GitSubscription{
				RepoURL:                 "https://github.com/akuity/kargo.git",
				CommitSelectionStrategy: "InvalidStrategy",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "git.commitSelectionStrategy", errs[0].Field)
				require.Equal(t, field.ErrorTypeNotSupported, errs[0].Type)
			},
		},
		{
			name: "invalid semverConstraint",
			sub: kargoapi.GitSubscription{
				RepoURL:          "https://github.com/akuity/kargo.git",
				SemverConstraint: "bogus",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "git.semverConstraint", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "DiscoveryLimit too small",
			sub: kargoapi.GitSubscription{
				RepoURL:        "https://github.com/akuity/kargo.git",
				DiscoveryLimit: 0,
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "git.discoveryLimit", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "DiscoveryLimit too large",
			sub: kargoapi.GitSubscription{
				RepoURL:        "https://github.com/akuity/kargo.git",
				DiscoveryLimit: 101,
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.NotNil(t, errs)
				require.True(t, len(errs) > 0)
				require.Equal(t, "git.discoveryLimit", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "valid",
			sub: kargoapi.GitSubscription{
				RepoURL:                 "https://github.com/akuity/kargo.git",
				Branch:                  "main",
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
				SemverConstraint:        "^1.0.0",
				DiscoveryLimit:          20,
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Nil(t, errs)
			},
		},
	}
	s := &gitSubscriber{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				s.ValidateSubscription(
					t.Context(),
					field.NewPath("git"),
					kargoapi.RepoSubscription{Git: &testCase.sub},
				),
			)
		})
	}
}
