package governance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_isMaintainer(t *testing.T) {
	cfg := config{
		MaintainerAssociations: []string{"MEMBER", "OWNER"},
	}
	testCases := []struct {
		name        string
		association string
		expected    bool
	}{
		{name: "MEMBER is maintainer", association: "MEMBER", expected: true},
		{name: "OWNER is maintainer", association: "OWNER", expected: true},
		{name: "case insensitive", association: "member", expected: true},
		{name: "NONE is not maintainer", association: "NONE", expected: false},
		{name: "CONTRIBUTOR is not maintainer", association: "CONTRIBUTOR", expected: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expected, isMaintainer(cfg, testCase.association))
		})
	}
}

func Test_needsLabel(t *testing.T) {
	testCases := []struct {
		name           string
		prefix         string
		existingLabels map[string]struct{}
		expected       bool
	}{
		{
			name:           "label present",
			prefix:         "kind",
			existingLabels: map[string]struct{}{"kind/bug": {}},
			expected:       false,
		},
		{
			name:           "label missing",
			prefix:         "kind",
			existingLabels: map[string]struct{}{"priority/high": {}},
			expected:       true,
		},
		{
			name:           "no labels at all",
			prefix:         "kind",
			existingLabels: map[string]struct{}{},
			expected:       true,
		},
		{
			name:           "prefix without slash does not match",
			prefix:         "kind",
			existingLabels: map[string]struct{}{"kinder": {}},
			expected:       true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := needsLabel(testCase.prefix, testCase.existingLabels)
			require.Equal(t, testCase.expected, result)
		})
	}
}
