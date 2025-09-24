package chart

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_baseSelector_semversToVersionStrings(t *testing.T) {
	testCases := []struct {
		name     string
		input    semver.Collection
		expected []string
	}{
		{
			name:     "empty collection",
			input:    semver.Collection{},
			expected: []string{},
		},
		{
			name: "valid semvers",
			input: semver.Collection{
				semver.MustParse("1.2.3"),
				semver.MustParse("4.5.6"),
				semver.MustParse("7.8.9"),
			},
			expected: []string{"1.2.3", "4.5.6", "7.8.9"},
		},
		{
			name: "prerelease versions",
			input: semver.Collection{
				semver.MustParse("1.2.3-alpha.1"),
				semver.MustParse("4.5.6-beta.2"),
				semver.MustParse("7.8.9-rc.3"),
			},
			expected: []string{"1.2.3-alpha.1", "4.5.6-beta.2", "7.8.9-rc.3"},
		},
		{
			name: "metadata versions",
			input: semver.Collection{
				semver.MustParse("1.2.3+build.1"),
				semver.MustParse("4.5.6+build.2"),
				semver.MustParse("7.8.9+build.3"),
			},
			expected: []string{"1.2.3+build.1", "4.5.6+build.2", "7.8.9+build.3"},
		},
	}
	s := &baseSelector{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, s.semversToVersionStrings(tc.input))
		})
	}
}

func Test_baseSelector_filterSemvers(t *testing.T) {
	testCases := []struct {
		name       string
		constraint string
		input      semver.Collection
		expected   semver.Collection
	}{
		{
			name:       "empty collection",
			constraint: "^1.2.3",
			input:      semver.Collection{},
			expected:   semver.Collection{},
		},
		{
			name:       "exact version constraint",
			constraint: "=4.5.6",
			input: semver.Collection{
				semver.MustParse("1.2.3"),
				semver.MustParse("4.5.6"),
				semver.MustParse("7.8.9"),
			},
			expected: semver.Collection{
				semver.MustParse("4.5.6"),
			},
		},
		{
			name:       "range constraint",
			constraint: ">=4.5.6 <7.8.9",
			input: semver.Collection{
				semver.MustParse("1.2.3"),
				semver.MustParse("4.5.6"),
				semver.MustParse("7.8.9"),
			},
			expected: semver.Collection{
				semver.MustParse("4.5.6"),
			},
		},
		{
			name:       "prerelease constraint",
			constraint: "1.2.x-0",
			input: semver.Collection{
				semver.MustParse("1.2.3-alpha.1"),
				semver.MustParse("1.2.3-beta.2"),
				semver.MustParse("1.3.0"),
			},
			expected: semver.Collection{
				semver.MustParse("1.2.3-alpha.1"),
				semver.MustParse("1.2.3-beta.2"),
			},
		},
		{
			name:       "multiple matches",
			constraint: "1.2.x",
			input: semver.Collection{
				semver.MustParse("1.2.3"),
				semver.MustParse("1.2.4"),
				semver.MustParse("1.3.0"),
			},
			expected: semver.Collection{
				semver.MustParse("1.2.3"),
				semver.MustParse("1.2.4"),
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s := &baseSelector{}
			if testCase.constraint != "" {
				var err error
				s.constraint, err = semver.NewConstraint(testCase.constraint)
				require.NoError(t, err)
			}
			assert.Equal(t, testCase.expected, s.filterSemvers(testCase.input))
		})
	}
}
