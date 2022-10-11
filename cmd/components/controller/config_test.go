package controller

import (
	"testing"

	"github.com/akuityio/k8sta/internal/bookkeeper"
	"github.com/stretchr/testify/require"
)

func TestBookkeeperClientConfig(t *testing.T) {
	const testAddress = "https://bookkeeper.example.com"
	testCases := []struct {
		name       string
		setup      func()
		assertions func(address string, opts bookkeeper.ClientOptions, err error)
	}{
		{
			name: "BOOKKEEPER_ADDRESS not set",
			assertions: func(_ string, _ bookkeeper.ClientOptions, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"value not found for required environment variable",
				)
				require.Contains(t, err.Error(), "BOOKKEEPER_ADDRESS")
			},
		},
		{
			name: "IGNORE_BOOKKEEPER_CERT_WARNINGS not a bool",
			setup: func() {
				t.Setenv("BOOKKEEPER_ADDRESS", testAddress)
				t.Setenv("BOOKKEEPER_IGNORE_CERT_WARNINGS", "nope")
			},
			assertions: func(_ string, _ bookkeeper.ClientOptions, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "was not parsable as a bool")
				require.Contains(t, err.Error(), "BOOKKEEPER_IGNORE_CERT_WARNINGS")
			},
		},
		{
			name: "success",
			setup: func() {
				t.Setenv("BOOKKEEPER_IGNORE_CERT_WARNINGS", "true")
			},
			assertions: func(
				address string,
				opts bookkeeper.ClientOptions,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, testAddress, address)
				require.Equal(
					t,
					bookkeeper.ClientOptions{
						AllowInsecureConnections: true,
					},
					opts,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.setup != nil {
				testCase.setup()
			}
			address, opts, err := bookkeeperClientConfig()
			testCase.assertions(address, opts, err)
		})
	}
}
