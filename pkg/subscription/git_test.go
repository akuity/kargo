package subscription

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/controller/git/commit"
	"github.com/akuity/kargo/pkg/credentials"
	libExec "github.com/akuity/kargo/pkg/exec"
	"github.com/akuity/kargo/pkg/telemetry"
	"github.com/akuity/kargo/pkg/telemetry/telemetrytest"
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
		name       string
		subscriber *gitSubscriber
		last       any
		assertions func(*testing.T, any, error)
	}{
		{
			name: "refs unchanged skips clone and reuses prior commits",
			subscriber: &gitSubscriber{
				credentialsDB: &credentials.FakeDB{},
				newSelectorFn: func(
					context.Context,
					kargoapi.GitSubscription,
					*git.RepoCredentials,
				) (commit.Selector, error) {
					return &fakeSelector{
						listRefsFn: func(context.Context) (*kargoapi.GitDiscoveryRefs, error) {
							return &kargoapi.GitDiscoveryRefs{BranchHead: "abc"}, nil
						},
						selectFn: func(context.Context) ([]kargoapi.DiscoveredCommit, error) {
							require.Fail(t, "Select must not run when refs are unchanged")
							return nil, nil
						},
					}, nil
				},
			},
			last: kargoapi.GitDiscoveryResult{
				RepoURL:      repoURL,
				Commits:      prevCommits,
				ObservedRefs: observed,
			},
			assertions: func(t *testing.T, res any, err error) {
				require.NoError(t, err)
				result, ok := res.(kargoapi.GitDiscoveryResult)
				require.True(t, ok)
				require.Equal(t, repoURL, result.RepoURL)
				require.Equal(t, prevCommits, result.Commits)
				require.Equal(t, observed, result.ObservedRefs)
				// Even the path that short-circuits the clone must stamp the
				// subscription's name on the result.
				require.Equal(t, "fake-sub", result.SubscriptionName)
			},
		},
		{
			name: "refs changed falls through to selection",
			subscriber: &gitSubscriber{
				credentialsDB: &credentials.FakeDB{},
				newSelectorFn: func(
					context.Context,
					kargoapi.GitSubscription,
					*git.RepoCredentials,
				) (commit.Selector, error) {
					return &fakeSelector{
						listRefsFn: func(context.Context) (*kargoapi.GitDiscoveryRefs, error) {
							return &kargoapi.GitDiscoveryRefs{BranchHead: "def"}, nil
						},
						selectFn: func(context.Context) ([]kargoapi.DiscoveredCommit, error) {
							return freshCommits, nil
						},
					}, nil
				},
			},
			last: kargoapi.GitDiscoveryResult{
				RepoURL:      repoURL,
				Commits:      prevCommits,
				ObservedRefs: observed,
			},
			assertions: func(t *testing.T, res any, err error) {
				require.NoError(t, err)
				result, ok := res.(kargoapi.GitDiscoveryResult)
				require.True(t, ok)
				require.Equal(t, repoURL, result.RepoURL)
				require.Equal(t, freshCommits, result.Commits)
				require.Equal(t, &kargoapi.GitDiscoveryRefs{BranchHead: "def"}, result.ObservedRefs)
				require.Equal(t, "fake-sub", result.SubscriptionName)
			},
		},
		{
			// A prior result for a different repo (e.g. were pairing ever wrong)
			// must never be reused, even if its observed refs happen to match.
			name: "mismatched prior RepoURL falls through to selection",
			subscriber: &gitSubscriber{
				credentialsDB: &credentials.FakeDB{},
				newSelectorFn: func(
					context.Context,
					kargoapi.GitSubscription,
					*git.RepoCredentials,
				) (commit.Selector, error) {
					return &fakeSelector{
						listRefsFn: func(context.Context) (*kargoapi.GitDiscoveryRefs, error) {
							return &kargoapi.GitDiscoveryRefs{BranchHead: "abc"}, nil
						},
						selectFn: func(context.Context) ([]kargoapi.DiscoveredCommit, error) {
							return freshCommits, nil
						},
					}, nil
				},
			},
			last: kargoapi.GitDiscoveryResult{
				RepoURL:      otherURL,
				Commits:      prevCommits,
				ObservedRefs: observed,
			},
			assertions: func(t *testing.T, res any, err error) {
				require.NoError(t, err)
				result, ok := res.(kargoapi.GitDiscoveryResult)
				require.True(t, ok)
				require.Equal(t, repoURL, result.RepoURL)
				require.Equal(t, freshCommits, result.Commits)
				require.Equal(t, "fake-sub", result.SubscriptionName)
			},
		},
		{
			name: "no prior result falls through to selection",
			subscriber: &gitSubscriber{
				credentialsDB: &credentials.FakeDB{},
				newSelectorFn: func(
					context.Context,
					kargoapi.GitSubscription,
					*git.RepoCredentials,
				) (commit.Selector, error) {
					return &fakeSelector{
						listRefsFn: func(context.Context) (*kargoapi.GitDiscoveryRefs, error) {
							return observed, nil
						},
						selectFn: func(context.Context) ([]kargoapi.DiscoveredCommit, error) {
							return freshCommits, nil
						},
					}, nil
				},
			},
			assertions: func(t *testing.T, res any, err error) {
				require.NoError(t, err)
				result, ok := res.(kargoapi.GitDiscoveryResult)
				require.True(t, ok)
				require.Equal(t, repoURL, result.RepoURL)
				require.Equal(t, freshCommits, result.Commits)
				require.Equal(t, "fake-sub", result.SubscriptionName)
			},
		},
		{
			name: "nil observation falls through even with prior result",
			subscriber: &gitSubscriber{
				credentialsDB: &credentials.FakeDB{},
				newSelectorFn: func(
					context.Context,
					kargoapi.GitSubscription,
					*git.RepoCredentials,
				) (commit.Selector, error) {
					return &fakeSelector{
						listRefsFn: func(context.Context) (*kargoapi.GitDiscoveryRefs, error) {
							return nil, nil
						},
						selectFn: func(context.Context) ([]kargoapi.DiscoveredCommit, error) {
							return freshCommits, nil
						},
					}, nil
				},
			},
			last: kargoapi.GitDiscoveryResult{
				RepoURL:      repoURL,
				Commits:      prevCommits,
				ObservedRefs: observed,
			},
			assertions: func(t *testing.T, res any, err error) {
				require.NoError(t, err)
				result, ok := res.(kargoapi.GitDiscoveryResult)
				require.True(t, ok)
				require.Equal(t, repoURL, result.RepoURL)
				require.Equal(t, freshCommits, result.Commits)
				require.Nil(t, result.ObservedRefs)
				require.Equal(t, "fake-sub", result.SubscriptionName)
			},
		},
		{
			name: "ls-remote error falls through to selection",
			subscriber: &gitSubscriber{
				credentialsDB: &credentials.FakeDB{},
				newSelectorFn: func(
					context.Context,
					kargoapi.GitSubscription,
					*git.RepoCredentials,
				) (commit.Selector, error) {
					return &fakeSelector{
						listRefsFn: func(context.Context) (*kargoapi.GitDiscoveryRefs, error) {
							return nil, errors.New("network down")
						},
						selectFn: func(context.Context) ([]kargoapi.DiscoveredCommit, error) {
							return freshCommits, nil
						},
					}, nil
				},
			},
			last: kargoapi.GitDiscoveryResult{
				RepoURL:      repoURL,
				Commits:      prevCommits,
				ObservedRefs: observed,
			},
			assertions: func(t *testing.T, res any, err error) {
				require.NoError(t, err)
				result, ok := res.(kargoapi.GitDiscoveryResult)
				require.True(t, ok)
				require.Equal(t, repoURL, result.RepoURL)
				require.Equal(t, freshCommits, result.Commits)
				require.Equal(t, "fake-sub", result.SubscriptionName)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			res, err := testCase.subscriber.DiscoverArtifacts(
				t.Context(),
				"fake-project",
				kargoapi.RepoSubscription{
					Name: "fake-sub",
					Git:  &kargoapi.GitSubscription{RepoURL: repoURL},
				},
				testCase.last,
			)
			testCase.assertions(t, res, err)
		})
	}
}

