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
		// Canonical docker hub images
		{"nginx short", "nginx", "nginx", false},
		{"nginx latest explicit", "nginx:latest", "nginx", false},
		{"nginx with tag", "nginx:1.21", "nginx:1.21", false},
		{"docker.io/library/nginx", "docker.io/library/nginx", "nginx", false},
		{"docker.io/library/nginx:latest", "docker.io/library/nginx:latest", "nginx", false},
		{"docker.io/library/nginx:1.21", "docker.io/library/nginx:1.21", "nginx:1.21", false},
		{"docker.io/nginx", "docker.io/nginx", "nginx", false},
		{"docker.io/nginx:1.21", "docker.io/nginx:1.21", "nginx:1.21", false},
		{"index.docker.io/library/nginx", "index.docker.io/library/nginx", "nginx", false},
		{"index.docker.io/library/nginx:1.21", "index.docker.io/library/nginx:1.21", "nginx:1.21", false},
		{"library/nginx", "library/nginx", "nginx", false},

		// Namespaced docker hub images
		{"docker.io/foo/bar", "docker.io/foo/bar", "foo/bar", false},
		{"docker.io/foo/bar:dev", "docker.io/foo/bar:dev", "foo/bar:dev", false},
		{"foo/bar", "foo/bar", "foo/bar", false},
		{"foo/bar:dev", "foo/bar:dev", "foo/bar:dev", false},

		// Other registries
		{"gcr.io/myproj/app", "gcr.io/myproj/app", "gcr.io/myproj/app", false},
		{"gcr.io/myproj/app:tag", "gcr.io/myproj/app:tag", "gcr.io/myproj/app:tag", false},
		{"quay.io/org/repo", "quay.io/org/repo", "quay.io/org/repo", false},
		{"quay.io/org/repo:tag", "quay.io/org/repo:tag", "quay.io/org/repo:tag", false},

		// Digest references
		{
			"nginx with digest",
			"nginx@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			"nginx@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			false,
		},
		{
			"docker.io/library/nginx with digest",
			"docker.io/library/nginx@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			"nginx@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			false,
		},
		{
			"gcr.io/myproj/app with digest",
			"gcr.io/myproj/app@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			"gcr.io/myproj/app@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			false,
		},

		// Edge cases
		{"trailing colon", "nginx:", "nginx", false},
		{"repo with dash", "foo-bar", "foo-bar", false},
		{"repo with underscore", "foo_bar", "foo_bar", false},
		{"repo with dot", "foo.bar", "foo.bar", false},

		// Invalid cases
		{"invalid chars", "invalid!name", "", true},
		{"invalid full ref", "docker.io/Invalid$$Name", "", true},
		{"empty", "", "", true},
		{"whitespace", "  nginx  ", "", true},
		{"uppercase", "UPPERCASE/INVALID", "", true},
		{"single char", "a", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeURL(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, got)
			}
		})
	}
}
