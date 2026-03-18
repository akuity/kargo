package commit

import (
	"fmt"

	"github.com/akuity/kargo/pkg/pattern"
)

func getPathSelectors(selectors []string) (pattern.Matcher, error) {
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
