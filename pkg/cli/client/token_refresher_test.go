package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/cli/config"
)

func TestNewTokenRefresher(t *testing.T) {
	tr := newTokenRefresher()
	require.NotNil(t, tr.redeemRefreshTokenFn)
	require.NotNil(t, tr.saveCLIConfigFn)
}

func TestRefreshToken(t *testing.T) {
	testCases := []struct {
		name                 string
		setup                func() config.CLIConfig
		redeemRefreshTokenFn func(
			ctx context.Context,
			serverAddress string,
			refreshToken string,
			insecureTLS bool,
		) (string, string, error)
		saveCLIConfigFn func(config.CLIConfig) error
		assertions      func(
			t *testing.T,
			originalCfg config.CLIConfig,
			updatedCfg config.CLIConfig,
			err error,
		)
	}{
		{
			name: "token is not a JWT",
			setup: func() config.CLIConfig {
				return config.CLIConfig{
					BearerToken: "not a JWT",
				}
			},
			assertions: func(t *testing.T, originalCfg, updatedCfg config.CLIConfig, err error) {
				require.NoError(t, err)
				require.Equal(t, originalCfg, updatedCfg)
			},
		},
		{
			name: "token is a non-expired JWT",
			setup: func() config.CLIConfig {
				cfg := config.CLIConfig{}
				var err error
				cfg.BearerToken, err = jwt.NewWithClaims(
					jwt.SigningMethodHS256,
					jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
					},
				).SignedString([]byte("signing key"))
				require.NoError(t, err)
				return cfg
			},
			assertions: func(t *testing.T, originalCfg, updatedCfg config.CLIConfig, err error,
			) {
				require.NoError(t, err)
				require.Equal(t, originalCfg, updatedCfg)
			},
		},
		{
			name: "token is an expired JWT; no refresh token present",
			setup: func() config.CLIConfig {
				cfg := config.CLIConfig{}
				var err error
				cfg.BearerToken, err = jwt.NewWithClaims(
					jwt.SigningMethodHS256,
					jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
					},
				).SignedString([]byte("signing key"))
				require.NoError(t, err)
				return cfg
			},
			assertions: func(t *testing.T, _, _ config.CLIConfig, err error) {
				require.Error(t, err)
				require.Equal(
					t,
					"your token is expired; please use `kargo login` to re-authenticate",
					err.Error(),
				)
			},
		},
		{
			name: "token is an expired JWT; refresh token present; tls warnings " +
				"ignored",
			setup: func() config.CLIConfig {
				cfg := config.CLIConfig{
					RefreshToken:          "refresh-token",
					InsecureSkipTLSVerify: true,
				}
				var err error
				cfg.BearerToken, err = jwt.NewWithClaims(
					jwt.SigningMethodHS256,
					jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
					},
				).SignedString([]byte("signing key"))
				require.NoError(t, err)
				return cfg
			},
			assertions: func(t *testing.T, _, _ config.CLIConfig, err error) {
				require.Error(t, err)
				require.Equal(
					t,
					"your token is expired; please use `kargo login` to re-authenticate",
					err.Error(),
				)
			},
		},
		{
			name: "token is an expired JWT; error redeeming refresh token",
			setup: func() config.CLIConfig {
				cfg := config.CLIConfig{
					RefreshToken: "refresh-token",
				}
				var err error
				cfg.BearerToken, err = jwt.NewWithClaims(
					jwt.SigningMethodHS256,
					jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
					},
				).SignedString([]byte("signing key"))
				require.NoError(t, err)
				return cfg
			},
			redeemRefreshTokenFn: func(
				context.Context,
				string,
				string,
				bool,
			) (string, string, error) {
				return "", "", errors.New("something went wrong")
			},
			assertions: func(t *testing.T, _, _ config.CLIConfig, err error) {
				require.Error(t, err)
				require.Equal(
					t,
					"error refreshing token; please use `kargo login` to re-authenticate",
					err.Error(),
				)
			},
		},
		{
			name: "token is an expired JWT; success redeeming refresh token",
			setup: func() config.CLIConfig {
				cfg := config.CLIConfig{
					RefreshToken: "refresh-token",
				}
				var err error
				cfg.BearerToken, err = jwt.NewWithClaims(
					jwt.SigningMethodHS256,
					jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
					},
				).SignedString([]byte("signing key"))
				require.NoError(t, err)
				return cfg
			},
			redeemRefreshTokenFn: func(
				context.Context,
				string,
				string,
				bool,
			) (string, string, error) {
				return "new-token", "new-refresh-token", nil
			},
			saveCLIConfigFn: func(config.CLIConfig) error {
				return nil
			},
			assertions: func(t *testing.T, _, newConfig config.CLIConfig, err error) {
				require.NoError(t, err)
				require.Equal(t, "new-token", newConfig.BearerToken)
				require.Equal(t, "new-refresh-token", newConfig.RefreshToken)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tf := &tokenRefresher{
				redeemRefreshTokenFn: testCase.redeemRefreshTokenFn,
				saveCLIConfigFn:      testCase.saveCLIConfigFn,
			}
			cfg := testCase.setup()
			newCfg, err :=
				tf.refreshToken(context.Background(), testCase.setup(), false)
			testCase.assertions(t, cfg, newCfg, err)
		})
	}
}
