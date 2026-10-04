package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
)

func init() {
	jwt.MarshalSingleStringAsArray = false
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
	logger := logging.LoggerFromContext(c.Request.Context()).
		WithValues("loginType", "admin")
	signedToken, err := s.newAdminToken(c.GetHeader("Authorization"))
	if err != nil {
		logger.Info("login failed", "error", err.Error())
		_ = c.Error(err)
		return
	}
	logger.Info("login successful")
	c.JSON(http.StatusOK, adminLoginResponse{
		IDToken: signedToken,
	})
}

// newAdminToken returns a signed ID token for the admin user if authHeader
// carries the admin password in the format "Bearer <password>".
func (s *server) newAdminToken(authHeader string) (string, error) {
	if s.cfg.AdminConfig == nil {
		return "", libhttp.Error(
			errors.New("admin user is not enabled"),
			http.StatusForbidden,
		)
	}

	if authHeader == "" {
		return "", libhttp.Error(
			errors.New("Authorization header is required"), // nolint: staticcheck
			http.StatusBadRequest,
		)
	}

	const bearerPrefix = "Bearer "
	if len(authHeader) <= len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", libhttp.Error(
			errors.New("Authorization header must be in format 'Bearer <password>'"), // nolint: staticcheck
			http.StatusBadRequest,
		)
	}
	password := authHeader[len(bearerPrefix):]

	if err := bcrypt.CompareHashAndPassword(
		[]byte(s.cfg.AdminConfig.HashedPassword),
		[]byte(password),
	); err != nil {
		return "", libhttp.Error(
			errors.New("invalid password"),
			http.StatusForbidden,
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
		return "", fmt.Errorf("error signing ID token: %w", err)
	}
	return signedToken, nil
}

type adminLoginResponse struct {
	IDToken string `json:"idToken"`
} // @name AdminLoginResponse
