package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
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

// @id AdminLogin
// @Summary Admin login
// @Description Authenticate as the admin user if enabled.
// @Security BearerAuth
// @Tags System
// @Produce json
// @Success 200 {object} adminLoginResponse
// @Router /v1beta1/login [post]
func (s *server) adminLogin(c *gin.Context) {
	if s.cfg.AdminConfig == nil {
		_ = c.Error(libhttp.Error(
			errors.New("admin user is not enabled"),
			http.StatusForbidden,
		))
		return
	}

	// Extract password from Authorization header (format: "Bearer <password>")
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		_ = c.Error(libhttp.Error(
			errors.New("Authorization header is required"), // nolint: staticcheck
			http.StatusBadRequest,
		))
		return
	}

	// Extract the password from "Bearer <password>" format
	const bearerPrefix = "Bearer "
	if len(authHeader) <= len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
		_ = c.Error(libhttp.Error(
			errors.New("Authorization header must be in format 'Bearer <password>'"), // nolint: staticcheck
			http.StatusBadRequest,
		))
		return
	}
	password := authHeader[len(bearerPrefix):]

	if err := bcrypt.CompareHashAndPassword(
		[]byte(s.cfg.AdminConfig.HashedPassword),
		[]byte(password),
	); err != nil {
		_ = c.Error(libhttp.Error(
			errors.New("invalid password"),
			http.StatusForbidden,
		))
		return
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
		_ = c.Error(fmt.Errorf("error signing ID token: %w", err))
		return
	}

	c.JSON(http.StatusOK, adminLoginResponse{
		IDToken: signedToken,
	})
}

type adminLoginResponse struct {
	IDToken string `json:"idToken"`
} // @name AdminLoginResponse
