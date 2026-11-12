// Package state holds process-wide telemetry state shared by package
// telemetry and its test helpers in package telemetrytest.
package state

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"sync/atomic"
)

// Keywords TRACING_CATEGORIES accepts besides category names.
const (
	// AllCategories selects every category.
	AllCategories = "all"
	// DefaultCategories selects the categories on by default.
	DefaultCategories = "default"
	// NoCategories selects no category.
	NoCategories = "none"
)

// Enabled records whether an exporting tracer provider is installed.
var Enabled atomic.Bool

// Selected holds the tracing categories to record. nil selects none.
var Selected atomic.Pointer[Categories]

// Categories is a selection of tracing categories.
type Categories struct {
	// all selects every category, including any not known when it was set.
	all      bool
	defaults bool
	names    map[string]struct{}
}

// ParseCategories parses TRACING_CATEGORIES entries: case-insensitive, spaces
// trimmed, empty entries dropped. No entries select the default categories.
// Names are not checked.
func ParseCategories(entries []string) (*Categories, error) {
	c := &Categories{names: map[string]struct{}{}}
	var none, some bool
	for _, entry := range entries {
		name := strings.ToLower(strings.TrimSpace(entry))
		switch name {
		case "":
			continue
		case AllCategories:
			c.all = true
		case DefaultCategories:
			c.defaults = true
		case NoCategories:
			none = true
		default:
			c.names[name] = struct{}{}
		}
		some = true
	}
	if none && (c.all || c.defaults || len(c.names) > 0) {
		return nil, errors.New("none can't be combined with other categories")
	}
	if !some {
		c.defaults = true
	}
	return c, nil
}

// Names returns the categories selected by name, sorted.
func (c *Categories) Names() []string {
	return slices.Sorted(maps.Keys(c.names))
}

// Has returns whether the category named name, on by default if byDefault, is
// selected.
func (c *Categories) Has(name string, byDefault bool) bool {
	if c == nil {
		return false
	}
	if c.all || (c.defaults && byDefault) {
		return true
	}
	_, ok := c.names[name]
	return ok
}
