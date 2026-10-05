package authn

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/server/config"
	libOIDC "github.com/akuity/kargo/pkg/server/oidc"
	"github.com/akuity/kargo/pkg/server/user"
)

// fakeAuthenticator recognizes tokens with a given issuer and answers with a
// fixed identity or error.
type fakeAuthenticator struct {
	issuer string
	id     user.Identity
	err    error
	calls  int
}

func (f *fakeAuthenticator) Authenticate(
	_ context.Context,
	rawToken string,
) (user.Identity, bool, error) {
	if issuer, _ := unverifiedIssuer(rawToken); issuer != f.issuer {
		return nil, false, nil
	}
	f.calls++
	return f.id, true, f.err
}

// tokenFrom returns a JWT claiming the issuer, signed with a key nobody trusts.
func tokenFrom(t *testing.T, issuer string) string {
	t.Helper()
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer: issuer,
	}).SignedString([]byte("any key"))
	require.NoError(t, err)
	return raw
}

func TestUnverifiedIssuer(t *testing.T) {
	t.Parallel()
	_, ok := unverifiedIssuer("not a jwt")
	require.False(t, ok)
	// Expired and signed with an unknown key: the issuer is read without
	// judging the token.
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "someone",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
	}).SignedString([]byte("any key"))
	require.NoError(t, err)
	issuer, ok := unverifiedIssuer(raw)
	require.True(t, ok)
	require.Equal(t, "someone", issuer)
}

func TestChain_Authenticate(t *testing.T) {
	t.Parallel()
	first := &fakeAuthenticator{issuer: "first", id: user.Admin{}}
	second := &fakeAuthenticator{issuer: "second", err: errors.New("broken")}
	chain := Chain{first, second}

	id, ok, err := chain.Authenticate(t.Context(), tokenFrom(t, "first"))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, user.Admin{}, id)
	require.Equal(t, 1, first.calls)
	require.Zero(t, second.calls)

	// An error from the Authenticator that recognized the token ends the chain.
	_, ok, err = chain.Authenticate(t.Context(), tokenFrom(t, "second"))
	require.True(t, ok)
	require.ErrorContains(t, err, "broken")

	// A token nobody recognizes.
	id, ok, err = chain.Authenticate(t.Context(), tokenFrom(t, "third"))
	require.NoError(t, err)
	require.False(t, ok)
	require.Nil(t, id)
}

func TestNew(t *testing.T) {
	t.Parallel()
	// Kubernetes is always last; the others are present only when configured.
	require.Len(t, New(t.Context(), config.ServerConfig{}, nil), 1)
	require.Len(t, New(t.Context(), config.ServerConfig{
		AdminConfig: &config.AdminConfig{TokenIssuer: "kargo"},
	}, nil), 2)
	require.Len(t, New(t.Context(), config.ServerConfig{
		AdminConfig: &config.AdminConfig{TokenIssuer: "kargo"},
		OIDCConfig:  &libOIDC.Config{IssuerURL: "http://127.0.0.1:1"},
	}, nil), 3)
}
