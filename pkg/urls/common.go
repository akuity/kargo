package urls

import (
	"strings"
	"unicode"
)

// SanitizeURL removes leading and trailing whitespace only from a string
// presumed to represent a URL. It additionally removes non-printable runes such
// as byte order marks (BOMs) from anywhere in a string. Leading whitespace and
// non-printable runes can easily be copied and pasted without a user realizing
// and are known to interfere with URL parsing.
func SanitizeURL(url string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, url))
}
