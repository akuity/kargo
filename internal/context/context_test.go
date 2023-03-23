package context

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetSet(t *testing.T) {
	type dummyKey struct{}
	testSets := map[string]struct {
		newContext    func() context.Context
		expectedValue string
		expectedOK    bool
	}{
		"empty context": {
			newContext:    context.Background,
			expectedValue: "",
			expectedOK:    false,
		},
		"context with value": {
			newContext: func() context.Context {
				return context.WithValue(context.Background(), dummyKey{}, "kargo")
			},
			expectedValue: "kargo",
			expectedOK:    true,
		},
	}
	for name, ts := range testSets {
		t.Run(name, func(t *testing.T) {
			ctx := ts.newContext()
			actualValue, actualOK := get[dummyKey, string](ctx, dummyKey{})
			require.Equal(t, ts.expectedOK, actualOK)
			require.Equal(t, ts.expectedValue, actualValue)
		})
	}
}
