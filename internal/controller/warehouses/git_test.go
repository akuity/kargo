package warehouses

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
)

func TestDiscoverCommits(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		subs       []kargoapi.RepoSubscription
		assertions func(*testing.T, []kargoapi.GitDiscoveryResult, error)
	}{
		{
			name: "error cloning repository",
			reconciler: &reconciler{
				credentialsDB: &credentials.FakeDB{},
				gitCloneFn: func(string, *git.ClientOptions, *git.CloneOptions) (git.Repo, error) {
					return nil, errors.New("something went wrong")
				},
			},
			subs: []kargoapi.RepoSubscription{
				{Git: &kargoapi.GitSubscription{}},
			},
			assertions: func(t *testing.T, _ []kargoapi.GitDiscoveryResult, err error) {
				require.ErrorContains(t, err, "failed to clone git repo")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error obtaining credentials",
			reconciler: &reconciler{
				credentialsDB: &credentials.FakeDB{
					GetFn: func(
						context.Context,
						string,
						credentials.Type,
						string,
					) (credentials.Credentials, bool, error) {
						return credentials.Credentials{}, false, errors.New("something went wrong")
					},
				},
			},
			subs: []kargoapi.RepoSubscription{
				{Git: &kargoapi.GitSubscription{}},
			},
			assertions: func(t *testing.T, _ []kargoapi.GitDiscoveryResult, err error) {
				require.ErrorContains(t, err, "error obtaining credentials for git repo")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "discovers tags",
			reconciler: &reconciler{
				credentialsDB: &credentials.FakeDB{},
				gitCloneFn: func(string, *git.ClientOptions, *git.CloneOptions) (git.Repo, error) {
					return nil, nil
				},
				discoverTagsFn: func(git.Repo, kargoapi.GitSubscription) ([]git.TagMetadata, error) {
					return []git.TagMetadata{
						{Tag: "v2.0.0"},
						{Tag: "v1.0.0"},
					}, nil
				},
			},
			subs: []kargoapi.RepoSubscription{
				{Git: &kargoapi.GitSubscription{
					RepoURL:                 "fake-repo",
					CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
				}},
			},
			assertions: func(t *testing.T, results []kargoapi.GitDiscoveryResult, err error) {
				require.NoError(t, err)
				require.Equal(t, []kargoapi.GitDiscoveryResult{
					{
						RepoURL: "fake-repo",
						Commits: []kargoapi.DiscoveredCommit{
							{Tag: "v2.0.0", CreatorDate: &metav1.Time{}},
							{Tag: "v1.0.0", CreatorDate: &metav1.Time{}},
						},
					},
				}, results)
			},
		},
		{
			name: "error discovering tags",
			reconciler: &reconciler{
				credentialsDB: &credentials.FakeDB{},
				gitCloneFn: func(string, *git.ClientOptions, *git.CloneOptions) (git.Repo, error) {
					return nil, nil
				},
				discoverTagsFn: func(git.Repo, kargoapi.GitSubscription) ([]git.TagMetadata, error) {
					return nil, errors.New("something went wrong")
				},
			},
			subs: []kargoapi.RepoSubscription{
				{Git: &kargoapi.GitSubscription{
					CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
				}},
			},
			assertions: func(t *testing.T, _ []kargoapi.GitDiscoveryResult, err error) {
				require.ErrorContains(t, err, "error listing tags from git repo")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "discovers branch history",
			reconciler: &reconciler{
				credentialsDB: &credentials.FakeDB{},
				gitCloneFn: func(string, *git.ClientOptions, *git.CloneOptions) (git.Repo, error) {
					return nil, nil
				},
				discoverBranchHistoryFn: func(git.Repo, kargoapi.GitSubscription) ([]git.CommitMetadata, error) {
					return []git.CommitMetadata{
						{ID: "abc"},
						{ID: "xyz"},
					}, nil
				},
			},
			subs: []kargoapi.RepoSubscription{
				{Git: &kargoapi.GitSubscription{
					RepoURL:                 "fake-repo",
					CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
				}},
			},
			assertions: func(t *testing.T, results []kargoapi.GitDiscoveryResult, err error) {
				require.NoError(t, err)
				require.Equal(t, []kargoapi.GitDiscoveryResult{
					{
						RepoURL: "fake-repo",
						Commits: []kargoapi.DiscoveredCommit{
							{ID: "abc", CreatorDate: &metav1.Time{}},
							{ID: "xyz", CreatorDate: &metav1.Time{}},
						},
					},
				}, results)
			},
		},
		{
			name: "error discovering branch history",
			reconciler: &reconciler{
				credentialsDB: &credentials.FakeDB{},
				gitCloneFn: func(string, *git.ClientOptions, *git.CloneOptions) (git.Repo, error) {
					return nil, nil
				},
				discoverBranchHistoryFn: func(git.Repo, kargoapi.GitSubscription) ([]git.CommitMetadata, error) {
					return nil, errors.New("something went wrong")
				},
			},
			subs: []kargoapi.RepoSubscription{
				{Git: &kargoapi.GitSubscription{
					CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
				}},
			},
			assertions: func(t *testing.T, _ []kargoapi.GitDiscoveryResult, err error) {
				require.ErrorContains(t, err, "error listing commits from git repo")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "discovers for multiple subscriptions",
			reconciler: &reconciler{
				credentialsDB: &credentials.FakeDB{},
				gitCloneFn: func(string, *git.ClientOptions, *git.CloneOptions) (git.Repo, error) {
					return nil, nil
				},
				discoverTagsFn: func(git.Repo, kargoapi.GitSubscription) ([]git.TagMetadata, error) {
					return []git.TagMetadata{
						{Tag: "v2.0.0"},
						{Tag: "v1.0.0"},
					}, nil
				},
				discoverBranchHistoryFn: func(git.Repo, kargoapi.GitSubscription) ([]git.CommitMetadata, error) {
					return []git.CommitMetadata{
						{ID: "abc"},
						{ID: "xyz"},
					}, nil
				},
			},
			subs: []kargoapi.RepoSubscription{
				{Git: &kargoapi.GitSubscription{
					RepoURL:                 "fake-repo-1",
					CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestTag,
				}},
				{Image: &kargoapi.ImageSubscription{}}, // Should be ignored
				{Git: &kargoapi.GitSubscription{
					RepoURL:                 "fake-repo-2",
					CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
				}},
			},
			assertions: func(t *testing.T, results []kargoapi.GitDiscoveryResult, err error) {
				require.NoError(t, err)
				require.Equal(t, []kargoapi.GitDiscoveryResult{
					{
						RepoURL: "fake-repo-1",
						Commits: []kargoapi.DiscoveredCommit{
							{Tag: "v2.0.0", CreatorDate: &metav1.Time{}},
							{Tag: "v1.0.0", CreatorDate: &metav1.Time{}},
						},
					},
					{
						RepoURL: "fake-repo-2",
						Commits: []kargoapi.DiscoveredCommit{
							{ID: "abc", CreatorDate: &metav1.Time{}},
							{ID: "xyz", CreatorDate: &metav1.Time{}},
						},
					},
				}, results)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := testCase.reconciler.discoverCommits(context.TODO(), "fake-ns", testCase.subs)
			testCase.assertions(t, results, err)
		})
	}
}

func TestDiscoverBranchHistory(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.GitSubscription
		reconciler *reconciler
		assertions func(*testing.T, []git.CommitMetadata, error)
	}{
		{
			name: "error listing commits",
			reconciler: &reconciler{
				listCommitsFn: func(git.Repo, uint, uint) ([]git.CommitMetadata, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []git.CommitMetadata, err error) {
				require.ErrorContains(t, err, "error listing commits")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "without path filters",
			reconciler: &reconciler{
				listCommitsFn: func(git.Repo, uint, uint) ([]git.CommitMetadata, error) {
					return []git.CommitMetadata{
						{ID: "abc"},
						{ID: "xyz"},
					}, nil
				},
			},
			assertions: func(t *testing.T, commits []git.CommitMetadata, err error) {
				require.NoError(t, err)
				require.Equal(t, []git.CommitMetadata{
					{ID: "abc"},
					{ID: "xyz"},
				}, commits)
			},
		},
		{
			name: "error getting diff path",
			sub: kargoapi.GitSubscription{
				IncludePaths: []string{regexpPrefix + "^.*third_path_to_a/file$"},
			},
			reconciler: &reconciler{
				listCommitsFn: func(git.Repo, uint, uint) ([]git.CommitMetadata, error) {
					return []git.CommitMetadata{
						{ID: "abc"},
						{ID: "xyz"},
					}, nil
				},
				getDiffPathsForCommitIDFn: func(_ git.Repo, _ string) ([]string, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []git.CommitMetadata, err error) {
				require.ErrorContains(t, err, "error getting diff paths for commit")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "with path filters",
			sub: kargoapi.GitSubscription{
				IncludePaths: []string{regexpPrefix + "^.*third_path_to_a/file$"},
			},
			reconciler: &reconciler{
				listCommitsFn: func(_ git.Repo, _ uint, skip uint) ([]git.CommitMetadata, error) {
					if skip > 0 {
						return nil, nil
					}
					return []git.CommitMetadata{
						{ID: "abc"},
						{ID: "xyz"},
					}, nil
				},
				getDiffPathsForCommitIDFn: func(_ git.Repo, id string) ([]string, error) {
					if id == "abc" {
						return []string{"third_path_to_a/file"}, nil
					}
					return []string{"first_path_to_a/file", "second_path_to_a/file"}, nil
				},
			},
			assertions: func(t *testing.T, commits []git.CommitMetadata, err error) {
				require.NoError(t, err)
				require.Equal(t, []git.CommitMetadata{
					{ID: "abc"},
				}, commits)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tags, err := testCase.reconciler.discoverBranchHistory(nil, testCase.sub)
			testCase.assertions(t, tags, err)
		})
	}
}

func TestDiscoverTags(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.GitSubscription
		reconciler *reconciler
		assertions func(*testing.T, []git.TagMetadata, error)
	}{
		{
			name: "error listing tags",
			reconciler: &reconciler{
				listTagsFn: func(git.Repo) ([]git.TagMetadata, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []git.TagMetadata, err error) {
				require.ErrorContains(t, err, "error listing tags")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "ignore tags",
			sub: kargoapi.GitSubscription{
				IgnoreTags:              []string{"abc"},
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestTag,
			},
			reconciler: &reconciler{
				listTagsFn: func(git.Repo) ([]git.TagMetadata, error) {
					return []git.TagMetadata{
						{Tag: "abc"},
						{Tag: "xyz"},
					}, nil
				},
			},
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Equal(t, []git.TagMetadata{{Tag: "xyz"}}, tags)
			},
		},
		{
			name: "allow tags compile error",
			sub: kargoapi.GitSubscription{
				AllowTags: "[",
			},
			reconciler: &reconciler{
				listTagsFn: func(git.Repo) ([]git.TagMetadata, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, _ []git.TagMetadata, err error) {
				require.ErrorContains(t, err, "failed to filter tags")
				require.ErrorContains(t, err, "error compiling regular expression")
			},
		},
		{
			name: "SemVer commit selection strategy",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
			},
			reconciler: &reconciler{
				listTagsFn: func(git.Repo) ([]git.TagMetadata, error) {
					return []git.TagMetadata{
						{Tag: "v1.0.0"},
						{Tag: "abc"},
						{Tag: "v2.0.0"},
						{Tag: "xyz"},
						{Tag: "v1.2.3"},
					}, nil
				},
			},
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Equal(t, []git.TagMetadata{
					{Tag: "v2.0.0"},
					{Tag: "v1.2.3"},
					{Tag: "v1.0.0"},
				}, tags)
			},
		},
		{
			name: "SemVer commit selection strategy with constraint",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
				SemverConstraint:        ">=2.0.0",
			},
			reconciler: &reconciler{
				listTagsFn: func(git.Repo) ([]git.TagMetadata, error) {
					return []git.TagMetadata{
						{Tag: "v1.0.0"},
						{Tag: "v2.0.0"},
						{Tag: "v1.2.3"},
					}, nil
				},
			},
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Equal(t, []git.TagMetadata{
					{Tag: "v2.0.0"},
				}, tags)
			},
		},
		{
			name: "SemVer commit selection strategy with invalid constraint",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
				SemverConstraint:        "invalid",
			},
			reconciler: &reconciler{
				listTagsFn: func(git.Repo) ([]git.TagMetadata, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, _ []git.TagMetadata, err error) {
				require.ErrorContains(t, err, "failed to select semver tags")
				require.ErrorContains(t, err, "error parsing semver constraint")
			},
		},
		{
			name: "lexicographical commit selection strategy",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategyLexical,
			},
			reconciler: &reconciler{
				listTagsFn: func(git.Repo) ([]git.TagMetadata, error) {
					return []git.TagMetadata{
						{Tag: "123"},
						{Tag: "abc"},
						{Tag: "xyz"},
					}, nil
				},
			},
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Equal(t, []git.TagMetadata{
					{Tag: "xyz"},
					{Tag: "abc"},
					{Tag: "123"},
				}, tags)
			},
		},
		{
			name: "more tags than limit",
			sub: kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestTag,
				DiscoveryLimit:          3,
			},
			reconciler: &reconciler{
				listTagsFn: func(git.Repo) ([]git.TagMetadata, error) {
					return []git.TagMetadata{
						{Tag: "a"}, {Tag: "b"}, {Tag: "c"}, {Tag: "d"}, {Tag: "e"},
					}, nil
				},
			},
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Len(t, tags, 3)
			},
		},
		{
			name: "with path filters",
			sub: kargoapi.GitSubscription{
				IncludePaths:   []string{regexpPrefix + "^.*third_path_to_a/file$"},
				DiscoveryLimit: 20,
			},
			reconciler: &reconciler{
				listTagsFn: func(git.Repo) ([]git.TagMetadata, error) {
					return []git.TagMetadata{
						{Tag: "v1.0.0"},
						{Tag: "abc", CommitID: "fake-commit-id"},
						{Tag: "v2.0.0"},
						{Tag: "xyz"},
						{Tag: "v1.2.3"},
					}, nil
				},
				getDiffPathsForCommitIDFn: func(_ git.Repo, id string) ([]string, error) {
					if id == "fake-commit-id" {
						return []string{"third_path_to_a/file"}, nil
					}
					return []string{"first_path_to_a/file", "second_path_to_a/file"}, nil
				},
			},
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Equal(t, []git.TagMetadata{
					{Tag: "abc", CommitID: "fake-commit-id"},
				}, tags)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tags, err := testCase.reconciler.discoverTags(
				nil,
				testCase.sub,
			)
			testCase.assertions(t, tags, err)
		})
	}
}

