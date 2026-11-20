package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServerConfigFromEnv_AuthCacheConfig(t *testing.T) {
	// Not parallel: manipulates the environment.
	testCases := []struct {
		name     string
		env      map[string]string
		expected AuthCacheConfig
	}{
		{
			name: "defaults",
			expected: AuthCacheConfig{
				TokenTTL:         2 * time.Minute,
				DecisionAllowTTL: 5 * time.Minute,
				DecisionDenyTTL:  30 * time.Second,
			},
		},
		{
			name: "overridden, including disabled",
			env: map[string]string{
				"AUTH_TOKEN_CACHE_TTL":          "0",
				"AUTH_DECISION_CACHE_ALLOW_TTL": "1m",
				"AUTH_DECISION_CACHE_DENY_TTL":  "0s",
			},
			expected: AuthCacheConfig{DecisionAllowTTL: time.Minute},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			for k, v := range testCase.env {
				t.Setenv(k, v)
			}
			require.Equal(t, testCase.expected, ServerConfigFromEnv().AuthCacheConfig)
		})
	}
}
