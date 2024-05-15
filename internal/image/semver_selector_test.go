package image

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSemVerSelector(t *testing.T) {
	testAllowRegex := regexp.MustCompile("fake-regex")
	testIgnore := []string{"fake-ignore"}
	testPlatform := &platformConstraint{
		os:   "linux",
		arch: "amd64",
	}
	testDiscoveryLimit := 10
	testCases := []struct {
		name       string
		constraint string
		assertions func(t *testing.T, s Selector, err error)
	}{
		{
			name:       "invalid semver constraint",
			constraint: "invalid",
			assertions: func(t *testing.T, _ Selector, err error) {
				require.ErrorContains(t, err, "error parsing semver constraint")
			},
		},
		{
			name:       "no semver constraint",
			constraint: "",
			assertions: func(t *testing.T, s Selector, err error) {
				require.NoError(t, err)
				selector, ok := s.(*semVerSelector)
				require.True(t, ok)
				require.Equal(t, testAllowRegex, selector.allowRegex)
				require.Equal(t, testIgnore, selector.ignore)
				require.Nil(t, selector.constraint)
				require.Equal(t, testPlatform, selector.platform)
				require.Equal(t, testDiscoveryLimit, selector.discoveryLimit)
			},
		},
		{
			name:       "valid semver constraint",
			constraint: "^1.24",
			assertions: func(t *testing.T, s Selector, err error) {
				require.NoError(t, err)
				selector, ok := s.(*semVerSelector)
				require.True(t, ok)
				require.Equal(t, testAllowRegex, selector.allowRegex)
				require.Equal(t, testIgnore, selector.ignore)
				require.NotNil(t, selector.constraint)
				require.Equal(t, testPlatform, selector.platform)
				require.Equal(t, testDiscoveryLimit, selector.discoveryLimit)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newSemVerSelector(
				nil,
				testAllowRegex,
				testIgnore,
				testCase.constraint,
				testPlatform,
				testDiscoveryLimit,
			)
			testCase.assertions(t, s, err)
		})
	}
}

func TestSortImagesBySemver(t *testing.T) {
	images := []Image{
		newImage("5.0.0", "", nil),
		newImage("0.0.1", "", nil),
		newImage("0.2.1", "", nil),
		newImage("0.1.1", "", nil),
		newImage("1.1.1", "", nil),
		newImage("7.0.6", "", nil),
		newImage("1.0.0", "", nil),
		newImage("1.0.2", "", nil),
	}
	sortImagesBySemVer(images)
	require.Equal(
		t,
		[]Image{
			newImage("7.0.6", "", nil),
			newImage("5.0.0", "", nil),
			newImage("1.1.1", "", nil),
			newImage("1.0.2", "", nil),
			newImage("1.0.0", "", nil),
			newImage("0.2.1", "", nil),
			newImage("0.1.1", "", nil),
			newImage("0.0.1", "", nil),
		},
		images,
	)
}
