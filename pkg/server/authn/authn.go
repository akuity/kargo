// Package authn resolves the bearer tokens presented to the Kargo API server
// to the identities behind them.
//
// A token may have been issued by the Kargo API server itself (its admin
// user), by Kargo's OpenID Connect identity provider, or by Kubernetes (a
// ServiceAccount token, or the cluster's own identity provider). Each kind
// has an Authenticator of its own, and a Chain tries them in order. An
// Authenticator recognizes its kind of token by the issuer the token claims,
// which is read before anything is verified and trusted only as a hint about
// how to verify it.
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

// Hint is what a token says about itself before it is verified: the claims
// that pick which Authenticator should verify it. Nothing in a Hint is
// trusted.
type Hint struct {
	// Issuer is the token's unverified "iss" claim.
	Issuer string
}

// HintFrom reads a Hint from a raw JWT without verifying it. It returns false
// if the token is not a JWT at all.
func HintFrom(rawToken string) (Hint, bool) {
	claims := jwt.RegisteredClaims{}
	if _, _, err := jwt.NewParser(jwt.WithoutClaimsValidation()).
		ParseUnverified(rawToken, &claims); err != nil {
		return Hint{}, false
	}
	return Hint{Issuer: claims.Issuer}, true
}

// Authenticator resolves tokens of one kind to the identity behind them.
type Authenticator interface {
	// Authenticate resolves the token to an Identity. It reports false when
	// the token is not of a kind it handles, so that the next Authenticator
	// may be tried. An error means the token is of its kind but was rejected
	// (ErrInvalidToken) or could not be checked.
	Authenticate(ctx context.Context, rawToken string, hint Hint) (user.Identity, bool, error)
}

// Chain tries each Authenticator in order and answers with the first that
// recognizes the token.
type Chain []Authenticator

var _ Authenticator = Chain(nil)

// Authenticate implements Authenticator.
func (c Chain) Authenticate(
	ctx context.Context,
	rawToken string,
	hint Hint,
) (user.Identity, bool, error) {
	for _, authenticator := range c {
		id, ok, err := authenticator.Authenticate(ctx, rawToken, hint)
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
