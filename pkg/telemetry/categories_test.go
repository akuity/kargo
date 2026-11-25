package telemetry

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/telemetry/internal/state"
)

func TestParseCategories(t *testing.T) {
	t.Parallel()
	// later stands in for a category registered after the selection is made.
	const later = "later"
	type parseCase struct {
		name    string
		entries []string
		assert  func(*testing.T, *state.Categories, error)
	}
	testCases := []parseCase{
		{
			name: "unset",
			assert: func(t *testing.T, s *state.Categories, err error) {
				require.NoError(t, err)
				require.True(t, databaseSelected(s))
			},
		},
		{
			name:    "empty",
			entries: []string{"", " "},
			assert: func(t *testing.T, s *state.Categories, err error) {
				require.NoError(t, err)
				require.True(t, databaseSelected(s))
			},
		},
		{
			name:    "default",
			entries: []string{"default"},
			assert: func(t *testing.T, s *state.Categories, err error) {
				require.NoError(t, err)
				require.True(t, databaseSelected(s))
				require.True(t, s.Has(later, true))
				require.False(t, s.Has(later, false))
			},
		},
		{
			name:    "all",
			entries: []string{"all"},
			assert: func(t *testing.T, s *state.Categories, err error) {
				require.NoError(t, err)
				require.True(t, databaseSelected(s))
				require.True(t, s.Has(later, false))
			},
		},
		{
			name:    "all with a category, redundantly",
			entries: []string{"all", "database"},
			assert: func(t *testing.T, s *state.Categories, err error) {
				require.NoError(t, err)
				require.True(t, databaseSelected(s))
				require.True(t, s.Has(later, false))
			},
		},
		{
			name:    "none",
			entries: []string{"none"},
			assert: func(t *testing.T, s *state.Categories, err error) {
				require.NoError(t, err)
				require.False(t, databaseSelected(s))
				require.False(t, s.Has(later, true))
			},
		},
		{
			name:    "none with a category",
			entries: []string{"none", "database"},
			assert: func(t *testing.T, s *state.Categories, err error) {
				require.EqualError(
					t,
					err,
					"invalid TRACING_CATEGORIES: "+
						"none can't be combined with other categories",
				)
				require.Nil(t, s)
			},
		},
		{
			name:    "category by name",
			entries: []string{"database"},
			assert: func(t *testing.T, s *state.Categories, err error) {
				require.NoError(t, err)
				require.True(t, databaseSelected(s))
				require.False(t, s.Has(later, true))
			},
		},
		{
			name:    "case and spaces are ignored, empty entries dropped",
			entries: []string{" DataBase ", "", "  "},
			assert: func(t *testing.T, s *state.Categories, err error) {
				require.NoError(t, err)
				require.True(t, databaseSelected(s))
				require.False(t, s.Has(later, true))
			},
		},
		{
			name:    "keywords ignore case and spaces",
			entries: []string{" NONE "},
			assert: func(t *testing.T, s *state.Categories, err error) {
				require.NoError(t, err)
				require.False(t, databaseSelected(s))
			},
		},
		{
			name:    "unknown category",
			entries: []string{"database", "Bogus"},
			assert: func(t *testing.T, s *state.Categories, err error) {
				require.ErrorContains(
					t,
					err,
					`unknown tracing category "bogus" in TRACING_CATEGORIES; `+
						"known categories: ",
				)
				require.ErrorContains(t, err, string(CategoryDatabase))
				require.ErrorContains(t, err, "; keywords: default, all, none")
				require.Nil(t, s)
			},
		},
	}
	for _, b := range []string{"true", "false", "1", "0", "yes", "no"} {
		testCases = append(testCases, parseCase{
			name:    b,
			entries: []string{b},
			assert: func(t *testing.T, s *state.Categories, err error) {
				require.ErrorContains(t, err, `unknown tracing category "`+b+`"`)
				require.Nil(t, s)
			},
		})
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			s, err := parseCategories(testCase.entries)
			testCase.assert(t, s, err)
		})
	}
}

func TestCategoryEnabled(t *testing.T) {
	// Not parallel: changes global telemetry state.
	prevEnabled := state.Enabled.Load()
	prevSelected := state.Selected.Load()
	t.Cleanup(func() {
		state.Enabled.Store(prevEnabled)
		state.Selected.Store(prevSelected)
	})
	all, err := state.ParseCategories([]string{"all"})
	require.NoError(t, err)

	state.Enabled.Store(false)
	state.Selected.Store(all)
	require.False(t, CategoryEnabled(CategoryDatabase))

	state.Enabled.Store(true)
	require.True(t, CategoryEnabled(CategoryDatabase))

	state.Selected.Store(nil)
	require.False(t, CategoryEnabled(CategoryDatabase))
}

// databaseSelected returns whether s selects CategoryDatabase.
func databaseSelected(s *state.Categories) bool {
	return s.Has(string(CategoryDatabase), categories[CategoryDatabase])
}
