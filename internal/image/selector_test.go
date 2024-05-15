package image

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSelector(t *testing.T) {
	testCases := []struct {
		name       string
		repoURL    string
		strategy   SelectionStrategy
		opts       *SelectorOptions
		assertions func(t *testing.T, s Selector, err error)
	}{
		{
			name:    "invalid allow regex",
			repoURL: "debian",
			opts: &SelectorOptions{
				AllowRegex: "(invalid", // Invalid regex due to unclosed parenthesis
			},
			assertions: func(t *testing.T, _ Selector, err error) {
				require.ErrorContains(t, err, "error compiling regular expression")
			},
		},
		{
			name:    "invalid platform constraint",
			repoURL: "debian",
			opts: &SelectorOptions{
				Platform: "invalid",
			},
			assertions: func(t *testing.T, _ Selector, err error) {
				require.ErrorContains(t, err, "error parsing platform constraint")
			},
		},
		{
			name:     "invalid selection strategy",
			strategy: SelectionStrategy("invalid"),
			repoURL:  "debian",
			opts: &SelectorOptions{
				Constraint: "invalid", // Not a semver
			},
			assertions: func(t *testing.T, _ Selector, err error) {
				require.ErrorContains(t, err, "invalid image selection strategy")
			},
		},
		{
			name:     "success with digest image selector",
			strategy: SelectionStrategyDigest,
			opts: &SelectorOptions{
				Constraint: "fake-constraint",
			},
			repoURL: "debian",
			assertions: func(t *testing.T, selector Selector, err error) {
				require.NoError(t, err)
				require.IsType(t, &digestSelector{}, selector)
			},
		},
		{
			name:     "success with lexical image selector",
			strategy: SelectionStrategyLexical,
			repoURL:  "debian",
			assertions: func(t *testing.T, selector Selector, err error) {
				require.NoError(t, err)
				require.IsType(t, &lexicalSelector{}, selector)
			},
		},
		{
			name:     "success with newest build image selector",
			strategy: SelectionStrategyNewestBuild,
			repoURL:  "debian",
			assertions: func(t *testing.T, selector Selector, err error) {
				require.NoError(t, err)
				require.IsType(t, &newestBuildSelector{}, selector)
			},
		},
		{
			name:     "success with semver image selector",
			strategy: SelectionStrategySemVer,
			repoURL:  "debian",
			assertions: func(t *testing.T, selector Selector, err error) {
				require.NoError(t, err)
				require.IsType(t, &semVerSelector{}, selector)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := NewSelector(
				testCase.repoURL,
				testCase.strategy,
				testCase.opts,
			)
			testCase.assertions(t, s, err)
		})
	}
}

func TestAllowsTag(t *testing.T) {
	testRegex := regexp.MustCompile("^[a-z]*$")
	testCases := []struct {
		name    string
		tag     string
		allowed bool
	}{
		{
			name:    "tag isn't allowed",
			tag:     "NO",
			allowed: false,
		},
		{
			name:    "tag is allowed",
			tag:     "yes",
			allowed: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.allowed,
				allowsTag(testCase.tag, testRegex),
			)
		})
	}
}

func TestIgnoresTag(t *testing.T) {
	testIgnore := []string{"ignore-me"}
	testCases := []struct {
		name    string
		tag     string
		ignored bool
	}{
		{
			name:    "tag isn't ignored",
			tag:     "allow-me",
			ignored: false,
		},
		{
			name:    "tag is ignored",
			tag:     "ignore-me",
			ignored: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.ignored,
				ignoresTag(testCase.tag, testIgnore),
			)
		})
	}
}
