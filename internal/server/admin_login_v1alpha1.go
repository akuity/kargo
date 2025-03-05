package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func init() {
	jwt.MarshalSingleStringAsArray = false
}

func (s *server) AdminLogin(
	_ context.Context,
	req *connect.Request[svcv1alpha1.AdminLoginRequest],
) (*connect.Response[svcv1alpha1.AdminLoginResponse], error) {
	if s.cfg.AdminConfig == nil {
		return nil, connect.NewError(
			connect.CodePermissionDenied,
			errors.New("admin user is not enabled"),
		)
	}

	password := req.Msg.GetPassword()
	if err := validateFieldNotEmpty("password", password); err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(s.cfg.AdminConfig.HashedPassword),
		[]byte(password),
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
			ID:        uuid.NewString(),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AdminConfig.TokenTTL)),
		},
	)

	signedToken, err := idToken.SignedString(s.cfg.AdminConfig.TokenSigningKey)
	if err != nil {
		return nil, connect.NewError(
			connect.CodeInternal,
			fmt.Errorf("error signing ID token: %w", err),
		)
	}

	return connect.NewResponse(&svcv1alpha1.AdminLoginResponse{
		IdToken: signedToken,
	}), nil
}
