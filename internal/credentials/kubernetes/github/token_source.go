package github

import (
	"crypto/rsa"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

const (
	// bearerTokenType is the token type for GitHub App tokens
	bearerTokenType = "Bearer"

	// defaultApplicationTokenExpiration is the default expiration time for the GitHub App token.
	// The expiration time of the JWT, after which it can't be used to request an installation token.
	// The time must be no more than 10 minutes into the future.
	defaultApplicationTokenExpiration = 10 * time.Minute
)

// applicationTokenSource represents a GitHub App token source that can handle
// both numeric app IDs and alphanumeric client IDs.
type applicationTokenSource struct {
	appID      string // Can be numeric app ID or alphanumeric client ID
	privateKey *rsa.PrivateKey
	expiration time.Duration
}

// Token generates a new GitHub App token for authenticating as a GitHub App.
func (t *applicationTokenSource) Token() (*oauth2.Token, error) {
	// To protect against clock drift, set the issuance time 60 seconds in the past.
	now := time.Now().Add(-60 * time.Second)
	expiresAt := now.Add(t.expiration)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		Issuer:    t.appID, // Use appID directly as string (works for both numeric and alphanumeric)
	})

	tokenString, err := token.SignedString(t.privateKey)
	if err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken: tokenString,
		TokenType:   bearerTokenType,
		Expiry:      expiresAt,
	}, nil
}