func TestFilterTags(t *testing.T) {
	testCases := []struct {
		name       string
		tags       []git.TagMetadata
		ignoreTags []string
		allow      string
		assertions func(*testing.T, []git.TagMetadata, error)
	}{
		{
			name: "no tags",
			tags: nil,
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Empty(t, tags)
			},
		},
		{
			name:  "invalid regular expression",
			allow: "[",
			assertions: func(t *testing.T, _ []git.TagMetadata, err error) {
				require.ErrorContains(t, err, "error compiling regular expression")
			},
		},
		{
			name: "without ignore tags or allow regex",
			tags: []git.TagMetadata{
				{Tag: "v1.0.0"},
				{Tag: "xyz"},
				{Tag: "foo"},
			},
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Equal(t, []git.TagMetadata{
					{Tag: "v1.0.0"},
					{Tag: "xyz"},
					{Tag: "foo"},
				}, tags)
			},
		},
		{
			name: "with ignore tags",
			tags: []git.TagMetadata{
				{Tag: "v1.0.0"},
				{Tag: "xyz"},
				{Tag: "foo"},
			},
			ignoreTags: []string{"v1.0.0", "foo"},
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Equal(t, []git.TagMetadata{
					{Tag: "xyz"},
				}, tags)
			},
		},
		{
			name: "with allow regex",
			tags: []git.TagMetadata{
				{Tag: "v1.0.0"},
				{Tag: "xyz"},
				{Tag: "foo"},
			},
			allow: "v.*",
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Equal(t, []git.TagMetadata{
					{Tag: "v1.0.0"},
				}, tags)
			},
		},
		{
			name: "with ignore tags and allow regex",
			tags: []git.TagMetadata{
				{Tag: "v1.0.0"},
				{Tag: "xyz"},
				{Tag: "foo"},
				{Tag: "v2.0.0"},
			},
			ignoreTags: []string{"v1.0.0"},
			allow:      "v.*",
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Equal(t, []git.TagMetadata{
					{Tag: "v2.0.0"},
				}, tags)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tags, err := filterTags(testCase.tags, testCase.ignoreTags, testCase.allow)
			testCase.assertions(t, tags, err)
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

func TestSelectSemVerTags(t *testing.T) {
	testCases := []struct {
		name       string
		constraint string
		tags       []git.TagMetadata
		assertions func(*testing.T, []git.TagMetadata, error)
	}{
		{
			name:       "error parsing constraint",
			constraint: "invalid",
			tags:       nil,
			assertions: func(t *testing.T, _ []git.TagMetadata, err error) {
				require.ErrorContains(t, err, "error parsing semver constraint")
			},
		},
		{
			name: "empty/nil tag list",
			tags: nil,
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Empty(t, tags)
			},
		},
		{
			name: "no semantic tags in tag list",
			tags: []git.TagMetadata{
				{Tag: "abc"},
				{Tag: "xyz"},
				{Tag: "foo"},
				{Tag: "bar"},
			},
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Empty(t, tags)
			},
		},
		{
			name:       "no constraint matches",
			constraint: ">=2.0.0",
			tags: []git.TagMetadata{
				{Tag: "v1.0.0"},
				{Tag: "v1.2.3"},
			},
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Empty(t, tags)
			},
		},
		{
			name: "success with no constraint",
			tags: []git.TagMetadata{
				{Tag: "v1.0.0"},
				{Tag: "v2.1.3"},
				{Tag: "v1.2.3"},
			},
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Equal(t, []git.TagMetadata{
					{Tag: "v2.1.3"},
					{Tag: "v1.2.3"},
					{Tag: "v1.0.0"},
				}, tags)
			},
		},
		{
			name:       "success with constraint",
			constraint: "<2.0.0",
			tags: []git.TagMetadata{
				{Tag: "v1.0.0"},
				{Tag: "v2.1.3"},
				{Tag: "v1.2.3"},
			},
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Equal(t, []git.TagMetadata{
					{Tag: "v1.2.3"},
					{Tag: "v1.0.0"},
				}, tags)
			},
		},
		{
			name: "success with equivalent versions",
			tags: []git.TagMetadata{
				{Tag: "1.0"},
				{Tag: "1.0.0"},
			},
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Equal(t, []git.TagMetadata{
					{Tag: "1.0.0"},
					{Tag: "1.0"},
				}, tags)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tags, err := selectSemVerTags(testCase.tags, false, testCase.constraint)
			testCase.assertions(t, tags, err)
		})
	}
}

