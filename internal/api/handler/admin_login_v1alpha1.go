package handler

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/akuity/kargo/internal/api/config"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type AdminLoginV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.AdminLoginRequest],
) (*connect.Response[svcv1alpha1.AdminLoginResponse], error)

func AdminLoginV1Alpha1(cfg *config.AdminConfig) AdminLoginV1Alpha1Func {
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

		if err := bcrypt.CompareHashAndPassword(
			[]byte(cfg.HashedPassword),
			[]byte(req.Msg.Password),
		); err != nil {
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
