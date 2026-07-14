package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/client/generated/models"
)

func TestFreightMarshal_NoPhantomEmptyObjects(t *testing.T) {
	testCases := []struct {
		name   string
		assert func(*testing.T, []byte)
	}{
		{
			name: "unset optional status is omitted, not an empty object",
			assert: func(t *testing.T, b []byte) {
				require.NotContains(t, string(b), `"status":{}`)
			},
		},
		{
			name: "unset optional metadata is omitted, not an empty object",
			assert: func(t *testing.T, b []byte) {
				require.NotContains(t, string(b), `"metadata":{}`)
			},
		},
	}

	freight := models.Freight{Alias: "my-freight"}
	b, err := json.Marshal(freight)
	require.NoError(t, err)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assert(t, b)
		})
	}
}
