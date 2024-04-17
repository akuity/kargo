package git

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeGitURL(t *testing.T) {
	// nolint: lll
	testCases := map[string]string{
		"git@github.com": "git@github.com", // No path
		// Variable features of each URL:
		//   1000. ssh:// prefix or not
		//   0100. Port number or not
		//   0010. .git suffix or not
		//   0001. Trailing slash or not
		"git@github.com:example/repo":                "git@github.com:example/repo",     // 0000
		"git@github.com:example/repo/":               "git@github.com:example/repo",     // 0001
		"git@github.com:example/repo.git":            "git@github.com:example/repo",     // 0010
		"git@github.com:example/repo.git/":           "git@github.com:example/repo",     // 0011
		"git@localhost:8443:example/repo":            "git@localhost:8443:example/repo", // 0100
		"git@localhost:8443:example/repo/":           "git@localhost:8443:example/repo", // 0101
		"git@localhost:8443:example/repo.git":        "git@localhost:8443:example/repo", // 0110
		"git@localhost:8443:example/repo.git/":       "git@localhost:8443:example/repo", // 0111
		"ssh://git@github.com:example/repo":          "git@github.com:example/repo",     // 1000
		"ssh://git@github.com:example/repo/":         "git@github.com:example/repo",     // 1001
		"ssh://git@github.com:example/repo.git":      "git@github.com:example/repo",     // 1010
		"ssh://git@github.com:example/repo.git/":     "git@github.com:example/repo",     // 1011
		"ssh://git@localhost:8443:example/repo":      "git@localhost:8443:example/repo", // 1100
		"ssh://git@localhost:8443:example/repo/":     "git@localhost:8443:example/repo", // 1101
		"ssh://git@localhost:8443:example/repo.git":  "git@localhost:8443:example/repo", // 1110
		"ssh://git@localhost:8443:example/repo.git/": "git@localhost:8443:example/repo", // 1111
		"https://github.com":                         "https://github.com",              // No path
		// Variable features of each URL:
		//   10000. Outbound proxy or not
		//   01000. Port number or not
		//   00100. .git suffix or not
		//   00010. Trailing slash or not
		//   00001. Query parameters or not
		"https://github.com/example/repo":                          "https://github.com/example/repo",                     // 00000
		"https://github.com/example/repo?foo=bar":                  "https://github.com/example/repo?foo=bar",             // 00001
		"https://github.com/example/repo/":                         "https://github.com/example/repo",                     // 00010
		"https://github.com/example/repo/?foo=bar":                 "https://github.com/example/repo?foo=bar",             // 00011
		"https://github.com/example/repo.git":                      "https://github.com/example/repo",                     // 00100
		"https://github.com/example/repo.git?foo=bar":              "https://github.com/example/repo?foo=bar",             // 00101
		"https://github.com/example/repo.git/":                     "https://github.com/example/repo",                     // 00110
		"https://github.com/example/repo.git/?foo=bar":             "https://github.com/example/repo?foo=bar",             // 00111
		"https://localhost:8443/example/repo":                      "https://localhost:8443/example/repo",                 // 01000
		"https://localhost:8443/example/repo?foo=bar":              "https://localhost:8443/example/repo?foo=bar",         // 01001
		"https://localhost:8443/example/repo/":                     "https://localhost:8443/example/repo",                 // 01010
		"https://localhost:8443/example/repo/?foo=bar":             "https://localhost:8443/example/repo?foo=bar",         // 01011
		"https://localhost:8443/example/repo.git":                  "https://localhost:8443/example/repo",                 // 01100
		"https://localhost:8443/example/repo.git?foo=bar":          "https://localhost:8443/example/repo?foo=bar",         // 01101
		"https://localhost:8443/example/repo.git/":                 "https://localhost:8443/example/repo",                 // 01110
		"https://localhost:8443/example/repo.git/?foo=bar":         "https://localhost:8443/example/repo?foo=bar",         // 01111
		"https://foo:bar@github.com/example/repo":                  "https://foo:bar@github.com/example/repo",             // 10000
		"https://foo:bar@github.com/example/repo?foo=bar":          "https://foo:bar@github.com/example/repo?foo=bar",     // 10001
		"https://foo:bar@github.com/example/repo/":                 "https://foo:bar@github.com/example/repo",             // 10010
		"https://foo:bar@github.com/example/repo/?foo=bar":         "https://foo:bar@github.com/example/repo?foo=bar",     // 10011
		"https://foo:bar@github.com/example/repo.git":              "https://foo:bar@github.com/example/repo",             // 10100
		"https://foo:bar@github.com/example/repo.git?foo=bar":      "https://foo:bar@github.com/example/repo?foo=bar",     // 10101
		"https://foo:bar@github.com/example/repo.git/":             "https://foo:bar@github.com/example/repo",             // 10110
		"https://foo:bar@github.com/example/repo.git/?foo=bar":     "https://foo:bar@github.com/example/repo?foo=bar",     // 10111
		"https://foo:bar@localhost:8443/example/repo":              "https://foo:bar@localhost:8443/example/repo",         // 11000
		"https://foo:bar@localhost:8443/example/repo?foo=bar":      "https://foo:bar@localhost:8443/example/repo?foo=bar", // 11001
		"https://foo:bar@localhost:8443/example/repo/":             "https://foo:bar@localhost:8443/example/repo",         // 11010
		"https://foo:bar@localhost:8443/example/repo/?foo=bar":     "https://foo:bar@localhost:8443/example/repo?foo=bar", // 11011
		"https://foo:bar@localhost:8443/example/repo.git":          "https://foo:bar@localhost:8443/example/repo",         // 11100
		"https://foo:bar@localhost:8443/example/repo.git?foo=bar":  "https://foo:bar@localhost:8443/example/repo?foo=bar", // 11101
		"https://foo:bar@localhost:8443/example/repo.git/":         "https://foo:bar@localhost:8443/example/repo",         // 11110
		"https://foo:bar@localhost:8443/example/repo.git/?foo=bar": "https://foo:bar@localhost:8443/example/repo?foo=bar", // 11111
	}
	for in, out := range testCases {
		t.Run(in, func(t *testing.T) {
			require.Equal(t, out, NormalizeGitURL(in))
		})
	}
}
