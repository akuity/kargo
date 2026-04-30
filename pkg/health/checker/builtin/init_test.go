package builtin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitialize(t *testing.T) {
	require.NotPanics(t, func() { Initialize(nil) })
	// Should panic if called more than once
	require.PanicsWithValue(
		t,
		"built-in health checkers already initialized",
		func() { Initialize(nil) },
	)
}
