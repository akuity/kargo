package sjson

import (
	"fmt"
	"strings"
)

// SplitKey splits a key string into parts separated by dots. It observes the
// same basic syntax rules as tidwall/sjson. Dots are separators unless escaped.
// Colons, unless escaped, are hints that a numeric-looking key part should be
// treated as a key in an object, rather than an index in a sequence.
func SplitKey(key string) ([]string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("empty key")
	}
	parts := make([]string, 0, strings.Count(key, ".")+1)
	currentPart := strings.Builder{}
	escaped := false
	for i := 0; i < len(key); i++ {
		char := key[i]
		if !escaped {
			switch char {
			case '\\':
				escaped = true // Enter escape mode.
			case '.':
				// We've reached the end of the current part.
				if currentPart.Len() == 0 {
					return nil, fmt.Errorf("empty key part in key %q", key)
				}
				parts = append(parts, currentPart.String())
				currentPart.Reset()
			case ':':
				if currentPart.Len() > 0 {
					// An unescaped colon is only valid as the first character of a key
					// part.
					return nil, fmt.Errorf("unexpected colon in key %q", key)
				}
				// We don't actually need to KEEP the colon, since the code that uses
				// the key parts returned from this function requires no hint that a
				// numeric-looking key part should be treated as a key in an object,
				// rather than an index in a sequence.
			default:
				// Any other character is added to the current part as is.
				if err := currentPart.WriteByte(char); err != nil {
					return nil, err
				}
			}
			continue
		}
		// If we get to here, we're currently in escape mode.
		switch char {
		case '.', ':':
			if err := currentPart.WriteByte(char); err != nil {
				return nil, err
			}
			escaped = false // Exit escape mode.
		default:
			return nil, fmt.Errorf("invalid escape sequence in key %q", key)
		}
	}
	// Don't forget about whatever is left over in currentPart.
	if currentPart.Len() == 0 {
		return nil, fmt.Errorf("empty key part in key %q", key)
	}
	return append(parts, currentPart.String()), nil
}
