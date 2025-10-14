package image

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewTagBasedSelector(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.ImageSubscription
		assertions func(*testing.T, *tagBasedSelector, error)
	}{
		{
			name: "error building base selector",
			sub:  kargoapi.ImageSubscription{}, // No RepoURL
			assertions: func(t *testing.T, _ *tagBasedSelector, err error) {
				require.ErrorContains(t, err, "error building base selector")
			},
		},
		{
			// TODO(v1.13.0): Remove this test once AllowTags is removed.
			name: "error compiling AllowedTags regex",
			sub: kargoapi.ImageSubscription{
				RepoURL:   "example/image",
				AllowTags: "[", // Invalid regex
			},
			assertions: func(t *testing.T, _ *tagBasedSelector, err error) {
				require.ErrorContains(t, err, "error compiling regular expression")
			},
		},
		{
			name: "error compiling AllowTagsRegex",
			sub: kargoapi.ImageSubscription{
				RepoURL:        "example/image",
				AllowTagsRegex: []string{"["}, // Invalid regex
			},
			assertions: func(t *testing.T, _ *tagBasedSelector, err error) {
				require.ErrorContains(t, err, "error compiling regular expression")
			},
		},
		{
			name: "error compiling IgnoreTagsRegex",
			sub: kargoapi.ImageSubscription{
				RepoURL:         "example/image",
				IgnoreTagsRegex: []string{"["}, // Invalid regex
			},
			assertions: func(t *testing.T, _ *tagBasedSelector, err error) {
				require.ErrorContains(t, err, "error compiling ignore tags regex")
			},
		},
		{
			// TODO(v1.13.0): Update this test once AllowTags and IgnoreTags are
			// removed.
			name: "success",
			sub: kargoapi.ImageSubscription{
				RepoURL: "example/image",
				AllowTags:      `^v1\.`,
				AllowTagsRegex: []string{`^v2\.`},
				IgnoreTags:      []string{"v1.0.0"},
				IgnoreTagsRegex: []string{`^v1\.0\..*`},
			},
			assertions: func(t *testing.T, s *tagBasedSelector, err error) {
				require.NoError(t, err)
				require.NotNil(t, s.baseSelector)
				require.Len(t, s.allowTagsRegex, 2)
				require.Equal(t, `^v2\.`, s.allowTagsRegex[0].String())
				require.Equal(t, `^v1\.`, s.allowTagsRegex[1].String())
				require.Len(t, s.ignoreTagsRegex, 2)
				require.Equal(t, `^v1\.0\..*`, s.ignoreTagsRegex[0].String())
				require.Equal(t, `^v1\.0\.0$`, s.ignoreTagsRegex[1].String())
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

func Test_tagBasedSelector_MatchesTag(t *testing.T) {
	testCases := []struct {
		name        string
		selector    *tagBasedSelector
		tag         string
		shouldMatch bool
	}{
		{
			name:        "no regex specified",
			selector:    &tagBasedSelector{},
			tag:         "abc",
			shouldMatch: true,
		},
		{
			name:        "regex matches",
			selector:    &tagBasedSelector{
				allowTagsRegex: []*regexp.Regexp{regexp.MustCompile("[a-z]+")},
			},
			tag:         "abc",
			shouldMatch: true,
		},
		{
			name:        "regex does not match",
			selector:    &tagBasedSelector{
				allowTagsRegex: []*regexp.Regexp{regexp.MustCompile("[a-z]+")},
			},
			tag:         "123",
			shouldMatch: false,
		},
		{
			name:        "ignored",
			selector:    &tagBasedSelector{ignoreTags: []string{"abc"}},
			tag:         "abc",
			shouldMatch: false,
		},
		{
			name:        "not ignored",
			selector:    &tagBasedSelector{ignoreTags: []string{"abc"}},
			tag:         "123",
			shouldMatch: true,
		},
		{
			name: "regex matches, but ignored",
			selector: &tagBasedSelector{
				allowTagsRegex: []*regexp.Regexp{regexp.MustCompile("[a-z]+")},
				ignoreTagsRegex: []*regexp.Regexp{regexp.MustCompile("^abc$")},
			},
			tag:         "abc",
			shouldMatch: false,
		},
		{
			name: "regex matches, not ignored by ignoreRegex",
			selector: &tagBasedSelector{
				allowTagsRegex:  []*regexp.Regexp{regexp.MustCompile("[a-z]+")},
				ignoreTagsRegex: []*regexp.Regexp{regexp.MustCompile("^def$")},
			},
			tag:         "abc",
			shouldMatch: true,
		},
		{
			name: "regex matches, but ignored by both ignoreTags and ignoreRegex",
			selector: &tagBasedSelector{
				allowTagsRegex:  []*regexp.Regexp{regexp.MustCompile("[a-z]+")},
				ignoreTags:      []string{"abc"},
				ignoreTagsRegex: []*regexp.Regexp{regexp.MustCompile("^abc$")},
			},
			tag:         "abc",
			shouldMatch: false,
		},
		{
			name: "regex matches, ignored by ignoreRegex but not ignoreTags",
			selector: &tagBasedSelector{
				allowTagsRegex:  []*regexp.Regexp{regexp.MustCompile("[a-z]+")},
				ignoreTags:      []string{"def"},
				ignoreTagsRegex: []*regexp.Regexp{regexp.MustCompile("^abc$")},
			},
			tag:         "abc",
			shouldMatch: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.shouldMatch,
				testCase.selector.MatchesTag(testCase.tag),
			)
		})
	}
}

func Test_tagBasedSelector_filterTags(t *testing.T) {
	testCases := []struct {
		name     string
		selector *tagBasedSelector
		input    []string
		expected []string
	}{
		{
			name: "basic filtering with allows and ignores",
			selector: &tagBasedSelector{
				allowTagsRegex: []*regexp.Regexp{regexp.MustCompile(`v1\.`)},
				ignoreTags:     []string{"v1.0.0"},
			},
			input: []string{
				"v1.0.0", // Allowed, but ignored
				"v1.1.0", // Allowed
				"v2.0.0", // Not allowed
			},
			expected: []string{"v1.1.0"},
		},
		{
			name: "filtering with ignoreRegex",
			selector: &tagBasedSelector{
				allowTagsRegex:  []*regexp.Regexp{regexp.MustCompile(`v1\.`)},
				ignoreTagsRegex: []*regexp.Regexp{regexp.MustCompile(`v1\.0\..*`)},
			},
			input: []string{
				"v1.0.0", // Allowed, but ignored by regex
				"v1.0.1", // Allowed, but ignored by regex
				"v1.1.0", // Allowed, not ignored
				"v2.0.0", // Not allowed
			},
			expected: []string{"v1.1.0"},
		},
		{
			name: "filtering with both ignoreTags and ignoreRegex",
			selector: &tagBasedSelector{
				allowTagsRegex:  []*regexp.Regexp{regexp.MustCompile(`v1\.`)},
				ignoreTags:      []string{"v1.0.0"},
				ignoreTagsRegex: []*regexp.Regexp{regexp.MustCompile(`v1\.1\..*`)},
			},
			input: []string{
				"v1.0.0", // Allowed, but ignored by ignoreTags
				"v1.0.1", // Allowed, not ignored
				"v1.1.0", // Allowed, but ignored by ignoreRegex
				"v1.1.1", // Allowed, but ignored by ignoreRegex
				"v2.0.0", // Not allowed
			},
			expected: []string{"v1.0.1"},
		},
		{
			name:     "no constraints",
			selector: &tagBasedSelector{},
			input: []string{
				"v1.0.0",
				"v1.1.0",
				"v2.0.0",
			},
			expected: []string{"v1.0.0", "v1.1.0", "v2.0.0"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			filtered := testCase.selector.filterTags(testCase.input)
			require.Equal(t, testCase.expected, filtered)
		})
	}
}
