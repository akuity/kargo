package dbreconcile

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnqueueKey(t *testing.T) {
	t.Parallel()
	require.Equal(t, []string{"a"}, EnqueueKey[string, item]().Keys(context.Background(), created("a")))
}

func TestEnqueueMapped(t *testing.T) {
	t.Parallel()
	// A child event, keyed by the child's name, maps to its parents' keys.
	handler := EnqueueMapped(func(_ context.Context, e Event[string, item]) []int {
		return []int{len(e.Key), strings.Count(e.Key, "-")}
	})
	require.Equal(t, []int{5, 1}, handler.Keys(context.Background(), created("ab-cd")))
}
