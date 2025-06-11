package image

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeRef(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		// Docker Hub normalization cases
		{"simple repo (nginx)", "nginx", "docker.io/library/nginx:latest", false},
		{"simple repo (Ubuntu, uppercase)", "Ubuntu", "docker.io/library/ubuntu:latest", false},
		{"LiBrArY/NGINX (mixed case library)", "LiBrArY/NGINX", "docker.io/library/nginx:latest", false},
		{"docker.io/library/nginx (explicit)", "docker.io/library/nginx", "docker.io/library/nginx:latest", false},
		{"docker.io/nginx (implicit library)", "docker.io/nginx", "docker.io/library/nginx:latest", false},
		{"nginx with tag", "nginx:1.25", "docker.io/library/nginx:1.25", false},
		{"docker.io/library/nginx with tag", "docker.io/library/nginx:1.25", "docker.io/library/nginx:1.25", false},
		{"docker.io/foo/bar (default tag)", "docker.io/foo/bar", "docker.io/foo/bar:latest", false},
		{"implicit docker.io with namespace", "foo/bar", "docker.io/foo/bar:latest", false},
		{"personal repo (foo/app)", "foo/app", "docker.io/foo/app:latest", false},
		{"personal repo with tag (foo/app:v1.2)", "foo/app:v1.2", "docker.io/foo/app:v1.2", false},
		{"fully qualified docker.io with user", "docker.io/foo/app:v2.0", "docker.io/foo/app:v2.0", false},
		{"explicit docker.io/library/alpine", "docker.io/library/alpine", "docker.io/library/alpine:latest", false},
		{"repo with multiple slashes and tag", "docker.io/org/namespace/app:dev", "docker.io/org/namespace/app:dev", false},
		{"UPPERCASE/INVALID (normalizes to lowercase)", "UPPERCASE/INVALID", "docker.io/uppercase/invalid:latest", false},
		{"single character repo", "a", "docker.io/library/a:latest", false},
		{"repo with dash", "foo-bar", "docker.io/library/foo-bar:latest", false},
		{"repo with underscore", "foo_bar", "docker.io/library/foo_bar:latest", false},
		{"repo with dot", "foo.bar", "docker.io/library/foo.bar:latest", false},
		{"repo with leading/trailing whitespace", "  nginx  ", "docker.io/library/nginx:latest", false},

		// Invalid cases for Docker Hub
		{"invalid characters in name", "invalid!name", "", true},
		{"invalid characters in full ref", "docker.io/Invalid$$Name", "", true},
		{"empty input", "", "", true},
		{"repo with trailing slash", "nginx/", "", true},
		{"repo with trailing colon but no tag", "nginx:", "", true},
		{"too many path components", "foo/bar/baz/qux", "docker.io/foo/bar/baz/qux:latest", false},
		{"malformed digest", "nginx@sha256", "", true},

		// Digest-based reference for Docker Hub
		{
			name:     "digest-based ref (nginx@sha256)",
			input:    "nginx@sha256:123abcdeffedcba3210987654321abcdefabcdefabcdefabcdefabcdefabcdef",
			expected: "docker.io/library/nginx@sha256:123abcdeffedcba3210987654321abcdefabcdefabcdefabcdefabcdefabcdef",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeRef(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, got)
			}
		})
	}
}
