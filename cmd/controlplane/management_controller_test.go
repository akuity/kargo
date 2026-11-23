package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/client/apiserver"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/logging"
)

func TestManagementControllerOptionsRun(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		opts   *managementControllerOptions
		assert func(*testing.T, error)
	}{
		{
			name: "no database",
			opts: &managementControllerOptions{
				APIServer: apiserver.ClientConfig{Address: "https://kargo-api"},
			},
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "DATABASE_URL")
				require.ErrorContains(t, err, "DATABASE_HOST")
			},
		},
		{
			name: "no API server",
			opts: &managementControllerOptions{
				Database: database.Config{URL: "postgres://localhost/kargo"},
			},
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "API_SERVER_ADDRESS")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.opts.Logger = logging.NewDiscardLoggerOrDie()
			testCase.assert(t, testCase.opts.run(t.Context()))
		})
	}
}
