package telemetrytest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/telemetry/internal/state"
)

func TestSetCategories(t *testing.T) {
	// Not parallel: changes global telemetry state.
	selected := func(name string, byDefault bool) bool {
		return state.Selected.Load().Has(name, byDefault)
	}
	prev := state.Selected.Load()
	t.Run("set until the test ends", func(t *testing.T) {
		SetCategories(t, "default,k8s")
		require.True(t, selected("k8s", false))
		require.True(t, selected("database", true))
		require.False(t, selected("patch", false))
		t.Run("tracer provider selects every category", func(t *testing.T) {
			UseTracerProvider(t, provider)
			require.True(t, selected("patch", false))
			SetCategories(t, " None ")
			require.False(t, selected("database", true))
		})
		require.True(t, selected("k8s", false))
		require.False(t, selected("patch", false))
	})
	require.Same(t, prev, state.Selected.Load())
}
