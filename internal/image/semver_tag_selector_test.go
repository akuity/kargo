package image

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSemVerTagSelector(t *testing.T) {
	testAllowRegex := regexp.MustCompile("fake-regex")
	testIgnore := []string{"fake-ignore"}
	testPlatform := &platformConstraint{
		os:   "linux",
		arch: "amd64",
	}
	testCases := []struct {
		name       string
		constraint string
		assertions func(s TagSelector, err error)
	}{
		{
			name:       "invalid semver constraint",
			constraint: "invalid",
			assertions: func(s TagSelector, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error parsing semver constraint")
			},
		},
		{
			name:       "no semver constraint",
			constraint: "",
			assertions: func(s TagSelector, err error) {
				require.NoError(t, err)
				selector, ok := s.(*semVerTagSelector)
				require.True(t, ok)
				require.Equal(t, testAllowRegex, selector.allowRegex)
				require.Equal(t, testIgnore, selector.ignore)
				require.Nil(t, selector.constraint)
				require.Equal(t, testPlatform, selector.platform)
			},
		},
		{
			name:       "valid semver constraint",
			constraint: "^1.24",
			assertions: func(s TagSelector, err error) {
				require.NoError(t, err)
				selector, ok := s.(*semVerTagSelector)
				require.True(t, ok)
				require.Equal(t, testAllowRegex, selector.allowRegex)
				require.Equal(t, testIgnore, selector.ignore)
				require.NotNil(t, selector.constraint)
				require.Equal(t, testPlatform, selector.platform)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				newSemVerTagSelector(
					nil,
					testAllowRegex,
					testIgnore,
					testCase.constraint,
					testPlatform,
				),
			)
		})
	}
}

func TestSortTagsBySemver(t *testing.T) {
	tags := []Tag{
		newTag("5.0.0", nil, ""),
		newTag("0.0.1", nil, ""),
		newTag("0.2.1", nil, ""),
		newTag("0.1.1", nil, ""),
		newTag("1.1.1", nil, ""),
		newTag("7.0.6", nil, ""),
		newTag("1.0.0", nil, ""),
		newTag("1.0.2", nil, ""),
	}
	sortTagsBySemVer(tags)
	require.Equal(
		t,
		[]Tag{
			newTag("7.0.6", nil, ""),
			newTag("5.0.0", nil, ""),
			newTag("1.1.1", nil, ""),
			newTag("1.0.2", nil, ""),
			newTag("1.0.0", nil, ""),
			newTag("0.2.1", nil, ""),
			newTag("0.1.1", nil, ""),
			newTag("0.0.1", nil, ""),
		},
		tags,
	)
}
