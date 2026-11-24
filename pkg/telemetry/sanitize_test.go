package telemetry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeURL(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"https://user:token@github.com/org/repo.git":            "https://github.com/org/repo.git",
		"https://s3.amazonaws.com/blob?X-Amz-Signature=abc#x":   "https://s3.amazonaws.com/blob",
		"oci://ghcr.io/org/chart":                               "oci://ghcr.io/org/chart",
		"git@github.com:org/repo.git":                           "github.com:org/repo.git",
		"ghcr.io/org/image":                                     "ghcr.io/org/image",
		"user:pass@registry.example.com/org/image?token=s3cr3t": "registry.example.com/org/image",
		// Malformed userinfo: unescaped '/', '?', '#', bad escapes, bad ports.
		"https://user:pa/ss@github.com/org/repo.git":        "https://github.com/org/repo.git",
		"https://user:p?ss@github.com/org/repo.git":         "https://github.com/org/repo.git",
		"https://user:p#ss@github.com/org/repo.git":         "https://github.com/org/repo.git",
		"https://user:12/ss@github.com/org/repo.git":        "https://github.com/org/repo.git",
		"https://user:1#ss@github.com/org/repo.git":         "https://github.com/org/repo.git",
		"https://user:pa%zz@github.com/org/repo.git":        "https://github.com/org/repo.git",
		"https://user:pass@github.com:badport/org/repo.git": "https://github.com:badport/org/repo.git",
		"user:pa/ss@registry.example.com/org/image":         "registry.example.com/org/image",
		// Malformed userinfo, with an '@' in the path that isn't userinfo.
		"https://user:pa%zz@host/@scope/pkg": "https://host/@scope/pkg",
		// An '@' that isn't userinfo is kept.
		"https://storage.googleapis.com/b/o?X-Goog-Credential=sa@p.iam": "https://storage.googleapis.com/b/o",
		"https://host/@scope/pkg":                   "https://host/@scope/pkg",
		"https://host:8080/@scope/pkg":              "https://host:8080/@scope/pkg",
		"https://localhost:5000/v2/repo@sha256:abc": "https://localhost:5000/v2/repo@sha256:abc",
		"ghcr.io/org/image@sha256:abc":              "ghcr.io/org/image@sha256:abc",
		// A Secret's name in a Kubernetes API path.
		"https://kube/api/v1/namespaces/p/secrets/creds?timeout=1s": "https://kube/api/v1/namespaces/p/secrets/{redacted}",
		"/api/v1/namespaces/p/secrets/repo-creds/status":            "/api/v1/namespaces/p/secrets/{redacted}/status",
		"https://kube/api/v1/namespaces/p/secrets?limit=500":        "https://kube/api/v1/namespaces/p/secrets",
		// Ambiguous with a password like "12/x": dropped up to the '@'.
		"https://host:5000/x@y/pkg": "https://y/pkg",
		"":                          "",
	} {
		require.Equal(t, want, SanitizeURL(in), in)
	}
}

func TestRedactURLs(t *testing.T) {
	t.Parallel()
	require.Equal(
		t,
		`error cloning "https://github.com/org/repo.git": auth failed; see https://example.com/doc`,
		RedactURLs(
			`error cloning "https://x-access-token:ghp_abc@github.com/org/repo.git": `+
				`auth failed; see https://example.com/doc?sig=1`,
		),
	)
	require.Equal(t, "no urls here", RedactURLs("no urls here"))
	// A backtick ends a URL, as quotes do.
	require.Equal(
		t,
		"Get `https://host/x`: EOF",
		RedactURLs("Get `https://user:tok@host/x`: EOF"),
	)
	// A password with an unescaped space.
	require.Equal(
		t,
		"error cloning https://github.com/org/repo: auth failed",
		RedactURLs("error cloning https://user:p ss@github.com/org/repo: auth failed"),
	)
	// A URL within another.
	require.Equal(
		t,
		"repos https://a/x,https://b/y failed",
		RedactURLs("repos https://a/x,https://user:tok@b/y failed"),
	)
	require.Equal(
		t,
		"Get http://proxy:3128/https://host/x: EOF",
		RedactURLs("Get http://proxy:3128/https://user:tok@host/x: EOF"),
	)
	require.Equal(
		t,
		`Get "https://kube/api/v1/namespaces/p/secrets/{redacted}": EOF`,
		RedactURLs(`Get "https://kube/api/v1/namespaces/p/secrets/repo-creds?timeout=10s": EOF`),
	)
	// Ambiguous with such a password: redacted up to the '@'.
	require.Equal(t, "dial https://b", RedactURLs("dial https://host:80 for a@b"))
}
