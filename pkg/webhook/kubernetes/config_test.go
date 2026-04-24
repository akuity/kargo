package webhook

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigFromEnv(t *testing.T) {
	testCases := map[string]struct {
		envs     map[string]string
		assertFn func(*testing.T, func() Config)
	}{
		"empty controlplane user regex should not panic": {
			envs: map[string]string{
				"KARGO_NAMESPACE":         "kargo",
				"CONTROLPLANE_USER_REGEX": "",
			},
			assertFn: func(t *testing.T, f func() Config) {
				var cfg Config
				require.NotPanics(t, func() {
					cfg = f()
				})
				require.Nil(t, cfg.ControlplaneUserRegex)
			},
		},
		"invalid controlplane user regex should panic": {
			envs: map[string]string{
				"KARGO_NAMESPACE":         "kargo",
				"CONTROLPLANE_USER_REGEX": "[",
			},
			assertFn: func(t *testing.T, f func() Config) {
				require.Panics(t, func() { f() })
			},
		},
		"default controlplane user regex in helm chart": {
			envs: map[string]string{
				"KARGO_NAMESPACE":         "kargo",
				"CONTROLPLANE_USER_REGEX": "^system:serviceaccount:kargo:(kargo-api|kargo-controller)$",
			},
			assertFn: func(t *testing.T, f func() Config) {
				var cfg Config
				require.NotPanics(t, func() {
					cfg = f()
				})
				require.NotNil(t, cfg.ControlplaneUserRegex)
				require.True(t, cfg.ControlplaneUserRegex.MatchString("system:serviceaccount:kargo:kargo-api"))
				require.True(t, cfg.ControlplaneUserRegex.MatchString("system:serviceaccount:kargo:kargo-controller"))
			},
		},
		"sample controlplane user regex in helm chart": {
			envs: map[string]string{
				"KARGO_NAMESPACE":         "kargo",
				"CONTROLPLANE_USER_REGEX": "^system:serviceaccount:kargo:[a-z0-9]([-a-z0-9]*[a-z0-9])?$",
			},
			assertFn: func(t *testing.T, f func() Config) {
				var cfg Config
				require.NotPanics(t, func() {
					cfg = f()
				})
				require.NotNil(t, cfg.ControlplaneUserRegex)
				require.True(t, cfg.ControlplaneUserRegex.MatchString("system:serviceaccount:kargo:kargo-api"))
				require.True(t, cfg.ControlplaneUserRegex.MatchString("system:serviceaccount:kargo:kargo-controller"))
			},
		},
		"management controller username is populated": {
			envs: map[string]string{
				"KARGO_NAMESPACE":                "kargo",
				"MANAGEMENT_CONTROLLER_USERNAME": "system:serviceaccount:kargo:kargo-management-controller",
			},
			assertFn: func(t *testing.T, f func() Config) {
				var cfg Config
				require.NotPanics(t, func() {
					cfg = f()
				})
				require.Equal(
					t,
					"system:serviceaccount:kargo:kargo-management-controller",
					cfg.ManagementControllerUsername,
				)
			},
		},
		"external webhooks server username is populated": {
			envs: map[string]string{
				"KARGO_NAMESPACE":                   "kargo",
				"EXTERNAL_WEBHOOKS_SERVER_USERNAME": "system:serviceaccount:kargo:kargo-external-webhooks-server",
			},
			assertFn: func(t *testing.T, f func() Config) {
				var cfg Config
				require.NotPanics(t, func() {
					cfg = f()
				})
				require.Equal(
					t,
					"system:serviceaccount:kargo:kargo-external-webhooks-server",
					cfg.ExternalWebhooksServerUsername,
				)
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}
			tc.assertFn(t, ConfigFromEnv)
		})
	}
}
