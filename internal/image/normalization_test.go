package image

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Official images on Docker Hub
		{"index.docker.io/library/busybox", "busybox"},
		{"index.docker.io/busybox", "busybox"},
		{"registry-1.io/library/busybox", "busybox"},
		{"registry-1.io/busybox", "busybox"},
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
		{"invalid url", "invalid url"},
	}

	for _, tc := range tests {
		got := NormalizeURL(tc.input)
		require.Equal(t, tc.expected, got, "input: %s", tc.input)
	}
}
