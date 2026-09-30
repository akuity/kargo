package database

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfig_ConnString(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		cfg    Config
		assert func(*testing.T, string, error)
	}{
		{
			name: "neither URL nor host",
			cfg:  Config{},
			assert: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "DATABASE_URL or DATABASE_HOST")
			},
		},
		{
			name: "URL is used verbatim",
			cfg: Config{
				URL:  "postgres://u:p@db:5432/x?sslmode=require",
				Host: "ignored",
			},
			assert: func(t *testing.T, s string, err error) {
				require.NoError(t, err)
				require.Equal(t, "postgres://u:p@db:5432/x?sslmode=require", s)
			},
		},
		{
			name: "composed from parts with the password escaped",
			cfg: Config{
				Host:     "kargo-postgres.kargo.svc",
				Port:     5432,
				Name:     "kargo",
				User:     "kargo",
				Password: "p@ss/w:rd",
				SSLMode:  "disable",
			},
			assert: func(t *testing.T, s string, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					"postgres://kargo:p%40ss%2Fw%3Ard@kargo-postgres.kargo.svc:5432/kargo?sslmode=disable",
					s,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			s, err := testCase.cfg.ConnString()
			testCase.assert(t, s, err)
		})
	}
}

func TestConfigFromEnv(t *testing.T) {
	// Not parallel: sets environment variables.
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DATABASE_HOST", "db")
	t.Setenv("DATABASE_PASSWORD", "secret")
	t.Setenv("DATABASE_CONNECT_TIMEOUT", "30s")
	cfg := ConfigFromEnv()
	require.Equal(t, "db", cfg.Host)
	require.Equal(t, 5432, cfg.Port)
	require.Equal(t, "kargo", cfg.Name)
	require.Equal(t, "kargo", cfg.User)
	require.Equal(t, "secret", cfg.Password)
	require.Equal(t, "disable", cfg.SSLMode)
	require.Equal(t, "30s", cfg.ConnectTimeout.String())
}
