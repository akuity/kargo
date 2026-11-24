package namer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultDescriptors(t *testing.T) {
	t.Parallel()
	descriptors := DefaultDescriptors()
	require.Equal(t, defaultDescriptors, descriptors)
	// The copy must be independent of the package's own list.
	descriptors[0] = "mutated"
	require.NotEqual(t, "mutated", defaultDescriptors[0])
}
