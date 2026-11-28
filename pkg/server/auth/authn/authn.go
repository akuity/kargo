// Package authn resolves the bearer tokens presented to the Kargo API server
// to the identities behind them.
//
// A token may have been issued by the Kargo API server itself (its admin
// user), by Kargo's OpenID Connect identity provider, or by Kubernetes (a
// ServiceAccount token, or the cluster's own identity provider). Each kind
// has an Authenticator of its own, and a Chain tries them in order. An
// Authenticator recognizes its kind of token by the issuer the token claims,
// which it reads before verifying anything and trusts only to decide whether
// the token is its to verify.
package authn

import (
	"context"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"sigs.k8s.io/controller-runtime/pkg/client"

	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/user"
)

// ErrInvalidToken is the only rejection reported to clients. It carries a 401
// and discloses nothing about which check rejected the credential. Where the
// underlying reason is useful, Authenticators wrap it so that the detail
// reaches the logs while the client's response remains opaque.
//
// Any other error an Authenticator returns, such as an unreachable API
// server or a misconfigured identity provider, carries no status code, which
// the error-handling middleware reports as an internal error. A client must
// not be able to mistake a broken control plane for a rejected token.
var ErrInvalidToken = libhttp.ErrorStr("invalid token", http.StatusUnauthorized)

// unverifiedIssuer returns the issuer a JWT claims without verifying the
// token, and false if the token is not a JWT at all. Anyone can write any
// issuer into a token, so it only says which Authenticator should verify the
// token, never that the token is valid.
func unverifiedIssuer(rawToken string) (string, bool) {
	claims, ok := unverifiedClaims(rawToken)
	return claims.Issuer, ok
}

// unverifiedClaims returns the registered claims of a JWT without verifying
// the token, and false if the token is not a JWT at all. Nothing in them is
// to be trusted until the token has been verified.
func unverifiedClaims(rawToken string) (jwt.RegisteredClaims, bool) {
	claims := jwt.RegisteredClaims{}
	if _, _, err := jwt.NewParser(jwt.WithoutClaimsValidation()).
		ParseUnverified(rawToken, &claims); err != nil {
		return jwt.RegisteredClaims{}, false
	}
	return claims, true
}

// Authenticator resolves tokens of one kind to the identity behind them.
type Authenticator interface {
	// Authenticate resolves the token to an Identity. It reports false when
	// the token is not of a kind it handles, so that the next Authenticator
	// may be tried. An error means the token is of its kind but was rejected
	// (ErrInvalidToken) or could not be checked.
	Authenticate(ctx context.Context, rawToken string) (user.Identity, bool, error)
}

// Chain tries each Authenticator in order and answers with the first that
// recognizes the token.
type Chain []Authenticator

var _ Authenticator = Chain(nil)

// Authenticate implements Authenticator.
func (c Chain) Authenticate(
	ctx context.Context,
	rawToken string,
) (user.Identity, bool, error) {
	for _, authenticator := range c {
		id, ok, err := authenticator.Authenticate(ctx, rawToken)
		if err != nil || ok {
			return id, ok, err
		}
	}
	return nil, false, nil
}

// New returns the Chain the API server authenticates with: the admin
// Authenticator when an admin account is configured, the OpenID Connect
// Authenticator when an identity provider is, and Kubernetes last, which
// handles whatever the others did not recognize.
func New(ctx context.Context, cfg config.ServerConfig, kube client.Client) Chain {
	chain := Chain{}
	if cfg.AdminConfig != nil {
		chain = append(chain, NewAdmin(cfg.AdminConfig))
	}
	if cfg.OIDCConfig != nil {
		chain = append(chain, NewOIDC(ctx, cfg, kube))
	}
	return append(chain, NewKubernetes(kube))
}
