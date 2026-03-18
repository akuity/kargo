package commit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
)

func TestNewNewestFromBranchSelector(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.GitSubscription
		assertions func(*testing.T, Selector, error)
	}{
		{
			name: "error building tag based selector",
			sub: kargoapi.GitSubscription{
				ExpressionFilter: "(1 + 2", // This will force an error
				Branch:           "main",
			},
			assertions: func(t *testing.T, _ Selector, err error) {
				require.ErrorContains(t, err, "error building base selector")
			},
		},
		{
			name: "success",
			sub:  kargoapi.GitSubscription{Branch: "main"},
			assertions: func(t *testing.T, s Selector, err error) {
				require.NoError(t, err)
				n, ok := s.(*newestFromBranchSelector)
				require.True(t, ok)
				require.NotNil(t, n.baseSelector)
				require.Equal(t, "main", n.branch)
				require.NotNil(t, n.selectCommitsFn)
				require.NotNil(t, n.listCommitsFn)
				require.NotNil(t, n.getDiffPathsForCommitIDFn)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newNewestFromBranchSelector(testCase.sub, nil)
			testCase.assertions(t, s, err)
		})
	}
}

func Test_newestFromBranchSelector_MatchesRef(t *testing.T) {
	const testBranchName = "my-cool-branch"
	var testRef = "refs/heads/" + testBranchName
	testSelector := &newestFromBranchSelector{
		branch: testBranchName,
	}
	testCases := []struct {
		name        string
		selector    *newestFromBranchSelector
		ref         string
		shouldMatch bool
	}{
		{
			name:        "input is missing branch prefix",
			selector:    testSelector,
			ref:         testBranchName,
			shouldMatch: false,
		},
		{
			name:        "input is has tag prefix",
			selector:    testSelector,
			ref:         "refs/tags/" + testBranchName,
			shouldMatch: false,
		},
		{
			name:        "input doesn't match selector branch",
			selector:    testSelector,
			ref:         "refs/heads/some-other-branch",
			shouldMatch: false,
		},
		{
			name:        "input matches selector branch",
			selector:    testSelector,
			ref:         testRef,
			shouldMatch: true,
		},
		{
			name:        "input is main, selector has no branch configured",
			selector:    &newestFromBranchSelector{},
			ref:         "refs/heads/main",
			shouldMatch: true,
		},
		{
			name:        "input is master, selector has no branch configured",
			selector:    &newestFromBranchSelector{},
			ref:         "refs/heads/master",
			shouldMatch: true,
		},
		{
			name:        "input is non-main, non-master, selector has no branch configured",
			selector:    &newestFromBranchSelector{},
			ref:         testRef,
			shouldMatch: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(
				t,
				testCase.shouldMatch,
				testCase.selector.MatchesRef(testCase.ref),
			)
		})
	}
}

