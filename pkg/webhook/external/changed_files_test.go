package external

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectChangedFiles(t *testing.T) {
	testCases := []struct {
		name     string
		diffs    []commitDiff
		expected []string
	}{
		{
			name: "nil diffs",
		},
		{
			name:  "empty diffs",
			diffs: []commitDiff{},
		},
		{
			name: "single diff",
			diffs: []commitDiff{{
				Added:    []string{"a.txt", "b.txt"},
				Modified: []string{"c.txt"},
				Removed:  []string{"d.txt"},
			}},
			expected: []string{"a.txt", "b.txt", "c.txt", "d.txt"},
		},
		{
			name: "multiple diffs with deduplication",
			diffs: []commitDiff{
				{
					Added:    []string{"a.txt", "b.txt"},
					Modified: []string{"c.txt"},
				},
				{
					Added:    []string{"b.txt", "d.txt"},
					Modified: []string{"a.txt"},
					Removed:  []string{"c.txt", "e.txt"},
				},
			},
			expected: []string{
				"a.txt", "b.txt", "c.txt", "d.txt", "e.txt",
			},
		},
		{
			name: "empty fields in diff",
			diffs: []commitDiff{
				{Added: []string{"a.txt"}},
				{Modified: []string{"b.txt"}},
				{Removed: []string{"c.txt"}},
			},
			expected: []string{"a.txt", "b.txt", "c.txt"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expected, collectChangedFiles(testCase.diffs))
		})
	}
}
