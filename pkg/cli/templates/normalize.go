package templates

import "strings"

const indentation = "  "

// Example returns a string without leading or trailing whitespace, and each
// line indented by two spaces.
func Example(s string) string {
	s = strings.TrimSpace(s)

	if s == "" {
		return ""
	}

	var indentedLines []string
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			trimmed = indentation + trimmed
		}
		indentedLines = append(indentedLines, trimmed)
	}

	return strings.Join(indentedLines, "\n")
}
