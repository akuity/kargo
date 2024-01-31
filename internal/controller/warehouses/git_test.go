package warehouses

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
)

func TestSelectCommits(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(commits []kargoapi.GitCommit, err error)
	}{
		{
			name: "error getting repo credentials",
			reconciler: &reconciler{
				credentialsDB: &credentials.FakeDB{
					GetFn: func(
						context.Context,
						string,
						credentials.Type,
						string,
					) (credentials.Credentials, bool, error) {
						return credentials.Credentials{}, false,
							errors.New("something went wrong")
					},
				},
			},
			assertions: func(commits []kargoapi.GitCommit, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error obtaining credentials for git repo",
				)
				require.Contains(t, err.Error(), "something went wrong")
				require.Empty(t, commits)
			},
		},

		{
			name: "error getting last commit ID",
			reconciler: &reconciler{
				credentialsDB: &credentials.FakeDB{
					GetFn: func(
						context.Context,
						string,
						credentials.Type,
						string,
					) (credentials.Credentials, bool, error) {
						return credentials.Credentials{}, false, nil
					},
				},
				selectCommitMetaFn: func(
					context.Context,
					kargoapi.GitSubscription,
					*git.RepoCredentials,
				) (*gitMeta, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(commits []kargoapi.GitCommit, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error determining latest commit ID of git repo",
				)
				require.Contains(t, err.Error(), "something went wrong")
				require.Empty(t, commits)
			},
		},

		{
			name: "success",
			reconciler: &reconciler{
				credentialsDB: &credentials.FakeDB{
					GetFn: func(
						context.Context,
						string,
						credentials.Type,
						string,
					) (credentials.Credentials, bool, error) {
						return credentials.Credentials{}, false, nil
					},
				},
				selectCommitMetaFn: func(
					context.Context,
					kargoapi.GitSubscription,
					*git.RepoCredentials,
				) (*gitMeta, error) {
					return &gitMeta{Commit: "fake-commit", Message: "message"}, nil
				},
			},
			assertions: func(commits []kargoapi.GitCommit, err error) {
				require.NoError(t, err)
				require.Len(t, commits, 1)
				require.Equal(
					t,
					kargoapi.GitCommit{
						RepoURL: "fake-url",
						ID:      "fake-commit",
						Message: "message",
					},
					commits[0],
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.reconciler.selectCommits(
					context.Background(),
					"fake-namespace",
					[]kargoapi.RepoSubscription{
						{
							Git: &kargoapi.GitSubscription{
								RepoURL: "fake-url",
							},
						},
					},
				),
			)
		})
	}
}

func TestSelectCommitMeta(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.GitSubscription
		reconciler *reconciler
		assertions func(*gitMeta, error)
	}{
		{
			name: "error cloning repo",
			sub: kargoapi.GitSubscription{
				RepoURL: "fake-url", // This should force a failure
			},
			reconciler: &reconciler{},
			assertions: func(_ *gitMeta, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error cloning git repo")
			},
		},
		{
			name: "success",
			sub: kargoapi.GitSubscription{
				RepoURL: "https://github.com/akuity/kargo.git",
			},
			reconciler: newReconciler(fake.NewClientBuilder().Build(), nil),
			assertions: func(gm *gitMeta, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, gm.Commit)
				require.NotEmpty(t, gm.Message)
				require.Len(t, strings.Split(gm.Message, "\n"), 1)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.reconciler.selectCommitMeta(
					context.Background(),
					testCase.sub,
					nil,
				),
			)
		})
	}
}

