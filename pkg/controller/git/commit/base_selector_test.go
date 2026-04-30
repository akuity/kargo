package commit

import (
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
)

func TestNewBaseSelector(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.GitSubscription
		creds      *git.RepoCredentials
		assertions func(*testing.T, *baseSelector, error)
	}{
		{
			name: "error parsing filter expression",
			sub:  kargoapi.GitSubscription{ExpressionFilter: "(1 + 2"},
			assertions: func(t *testing.T, _ *baseSelector, err error) {
				require.ErrorContains(t, err, "error compiling filter expression")
			},
		},
		{
			name: "error parsing include path selectors",
			sub: kargoapi.GitSubscription{
				IncludePaths: []string{"regex:["}, // Bad regex
			},
			assertions: func(t *testing.T, _ *baseSelector, err error) {
				require.ErrorContains(t, err, "error parsing include path selectors")
			},
		},
		{
			name: "error parsing exclude path selectors",
			sub: kargoapi.GitSubscription{
				ExcludePaths: []string{"regex:["}, // Bad regex
			},
			assertions: func(t *testing.T, _ *baseSelector, err error) {
				require.ErrorContains(t, err, "error parsing exclude path selectors")
			},
		},
		{
			name: "success",
			sub: kargoapi.GitSubscription{
				RepoURL:               "https://github.com/example/repo.git",
				InsecureSkipTLSVerify: true,
				ExpressionFilter:      "false",
				IncludePaths:          []string{"apps/"},
				ExcludePaths:          []string{"hack/"},
				DiscoveryLimit:        5,
			},
			creds: &git.RepoCredentials{
				Username: "foo",
				Password: "bar",
			},
			assertions: func(t *testing.T, s *baseSelector, err error) {
				require.NoError(t, err)
				require.Equal(t, "https://github.com/example/repo.git", s.repoURL)
				require.Equal(
					t,
					&git.RepoCredentials{
						Username: "foo",
						Password: "bar",
					},
					s.creds,
				)
				require.True(t, s.insecureSkipTLSVerify)
				require.NotNil(t, s.filterExpression)
				require.NotNil(t, s.includePaths)
				require.NotNil(t, s.excludePaths)
				require.Equal(t, 5, s.discoveryLimit)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newBaseSelector(testCase.sub, testCase.creds)
			testCase.assertions(t, s, err)
		})
	}
}

func TestBaseSelector_MatchesPaths(t *testing.T) {
	const (
		regexpPrefix = "regexp:"
		regexPrefix  = "regex:"
		globPrefix   = "glob:"
	)
	testCases := []struct {
		name         string
		includePaths []string
		excludePaths []string
		diffs        []string
		assertions   func(*testing.T, bool)
	}{
		{
			name:         "success with no includePaths configured",
			excludePaths: []string{regexpPrefix + "nonexistent"},
			diffs:        []string{"path1/values.yaml", "path2/_helpers.tpl"},
			assertions: func(t *testing.T, matchFound bool) {
				require.Equal(t, true, matchFound)
			},
		},
		{
			name:         "success with a matching regexp filters configuration",
			includePaths: []string{regexpPrefix + "values\\.ya?ml$"},
			excludePaths: []string{regexPrefix + "nonexistent"},
			diffs:        []string{"path1/values.yaml", "path2/_helpers.tpl"},
			assertions: func(t *testing.T, matchFound bool) {
				require.Equal(t, true, matchFound)
			},
		},
		{
			name:         "success with unmatching regexp filters configuration",
			includePaths: []string{regexpPrefix + "values\\.ya?ml$"},
			excludePaths: []string{regexPrefix + "nonexistent", regexpPrefix + ".*val.*"},
			diffs:        []string{"path1/values.yaml", "path2/_helpers.tpl"},
			assertions: func(t *testing.T, matchFound bool) {
				require.Equal(t, false, matchFound)
			},
		},
		{
			name:         "success with matching glob filters configuration",
			includePaths: []string{"glob:path2/*.tpl"},
			excludePaths: []string{"nonexistent"},
			diffs:        []string{"path1/values.yaml", "path2/_helpers.tpl"},
			assertions: func(t *testing.T, matchFound bool) {
				require.Equal(t, true, matchFound)
			},
		},
		{
			name:         "success with unmatching glob filters configuration",
			includePaths: []string{"path2/*.tpl"},
			excludePaths: []string{regexPrefix + "nonexistent", "*/?helpers.tpl"},
			diffs:        []string{"path1/values.yaml", "path2/_helpers.tpl"},
			assertions: func(t *testing.T, matchFound bool) {
				require.Equal(t, false, matchFound)
			},
		},
		{
			name:         "success with matching prefix filters configuration",
			includePaths: []string{"path1/"},
			excludePaths: []string{"nonexistent"},
			diffs:        []string{"path1/values.yaml", "path2/_helpers.tpl"},
			assertions: func(t *testing.T, matchFound bool) {
				require.Equal(t, true, matchFound)
			},
		},
		{
			name:         "success with unmatching prefix filters configuration",
			includePaths: []string{"path3/"},
			excludePaths: []string{regexPrefix + "nonexistent", "*/?helpers.tpl"},
			diffs:        []string{"path1/values.yaml", "path2/_helpers.tpl"},
			assertions: func(t *testing.T, matchFound bool) {
				require.Equal(t, false, matchFound)
			},
		},
		{
			name:         "success unmatching mix1",
			includePaths: []string{"path3", regexPrefix + "nonexistent", globPrefix + "nonexistent"},
			excludePaths: []string{regexPrefix + "nonexistent", "*/?helpers.tpl", globPrefix + "nonexistent"},
			diffs:        []string{"path1/values.yaml", "path2/_helpers.tpl"},
			assertions: func(t *testing.T, matchFound bool) {
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
			assertions: func(t *testing.T, matchFound bool) {
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
			assertions: func(t *testing.T, matchFound bool) {
				require.Equal(t, false, matchFound)
			},
		},
		{
			name:         "success unmatching mix2; no config",
			includePaths: []string{},
			excludePaths: []string{},
			diffs:        []string{},
			assertions: func(t *testing.T, matchFound bool) {
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
			assertions: func(t *testing.T, matchFound bool) {
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
			assertions: func(t *testing.T, matchFound bool) {
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
			b := &baseSelector{
				includePaths: includeSelectors,
				excludePaths: excludeSelectors,
			}
			matchFound := b.MatchesPaths(testCase.diffs)
			testCase.assertions(t, matchFound)
		})
	}
}
