package strings

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
)

// SplitLast splits a specified string on the last occurrence of the specified
// separator and returns two strings -- the first containing everything that
// preceded the separator and the second containing everything that followed the
// separator. If the specified string contains no occurrences of the specified
// separator, an error is returned.
func SplitLast(s, sep string) (string, string, error) {
	if sep == "" {
		return "", "", errors.New("no separator was specified")
	}
	i := strings.LastIndex(s, sep)
	if i < 0 {
		return "", "", fmt.Errorf(
			"string %q contains no occurrences of separator %q",
			s,
			sep,
		)
	}
	return s[:i], s[i+1:], nil
}

// WARNING: Do not change this. See comment on strings.HashShorten().
const defaultShortHashLen = 8

// HashShorten deterministically shortens the provided string to the specified
// length by retaining as many of the leading characters as possible and
// replacing as many trailing characters as necessary with a short hash of the
// entire input. The preserved characters of the input string and the short hash
// will be separated by the specified separator. If no separator is specified,
// the separator will default to a single dash. If the length of the input
// string is already less than or equal to the specified maximum, or if the
// string cannot possibly be shortened to the specified maximum (for instance,
// by specifying a maximum less than the length of a short hash), then the
// original string is returned as is. A second return value, a boolean, will be
// false only when the provided string could not be shortened, thereby enabling
// callers to choose how to deal with such a scenario.
//
// WARNING: Altering this algorithm could have profound consequences. This
// function is often used to shorten long identifiers while maintaining
// requisite uniqueness. If this algorithm is altered, the ability to shorten
// identifiers and match the result to an identifier previously shortened with
// the older algorithm will be compromised.
func HashShorten(
	s string,
	maxLen int,
	sep string,
	shortHashLen int,
) (string, bool) {
	if len(s) <= maxLen {
		return s, true // Nothing to do
	}
	if shortHashLen <= 0 {
		shortHashLen = defaultShortHashLen
	}
	if maxLen < shortHashLen {
		// We cannot possibly shorten to the specified length
		return s, false
	}
	if sep == "" {
		sep = "-" // Default separator
	}
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
	shortHash := sum[:shortHashLen]
	if maxLen <= shortHashLen+len(sep) {
		// No room for at least one original character + separator + short hash...
		return shortHash, true // Return just the hash
	}
	prefix := s[:maxLen-(shortHashLen+len(sep))]
	// Remove any trailing characters exactly matching the separator to avoid
	// a double separator
	prefix = strings.TrimRight(prefix, sep)
	return prefix + sep + shortHash, true
}
