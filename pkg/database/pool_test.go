package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPool(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "invalid configuration does not expose credentials", url: "postgres://secret-password@[%", wantErr: true},
		{name: "unreachable database does not prevent startup", url: "postgres://kargo:kargo@127.0.0.1:1/kargo"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			pool, err := NewPool(context.Background(), testCase.url)
			if testCase.wantErr {
				require.Error(t, err)
				require.NotContains(t, err.Error(), "secret-password")
				require.Nil(t, pool)
				return
			}
			require.NoError(t, err)
			defer pool.Close()
			require.EqualValues(t, 8, pool.Config().MaxConns)
			require.Zero(t, pool.Config().MinConns)
			require.Equal(t, operationTimeout, pool.Config().ConnConfig.ConnectTimeout)
		})
	}
}
