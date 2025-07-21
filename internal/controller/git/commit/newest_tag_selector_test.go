package commit

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
)

func TestNewNewestTagSelector(t *testing.T) {
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
			name: "success",
			sub:  kargoapi.GitSubscription{},
			assertions: func(t *testing.T, s Selector, err error) {
				require.NoError(t, err)
				n, ok := s.(*newestTagSelector)
				require.True(t, ok)
				require.NotNil(t, n.tagBasedSelector)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newNewestTagSelector(testCase.sub, nil)
			testCase.assertions(t, s, err)
		})
	}
}

func Test_newestTagSelector_Select(t *testing.T) {
	nonBoolExpression, err := expr.Compile(`"invalid"`)
	require.NoError(t, err)

	allowAlphas, err := regexp.Compile("^[a-zA-Z]+$")
	require.NoError(t, err)

	testCases := []struct {
		name       string
		selector   *newestTagSelector
		assertions func(*testing.T, []kargoapi.DiscoveredCommit, error)
	}{
		{
			name: "error cloning repo",
			selector: &newestTagSelector{
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
			selector: &newestTagSelector{
				tagBasedSelector: &tagBasedSelector{
					baseSelector: &baseSelector{
						gitCloneFn: func(
							string,
							*git.ClientOptions,
							*git.CloneOptions,
						) (git.Repo, error) {
							return nil, nil
						},
					},
					listTagsFn: func(git.Repo) ([]git.TagMetadata, error) {
						return nil, errors.New("something went wrong")
					},
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.DiscoveredCommit, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error filtering tags with expression",
			selector: &newestTagSelector{
				tagBasedSelector: &tagBasedSelector{
					baseSelector: &baseSelector{
						gitCloneFn: func(
							string,
							*git.ClientOptions,
							*git.CloneOptions,
						) (git.Repo, error) {
							return nil, nil
						},
						filterExpression: nonBoolExpression,
					},
					listTagsFn: func(git.Repo) ([]git.TagMetadata, error) {
						return []git.TagMetadata{{}}, nil
					},
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.DiscoveredCommit, err error) {
				require.ErrorContains(t, err, "error filtering tags by expression")
			},
		},
		{
			name: "error filtering tags by diff paths",
			selector: &newestTagSelector{
				tagBasedSelector: &tagBasedSelector{
					baseSelector: &baseSelector{
						gitCloneFn: func(
							string,
							*git.ClientOptions,
							*git.CloneOptions,
						) (git.Repo, error) {
							return nil, nil
						},
					},
					listTagsFn: func(git.Repo) ([]git.TagMetadata, error) {
						return []git.TagMetadata{{}}, nil
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
			name: "tags not allowed are filtered out",
			selector: &newestTagSelector{
				tagBasedSelector: &tagBasedSelector{
					baseSelector: &baseSelector{
						gitCloneFn: func(
							string,
							*git.ClientOptions,
							*git.CloneOptions,
						) (git.Repo, error) {
							return nil, nil
						},
					},
					allows: allowAlphas,
					listTagsFn: func(git.Repo) ([]git.TagMetadata, error) {
						return []git.TagMetadata{
							{Tag: "123"},
							{Tag: "abc"},
						}, nil
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
				require.Equal(t, "abc", commits[0].Tag)
			},
		},
		{
			name: "ignored tags are filtered out",
			selector: &newestTagSelector{
				tagBasedSelector: &tagBasedSelector{
					baseSelector: &baseSelector{
						gitCloneFn: func(
							string,
							*git.ClientOptions,
							*git.CloneOptions,
						) (git.Repo, error) {
							return nil, nil
						},
					},
					ignores: []string{"123"},
					listTagsFn: func(git.Repo) ([]git.TagMetadata, error) {
						return []git.TagMetadata{
							{Tag: "123"},
							{Tag: "abc"},
						}, nil
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
				require.Equal(t, "abc", commits[0].Tag)
			},
		},
		{
			name: "ignored tags have precedence over allowed tags",
			selector: &newestTagSelector{
				tagBasedSelector: &tagBasedSelector{
					baseSelector: &baseSelector{
						gitCloneFn: func(
							string,
							*git.ClientOptions,
							*git.CloneOptions,
						) (git.Repo, error) {
							return nil, nil
						},
					},
					allows:  allowAlphas,
					ignores: []string{"ABC"},
					listTagsFn: func(git.Repo) ([]git.TagMetadata, error) {
						return []git.TagMetadata{
							{Tag: "ABC"},
							{Tag: "abc"},
						}, nil
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
				require.Equal(t, "abc", commits[0].Tag)
			},
		},
		{
			name: "results are trimmed to discovery limit",
			selector: &newestTagSelector{
				tagBasedSelector: &tagBasedSelector{
					baseSelector: &baseSelector{
						gitCloneFn: func(
							string,
							*git.ClientOptions,
							*git.CloneOptions,
						) (git.Repo, error) {
							return nil, nil
						},
						discoveryLimit: 3,
					},
					listTagsFn: func(git.Repo) ([]git.TagMetadata, error) {
						return []git.TagMetadata{{}, {}, {}, {}, {}}, nil
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
