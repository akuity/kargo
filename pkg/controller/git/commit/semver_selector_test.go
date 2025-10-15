package commit

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
)

func TestNewSemverSelectorTest(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.GitSubscription
		assertions func(*testing.T, Selector, error)
	}{
		{
			name: "error building tag based selector",
			sub: kargoapi.GitSubscription{
				ExpressionFilter: "(1 + 2", // This will force an error
			},
			assertions: func(t *testing.T, _ Selector, err error) {
				require.ErrorContains(t, err, "error building tag based selector")
			},
		},
		{
			name: "error parsing semver constraint",
			sub:  kargoapi.GitSubscription{SemverConstraint: "invalid"},
			assertions: func(t *testing.T, _ Selector, err error) {
				require.ErrorContains(t, err, "error parsing semver constraint")
			},
		},
		{
			name: "success",
			sub:  kargoapi.GitSubscription{SemverConstraint: "^v1.0.0"},
			assertions: func(t *testing.T, sel Selector, err error) {
				require.NoError(t, err)
				s, ok := sel.(*semverSelector)
				require.True(t, ok)
				require.NotNil(t, s.tagBasedSelector)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newSemverSelector(testCase.sub, nil)
			testCase.assertions(t, s, err)
		})
	}
}

func Test_semVerSelector_MatchesRef(t *testing.T) {
	testCases := []struct {
		name        string
		selector    *semverSelector
		ref         string
		shouldMatch bool
	}{
		{
			name:        "invalid tag ref",
			selector:    &semverSelector{tagBasedSelector: &tagBasedSelector{}},
			ref:         "invalid",
			shouldMatch: false,
		},
		{
			name:        "non semver ignored",
			selector:    &semverSelector{tagBasedSelector: &tagBasedSelector{}},
			ref:         "refs/tags/foo",
			shouldMatch: false,
		},
		{
			name: "non-strict semver ignored",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{},
				strictSemvers:    true,
			},
			ref:         "refs/tags/v1.0", // Not strict
			shouldMatch: false,
		},
		{
			name:        "no regex specified",
			selector:    &semverSelector{tagBasedSelector: &tagBasedSelector{}},
			ref:         "refs/tags/v1.0.0",
			shouldMatch: true,
		},
		{
			name: "regex matches",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{
					allowTagsRegexes: []*regexp.Regexp{regexp.MustCompile(`^v1\.`)},
				},
			},
			ref:         "refs/tags/v1.0.0",
			shouldMatch: true,
		},
		{
			name: "regex does not match",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{
					allowTagsRegexes: []*regexp.Regexp{regexp.MustCompile(`^v1\.`)},
				},
			},
			ref:         "refs/tags/v2.0.0",
			shouldMatch: false,
		},
		{
			name: "ignored",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{
					ignoreTagsRegexes: []*regexp.Regexp{regexp.MustCompile(`^v1\.0\.0$`)},
				},
			},
			ref:         "refs/tags/v1.0.0",
			shouldMatch: false,
		},
		{
			name: "not ignored",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{
					ignoreTagsRegexes: []*regexp.Regexp{regexp.MustCompile(`^v2\.0\.0$`)},
				},
			},
			ref:         "refs/tags/v1.0.0",
			shouldMatch: true,
		},
		{
			name: "regex matches, but ignored",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{
					allowTagsRegexes:  []*regexp.Regexp{regexp.MustCompile(`^v1\.`)},
					ignoreTagsRegexes: []*regexp.Regexp{regexp.MustCompile(`^v1\.0\.0$`)},
				},
			},
			ref:         "refs/tags/v1.0.0",
			shouldMatch: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.shouldMatch,
				testCase.selector.MatchesRef(testCase.ref),
			)
		})
	}
}

