package strings

import (
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
