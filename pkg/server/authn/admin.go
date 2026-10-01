package authn

import (
	"context"

	"github.com/golang-jwt/jwt/v5"

	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/user"
)

// admin recognizes the tokens the Kargo API server issues to its own admin
// user when they log in with the admin password.
type admin struct {
	cfg *config.AdminConfig
}

var _ Authenticator = (*admin)(nil)

// NewAdmin returns an Authenticator for tokens the API server issued itself.
func NewAdmin(cfg *config.AdminConfig) Authenticator {
	return &admin{cfg: cfg}
}

// Authenticate implements Authenticator. A token claiming the server's own
// issuer is of this kind; it is valid if the server's signing key signed it
// and it has not expired.
func (a *admin) Authenticate(
	_ context.Context,
	rawToken string,
	hint Hint,
) (user.Identity, bool, error) {
	if hint.Issuer != a.cfg.TokenIssuer {
		return nil, false, nil
	}
	if _, err := jwt.NewParser().Parse(
		rawToken,
		func(*jwt.Token) (any, error) {
			return a.cfg.TokenSigningKey, nil
		},
	); err != nil {
		return nil, true, ErrInvalidToken
	}
	return user.Admin{}, true, nil
}
