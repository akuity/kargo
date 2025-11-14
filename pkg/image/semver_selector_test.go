package image

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewSemverSelector(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.ImageSubscription
		assertions func(*testing.T, Selector, error)
	}{
		{
			name: "error building tag based selector",
			sub:  kargoapi.ImageSubscription{}, // No RepoURL
			assertions: func(t *testing.T, _ Selector, err error) {
				require.ErrorContains(t, err, "error building tag based selector")
			},
		},
		{
			name: "error parsing semver constraint",
			sub: kargoapi.ImageSubscription{
				RepoURL:    "example/image",
				Constraint: "invalid",
			},
			assertions: func(t *testing.T, _ Selector, err error) {
				require.ErrorContains(t, err, "error parsing semver constraint")
			},
		},
		{
			name: "success -- with constraint specified",
			sub: kargoapi.ImageSubscription{
				RepoURL:    "example/image",
				Constraint: "^v1.0.0",
			},
			assertions: func(t *testing.T, sel Selector, err error) {
				require.NoError(t, err)
				s, ok := sel.(*semverSelector)
				require.True(t, ok)
				require.NotNil(t, s.tagBasedSelector)
			},
		},
		{
			name: "success -- with no constraint specified",
			sub: kargoapi.ImageSubscription{
				RepoURL: "example/image",
			},
			assertions: func(t *testing.T, selector Selector, err error) {
				require.NoError(t, err)
				s, ok := selector.(*semverSelector)
				require.True(t, ok)
				require.Nil(t, s.constraint)
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

func Test_semverSelector_MatchesTag(t *testing.T) {
	testCases := []struct {
		name        string
		selector    *semverSelector
		tag         string
		shouldMatch bool
	}{
		{
			name:        "non semver ignored",
			selector:    &semverSelector{tagBasedSelector: &tagBasedSelector{}},
			tag:         "foo",
			shouldMatch: false,
		},
		{
			name: "non strict semver ignored",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{},
				strictSemvers:    true,
			},
			tag:         "v1.0", // Not strict
			shouldMatch: false,
		},
		{
			name:        "no regex specified",
			selector:    &semverSelector{tagBasedSelector: &tagBasedSelector{}},
			tag:         "v1.0.0",
			shouldMatch: true,
		},
		{
			name: "regex matches",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{
					allowTagsRegexes: []*regexp.Regexp{regexp.MustCompile(`^v1\.`)},
				},
			},
			tag:         "v1.0.0",
			shouldMatch: true,
		},
		{
			name: "regex does not match",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{
					allowTagsRegexes: []*regexp.Regexp{regexp.MustCompile(`^v1\.`)},
				},
			},
			tag:         "v2.0.0",
			shouldMatch: false,
		},
		{
			name: "ignored",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{
					ignoreTagsRegexes: []*regexp.Regexp{regexp.MustCompile(`^v2\.0\.0$`)},
				},
			},
			tag:         "v2.0.0",
			shouldMatch: false,
		},
		{
			name: "not ignored",
			selector: &semverSelector{
				tagBasedSelector: &tagBasedSelector{
					ignoreTagsRegexes: []*regexp.Regexp{regexp.MustCompile(`^v2\.0\.0$`)},
				},
			},
			tag:         "v1.0.0",
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
			tag:         "v1.0.0",
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

func Test_semVerSelector_sort(t *testing.T) {
	unsorted := []string{
		"5.0.0",
		"0.0.1",
		"0.2.1",
		"0.1.1",
		"1.1.1",
		"7.0.6",
		"1.0.0",
		"1.0.2",
	}

	sorted := (&semverSelector{}).sort(unsorted)

	require.Equal(
		t,
		[]string{
			"7.0.6",
			"5.0.0",
			"1.1.1",
			"1.0.2",
			"1.0.0",
			"0.2.1",
			"0.1.1",
			"0.0.1",
		},
		sorted,
	)
}