func Test_newestFromBranchSelector_Select(t *testing.T) {
	testCases := []struct {
		name       string
		selector   *newestFromBranchSelector
		assertions func(*testing.T, []kargoapi.DiscoveredCommit, error)
	}{
		{
			name: "error cloning repo",
			selector: &newestFromBranchSelector{
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
			assertions: func(t *testing.T, _ []kargoapi.DiscoveredCommit, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error selecting commits",
			selector: &newestFromBranchSelector{
				baseSelector: &baseSelector{
					gitCloneFn: func(
						string,
						*git.ClientOptions,
						*git.CloneOptions,
					) (git.Repo, error) {
						return &git.MockRepo{}, nil
					},
				},
				selectCommitsFn: func(git.Repo) ([]git.CommitMetadata, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.DiscoveredCommit, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			selector: &newestFromBranchSelector{
				baseSelector: &baseSelector{
					gitCloneFn: func(
						string,
						*git.ClientOptions,
						*git.CloneOptions,
					) (git.Repo, error) {
						return &git.MockRepo{}, nil
					},
				},
				selectCommitsFn: func(git.Repo) ([]git.CommitMetadata, error) {
					return []git.CommitMetadata{{}, {}, {}, {}, {}}, nil
				},
			},
			assertions: func(t *testing.T, commits []kargoapi.DiscoveredCommit, err error) {
				require.NoError(t, err)
				require.Len(t, commits, 5)
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

func Test_newestFromBranchSelector_evaluateCommitExpression(t *testing.T) {
	testCases := []struct {
		name       string
		commit     git.CommitMetadata
		expression string
		assertions func(*testing.T, bool, error)
	}{
		{
			name:       "boolean true expression",
			expression: "true",
			commit:     git.CommitMetadata{ID: "abc"},
			assertions: func(t *testing.T, result bool, err error) {
				require.NoError(t, err)
				require.True(t, result)
			},
		},
		{
			name:       "boolean false expression",
			expression: "false",
			commit:     git.CommitMetadata{ID: "abc"},
			assertions: func(t *testing.T, result bool, err error) {
				require.NoError(t, err)
				require.False(t, result)
			},
		},
		{
			name:       "expression with commit data",
			expression: "id == 'abc' && author == 'test-author'",
			commit: git.CommitMetadata{
				ID:     "abc",
				Author: "test-author",
			},
			assertions: func(t *testing.T, result bool, err error) {
				require.NoError(t, err)
				require.True(t, result)
			},
		},
		{
			name:       "expression with commit data false",
			expression: "id == 'xyz'",
			commit: git.CommitMetadata{
				ID: "abc",
			},
			assertions: func(t *testing.T, result bool, err error) {
				require.NoError(t, err)
				require.False(t, result)
			},
		},
		{
			name:       "non-boolean result converted to true",
			expression: "1",
			commit:     git.CommitMetadata{ID: "abc"},
			assertions: func(t *testing.T, result bool, err error) {
				require.NoError(t, err)
				require.True(t, result)
			},
		},
		{
			name:       "non-boolean result converted to false",
			expression: "0",
			commit:     git.CommitMetadata{ID: "abc"},
			assertions: func(t *testing.T, result bool, err error) {
				require.NoError(t, err)
				require.False(t, result)
			},
		},
		{
			name:       "error parsing non-boolean result",
			expression: "'invalid'",
			commit:     git.CommitMetadata{ID: "abc"},
			assertions: func(t *testing.T, _ bool, err error) {
				require.ErrorContains(t, err, "invalid syntax")
			},
		},
		{
			name:       "expression with date comparison",
			expression: "commitDate.Year() >= 2023",
			commit: git.CommitMetadata{
				ID:         "abc",
				CommitDate: time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC),
			},
			assertions: func(t *testing.T, result bool, err error) {
				require.NoError(t, err)
				require.True(t, result)
			},
		},
		{
			name:       "expression with subject matching",
			expression: "subject contains 'fix:'",
			commit: git.CommitMetadata{
				ID:      "abc",
				Subject: "fix: resolve issue with authentication",
			},
			assertions: func(t *testing.T, result bool, err error) {
				require.NoError(t, err)
				require.True(t, result)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			expression, err := expr.Compile(testCase.expression)
			require.NoError(t, err)
			s := &newestFromBranchSelector{
				baseSelector: &baseSelector{
					filterExpression: expression,
				},
			}
			result, err := s.evaluateCommitExpression(testCase.commit)
			testCase.assertions(t, result, err)
		})
	}
}

func Test_newestFromBranchSelector_selectCommits(t *testing.T) {
	nonBoolExpression, err := expr.Compile(`"invalid"`)
	require.NoError(t, err)

	idFilterExpression, err := expr.Compile(`id != "A"`)
	require.NoError(t, err)

	includePaths, err := getPathSelectors([]string{"some-path/"})
	require.NoError(t, err)

	testCases := []struct {
		name       string
		selector   *newestFromBranchSelector
		assertions func(*testing.T, []git.CommitMetadata, error)
	}{
		{
			name: "error listing commits",
			selector: &newestFromBranchSelector{
				baseSelector: &baseSelector{},
				listCommitsFn: func(git.Repo, uint, uint) ([]git.CommitMetadata, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []git.CommitMetadata, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "no filters specified",
			selector: &newestFromBranchSelector{
				baseSelector: &baseSelector{
					discoveryLimit: 3,
				},
				listCommitsFn: func(git.Repo, uint, uint) ([]git.CommitMetadata, error) {
					return []git.CommitMetadata{{}, {}, {}, {}, {}}, nil
				},
			},
			assertions: func(t *testing.T, commits []git.CommitMetadata, err error) {
				require.NoError(t, err)
				require.Len(t, commits, 3)
			},
		},
		{
			name: "error evaluating expression",
			selector: &newestFromBranchSelector{
				baseSelector: &baseSelector{
					filterExpression: nonBoolExpression,
				},
				listCommitsFn: func(git.Repo, uint, uint) ([]git.CommitMetadata, error) {
					return []git.CommitMetadata{{}}, nil
				},
			},
			assertions: func(t *testing.T, _ []git.CommitMetadata, err error) {
				require.ErrorContains(t, err, "error evaluating filter expression")
			},
		},
		{
			name: "error applying path filters",
			selector: &newestFromBranchSelector{
				baseSelector: &baseSelector{
					includePaths: includePaths,
				},
				listCommitsFn: func(git.Repo, uint, uint) ([]git.CommitMetadata, error) {
					return []git.CommitMetadata{{}}, nil
				},
				getDiffPathsForCommitIDFn: func(git.Repo, string) ([]string, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []git.CommitMetadata, err error) {
				require.ErrorContains(t, err, "error getting diff paths for commit")
			},
		},
		{
			name: "filter expression filters out commits",
			selector: &newestFromBranchSelector{
				baseSelector: &baseSelector{
					filterExpression: idFilterExpression,
					discoveryLimit:   3,
				},
				listCommitsFn: func(git.Repo, uint, uint) ([]git.CommitMetadata, error) {
					return []git.CommitMetadata{
						{ID: "A"},
						{ID: "B"},
						{ID: "C"},
						{ID: "D"},
						{ID: "E"},
					}, nil
				},
			},
			assertions: func(t *testing.T, commits []git.CommitMetadata, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]git.CommitMetadata{{ID: "B"}, {ID: "C"}, {ID: "D"}},
					commits,
				)
			},
		},
		{
			name: "path filter filters out commits",
			selector: &newestFromBranchSelector{
				baseSelector: &baseSelector{
					includePaths:   includePaths,
					discoveryLimit: 3,
				},
				listCommitsFn: func(git.Repo, uint, uint) ([]git.CommitMetadata, error) {
					return []git.CommitMetadata{
						{ID: "A"},
						{ID: "B"},
						{ID: "C"},
						{ID: "D"},
						{ID: "E"},
					}, nil
				},
				getDiffPathsForCommitIDFn: func(
					_ git.Repo,
					commitID string,
				) ([]string, error) {
					if commitID == "A" {
						return nil, nil
					}
					return []string{"some-path/some-file"}, nil
				},
			},
			assertions: func(t *testing.T, commits []git.CommitMetadata, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]git.CommitMetadata{{ID: "B"}, {ID: "C"}, {ID: "D"}},
					commits,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			commits, err := testCase.selector.selectCommits(nil)
			testCase.assertions(t, commits, err)
		})
	}
}
