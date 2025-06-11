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
		// Official images, docker.io registry, with tag/digest
		{"nginx:latest", "nginx"},
		{"docker.io/library/nginx:1.21", "nginx"},
		{"docker.io/library/nginx@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "nginx"},
		// Official images, index.docker.io registry
		{"index.docker.io/library/redis:7", "redis"},
		// Custom repo in docker.io
		{"docker.io/myuser/myapp:dev", "myuser/myapp"},
		// Other registries
		{"gcr.io/myproj/app:tag", "gcr.io/myproj/app"},
		{"quay.io/org/repo:tag", "quay.io/org/repo"},
		// No tag/digest
		{"docker.io/library/busybox", "busybox"},
		{"gcr.io/myproj/app", "gcr.io/myproj/app"},
	}

	for _, tc := range tests {
		got, err := NormalizeURL(tc.input)
		require.NoError(t, err, "input: %s", tc.input)
		require.Equal(t, tc.expected, got, "input: %s", tc.input)
	}

	// Invalid input
	_, err := NormalizeURL("not a valid ref")
	require.Error(t, err)
}