func Test_gitSubscriber_DiscoverArtifacts_tracing(t *testing.T) {
	const repoURL = "https://example.com/repo"
	observed := &kargoapi.GitDiscoveryRefs{BranchHead: "abc"}
	last := kargoapi.GitDiscoveryResult{RepoURL: repoURL, ObservedRefs: observed}
	newSubscriber := func(listRefsErr, selectErr error) *gitSubscriber {
		return &gitSubscriber{
			credentialsDB: &credentials.FakeDB{},
			newSelectorFn: func(
				context.Context,
				kargoapi.GitSubscription,
				*git.RepoCredentials,
			) (commit.Selector, error) {
				return &fakeSelector{
					listRefsFn: func(context.Context) (*kargoapi.GitDiscoveryRefs, error) {
						return observed, listRefsErr
					},
					selectFn: func(context.Context) ([]kargoapi.DiscoveredCommit, error) {
						return nil, selectErr
					},
				}, nil
			},
		}
	}
	testCases := []struct {
		name       string
		subscriber *gitSubscriber
		last       any
		cancel     bool
		wantErr    bool
		assert     func(*testing.T, []sdktrace.ReadOnlySpan)
	}{
		{
			name:       "refs unchanged",
			subscriber: newSubscriber(nil, nil),
			last:       last,
			assert: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				telemetrytest.EndedSpan(t, spans, "Git ls-remote")
				for _, s := range spans {
					require.NotEqual(t, "Git select commits", s.Name())
				}
				require.Contains(
					t,
					telemetrytest.EndedSpan(t, spans, "parent").Attributes(),
					telemetry.CloneSkippedKey.Bool(true),
				)
			},
		},
		{
			name:       "refs changed",
			subscriber: newSubscriber(nil, nil),
			assert: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				telemetrytest.EndedSpan(t, spans, "Git select commits")
				require.Contains(
					t,
					telemetrytest.EndedSpan(t, spans, "parent").Attributes(),
					telemetry.CloneSkippedKey.Bool(false),
				)
			},
		},
		{
			// A failed ls-remote falls back to cloning: not the span's error.
			name:       "ls-remote fails",
			subscriber: newSubscriber(errors.New("something went wrong"), nil),
			last:       last,
			assert: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				lsRemote := telemetrytest.EndedSpan(t, spans, "Git ls-remote")
				require.Equal(t, codes.Unset, lsRemote.Status().Code)
				require.Empty(t, lsRemote.Events())
				telemetrytest.EndedSpan(t, spans, "Git select commits")
			},
		},
		{
			// A git killed on cancellation fails with an ExitError.
			name:       "select canceled",
			subscriber: newSubscriber(nil, &libExec.ExitError{ExitCode: 143}),
			cancel:     true,
			wantErr:    true,
			assert: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				span := telemetrytest.EndedSpan(t, spans, "Git select commits")
				require.Equal(t, codes.Unset, span.Status().Code)
				require.Empty(t, span.Events())
			},
		},
		{
			name:       "select fails",
			subscriber: newSubscriber(nil, errors.New("something went wrong")),
			wantErr:    true,
			assert: func(t *testing.T, spans []sdktrace.ReadOnlySpan) {
				span := telemetrytest.EndedSpan(t, spans, "Git select commits")
				require.Equal(t, codes.Error, span.Status().Code)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Not parallel: installs a global tracer provider.
			ended := telemetrytest.RecordSpans(t)
			ctx, parent := otel.Tracer("test").Start(t.Context(), "parent")
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			if testCase.cancel {
				cancel()
			}
			_, err := testCase.subscriber.DiscoverArtifacts(
				ctx,
				"fake-project",
				kargoapi.RepoSubscription{
					Name: "fake-sub",
					Git:  &kargoapi.GitSubscription{RepoURL: repoURL},
				},
				testCase.last,
			)
			if testCase.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			parent.End()
			testCase.assert(t, ended())
		})
	}
}

