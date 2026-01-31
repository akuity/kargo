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
		// Edge cases for unusual whitespace characters
		{
			name:     "leading BOM",
			input:    "\ufeffoci://example/repo",
			expected: "example/repo",
		},
		{
			name:     "trailing BOM",
			input:    "oci://example/repo\ufeff",
			expected: "example/repo",
		},
		{
			name:     "leading and trailing spaces",
			input:    "  oci://example/repo  ",
			expected: "example/repo",
		},
		{
			name:     "leading and trailing tabs",
			input:    "\toci://example/repo\t",
			expected: "example/repo",
		},
		{
			name:     "leading and trailing newlines",
			input:    "\noci://example/repo\n",
			expected: "example/repo",
		},
		{
			name:     "leading and trailing carriage returns",
			input:    "\roci://example/repo\r",
			expected: "example/repo",
		},
		{
			name:     "mixed whitespace",
			input:    " \t\noci://example/repo\t\n ",
			expected: "example/repo",
		},
		{
			name:     "non-breaking spaces",
			input:    "\u00A0oci://example/repo\u00A0",
			expected: "example/repo",
		},
		{
			name:     "zero-width spaces",
			input:    "\u200Boci://example/repo\u200B",
			expected: "example/repo",
		},
		{
			name:     "multiple BOMs",
			input:    "\ufeff\ufeffoci://example/repo",
			expected: "example/repo",
		},
		{
			name:     "internal spaces",
			input:    "oci://example /repo",
			expected: "example/repo",
		},
		{
			name:     "encoded spaces",
			input:    "oci://example%20repo",
			expected: "examplerepo",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := NormalizeChart(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
