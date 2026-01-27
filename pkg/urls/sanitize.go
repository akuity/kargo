package urls

import (
	"strings"
	"unicode"
)

const (
	zeroWidthNoBreakSpace = '\uFEFF' // BOM
	zeroWidthSpace        = '\u200B'
	noBreakSpace          = '\u00A0'
)

// sanitize performs basic sanitization of a repository URL by trimming
// leading and trailing whitespace, converting to lowercase, and removing
// unusual whitespace characters.
func sanitize(repo string) string {
	repo = strings.TrimSpace(strings.ToLower(repo))
	repo = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == zeroWidthNoBreakSpace || r == zeroWidthSpace || r == noBreakSpace {
			return -1 // Remove the character
		}
		return r
	}, repo)
	return repo
}
