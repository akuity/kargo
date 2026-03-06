package urls

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeURL(t *testing.T) {
	testCases := map[string]string{
		// Leading and trailing whitespace
		"  https://example.com/repo  ":     "https://example.com/repo",
		"\n\thttps://example.com/repo\t\n": "https://example.com/repo",
		// Non-printable runes (e.g., BOMs)
		"\uFEFFhttps://example.com/repo":               "https://example.com/repo",
		"https://example.com/\u200Brepo":               "https://example.com/repo",
		"https://example.com/repo\uFEFF":               "https://example.com/repo",
		"\uFEFF https://example.com/\u200Brepo \uFEFF": "https://example.com/repo",
		// Combination of both
		"\uFEFF \n https://example.com/\trepo \u200B \n ": "https://example.com/repo",
		// No changes needed
		"https://example.com/repo":   "https://example.com/repo",
		"ftp://example.com/resource": "ftp://example.com/resource",
		"   ":                        "",
		"":                           "",
		// Internal whitespace should not be removed
		"https://example.com/ myrepo": "https://example.com/ myrepo",
		// Internal non-printable runes should be removed
		"https://example.com/\u200Bmyrepo": "https://example.com/myrepo",
	}
	for in, out := range testCases {
		t.Run(in, func(t *testing.T) {
			require.Equal(t, out,
				SanitizeURL(in),
			)
		})
	}
}
