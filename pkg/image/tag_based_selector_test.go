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
			name: "error compiling AllowTagsRegexes",
			sub: kargoapi.ImageSubscription{
				RepoURL:          "example/image",
				AllowTagsRegexes: []string{"["}, // Invalid regex
			},
			assertions: func(t *testing.T, _ *tagBasedSelector, err error) {
				require.ErrorContains(t, err, "error compiling regular expression")
			},
		},
		{
			name: "error compiling IgnoreTagsRegexes",
			sub: kargoapi.ImageSubscription{
				RepoURL:           "example/image",
				IgnoreTagsRegexes: []string{"["}, // Invalid regex
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
				RepoURL:           "example/image",
				AllowTags:         `^v1\.`,
				AllowTagsRegexes:  []string{`^v2\.`},
				IgnoreTags:        []string{`v1.0.0`},
				IgnoreTagsRegexes: []string{`^v1\.0\..*`},
			},
			assertions: func(t *testing.T, s *tagBasedSelector, err error) {
				require.NoError(t, err)
				require.NotNil(t, s.baseSelector)
				require.Len(t, s.allowTagsRegexes, 2)
				require.Equal(t, `^v2\.`, s.allowTagsRegexes[0].String())
				require.Equal(t, `^v1\.`, s.allowTagsRegexes[1].String())
				require.Len(t, s.ignoreTagsRegexes, 2)
				require.Equal(t, `^v1\.0\..*`, s.ignoreTagsRegexes[0].String())
				require.Equal(t, `^v1\.0\.0$`, s.ignoreTagsRegexes[1].String())
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
			name: "regex matches",
			selector: &tagBasedSelector{
				allowTagsRegexes: []*regexp.Regexp{regexp.MustCompile("[a-z]+")},
			},
			tag:         "abc",
			shouldMatch: true,
		},
		{
			name: "regex does not match",
			selector: &tagBasedSelector{
				allowTagsRegexes: []*regexp.Regexp{regexp.MustCompile("[a-z]+")},
			},
			tag:         "123",
			shouldMatch: false,
		},
		{
			name: "ignored",
			selector: &tagBasedSelector{
				ignoreTagsRegexes: []*regexp.Regexp{regexp.MustCompile("^abc$")},
			},
			tag:         "abc",
			shouldMatch: false,
		},
		{
			name: "not ignored",
			selector: &tagBasedSelector{
				ignoreTagsRegexes: []*regexp.Regexp{regexp.MustCompile("^abc$")},
			},
			tag:         "123",
			shouldMatch: true,
		},
		{
			name: "regex matches, but ignored",
			selector: &tagBasedSelector{
				allowTagsRegexes:  []*regexp.Regexp{regexp.MustCompile("[a-z]+")},
				ignoreTagsRegexes: []*regexp.Regexp{regexp.MustCompile("^abc$")},
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
	filtered := (&tagBasedSelector{
		allowTagsRegexes:  []*regexp.Regexp{regexp.MustCompile(`v1\.`)},
		ignoreTagsRegexes: []*regexp.Regexp{regexp.MustCompile(`^v1\.0\.0$`)},
	}).filterTags([]string{
		"v1.0.0", // Allowed, but ignored
		"v1.1.0", // Allowed
		"v2.0.0", // Not allowed
	})
	require.Equal(t, []string{"v1.1.0"}, filtered)
}
