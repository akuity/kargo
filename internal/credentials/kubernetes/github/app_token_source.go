package github

import (
	"crypto/rsa"
	"errors"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

// appTokenSource is an implementation of ouath2.TokenSource that returns tokens
// for a GitHub App to access portions of the GitHub API. We've implemented our
// own instead of using the implementation from github.com/jferrl/go-githubauth
// because that package's NewInstallationTokenSource() function and the
// ouath2.TokenSource it returns do not yet allow for the possibility of using
// the alphanumeric client ID (string) as an alternative to the numeric (int64)
// App ID as the token issuer. GitHub recommends using client ID when possible.
// Our implementation is nearly identical to theirs, but uses a string type
// for the issuer.
type appTokenSource struct {
	issuer     string
	privateKey *rsa.PrivateKey
	expiration time.Duration
}

// newApplicationTokenSource creates a new GitHub App token source using the
// provided issuer identifier and private key. The issuer may be an alphanumeric
// client ID or a string representation of a numeric App ID.
func newApplicationTokenSource(
	issuer string,
	privateKey []byte,
) (oauth2.TokenSource, error) {
	if issuer == "" {
		return nil, errors.New("issuer is required")
	}

	if len(privateKey) == 0 {
		return nil, errors.New("private key is required")
	}

	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKey)
	if err != nil {
		return nil, err
	}

	t := &appTokenSource{
		issuer:     issuer,
		privateKey: privKey,
		expiration: 10 * time.Minute,
	}

	return t, nil
}

// Token implements oauth2.TokenSource.
func (a *appTokenSource) Token() (*oauth2.Token, error) {
	// GitHub recommends setting the iat claim (issued at time) to 60 seconds in
	// the past to guard against issues arising from clock drift.
	//
	// nolint: lll
	//   https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-json-web-token-jwt-for-a-github-app#about-json-web-tokens-jwts
	now := time.Now().Add(-60 * time.Second)
	expiresAt := now.Add(a.expiration)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
		Issuer:    a.issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	})

	tokenString, err := token.SignedString(a.privateKey)
	if err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		Expiry:      expiresAt,
	}, nil
}
