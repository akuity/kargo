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
			name: "error parsing allowed tags regex",
			sub: kargoapi.ImageSubscription{
				RepoURL:   "example/image",
				AllowTags: "[", // Invalid regex
			},
			assertions: func(t *testing.T, _ *tagBasedSelector, err error) {
				require.ErrorContains(t, err, "error compiling regular expression")
			},
		},
		{
			name: "success",
			sub: kargoapi.ImageSubscription{
				RepoURL:        "example/image",
				AllowTags:      `^v1\.`,
				IgnoreTags:     []string{"v1.0.0"},
				DiscoveryLimit: 5,
			},
			assertions: func(t *testing.T, s *tagBasedSelector, err error) {
				require.NoError(t, err)
				require.NotNil(t, s.baseSelector)
				require.NotNil(t, s.allows)
				require.Equal(t, []string{"v1.0.0"}, s.ignores)
				require.Equal(t, 5, s.discoveryLimit)
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
			selector:    &tagBasedSelector{allows: regexp.MustCompile("[a-z]+")},
			tag:         "abc",
			shouldMatch: true,
		},
		{
			name:        "regex does not match",
			selector:    &tagBasedSelector{allows: regexp.MustCompile("[a-z]+")},
			tag:         "123",
			shouldMatch: false,
		},
		{
			name:        "ignored",
			selector:    &tagBasedSelector{ignores: []string{"abc"}},
			tag:         "abc",
			shouldMatch: false,
		},
		{
			name:        "not ignored",
			selector:    &tagBasedSelector{ignores: []string{"abc"}},
			tag:         "123",
			shouldMatch: true,
		},
		{
			name: "regex matches, but ignored",
			selector: &tagBasedSelector{
				allows:  regexp.MustCompile("[a-z]+"),
				ignores: []string{"abc"},
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
		allows:  regexp.MustCompile(`v1\.`),
		ignores: []string{"v1.0.0"},
	}).filterTags([]string{
		"v1.0.0", // Allowed, but ignored
		"v1.1.0", // Allowed
		"v2.0.0", // Not allowed
	})
	require.Equal(t, []string{"v1.1.0"}, filtered)
}
