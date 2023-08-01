package handler

import (
	"context"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

// AdminConfig represents configuration for an admin account.
type AdminConfig struct {
	// Password is the password for the admin account.
	Password string `envconfig:"ADMIN_ACCOUNT_PASSWORD" required:"true"`
	// TokenSigningKey is the key used to sign ID tokens for the admin account.
	TokenSigningKey []byte `envconfig:"TOKEN_SIGNING_KEY" required:"true"`
}

// AdminConfigFromEnv returns an AdminConfig populated from environment
// variables.
func AdminConfigFromEnv() AdminConfig {
	cfg := AdminConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

type AdminLoginV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.AdminLoginRequest],
) (*connect.Response[svcv1alpha1.AdminLoginResponse], error)

func AdminLoginV1Alpha1(cfg *AdminConfig) AdminLoginV1Alpha1Func {
	return func(
		_ context.Context,
		req *connect.Request[svcv1alpha1.AdminLoginRequest],
	) (*connect.Response[svcv1alpha1.AdminLoginResponse], error) {
		if cfg == nil {
			return nil, connect.NewError(
				connect.CodePermissionDenied,
				errors.New("admin user is not enabled"),
			)
		}

		if req.Msg.Password != cfg.Password {
			return nil, connect.NewError(
				connect.CodePermissionDenied,
				errors.New("invalid password"),
			)
		}

		now := time.Now()
		idToken := jwt.NewWithClaims(
			jwt.SigningMethodHS256,
			jwt.RegisteredClaims{
				IssuedAt:  jwt.NewNumericDate(now),
				Issuer:    "kargo",
				NotBefore: jwt.NewNumericDate(now),
				Subject:   "admin",
				ID:        uuid.NewV4().String(),
				ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			},
		)

		signedToken, err := idToken.SignedString(cfg.TokenSigningKey)
		if err != nil {
			return nil, connect.NewError(
				connect.CodeInternal,
				errors.Wrap(err, "error signing ID token"),
			)
		}

		return connect.NewResponse(&svcv1alpha1.AdminLoginResponse{
			IdToken: signedToken,
		}), nil
	}
}
