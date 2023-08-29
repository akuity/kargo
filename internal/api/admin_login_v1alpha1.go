package api

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func init() {
	jwt.MarshalSingleStringAsArray = false
}

func (s *server) AdminLogin(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.AdminLoginRequest],
) (*connect.Response[svcv1alpha1.AdminLoginResponse], error) {
	if s.cfg.AdminConfig == nil {
		return nil, connect.NewError(
			connect.CodePermissionDenied,
			errors.New("admin user is not enabled"),
		)
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(s.cfg.AdminConfig.HashedPassword),
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
			Issuer:    s.cfg.AdminConfig.TokenIssuer,
			Audience:  []string{s.cfg.AdminConfig.TokenAudience},
			NotBefore: jwt.NewNumericDate(now),
			Subject:   "admin",
			ID:        uuid.NewV4().String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AdminConfig.TokenTTL)),
		},
	)

	signedToken, err := idToken.SignedString(s.cfg.AdminConfig.TokenSigningKey)
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
