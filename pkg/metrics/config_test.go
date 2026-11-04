package metrics

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigFromEnv(t *testing.T) {
	testCases := []struct {
		name    string
		disable string
		assert  func(*testing.T, Config, error)
	}{
		{
			name: "unset",
			assert: func(t *testing.T, cfg Config, err error) {
				require.NoError(t, err)
				require.True(t, cfg.Enabled(FamilyRESTClient))
				require.True(t, cfg.Enabled(FamilyLifecycle))
			},
		},
		{
			name:    "rest-client disabled",
			disable: "rest-client",
			assert: func(t *testing.T, cfg Config, err error) {
				require.NoError(t, err)
				require.False(t, cfg.Enabled(FamilyRESTClient))
			},
		},
		{
			name:    "case and spaces ignored",
			disable: " Rest-Client ,",
			assert: func(t *testing.T, cfg Config, err error) {
				require.NoError(t, err)
				require.False(t, cfg.Enabled(FamilyRESTClient))
			},
		},
		{
			name:    "lifecycle disabled",
			disable: "lifecycle",
			assert: func(t *testing.T, cfg Config, err error) {
				require.NoError(t, err)
				require.False(t, cfg.Enabled(FamilyLifecycle))
				require.True(t, cfg.Enabled(FamilyRESTClient))
			},
		},
		{
			name:    "all",
			disable: "all",
			assert: func(t *testing.T, cfg Config, err error) {
				require.NoError(t, err)
				require.False(t, cfg.Enabled(FamilyRESTClient))
				require.False(t, cfg.Enabled(FamilyLifecycle))
			},
		},
		{
			name:    "ALL",
			disable: "ALL",
			assert: func(t *testing.T, cfg Config, err error) {
				require.NoError(t, err)
				require.False(t, cfg.Enabled(FamilyRESTClient))
			},
		},
		{
			// Redundant, but valid.
			name:    "all and a family",
			disable: " all ,rest-client",
			assert: func(t *testing.T, cfg Config, err error) {
				require.NoError(t, err)
				require.False(t, cfg.Enabled(FamilyRESTClient))
			},
		},
		{
			// A boolean is not a family.
			name:    "true",
			disable: "true",
			assert: func(t *testing.T, _ Config, err error) {
				require.ErrorContains(t, err, `unknown metrics family "true" in METRICS_DISABLE`)
			},
		},
		{
			name:    "unknown family",
			disable: "rest-client,rest-clients",
			assert: func(t *testing.T, _ Config, err error) {
				require.ErrorContains(t, err, `unknown metrics family "rest-clients" in METRICS_DISABLE`)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Not parallel: manipulates the environment.
			t.Setenv("METRICS_DISABLE", testCase.disable)
			cfg, err := ConfigFromEnv()
			testCase.assert(t, cfg, err)
		})
	}
}
