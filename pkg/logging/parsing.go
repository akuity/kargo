package logging

import (
	"fmt"
	"strings"
)

// ParseLevel parses a string representation of a log level and returns the
// corresponding Level value.
func ParseLevel(levelStr string) (Level, error) {
	switch strings.ToLower(levelStr) {
	case "discard":
		return DiscardLevel, nil
	case "error":
		return ErrorLevel, nil
	case "info":
		return InfoLevel, nil
	case "debug":
		return DebugLevel, nil
	case "trace":
		return TraceLevel, nil
	default:
		return InfoLevel, fmt.Errorf("invalid log level %q", levelStr)
	}
}

// ParseFormat parses a string representation of a log format and returns the
// corresponding Format value or an error if it isn't recognized
func ParseFormat(f string) (Format, error) {
	format := Format(strings.TrimSpace(strings.ToLower(f)))
	switch format {
	case JSONFormat, ConsoleFormat:
		return format, nil
	default:
		return "", fmt.Errorf("invalid log format %q", f)
	}
}
