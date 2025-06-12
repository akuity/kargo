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
		// Official images, docker.io registry
		{"docker.io/library/busybox", "busybox"},
		// Official images, index.docker.io registry
		{"index.docker.io/library/redis", "redis"},
		// Custom repo in docker.io
		{"docker.io/myuser/myapp", "myuser/myapp"},
		// Other registries
		{"gcr.io/myproj/app", "gcr.io/myproj/app"},
		{"quay.io/org/repo", "quay.io/org/repo"},
	}

	for _, tc := range tests {
		got := NormalizeURL(tc.input)
		require.Equal(t, tc.expected, got, "input: %s", tc.input)
	}

	// Invalid input: should return input as-is
	input := "not a valid ref"
	got := NormalizeURL(input)
	require.Equal(t, input, got)
}
