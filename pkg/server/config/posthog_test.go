package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServerConfigFromEnv_PostHogConfig(t *testing.T) {
	// Not parallel: manipulates the environment.
	testCases := []struct {
		name   string
		env    map[string]string
		assert func(*testing.T, *PostHogConfig)
	}{
		{
			name: "unset",
			assert: func(t *testing.T, cfg *PostHogConfig) {
				require.Nil(t, cfg)
			},
		},
		{
			name: "disabled",
			env: map[string]string{
				"POSTHOG_ENABLED":        "false",
				"POSTHOG_PUBLIC_API_KEY": "phc_test",
			},
			assert: func(t *testing.T, cfg *PostHogConfig) {
				require.Nil(t, cfg)
			},
		},
		{
			name: "enabled with default host",
			env: map[string]string{
				"POSTHOG_ENABLED":        "true",
				"POSTHOG_PUBLIC_API_KEY": "phc_test",
			},
			assert: func(t *testing.T, cfg *PostHogConfig) {
				require.Equal(
					t,
					&PostHogConfig{
						PublicAPIKey: "phc_test",
						Host:         "https://us.i.posthog.com",
					},
					cfg,
				)
			},
		},
		{
			name: "enabled with custom host",
			env: map[string]string{
				"POSTHOG_ENABLED":        "true",
				"POSTHOG_PUBLIC_API_KEY": "phc_test",
				"POSTHOG_HOST":           "https://posthog.example.com",
			},
			assert: func(t *testing.T, cfg *PostHogConfig) {
				require.Equal(
					t,
					&PostHogConfig{
						PublicAPIKey: "phc_test",
						Host:         "https://posthog.example.com",
					},
					cfg,
				)
			},
		},
		{
			name: "enabled with session replay",
			env: map[string]string{
				"POSTHOG_ENABLED":                "true",
				"POSTHOG_PUBLIC_API_KEY":         "phc_test",
				"POSTHOG_SESSION_REPLAY_ENABLED": "true",
			},
			assert: func(t *testing.T, cfg *PostHogConfig) {
				require.Equal(
					t,
					&PostHogConfig{
						PublicAPIKey:         "phc_test",
						Host:                 "https://us.i.posthog.com",
						SessionReplayEnabled: true,
					},
					cfg,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			for k, v := range testCase.env {
				t.Setenv(k, v)
			}
			testCase.assert(t, ServerConfigFromEnv().PostHogConfig)
		})
	}
}

func TestPostHogConfigFromEnv_MissingPublicAPIKey(t *testing.T) {
	// Not parallel: manipulates the environment. Setting the variable first
	// has the test restore it afterward; a variable that is set but empty
	// would satisfy envconfig, so it is then unset.
	t.Setenv("POSTHOG_PUBLIC_API_KEY", "")
	require.NoError(t, os.Unsetenv("POSTHOG_PUBLIC_API_KEY"))
	require.Panics(t, func() { PostHogConfigFromEnv() })
}
