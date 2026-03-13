package commit

import (
	"fmt"

	"github.com/akuity/kargo/pkg/pattern"
)

// GetPathSelectors parses a slice of path selector strings into a
// pattern.Matcher that can match file paths against include/exclude rules.
// Returns nil if selectors is empty. Each selector can use glob:, regex:, or
// regexp: prefixes, or defaults to base directory matching.
func GetPathSelectors(
	selectors []string,
) (pattern.Matcher, error) {
	if len(selectors) == 0 {
		return nil, nil
	}
	matchers := make(pattern.Matchers, len(selectors))
	for i := range selectors {
		matcher, err := pattern.ParsePathPattern(selectors[i])
		if err != nil {
			return nil, fmt.Errorf("parse error path selector %q: %w", selectors[i], err)
		}
		matchers[i] = matcher
	}
	return matchers, nil
}

// MatchesPathsFilters returns true if any path in diffs passes both the
// include and exclude filters. If include is nil, all paths are implicitly
// included. If exclude is nil, no paths are excluded.
func MatchesPathsFilters(
	include pattern.Matcher,
	exclude pattern.Matcher,
	diffs []string,
) bool {
	for _, path := range diffs {
		// If include is nil, all paths are implicitly included
		// Otherwise, check if the path matches the include pattern
		if include != nil && !include.Matches(path) {
			// Path not included, skip to next path
			continue
		}
		// Path is included (either implicitly or explicitly)
		// Now check if it should be excluded
		if exclude != nil && exclude.Matches(path) {
			// Path is explicitly excluded, skip to next path
			continue
		}
		// If we reach here, the path is included and not excluded
		return true
	}
	// None of the paths match our criteria
	return false
}

func shortenString(str string, length int) string {
	if length >= 3 && len(str) > length {
		return str[:length-3] + "..."
	}
	return str
}

func trimSlice[T any](slice []T, limit int) []T {
	if limit <= 0 || len(slice) <= limit {
		return slice
	}
	return slice[:limit]
}
