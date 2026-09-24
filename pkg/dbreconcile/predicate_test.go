package dbreconcile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPredicateAllow(t *testing.T) {
	t.Parallel()
	phaseChanged := Predicate[string, item]{
		Update: func(e Event[string, item]) bool { return e.Old.Phase != e.New.Phase },
	}
	nothing := Predicate[string, item]{
		Create: func(Event[string, item]) bool { return false },
		Update: func(Event[string, item]) bool { return false },
		Delete: func(Event[string, item]) bool { return false },
	}
	testCases := []struct {
		name      string
		predicate Predicate[string, item]
		event     Event[string, item]
		allowed   bool
	}{
		{
			name:    "an empty predicate lets a create through",
			event:   created("a"),
			allowed: true,
		},
		{
			name:    "an empty predicate lets an update through",
			event:   updated("a", "Pending", "Pending"),
			allowed: true,
		},
		{
			name:    "an empty predicate lets a delete through",
			event:   deleted("a"),
			allowed: true,
		},
		{
			name:      "an unset function lets its kind through",
			predicate: phaseChanged,
			event:     created("a"),
			allowed:   true,
		},
		{
			name:      "an update the predicate wants",
			predicate: phaseChanged,
			event:     updated("a", "Pending", "Running"),
			allowed:   true,
		},
		{
			name:      "an update the predicate does not want",
			predicate: phaseChanged,
			event:     updated("a", "Pending", "Pending"),
		},
		{
			name:      "a create the predicate does not want",
			predicate: nothing,
			event:     created("a"),
		},
		{
			name:      "a delete the predicate does not want",
			predicate: nothing,
			event:     deleted("a"),
		},
		{
			name:  "an unknown kind is never let through",
			event: Event[string, item]{Kind: "Renamed", Key: "a"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.allowed, testCase.predicate.Allow(testCase.event))
		})
	}
}
