package urls

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// These tests cover the Normalize function which normalizes URLs based on their type.
// the underlying normalization functions are tested exhaustively in their respective test files.
func TestNormalize(t *testing.T) {
	tests := []struct {
		name     string
		urlType  UrlType
		url      string
		expected string
	}{
		{
			name:     "git url",
			urlType:  UrlTypeGit,
			url:      "https://github.com/user/repo.git",
			expected: "https://github.com/user/repo",
		},
		{
			name:     "image url",
			urlType:  UrlTypeImage,
			url:      "index.docker.io/library/busybox",
			expected: "busybox",
		},
		{
			name:     "chart url",
			urlType:  UrlTypeChart,
			url:      "oci://registry-1.docker.io/example/repo",
			expected: "example/repo",
		},
		{
			name:     "unknown url type",
			urlType:  "unknown",
			url:      " some-random-url ",
			expected: " some-random-url ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, Normalize(tt.urlType, tt.url))
		})
	}
}
