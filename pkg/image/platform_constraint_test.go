package image

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlatformConstraintString(t *testing.T) {
	testCases := []struct {
		name     string
		platform *platformConstraint
		expected string
	}{
		{
			name: "without variant",
			platform: &platformConstraint{
				os:   "linux",
				arch: "amd64",
			},
			expected: "linux/amd64",
		},
		{
			name: "with variant",
			platform: &platformConstraint{
				os:      "linux",
				arch:    "amd64",
				variant: "fake-variant",
			},
			expected: "linux/amd64/fake-variant",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expected, testCase.platform.String())
		})
	}
}

func TestValidatePlatformConstraint(t *testing.T) {
	testCases := []struct {
		name        string
		platformStr string
		valid       bool
	}{
		{
			name:        "invalid",
			platformStr: "invalid",
			valid:       false,
		},
		{
			name:        "valid without variant",
			platformStr: "linux/amd64",
			valid:       true,
		},
		{
			name:        "valid with variant",
			platformStr: "linux/amd64/fake-variant",
			valid:       true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.valid,
				ValidatePlatformConstraint(testCase.platformStr),
			)
		})
	}
}

func TestParsePlatformConstraint(t *testing.T) {
	testCases := []struct {
		name        string
		platformStr string
		assertions  func(t *testing.T, p *platformConstraint, err error)
	}{
		{
			name:        "invalid",
			platformStr: "invalid",
			assertions: func(t *testing.T, _ *platformConstraint, err error) {
				require.ErrorContains(t, err, "error parsing platform constraint")
			},
		},
		{
			name:        "valid without variant",
			platformStr: "linux/amd64",
			assertions: func(t *testing.T, p *platformConstraint, err error) {
				require.NoError(t, err)
				require.Equal(t, "linux", p.os)
				require.Equal(t, "amd64", p.arch)
				require.Empty(t, p.variant)
			},
		},
		{
			name:        "valid with variant",
			platformStr: "linux/amd64/fake-variant",
			assertions: func(t *testing.T, p *platformConstraint, err error) {
				require.NoError(t, err)
				require.Equal(t, "linux", p.os)
				require.Equal(t, "amd64", p.arch)
				require.Equal(t, "fake-variant", p.variant)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			p, err := parsePlatformConstraint(testCase.platformStr)
			testCase.assertions(t, p, err)
		})
	}
}

func TestPlatformConstraintMatches(t *testing.T) {
	testCases := []struct {
		name       string
		os         string
		arch       string
		variant    string
		constraint *platformConstraint
		matches    bool
	}{
		{
			name: "matches",
			os:   "linux",
			arch: "amd64",
			constraint: &platformConstraint{
				os:   "linux",
				arch: "amd64",
			},
			matches: true,
		},
		{
			name: "does not match",
			os:   "linux",
			arch: "arm64",
			constraint: &platformConstraint{
				os:   "linux",
				arch: "amd64",
			},
			matches: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.matches,
				testCase.constraint.matches(
					testCase.os,
					testCase.arch,
					testCase.variant,
				),
			)
		})
	}
}
