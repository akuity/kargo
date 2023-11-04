package image

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/distribution/distribution/v3/registry/client/auth/challenge"
	"github.com/stretchr/testify/require"
)

func TestNewTagSelector(t *testing.T) {
	getChallengeManagerBackup := getChallengeManager
	getChallengeManager = func(
		string,
		http.RoundTripper,
	) (challenge.Manager, error) {
		return challenge.NewSimpleManager(), nil
	}
	defer func() {
		getChallengeManager = getChallengeManagerBackup
	}()

	testCases := []struct {
		name       string
		repoURL    string
		strategy   TagSelectionStrategy
		opts       *TagSelectorOptions
		assertions func(s TagSelector, err error)
	}{
		{
			name:    "invalid allow regex",
			repoURL: "debian",
			opts: &TagSelectorOptions{
				AllowRegex: "(invalid", // Invalid regex due to unclosed parenthesis
			},
			assertions: func(_ TagSelector, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error compiling regular expression")
			},
		},
		{
			name:    "invalid platform constraint",
			repoURL: "debian",
			opts: &TagSelectorOptions{
				Platform: "invalid",
			},
			assertions: func(_ TagSelector, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error parsing platform constraint")
			},
		},
		{
			name:     "invalid selection strategy",
			strategy: TagSelectionStrategy("invalid"),
			repoURL:  "debian",
			opts: &TagSelectorOptions{
				Constraint: "invalid", // Not a semver
			},
			assertions: func(_ TagSelector, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid tag selection strategy")
			},
		},
		{
			name:     "success with digest tag selector",
			strategy: TagSelectionStrategyDigest,
			opts: &TagSelectorOptions{
				Constraint: "fake-constraint",
			},
			repoURL: "debian",
			assertions: func(selector TagSelector, err error) {
				require.NoError(t, err)
				require.IsType(t, &digestTagSelector{}, selector)
			},
		},
		{
			name:     "success with lexical tag selector",
			strategy: TagSelectionStrategyLexical,
			repoURL:  "debian",
			assertions: func(selector TagSelector, err error) {
				require.NoError(t, err)
				require.IsType(t, &lexicalTagSelector{}, selector)
			},
		},
		{
			name:     "success with newest build tag selector",
			strategy: TagSelectionStrategyNewestBuild,
			repoURL:  "debian",
			assertions: func(selector TagSelector, err error) {
				require.NoError(t, err)
				require.IsType(t, &newestBuildTagSelector{}, selector)
			},
		},
		{
			name:     "success with semver tag selector",
			strategy: TagSelectionStrategySemVer,
			repoURL:  "debian",
			assertions: func(selector TagSelector, err error) {
				require.NoError(t, err)
				require.IsType(t, &semVerTagSelector{}, selector)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := NewTagSelector(
				testCase.repoURL,
				testCase.strategy,
				testCase.opts,
			)
			testCase.assertions(s, err)
		})
	}
}

func TestAllows(t *testing.T) {
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
				allows(testCase.tag, testRegex),
			)
		})
	}
}

func TestIgnores(t *testing.T) {
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
				ignores(testCase.tag, testIgnore),
			)
		})
	}
}
