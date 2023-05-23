package kubeclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCredentialContext(t *testing.T) {
	testSets := map[string]struct {
		ctx          context.Context
		expectedCred string
		expectedOk   bool
	}{
		"empty context": {
			ctx:        context.Background(),
			expectedOk: false,
		},
		"context with credential": {
			ctx:          SetCredentialToContext(context.Background(), "Bearer token"),
			expectedCred: "Bearer token",
			expectedOk:   true,
		},
		"context with empty credential": {
			ctx:          SetCredentialToContext(context.Background(), ""),
			expectedCred: "",
			expectedOk:   true,
		},
	}
	for name, ts := range testSets {
		ts := ts
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cred, ok := GetCredentialFromContext(ts.ctx)
			require.Equal(t, ts.expectedCred, cred)
			require.Equal(t, ts.expectedOk, ok)
		})
	}
}
