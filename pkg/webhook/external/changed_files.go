package external

// commitDiff represents the files changed in a single commit. This is a
// provider-agnostic representation used to deduplicate changed file collection
// across GitHub, GitLab, and Gitea webhook handlers.
type commitDiff struct {
	Added    []string
	Modified []string
	Removed  []string
}

// collectChangedFiles deduplicates and returns all file paths from the given
// commit diffs. The returned slice preserves insertion order. A nil input
// returns nil, which signals to downstream consumers that no file-level data
// is available (graceful degradation for path filtering).
func collectChangedFiles(diffs []commitDiff) []string {
	if len(diffs) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	var files []string
	addFile := func(f string) {
		if _, ok := seen[f]; !ok {
			seen[f] = struct{}{}
			files = append(files, f)
		}
	}
	for _, d := range diffs {
		for _, f := range d.Added {
			addFile(f)
		}
		for _, f := range d.Modified {
			addFile(f)
		}
		for _, f := range d.Removed {
			addFile(f)
		}
	}
	return files
}
