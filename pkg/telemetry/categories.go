package telemetry

import (
	"fmt"
	"slices"
	"strings"

	"github.com/akuity/kargo/pkg/telemetry/internal/state"
)

// Category names a group of spans TRACING_CATEGORIES selects.
type Category string

const (
	// CategoryDatabase covers database query and batch spans.
	CategoryDatabase Category = "database"
	// CategoryK8s covers Kubernetes API request spans.
	CategoryK8s Category = "k8s"
	// CategoryPatch covers status patch spans.
	CategoryPatch Category = "patch"
)

// categories maps every Category to whether it is on by default.
var categories = map[Category]bool{
	CategoryDatabase: true,
	CategoryK8s:      false,
	CategoryPatch:    false,
}

// CategoryEnabled returns whether tracing is on and c is selected by
// TRACING_CATEGORIES.
func CategoryEnabled(c Category) bool {
	return Enabled() && state.Selected.Load().Has(string(c), categories[c])
}

// parseCategories parses TRACING_CATEGORIES entries. An unknown name is an
// error.
func parseCategories(entries []string) (*state.Categories, error) {
	s, err := state.ParseCategories(entries)
	if err != nil {
		return nil, fmt.Errorf("invalid TRACING_CATEGORIES: %w", err)
	}
	for _, name := range s.Names() {
		if _, ok := categories[Category(name)]; !ok {
			known := make([]string, 0, len(categories))
			for c := range categories {
				known = append(known, string(c))
			}
			slices.Sort(known)
			return nil, fmt.Errorf(
				"unknown tracing category %q in TRACING_CATEGORIES; "+
					"known categories: %s; keywords: %s, %s, %s",
				name,
				strings.Join(known, ", "),
				state.DefaultCategories,
				state.AllCategories,
				state.NoCategories,
			)
		}
	}
	return s, nil
}