func TestMatchesPathsFilters(t *testing.T) {
	testCases := []struct {
		name         string
		includePaths []string
		excludePaths []string
		diffs        []string
		assertions   func(*testing.T, bool, error)
	}{
		{
			name:         "success with no includePaths configured",
			excludePaths: []string{regexpPrefix + "nonexistent"},
			diffs:        []string{"path1/values.yaml", "path2/_helpers.tpl"},
			assertions: func(t *testing.T, matchFound bool, err error) {
				require.NoError(t, err)
				require.Equal(t, true, matchFound)
			},
		},
		{
			name:         "success with a matching regexp filters configuration",
			includePaths: []string{regexpPrefix + "values\\.ya?ml$"},
			excludePaths: []string{regexPrefix + "nonexistent"},
			diffs:        []string{"path1/values.yaml", "path2/_helpers.tpl"},
			assertions: func(t *testing.T, matchFound bool, err error) {
				require.NoError(t, err)
				require.Equal(t, true, matchFound)
			},
		},
		{
			name:         "success with unmatching regexp filters configuration",
			includePaths: []string{regexpPrefix + "values\\.ya?ml$"},
			excludePaths: []string{regexPrefix + "nonexistent", regexpPrefix + ".*val.*"},
			diffs:        []string{"path1/values.yaml", "path2/_helpers.tpl"},
			assertions: func(t *testing.T, matchFound bool, err error) {
				require.NoError(t, err)
				require.Equal(t, false, matchFound)
			},
		},
		{
			name:         "success with matching glob filters configuration",
			includePaths: []string{"glob:path2/*.tpl"},
			excludePaths: []string{"nonexistent"},
			diffs:        []string{"path1/values.yaml", "path2/_helpers.tpl"},
			assertions: func(t *testing.T, matchFound bool, err error) {
				require.NoError(t, err)
				require.Equal(t, true, matchFound)
			},
		},
		{
			name:         "success with unmatching glob filters configuration",
			includePaths: []string{"path2/*.tpl"},
			excludePaths: []string{regexPrefix + "nonexistent", "*/?helpers.tpl"},
			diffs:        []string{"path1/values.yaml", "path2/_helpers.tpl"},
			assertions: func(t *testing.T, matchFound bool, err error) {
				require.NoError(t, err)
				require.Equal(t, false, matchFound)
			},
		},
		{
			name:         "success with matching prefix filters configuration",
			includePaths: []string{"path1/"},
			excludePaths: []string{"nonexistent"},
			diffs:        []string{"path1/values.yaml", "path2/_helpers.tpl"},
			assertions: func(t *testing.T, matchFound bool, err error) {
				require.NoError(t, err)
				require.Equal(t, true, matchFound)
			},
		},
		{
			name:         "success with unmatching prefix filters configuration",
			includePaths: []string{"path3/"},
			excludePaths: []string{regexPrefix + "nonexistent", "*/?helpers.tpl"},
			diffs:        []string{"path1/values.yaml", "path2/_helpers.tpl"},
			assertions: func(t *testing.T, matchFound bool, err error) {
				require.NoError(t, err)
				require.Equal(t, false, matchFound)
			},
		},
		{
			name:         "error with invalid glob syntax",
			includePaths: []string{"glob:path2/*.tpl["},
			diffs:        []string{"path1/values.yaml", "path2/_helpers.tpl"},
			assertions: func(t *testing.T, _ bool, err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "syntax error in pattern")
			},
		},
		{
			name:         "success unmatching mix1",
			includePaths: []string{"path3", regexPrefix + "nonexistent", globPrefix + "nonexistent"},
			excludePaths: []string{regexPrefix + "nonexistent", "*/?helpers.tpl", globPrefix + "nonexistent"},
			diffs:        []string{"path1/values.yaml", "path2/_helpers.tpl"},
			assertions: func(t *testing.T, matchFound bool, err error) {
				require.NoError(t, err)
				require.Equal(t, false, matchFound)
			},
		},
		{
			name:         "success unmatching mix2",
			includePaths: []string{"path1", regexPrefix + "[_-]", globPrefix + "nonexistent"},
			excludePaths: []string{
				regexPrefix + "nonexistent",
				"path1",
				"path1",
				globPrefix + "*.tpl",
				globPrefix + "*/*.tpl",
				globPrefix + "*.tpl",
				"path1",
			},
			diffs: []string{"path1/values.yaml", "path2/_helpers.tpl", "path2/ingress.yaml"},
			assertions: func(t *testing.T, matchFound bool, err error) {
				require.NoError(t, err)
				require.Equal(t, false, matchFound)
			},
		},
		{
			name: "success unmatching mix3",
			includePaths: []string{
				"path1/f",
				regexpPrefix + "path[1-3]",
				globPrefix + "file*",
			},
			excludePaths: []string{
				regexPrefix + "\\d",
				"yaml",
				globPrefix + "*.tpl",
				globPrefix + "*.tpl",
				"nonexistent",
			},
			diffs: []string{"path1/file1", "path2/file2", "path3/file3"},
			assertions: func(t *testing.T, matchFound bool, err error) {
				require.NoError(t, err)
				require.Equal(t, false, matchFound)
			},
		},
		{
			name:         "success unmatching mix2; no config",
			includePaths: []string{},
			excludePaths: []string{},
			diffs:        []string{},
			assertions: func(t *testing.T, matchFound bool, err error) {
				require.NoError(t, err)
				require.Equal(t, false, matchFound)
			},
		},
		{
			name: "success matching mix1",
			includePaths: []string{
				"path1",
				regexPrefix + "[_-]",
				globPrefix + "nonexistent",
				regexPrefix + "no",
				globPrefix + "*/*/*/abe/*",
			},
			excludePaths: []string{
				regexPrefix + "nonexistent",
				"path1",
				"path1",
				globPrefix + "*.tpl",
				globPrefix + "*/*.tpl",
				globPrefix + "*.tpl",
				regexpPrefix + ".*q",
			},
			diffs: []string{
				"path1/values.yaml",
				"path2/_helpers.tpl",
				"abc",
				"abb",
				"aba",
				"abz",
				"aby",
				"abx",
				"abw",
				"abv",
				"abu",
				"path3/abc/abd/abe/deployment.yaml",
				"path4.txt",
			},
			assertions: func(t *testing.T, matchFound bool, err error) {
				require.NoError(t, err)
				require.Equal(t, true, matchFound)
			},
		},
		{
			name: "success matching mix3; no includePaths",
			excludePaths: []string{
				regexpPrefix + "ab[cbazxwvu]",
				"helpers.tpl",
				globPrefix + "path*/*"},
			diffs: []string{
				"path1/values.yaml",
				"path2/_helpers.tpl",
				"abc",
				"abb",
				"aba",
				"abz",
				"aby",
				"abx",
				"abw",
				"abv",
				"abu",
			},
			assertions: func(t *testing.T, matchFound bool, err error) {
				require.NoError(t, err)
				require.Equal(t, true, matchFound)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			includeSelectors, err := getPathSelectors(testCase.includePaths)
			require.NoError(t, err)
			excludeSelectors, err := getPathSelectors(testCase.excludePaths)
			require.NoError(t, err)

			matchFound, err := matchesPathsFilters(includeSelectors, excludeSelectors, testCase.diffs)
			testCase.assertions(t, matchFound, err)
		})
	}
}

func TestShortenString(t *testing.T) {
	testCases := []struct {
		name   string
		str    string
		length int
		want   string
	}{
		{
			name:   "exceeds length",
			str:    "this is a long string",
			length: 10,
			want:   "this is a ...",
		},
		{
			name:   "equal length",
			str:    "this is a long string",
			length: 21,
			want:   "this is a long string",
		},
		{
			name:   "shorter length",
			str:    "this is a long string",
			length: 30,
			want:   "this is a long string",
		},
		{
			name:   "empty string",
			str:    "",
			length: 10,
			want:   "",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.want, shortenString(testCase.str, testCase.length))
		})
	}
}
