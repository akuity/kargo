package server

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_effectiveResourceVersion(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name         string
		rv           string
		itemVersions []string
		expected     string
	}{
		{
			name:     "real list-level rv passthrough",
			rv:       "12345",
			expected: "12345",
		},
		{
			name:         "zero list-level rv falls back to max item rv",
			rv:           "0",
			itemVersions: []string{"100", "300", "200"},
			expected:     "300",
		},
		{
			name:         "empty list-level rv falls back to max item rv",
			rv:           "",
			itemVersions: []string{"100", "300", "200"},
			expected:     "300",
		},
		{
			name:         "zero list-level rv with no items returns empty",
			rv:           "0",
			itemVersions: []string{},
			expected:     "",
		},
		{
			name:         "items with non-numeric rvs are skipped",
			rv:           "0",
			itemVersions: []string{"abc", "100", "def"},
			expected:     "100",
		},
		{
			name:         "all invalid item rvs returns empty",
			rv:           "",
			itemVersions: []string{"", "abc"},
			expected:     "",
		},
		{
			name:         "real list-level rv wins over items",
			rv:           "999",
			itemVersions: []string{"100", "200"},
			expected:     "999",
		},
		{
			name:         "zero item rvs are skipped",
			rv:           "0",
			itemVersions: []string{"0", "0", "100"},
			expected:     "100",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expected, effectiveResourceVersion(tc.rv, tc.itemVersions))
		})
	}
}
