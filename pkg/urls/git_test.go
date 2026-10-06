package urls

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeGit(t *testing.T) {
	testCases := map[string]string{
		// Anything we can't normalize should be returned as-is
		"https://not a url":                      "https://not a url",
		"http://github.com/example/repo?foo=bar": "http://github.com/example/repo?foo=bar",
		"ssh://not a url":                        "ssh://not a url",
		"ssh://github.com/example/repo?foo=bar":  "ssh://github.com/example/repo?foo=bar",
		"not even remotely a url":                "not even remotely a url",
		// URLs of the form http[s]://[proxy-user:proxy-pass@]host.xz[:port][/path/to/repo[.git][/]]
		"https://github.com":          "https://github.com",
		"https://github.com/":         "https://github.com",
		"https://foo:bar@github.com":  "https://github.com",
		"https://foo:bar@github.com/": "https://github.com",
		// Variable features of the following URLs:
		//   1000. Outbound proxy or not
		//   0100. Port number or not
		//   0010. .git suffix or not
		//   0001. Trailing slash or not
		"https://github.com/example/repo":                  "https://github.com/example/repo",     // 0000
		"https://github.com/example/repo/":                 "https://github.com/example/repo",     // 0001
		"https://github.com/example/repo.git":              "https://github.com/example/repo",     // 0010
		"https://github.com/example/repo.git/":             "https://github.com/example/repo",     // 0011
		"https://localhost:8443/example/repo":              "https://localhost:8443/example/repo", // 0100
		"https://localhost:8443/example/repo/":             "https://localhost:8443/example/repo", // 0101
		"https://localhost:8443/example/repo.git":          "https://localhost:8443/example/repo", // 0110
		"https://localhost:8443/example/repo.git/":         "https://localhost:8443/example/repo", // 0111
		"https://foo:bar@github.com/example/repo":          "https://github.com/example/repo",     // 1000
		"https://foo:bar@github.com/example/repo/":         "https://github.com/example/repo",     // 1001
		"https://foo:bar@github.com/example/repo.git":      "https://github.com/example/repo",     // 1010
		"https://foo:bar@github.com/example/repo.git/":     "https://github.com/example/repo",     // 1011
		"https://foo:bar@localhost:8443/example/repo":      "https://localhost:8443/example/repo", // 1100
		"https://foo:bar@localhost:8443/example/repo/":     "https://localhost:8443/example/repo", // 1101
		"https://foo:bar@localhost:8443/example/repo.git":  "https://localhost:8443/example/repo", // 1110
		"https://foo:bar@localhost:8443/example/repo.git/": "https://localhost:8443/example/repo", // 1111
		// SSH and SCP-style URLs are not supported and are returned as-is
		"ssh://git@github.com/example/repo.git": "ssh://git@github.com/example/repo.git",
		"git@github.com:example/repo.git":       "git@github.com:example/repo.git",
	}
	for in, out := range testCases {
		t.Run(in, func(t *testing.T) {
			require.Equal(t, out, NormalizeGit(in))
		})
	}
}
