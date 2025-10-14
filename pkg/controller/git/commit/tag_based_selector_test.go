package commit

import (
	"regexp"
	"testing"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
)

func TestNewTagBasedSelector(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.GitSubscription
		assertions func(*testing.T, *tagBasedSelector, error)
	}{
		{
			name: "error building base selector",
			sub: kargoapi.GitSubscription{
				ExpressionFilter: "(1 + 2", // This will force an error
			}, // No RepoURL
			assertions: func(t *testing.T, _ *tagBasedSelector, err error) {
				require.ErrorContains(t, err, "error building base selector")
			},
		},
		{
			// TODO(v1.13.0): Remove this test once AllowTags is removed.
			name: "error compiling AllowTags regex",
			sub: kargoapi.GitSubscription{
				AllowTags: "[", // Invalid regex
			},
			assertions: func(t *testing.T, _ *tagBasedSelector, err error) {
				require.ErrorContains(t, err, "error compiling regular expression")
			},
		},
		{
			name: "error compiling AllowTagsRegex",
			sub: kargoapi.GitSubscription{
				RepoURL:        "https://github.com/example/repo.git",
				AllowTagsRegex: []string{"["}, // Invalid regex
			},
			assertions: func(t *testing.T, _ *tagBasedSelector, err error) {
				require.ErrorContains(t, err, "error compiling regular expression")
			},
		},
		{
			name: "error compiling IgnoreTagsRegex",
			sub: kargoapi.GitSubscription{
				RepoURL:         "https://github.com/example/repo.git",
				IgnoreTagsRegex: []string{"["}, // Invalid regex
			},
			assertions: func(t *testing.T, _ *tagBasedSelector, err error) {
				require.ErrorContains(t, err, "error compiling regular expression")
			},
		},
		{
			// TODO(v1.13.0): Update this test once AllowTags and IgnoreTags are
			// removed.
			name: "success",
			sub: kargoapi.GitSubscription{
				RepoURL:         "https://github.com/foo/bar",
				AllowTags:       "^v1\.",
				AllowTagsRegex:  []string{"^v2\."},
				IgnoreTags:      []string{"v1.0.0"},
				IgnoreTagsRegex: []string{"^v1\.0\..*"},
			},
			assertions: func(t *testing.T, s *tagBasedSelector, err error) {
				require.NoError(t, err)
				require.NotNil(t, s.baseSelector)
				require.Len(t, s.allowTagsRegex, 2)
				require.Equal(t, "^v2\.", s.allowTagsRegex[0].String())
				require.Equal(t, "^v1\.", s.allowTagsRegex[1].String())
				require.Len(t, s.ignoreTagsRegex, 2)
				require.Equal(t, "^v1\.0\..*", s.ignoreTagsRegex[0].String())
				require.Equal(t, "^v1\.0\.0$", s.ignoreTagsRegex[1].String())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newTagBasedSelector(testCase.sub, nil)
			testCase.assertions(t, s, err)
		})
	}
}