func Test_semverSelector_Select(t *testing.T) {
	nonBoolExpression, err := expr.Compile(`"invalid"`)
	require.NoError(t, err)

	allowMajorV1, err := regexp.Compile(`^v?1\.`)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		selector   *semverSelector
		assertions func(*testing.T, []kargoapi.DiscoveredCommit, error)
	}{
		{
			name: "error cloning repo",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{
					baseSelector: &baseSelector{
						gitCloneFn: func(
							string,
							*git.ClientOptions,
							*git.CloneOptions,
						) (git.Repo, error) {
							return nil, errors.New("something went wrong")
						},
					},
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.DiscoveredCommit, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error listing tags",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{
					baseSelector: &baseSelector{
						gitCloneFn: func(
							string,
							*git.ClientOptions,
							*git.CloneOptions,
						) (git.Repo, error) {
							return &git.MockRepo{
								ListTagsFn: func() ([]git.TagMetadata, error) {
									return nil, errors.New("something went wrong")
								},
							}, nil
						},
					},
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.DiscoveredCommit, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error filtering tags with expression",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{
					baseSelector: &baseSelector{
						gitCloneFn: func(
							string,
							*git.ClientOptions,
							*git.CloneOptions,
						) (git.Repo, error) {
							return &git.MockRepo{
								ListTagsFn: func() ([]git.TagMetadata, error) {
									return []git.TagMetadata{{Tag: "v1.0.0"}}, nil
								},
							}, nil
						},
						filterExpression: nonBoolExpression,
					},
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.DiscoveredCommit, err error) {
				require.ErrorContains(t, err, "error filtering tags by expression")
			},
		},
		{
			name: "error filtering tags by diff paths",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{
					baseSelector: &baseSelector{
						gitCloneFn: func(
							string,
							*git.ClientOptions,
							*git.CloneOptions,
						) (git.Repo, error) {
							return &git.MockRepo{
								ListTagsFn: func() ([]git.TagMetadata, error) {
									return []git.TagMetadata{{}}, nil
								},
							}, nil
						},
					},
					filterTagsByDiffPathsFn: func(
						git.Repo,
						[]git.TagMetadata,
					) ([]git.TagMetadata, error) {
						return nil, errors.New("something went wrong")
					},
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.DiscoveredCommit, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "non-semver tags are filtered out",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{
					baseSelector: &baseSelector{
						gitCloneFn: func(
							string,
							*git.ClientOptions,
							*git.CloneOptions,
						) (git.Repo, error) {
							return &git.MockRepo{
								ListTagsFn: func() ([]git.TagMetadata, error) {
									return []git.TagMetadata{
										{Tag: "foo"},
										{Tag: "v1.2.3"},
										{Tag: "bar"},
									}, nil
								},
							}, nil
						},
					},
					filterTagsByDiffPathsFn: func(
						_ git.Repo,
						tags []git.TagMetadata,
					) ([]git.TagMetadata, error) {
						return tags, nil
					},
				},
			},
			assertions: func(
				t *testing.T,
				commits []kargoapi.DiscoveredCommit,
				err error,
			) {
				require.NoError(t, err)
				require.Len(t, commits, 1)
				require.Equal(t, "v1.2.3", commits[0].Tag)
			},
		},
		{
			name: "tags not allowed are filtered out",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{
					baseSelector: &baseSelector{
						gitCloneFn: func(
							string,
							*git.ClientOptions,
							*git.CloneOptions,
						) (git.Repo, error) {
							return &git.MockRepo{
								ListTagsFn: func() ([]git.TagMetadata, error) {
									return []git.TagMetadata{
										{Tag: "v1.0.0"},
										{Tag: "v2.0.0"},
										{Tag: "v3.0.0"},
									}, nil
								},
							}, nil
						},
					},
					allowTagsRegexes: []*regexp.Regexp{allowMajorV1},
					filterTagsByDiffPathsFn: func(
						_ git.Repo,
						tags []git.TagMetadata,
					) ([]git.TagMetadata, error) {
						return tags, nil
					},
				},
			},
			assertions: func(
				t *testing.T,
				commits []kargoapi.DiscoveredCommit,
				err error,
			) {
				require.NoError(t, err)
				require.Len(t, commits, 1)
				require.Equal(t, "v1.0.0", commits[0].Tag)
			},
		},
		{
			name: "ignored tags are filtered out",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{
					baseSelector: &baseSelector{
						gitCloneFn: func(
							string,
							*git.ClientOptions,
							*git.CloneOptions,
						) (git.Repo, error) {
							return &git.MockRepo{
								ListTagsFn: func() ([]git.TagMetadata, error) {
									return []git.TagMetadata{
										{Tag: "v1.0.0"},
										{Tag: "v1.1.0"},
									}, nil
								},
							}, nil
						},
					},
					ignoreTagsRegexes: []*regexp.Regexp{regexp.MustCompile(`^v1\.0\.0$`)},
					filterTagsByDiffPathsFn: func(
						_ git.Repo,
						tags []git.TagMetadata,
					) ([]git.TagMetadata, error) {
						return tags, nil
					},
				},
			},
			assertions: func(
				t *testing.T,
				commits []kargoapi.DiscoveredCommit,
				err error,
			) {
				require.NoError(t, err)
				require.Len(t, commits, 1)
				require.Equal(t, "v1.1.0", commits[0].Tag)
			},
		},
		{
			name: "ignored tags have precedence over allowed tags",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{
					baseSelector: &baseSelector{
						gitCloneFn: func(
							string,
							*git.ClientOptions,
							*git.CloneOptions,
						) (git.Repo, error) {
							return &git.MockRepo{
								ListTagsFn: func() ([]git.TagMetadata, error) {
									return []git.TagMetadata{
										{Tag: "v1.1.0"},
										{Tag: "v1.0.0"},
									}, nil
								},
							}, nil
						},
					},
					allowTagsRegexes:  []*regexp.Regexp{allowMajorV1},
					ignoreTagsRegexes: []*regexp.Regexp{regexp.MustCompile(`^v1\.0\.0$`)},
					filterTagsByDiffPathsFn: func(
						_ git.Repo,
						tags []git.TagMetadata,
					) ([]git.TagMetadata, error) {
						return tags, nil
					},
				},
			},
			assertions: func(
				t *testing.T,
				commits []kargoapi.DiscoveredCommit,
				err error,
			) {
				require.NoError(t, err)
				require.Len(t, commits, 1)
				require.Equal(t, "v1.1.0", commits[0].Tag)
			},
		},
		{
			name: "tags are correctly sorted",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{
					baseSelector: &baseSelector{
						gitCloneFn: func(
							string,
							*git.ClientOptions,
							*git.CloneOptions,
						) (git.Repo, error) {
							return &git.MockRepo{
								ListTagsFn: func() ([]git.TagMetadata, error) {
									return []git.TagMetadata{
										{Tag: "v3.0.0"},
										{Tag: "v5.0.0"},
										{Tag: "v4.0.0"},
										{Tag: "v1.0.0"},
										{Tag: "v2.0.0"},
									}, nil
								},
							}, nil
						},
					},
					filterTagsByDiffPathsFn: func(
						_ git.Repo,
						tags []git.TagMetadata,
					) ([]git.TagMetadata, error) {
						return tags, nil
					},
				},
			},
			assertions: func(
				t *testing.T,
				commits []kargoapi.DiscoveredCommit,
				err error,
			) {
				require.NoError(t, err)
				require.Len(t, commits, 5)
				oldest, err := semver.NewVersion(commits[0].Tag)
				require.NoError(t, err)
				for i := 1; i < len(commits); i++ {
					next, err := semver.NewVersion(commits[i].Tag)
					require.NoError(t, err)
					require.Less(t, next.Compare(oldest), 0)
					oldest = next
				}
			},
		},
		{
			name: "results are trimmed to discovery limit",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{
					baseSelector: &baseSelector{
						gitCloneFn: func(
							string,
							*git.ClientOptions,
							*git.CloneOptions,
						) (git.Repo, error) {
							return &git.MockRepo{
								ListTagsFn: func() ([]git.TagMetadata, error) {
									return []git.TagMetadata{
										{Tag: "v3.0.0"},
										{Tag: "v5.0.0"},
										{Tag: "v4.0.0"},
										{Tag: "v1.0.0"},
										{Tag: "v2.0.0"},
									}, nil
								},
							}, nil
						},
						discoveryLimit: 3,
					},
					filterTagsByDiffPathsFn: func(
						_ git.Repo,
						tags []git.TagMetadata,
					) ([]git.TagMetadata, error) {
						return tags, nil
					},
				},
			},
			assertions: func(
				t *testing.T,
				commits []kargoapi.DiscoveredCommit,
				err error,
			) {
				require.NoError(t, err)
				require.Len(t, commits, 3)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			commits, err := testCase.selector.Select(context.Background())
			testCase.assertions(t, commits, err)
		})
	}
}
