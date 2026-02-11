package urls

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeChart(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single word",
			input:    "repo",
			expected: "repo",
		},
		{
			name:     "leading and trailing whitespace",
			input:    "  repo  ",
			expected: "repo",
		},
		{
			name:     "mixed case",
			input:    "REpo",
			expected: "repo",
		},
		{
			name:     "oci prefix",
			input:    "oci://repo",
			expected: "repo",
		},
		{
			name:     "oci prefix with whitespace",
			input:    "  oci://repo  ",
			expected: "repo",
		},
		{
			name:     "oci prefix with mixed case",
			input:    "OCI://Repo",
			expected: "repo",
		},
		// Check correct normalization of Docker Hub URLs
		{
			name:     "docker.io URL with oci prefix",
			input:    "oci://docker.io/example/repo",
			expected: "example/repo",
		},
		{
			name:     "docker.io URL without oci prefix",
			input:    "docker.io/example/repo",
			expected: "example/repo",
		},
		{
			name:     "index.docker.io URL with oci prefix",
			input:    "oci://index.docker.io/example/repo",
			expected: "example/repo",
		},
		{
			name:     "index.docker.io URL without oci prefix",
			input:    "index.docker.io/example/repo",
			expected: "example/repo",
		},
		{
			name:     "registry-1.docker.io URL with oci prefix",
			input:    "oci://registry-1.docker.io/example/repo",
			expected: "example/repo",
		},
		{
			name:     "registry-1.docker.io URL without oci prefix",
			input:    "registry-1.docker.io/example/repo",
			expected: "example/repo",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := NormalizeChart(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
