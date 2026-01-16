package external

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchesPathFilters(t *testing.T) {
	testCases := []struct {
		name         string
		includePaths []string
		excludePaths []string
		changedPaths []string
		expected     bool
	}{
		{
			name:         "no paths provided - allows refresh",
			includePaths: []string{"apps/project-a"},
			excludePaths: nil,
			changedPaths: []string{},
			expected:     true, // No paths to filter, allow refresh
		},
		{
			name:         "path matches includePaths",
			includePaths: []string{"apps/project-a"},
			excludePaths: nil,
			changedPaths: []string{"apps/project-a/deployment.yaml"},
			expected:     true,
		},
		{
			name:         "path doesn't match includePaths",
			includePaths: []string{"apps/project-a"},
			excludePaths: nil,
			changedPaths: []string{"apps/project-b/deployment.yaml"},
			expected:     false,
		},
		{
			name:         "multiple paths - one matches",
			includePaths: []string{"apps/project-a"},
			excludePaths: nil,
			changedPaths: []string{
				"apps/project-b/deployment.yaml",
				"apps/project-a/config.yaml",
				"apps/project-c/service.yaml",
			},
			expected: true,
		},
		{
			name:         "multiple paths - none match",
			includePaths: []string{"apps/project-a"},
			excludePaths: nil,
			changedPaths: []string{
				"apps/project-b/deployment.yaml",
				"apps/project-c/service.yaml",
			},
			expected: false,
		},
		{
			name:         "path matches but is excluded",
			includePaths: []string{"apps/project-a"},
			excludePaths: []string{"apps/project-a/.kargo"},
			changedPaths: []string{"apps/project-a/.kargo/metadata.yaml"},
			expected:     false,
		},
		{
			name:         "path matches include and not excluded",
			includePaths: []string{"apps/project-a"},
			excludePaths: []string{"apps/project-a/.kargo"},
			changedPaths: []string{"apps/project-a/deployment.yaml"},
			expected:     true,
		},
		{
			name:         "no includePaths - all paths included by default",
			includePaths: nil,
			excludePaths: []string{"*/.kargo"},
			changedPaths: []string{"apps/project-a/deployment.yaml"},
			expected:     true,
		},
		{
			name:         "no includePaths but path is excluded",
			includePaths: nil,
			excludePaths: []string{"apps/project-a/.kargo"},
			changedPaths: []string{"apps/project-a/.kargo/metadata.yaml"},
			expected:     false,
		},
		{
			name:         "glob pattern in includePaths",
			includePaths: []string{"glob:apps/*/deployment.yaml"},
			excludePaths: nil,
			changedPaths: []string{"apps/project-a/deployment.yaml"},
			expected:     true,
		},
		{
			name:         "multiple include patterns",
			includePaths: []string{"apps/project-a", "apps/project-b"},
			excludePaths: nil,
			changedPaths: []string{"apps/project-b/service.yaml"},
			expected:     true,
		},
		{
			name:         "complex real-world scenario",
			includePaths: []string{"apps/blueprint"},
			excludePaths: []string{"apps/blueprint/.kargo"},
			changedPaths: []string{
				"apps/other-project/deployment.yaml",
				"apps/blueprint/config.yaml",
				"README.md",
			},
			expected: true,
		},
		{
			name:         "complex real-world scenario - only excluded files",
			includePaths: []string{"apps/blueprint"},
			excludePaths: []string{"apps/blueprint/.kargo"},
			changedPaths: []string{
				"apps/other-project/deployment.yaml",
				"apps/blueprint/.kargo/metadata.yaml",
				"README.md",
			},
			expected: false,
		},
		{
			name:         "invalid glob include pattern - allows refresh as fallback",
			includePaths: []string{"glob:["},
			excludePaths: nil,
			changedPaths: []string{"apps/project-a/deployment.yaml"},
			expected:     true, // Should allow refresh when pattern is invalid
		},
		{
			name:         "invalid regex exclude pattern - allows refresh as fallback",
			includePaths: []string{"apps/project-a"},
			excludePaths: []string{"regex:[a-z"},
			changedPaths: []string{"apps/project-a/deployment.yaml"},
			expected:     true, // Should allow refresh when pattern is invalid
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matchesPathFilters(tc.includePaths, tc.excludePaths, tc.changedPaths)
			require.Equal(t, tc.expected, result,
				"includePaths: %v, excludePaths: %v, changedPaths: %v",
				tc.includePaths, tc.excludePaths, tc.changedPaths)
		})
	}
}