func Test_gitSubscriber_DiscoverArtifacts_categoryNotEnabled(t *testing.T) {
	const repoURL = "https://example.com/repo"
	observed := &kargoapi.GitDiscoveryRefs{BranchHead: "abc"}
	subscriber := &gitSubscriber{
		credentialsDB: &credentials.FakeDB{},
		newSelectorFn: func(
			context.Context,
			kargoapi.GitSubscription,
			*git.RepoCredentials,
		) (commit.Selector, error) {
			return &fakeSelector{
				listRefsFn: func(context.Context) (*kargoapi.GitDiscoveryRefs, error) {
					return observed, nil
				},
				selectFn: func(context.Context) ([]kargoapi.DiscoveredCommit, error) {
					return nil, nil
				},
			}, nil
		},
	}
	testCases := []struct {
		name string
		last any
	}{
		{
			name: "refs unchanged",
			last: kargoapi.GitDiscoveryResult{RepoURL: repoURL, ObservedRefs: observed},
		},
		{name: "refs changed"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Not parallel: installs a global tracer provider.
			ended := telemetrytest.RecordSpans(t)
			// discovery is off by default.
			telemetrytest.SetCategories(t, "default")
			ctx, parent := otel.Tracer("test").Start(t.Context(), "parent")
			_, err := subscriber.DiscoverArtifacts(
				ctx,
				"fake-project",
				kargoapi.RepoSubscription{
					Name: "fake-sub",
					Git:  &kargoapi.GitSubscription{RepoURL: repoURL},
				},
				testCase.last,
			)
			require.NoError(t, err)
			parent.End()
			// No spans of its own, and nothing on the caller's span.
			spans := ended()
			require.Len(t, spans, 1)
			require.Equal(t, "parent", spans[0].Name())
			require.Empty(t, spans[0].Attributes())
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

func Test_gitSchema_repoURL(t *testing.T) {
	cases := map[string]bool{
		"":             false,
		":":            false,
		"/etc/passwd":  false,
		"//etc/passwd": false,
		"https:":       false,

		"https://not a url":                      false,
		"http://github.com/example/repo?foo=bar": true,
		"ssh://not a url":                        false,
		"ssh://github.com/example/repo?foo=bar":  false,
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
		// SSH and SCP-style URLs are not supported
		"ssh://git@github.com/example/repo":     false,
		"ssh://git@localhost:2222/example/repo": false,
		"git@github.com:example/repo":           false,
		"git@github.com:example/repo.git":       false,
		"github.com:example/repo":               false,
	}
	for input, expected := range cases {
		t.Run(input, func(t *testing.T) {
			require.Equal(t, expected, schemaAccepts(t, "git", "repoURL", input))
		})
	}
}

func Test_gitSchema_branch(t *testing.T) {
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
			require.Equal(t, expected, schemaAccepts(t, "git", "branch", input))
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
			name: "invalid semverConstraint",
			sub: kargoapi.GitSubscription{
				RepoURL:          "https://github.com/akuity/kargo.git",
				SemverConstraint: "bogus",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 1)
				require.Equal(t, "git.semverConstraint", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "invalid expressionFilter",
			sub: kargoapi.GitSubscription{
				RepoURL:          "https://github.com/akuity/kargo.git",
				ExpressionFilter: "(1 + 2",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 1)
				require.Equal(t, "git.expressionFilter", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
			},
		},
		{
			name: "invalid semverConstraint and expressionFilter",
			sub: kargoapi.GitSubscription{
				RepoURL:          "https://github.com/akuity/kargo.git",
				SemverConstraint: "bogus",
				ExpressionFilter: "(1 + 2",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 2)
				require.Equal(t, "git.semverConstraint", errs[0].Field)
				require.Equal(t, "git.expressionFilter", errs[1].Field)
			},
		},
		{
			name: "expressionFilter with misspelled variable",
			sub: kargoapi.GitSubscription{
				RepoURL:          "https://github.com/akuity/kargo.git",
				ExpressionFilter: "authr == 'kent'",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 1)
				require.Equal(t, "git.expressionFilter", errs[0].Field)
				require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)
				require.Contains(t, errs[0].Detail, "unknown name authr")
			},
		},
		{
			name: "expressionFilter with variable unavailable to strategy",
			sub: kargoapi.GitSubscription{
				RepoURL:                 "https://github.com/akuity/kargo.git",
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
				ExpressionFilter:        "commitDate > date('2025-01-01')",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Len(t, errs, 1)
				require.Equal(t, "git.expressionFilter", errs[0].Field)
				require.Contains(t, errs[0].Detail, "unknown name commitDate")
			},
		},
		{
			name: "valid tag-based expressionFilter",
			sub: kargoapi.GitSubscription{
				RepoURL:                 "https://github.com/akuity/kargo.git",
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
				ExpressionFilter:        "creatorDate > date('2025-01-01')",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
			},
		},
		{
			name: "valid",
			sub: kargoapi.GitSubscription{
				RepoURL:          "https://github.com/akuity/kargo.git",
				SemverConstraint: "^1.0.0",
				ExpressionFilter: "!(subject contains 'skip-ci')",
			},
			assertions: func(t *testing.T, errs field.ErrorList) {
				require.Empty(t, errs)
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