func Test_tagBasedSelector_MatchesRef(t *testing.T) {
	testCases := []struct {
		name        string
		selector    *tagBasedSelector
		ref         string
		shouldMatch bool
	}{
		{
			name:        "invalid tag ref",
			selector:    &tagBasedSelector{},
			ref:         "invalid",
			shouldMatch: false,
		},
		{
			name:        "no regex specified",
			selector:    &tagBasedSelector{},
			ref:         "refs/tags/abc",
			shouldMatch: true,
		},
		{
			name:        "regex matches",
			selector:    &tagBasedSelector{
				allowTagsRegex: []*regexp.Regexp{regexp.MustCompile("[a-z]+")},
			},
			ref:         "refs/tags/abc",
			shouldMatch: true,
		},
		{
			name:        "regex does not match",
			selector:    &tagBasedSelector{
				allowTagsRegex: []*regexp.Regexp{regexp.MustCompile("[a-z]+")},
			},
			ref:         "refs/tags/123",
			shouldMatch: false,
		},
		{
			name:        "ignored",
			selector:    &tagBasedSelector{
				ignoreTagsRegex: []*regexp.Regexp{regexp.MustCompile("^abc$")},
			},
			ref:         "refs/tags/abc",
			shouldMatch: false,
		},
		{
			name:        "not ignored",
			selector:    &tagBasedSelector{
				ignoreTags: []*regexp.Regexp{regexp.MustCompile("^abc$")},
			},
			ref:         "refs/tags/123",
			shouldMatch: true,
		},
		{
			name: "regex matches, but ignored",
			selector: &tagBasedSelector{
				allowTagsRegex:  []*regexp.Regexp{regexp.MustCompile("[a-z]+")},
				ignoreTagsRegex: []*regexp.Regexp{regexp.MustCompile("^abc$")},
			},
			ref:         "refs/tags/abc",
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

func Test_tagBasedSelector_filterTagsByExpression(t *testing.T) {
	testCases := []struct {
		name       string
		tags       []git.TagMetadata
		expression string
		assertions func(*testing.T, []git.TagMetadata, error)
	}{
		{
			name:       "empty expression returns all tags",
			expression: "",
			tags: []git.TagMetadata{
				{Tag: "v1.0.0"},
				{Tag: "v2.0.0"},
			},
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Len(t, tags, 2)
			},
		},
		{
			name:       "boolean expression filters correctly",
			expression: "hasPrefix(tag, 'v1.')",
			tags: []git.TagMetadata{
				{Tag: "v1.0.0"},
				{Tag: "v2.0.0"},
				{Tag: "v1.2.3"},
			},
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Equal(t, []git.TagMetadata{
					{Tag: "v1.0.0"},
					{Tag: "v1.2.3"},
				}, tags)
			},
		},
		{
			name:       "non-boolean expression converted to boolean",
			expression: "1", // Should be converted to true
			tags: []git.TagMetadata{
				{Tag: "v1.0.0"},
				{Tag: "v2.0.0"},
			},
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Len(t, tags, 2)
			},
		},
		{
			name:       "non-boolean expression false",
			expression: "0", // Should be converted to false
			tags: []git.TagMetadata{
				{Tag: "v1.0.0"},
				{Tag: "v2.0.0"},
			},
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Empty(t, tags)
			},
		},
		{
			name:       "error parsing non-boolean result",
			expression: "'invalid'", // Can't be parsed as boolean
			tags:       []git.TagMetadata{{Tag: "v1.0.0"}},
			assertions: func(t *testing.T, _ []git.TagMetadata, err error) {
				require.ErrorContains(t, err, "error parsing expression result")
			},
		},
		{
			name:       "complex expression with tag metadata",
			expression: "tag contains '1.' && author == 'test-author'",
			tags: []git.TagMetadata{
				{Tag: "v1.0.0", Author: "test-author"},
				{Tag: "v1.2.3", Author: "other-author"},
				{Tag: "v2.0.0", Author: "test-author"},
			},
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Equal(t, []git.TagMetadata{
					{Tag: "v1.0.0", Author: "test-author"},
				}, tags)
			},
		},
		{
			name:       "expression with date comparison",
			expression: "creatorDate.Year() >= 2023",
			tags: []git.TagMetadata{
				{Tag: "v1.0.0", CreatorDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
				{Tag: "v2.0.0", CreatorDate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
			assertions: func(t *testing.T, tags []git.TagMetadata, err error) {
				require.NoError(t, err)
				require.Equal(t, []git.TagMetadata{
					{Tag: "v1.0.0", CreatorDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
				}, tags)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var expression *vm.Program
			if testCase.expression != "" {
				var err error
				expression, err = expr.Compile(testCase.expression)
				require.NoError(t, err)
			}
			s := &tagBasedSelector{
				baseSelector: &baseSelector{
					filterExpression: expression,
				},
			}
			tags, err := s.filterTagsByExpression(testCase.tags)
			testCase.assertions(t, tags, err)
		})
	}
}
