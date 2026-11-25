package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServerConfigFromEnv_TracingEnabled(t *testing.T) {
	// Not parallel: manipulates the environment.
	testCases := []struct {
		name     string
		value    string
		expected bool
	}{
		{name: "unset", expected: false},
		{name: "false", value: "false", expected: false},
		{name: "true", value: "true", expected: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.value != "" {
				t.Setenv("TRACING_ENABLED", testCase.value)
			}
			require.Equal(t, testCase.expected, ServerConfigFromEnv().TracingEnabled)
		})
	}
}
