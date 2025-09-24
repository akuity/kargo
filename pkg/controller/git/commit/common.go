package commit

import (
	"fmt"

	"github.com/akuity/kargo/pkg/pattern"
)

func getPathSelectors(
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

func matchesPathsFilters(
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
