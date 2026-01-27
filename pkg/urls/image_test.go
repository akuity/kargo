package urls

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeImage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Official images on Docker Hub
		{"index.docker.io/library/busybox", "busybox"},
		{"index.docker.io/busybox", "busybox"},
		{"registry-1.docker.io/library/busybox", "busybox"},
		{"registry-1.docker.io/busybox", "busybox"},
		{"docker.io/library/busybox", "busybox"},
		{"docker.io/busybox", "busybox"},
		{"library/busybox", "busybox"},
		{"busybox", "busybox"},

		// Other images on Docker Hub
		{"index.docker.io/example/repo", "example/repo"},
		{"docker.io/example/repo", "example/repo"},
		{"example/repo", "example/repo"},

		// Images from other registries
		{"ghcr.io/example/repo", "ghcr.io/example/repo"},
		{"quay.io/example/repo", "quay.io/example/repo"},

		// Input that cannot be normalized (invalid URL)
		{"invalid url", "invalidurl"},

		// Edge cases for unusual whitespace characters
		{" \ufeffdocker.io/example/repo", "example/repo"},      // Leading BOM
		{"docker.io/example/repo\ufeff", "example/repo"},       // Trailing BOM
		{"  docker.io/example/repo  ", "example/repo"},         // Leading and trailing spaces
		{"\tdocker.io/example/repo\t", "example/repo"},         // Leading and trailing tabs
		{"\ndocker.io/example/repo\n", "example/repo"},         // Leading and trailing newlines
		{"\rdocker.io/example/repo\r", "example/repo"},         // Leading and trailing carriage returns
		{" \t\ndocker.io/example/repo\t\n ", "example/repo"},   // Mixed whitespace
		{"\u00A0docker.io/example/repo\u00A0", "example/repo"}, // Non-breaking spaces
		{"\u200Bdocker.io/example/repo\u200B", "example/repo"}, // Zero-width spaces
		{"\ufeff\ufeffdocker.io/example/repo", "example/repo"}, // Multiple BOMs
		{"docker.io/example /repo", "example/repo"},            // Internal spaces
		{"docker.io/example%20repo", "examplerepo"},            // Encoded spaces
		{"", ""},       // Empty string
		{"   ", ""},    // Whitespace-only string
		{"\t\n\r", ""}, // Whitespace-only string with tabs/newlines
	}

	for _, tc := range tests {
		got := NormalizeImage(tc.input)
		require.Equal(t, tc.expected, got, "input: %s", tc.input)
	}
}