func TestSelectCommitID(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.GitSubscription
		reconciler *reconciler
		assertions func(tag string, commit string, err error)
	}{
		{
			name: "newest from branch; error getting commit ID",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
			},
			reconciler: &reconciler{
				getLastCommitIDFn: func(git.Repo) (string, error) {
					return "", errors.New("something went wrong")
				},
			},
			assertions: func(_, _ string, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error determining commit ID at head of branch",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "newest from branch; success",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
			},
			reconciler: &reconciler{
				getLastCommitIDFn: func(git.Repo) (string, error) {
					return "fake-commit", nil
				},
			},
			assertions: func(tag, commit string, err error) {
				require.NoError(t, err)
				require.Empty(t, tag)
				require.Equal(t, "fake-commit", commit)
			},
		},
		{
			name: "error listing tags",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategyLexical,
			},
			reconciler: &reconciler{
				listTagsFn: func(git.Repo) ([]string, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error listing tags from git repo")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "error compiling allow regex",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategyLexical,
				AllowTags:               "[", // This should force a failure
			},
			reconciler: &reconciler{
				listTagsFn: func(git.Repo) ([]string, error) {
					return []string{"abc"}, nil
				},
			},
			assertions: func(_, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error compiling regular expression")
			},
		},
		{
			name: "all tags get filtered out",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategyLexical,
				IgnoreTags:              []string{"abc"},
			},
			reconciler: &reconciler{
				listTagsFn: func(git.Repo) ([]string, error) {
					return []string{"abc"}, nil
				},
			},
			assertions: func(_, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "found no applicable tags in repo")
			},
		},
		{
			name: "unknown selection strategy",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategy("invalid"),
			},
			reconciler: &reconciler{
				listTagsFn: func(git.Repo) ([]string, error) {
					return []string{"abc"}, nil
				},
			},
			assertions: func(_, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unknown commit selection strategy")
			},
		},
		{
			name: "error checking out tag",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategyLexical,
			},
			reconciler: &reconciler{
				listTagsFn: func(git.Repo) ([]string, error) {
					return []string{"abc"}, nil
				},
				checkoutTagFn: func(git.Repo, string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(_, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error checking out tag")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "error getting commit ID",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategyLexical,
			},
			reconciler: &reconciler{
				listTagsFn: func(git.Repo) ([]string, error) {
					return []string{"abc"}, nil
				},
				checkoutTagFn: func(git.Repo, string) error {
					return nil
				},
				getLastCommitIDFn: func(git.Repo) (string, error) {
					return "", errors.New("something went wrong")
				},
			},
			assertions: func(_, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error determining commit ID of tag")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "lexical success",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategyLexical,
			},
			reconciler: &reconciler{
				listTagsFn: func(git.Repo) ([]string, error) {
					return []string{"abc", "xyz"}, nil
				},
				checkoutTagFn: func(git.Repo, string) error {
					return nil
				},
				getLastCommitIDFn: func(git.Repo) (string, error) {
					return "fake-commit", nil
				},
			},
			assertions: func(tag, commit string, err error) {
				require.NoError(t, err)
				require.Equal(t, "xyz", tag)
				require.Equal(t, "fake-commit", commit)
			},
		},
		{
			name: "newest tag success",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestTag,
			},
			reconciler: &reconciler{
				listTagsFn: func(git.Repo) ([]string, error) {
					return []string{"abc", "xyz"}, nil
				},
				checkoutTagFn: func(git.Repo, string) error {
					return nil
				},
				getLastCommitIDFn: func(git.Repo) (string, error) {
					return "fake-commit", nil
				},
			},
			assertions: func(tag, commit string, err error) {
				require.Equal(t, "abc", tag)
				require.NoError(t, err)
				require.Equal(t, "fake-commit", commit)
			},
		},
		{
			name: "semver error selecting tag",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
				SemverConstraint:        "invalid", // This should force a failure
			},
			reconciler: &reconciler{
				listTagsFn: func(git.Repo) ([]string, error) {
					return []string{"1.0.0", "2.0.0"}, nil
				},
			},
			assertions: func(_, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error parsing semver constraint")
			},
		},
		{
			name: "semver success",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
			},
			reconciler: &reconciler{
				listTagsFn: func(git.Repo) ([]string, error) {
					return []string{"1.0.0", "2.0.0"}, nil
				},
				checkoutTagFn: func(git.Repo, string) error {
					return nil
				},
				getLastCommitIDFn: func(git.Repo) (string, error) {
					return "fake-commit", nil
				},
			},
			assertions: func(tag, commit string, err error) {
				require.NoError(t, err)
				require.Equal(t, "2.0.0", tag)
				require.Equal(t, "fake-commit", commit)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.reconciler.selectTagAndCommitID(
					nil,
					testCase.sub,
				),
			)
		})
	}
}

func TestAllows(t *testing.T) {
	testCases := []struct {
		name    string
		regex   *regexp.Regexp
		tag     string
		allowed bool
	}{
		{
			name:    "no regex specified",
			tag:     "abc",
			allowed: true,
		},
		{
			name:    "allowed",
			regex:   regexp.MustCompile("[a-z]+"),
			tag:     "abc",
			allowed: true,
		},
		{
			name:    "not allowed",
			regex:   regexp.MustCompile("[a-z]+"),
			tag:     "123",
			allowed: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.allowed,
				allows(testCase.tag, testCase.regex),
			)
		})
	}
}

func TestIgnores(t *testing.T) {
	testCases := []struct {
		name    string
		ignore  []string
		tag     string
		ignored bool
	}{
		{
			name:    "ignored",
			ignore:  []string{"abc"},
			tag:     "abc",
			ignored: true,
		},
		{
			name:    "not ignored",
			ignore:  []string{"abc"},
			tag:     "123",
			ignored: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.ignored,
				ignores(testCase.tag, testCase.ignore),
			)
		})
	}
}

func TestSelectLexicallyLastTag(t *testing.T) {
	testCases := []struct {
		name     string
		tags     []string
		expected string
	}{
		{
			name:     "empty/nil tag list",
			tags:     nil,
			expected: "",
		},
		{
			name:     "non-empty tag list",
			tags:     []string{"abc", "xyz", "foo", "bar"},
			expected: "xyz",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.expected,
				selectLexicallyLastTag(testCase.tags),
			)
		})
	}
}

func TestSelectSemverTag(t *testing.T) {
	testCases := []struct {
		name       string
		constraint string
		tags       []string
		assertions func(string, error)
	}{
		{
			name:       "error parsing constraint",
			constraint: "invalid",
			tags:       nil,
			assertions: func(_ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error parsing semver constraint")
			},
		},
		{
			name: "empty/nil tag list",
			tags: nil,
			assertions: func(tag string, err error) {
				require.NoError(t, err)
				require.Empty(t, tag)
			},
		},
		{
			name: "no semantic tags in tag list",
			tags: []string{"abc", "xyz", "foo", "bar"},
			assertions: func(tag string, err error) {
				require.NoError(t, err)
				require.Empty(t, tag)
			},
		},
		{
			name:       "no constraint matches",
			constraint: ">=2.0.0",
			tags:       []string{"v1.0.0", "v1.2.3"},
			assertions: func(tag string, err error) {
				require.NoError(t, err)
				require.Empty(t, tag)
			},
		},
		{
			name: "success with no constraint",
			tags: []string{"v1.0.0", "v1.2.3"},
			assertions: func(tag string, err error) {
				require.NoError(t, err)
				require.Equal(t, "v1.2.3", tag)
			},
		},
		{
			name:       "success with constraint",
			constraint: "<2.0.0",
			tags:       []string{"v1.0.0", "v2.2.3"},
			assertions: func(tag string, err error) {
				require.NoError(t, err)
				require.Equal(t, "v1.0.0", tag)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				selectSemverTag(testCase.tags, testCase.constraint),
			)
		})
	}
}
