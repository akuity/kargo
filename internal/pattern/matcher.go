package pattern

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

const (
	globPrefix   = "glob:"
	regexPrefix  = "regex:"
	regexpPrefix = "regexp:"
)

// Matcher is an interface that defines a method for matching strings against a
// specific pattern. It can be implemented by different types of patterns, such
// as glob patterns, regular expressions, or base directory patterns.
type Matcher interface {
	// Matches checks if the given string matches the pattern.
	Matches(string) bool
	// String returns the string representation of the pattern.
	String() string
}

// ParseNamePattern parses a pattern string and returns a Matcher.
// It recognizes glob patterns (with "glob:" prefix), regular expressions (with
// "regex:" or "regexp:" prefix), and exact string matches (without any prefix).
func ParseNamePattern(pattern string) (Matcher, error) {
	switch {
	case strings.HasPrefix(pattern, globPrefix):
		return NewGlobPattern(strings.TrimPrefix(pattern, globPrefix))
	case strings.HasPrefix(pattern, regexPrefix):
		return NewRegexpMatcher(strings.TrimPrefix(pattern, regexPrefix))
	case strings.HasPrefix(pattern, regexpPrefix):
		return NewRegexpMatcher(strings.TrimPrefix(pattern, regexpPrefix))
	default:
		return NewExactMatcher(pattern)
	}
}

// ParsePathPattern parses a pattern string and returns a Matcher.
// It recognizes glob patterns (with "glob:" prefix), regular expressions (with
// "regex:" or "regexp:" prefix), and base directory patterns (without any prefix).
// It is important to note that when using this pattern for paths, you should
// use '/' as the path separator to avoid issues with OS-specific path separators.
func ParsePathPattern(pattern string) (Matcher, error) {
	switch {
	case strings.HasPrefix(pattern, globPrefix):
		return NewGlobPattern(strings.TrimPrefix(pattern, globPrefix))
	case strings.HasPrefix(pattern, regexPrefix):
		return NewRegexpMatcher(strings.TrimPrefix(pattern, regexPrefix))
	case strings.HasPrefix(pattern, regexpPrefix):
		return NewRegexpMatcher(strings.TrimPrefix(pattern, regexpPrefix))
	default:
		return NewBaseDirMatcher(pattern)
	}
}

type Matchers []Matcher

// Matches checks if any of the matchers match the given string.
func (m Matchers) Matches(str string) bool {
	for _, matcher := range m {
		if matcher.Matches(str) {
			return true
		}
	}
	return false
}

// String returns a string representation of all matchers.
func (m Matchers) String() string {
	var sb strings.Builder
	for i, matcher := range m {
		if i > 0 {
			_, _ = sb.WriteString(", ")
		}
		_, _ = sb.WriteString(matcher.String())
	}
	return sb.String()
}

// GlobMatcher is a pattern that uses glob syntax to match strings.
// When using this pattern for paths, make sure to use '/' as the path separator
// to avoid issues with OS-specific path separators.
type GlobMatcher struct {
	pattern string
}

// NewGlobPattern creates a new GlobMatcher with the given pattern.
// If the pattern is invalid, it returns an error.
func NewGlobPattern(pattern string) (*GlobMatcher, error) {
	if !doublestar.ValidatePattern(pattern) {
		return nil, doublestar.ErrBadPattern
	}
	return &GlobMatcher{
		pattern: pattern,
	}, nil
}

// Matches checks if the given string matches the glob pattern.
func (p *GlobMatcher) Matches(str string) bool {
	return doublestar.MatchUnvalidated(p.pattern, str)
}

// String returns the string representation of the glob pattern.
func (p *GlobMatcher) String() string {
	return p.pattern
}

// RegexpMatcher is a pattern that uses regular expressions to match strings.
// When using this pattern for paths, make sure to use '/' as the path separator
// to avoid issues with OS-specific path separators.
type RegexpMatcher struct {
	pattern string
	regexp  *regexp.Regexp
}

// NewRegexpMatcher creates a new RegexpMatcher with the given pattern.
// If the pattern is invalid, it returns an error.
func NewRegexpMatcher(pattern string) (*RegexpMatcher, error) {
	r, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &RegexpMatcher{
		pattern: pattern,
		regexp:  r,
	}, nil
}

// Matches checks if the given string matches the regexp pattern.
func (p *RegexpMatcher) Matches(str string) bool {
	return p.regexp.MatchString(str)
}

// String returns the string representation of the regexp pattern.
func (p *RegexpMatcher) String() string {
	return p.pattern
}

// BaseDirMatcher is a pattern that matches any path under a given base directory.
type BaseDirMatcher struct {
	basePath string
}

// NewBaseDirMatcher creates a new BaseDirMatcher with the given base path.
func NewBaseDirMatcher(basePath string) (*BaseDirMatcher, error) {
	return &BaseDirMatcher{
		basePath: basePath,
	}, nil
}

// Matches checks if the given string matches the base directory pattern.
func (p *BaseDirMatcher) Matches(str string) bool {
	relPath, err := filepath.Rel(p.basePath, str)
	if err != nil {
		return false
	}
	return !strings.Contains(relPath, "..")
}

// String returns the string representation of the base directory pattern.
func (p *BaseDirMatcher) String() string {
	return p.basePath
}

// ExactMatcher is a pattern that matches a string exactly.
type ExactMatcher struct {
	pattern string
}

// NewExactMatcher creates a new ExactMatcher with the given pattern.
func NewExactMatcher(pattern string) (*ExactMatcher, error) {
	return &ExactMatcher{
		pattern: pattern,
	}, nil
}

// Matches checks if the given string matches the exact pattern.
func (p *ExactMatcher) Matches(str string) bool {
	return p.pattern == str
}

// String returns the string representation of the exact pattern.
func (p *ExactMatcher) String() string {
	return p.pattern
}
