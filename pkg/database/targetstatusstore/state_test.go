package targetstatusstore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStates(t *testing.T) {
	t.Parallel()
	// The order is the precedence a Target governed by several Stages is
	// counted by, so it is pinned here.
	require.Equal(
		t,
		[]State{
			StatePromoting,
			StatePending,
			StateFailed,
			StateNotAttempted,
			StateNeverPromoted,
			StateBehind,
			StateUpToDate,
		},
		States(),
	)
	// A caller may change what it is given without changing the vocabulary.
	States()[0] = "changed"
	require.Equal(t, StatePromoting, States()[0])
}

func TestStateValid(t *testing.T) {
	t.Parallel()
	for _, state := range States() {
		require.True(t, state.Valid(), state)
	}
	for _, state := range []State{"", "Healthy", "pending", "Succeeded"} {
		require.False(t, state.Valid(), state)
	}
}
