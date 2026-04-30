package external

import "slices"

// collectPaths is a helper function that collects and deduplicates file paths
// from a slice of commits.
func collectPaths[T any](
	commits []T,
	getPaths func(commit T) []string,
) []string {
	var paths []string
	for _, commit := range commits {
		paths = append(paths, getPaths(commit)...)
	}
	slices.Sort(paths)
	return slices.Compact(paths)
}
