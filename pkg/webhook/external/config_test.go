package external

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
		{name: "unset", expected: true},
		{name: "false", value: "false", expected: true},
		{name: "true", value: "true", expected: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("EXTERNAL_WEBHOOK_SERVER_BASE_URL", "https://webhooks.example.com")
			t.Setenv("OTEL_SDK_DISABLED", testCase.value)
			require.Equal(t, testCase.expected, ServerConfigFromEnv().TracingEnabled)
		})
	}
}
