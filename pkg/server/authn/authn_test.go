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
	_ string,
	hint Hint,
) (user.Identity, bool, error) {
	if hint.Issuer != f.issuer {
		return nil, false, nil
	}
	f.calls++
	return f.id, true, f.err
}

func TestHintFrom(t *testing.T) {
	t.Parallel()
	_, ok := HintFrom("not a jwt")
	require.False(t, ok)
	// Expired and unsigned: a hint reads the claims without judging them.
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "someone",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
	}).SignedString([]byte("any key"))
	require.NoError(t, err)
	hint, ok := HintFrom(raw)
	require.True(t, ok)
	require.Equal(t, Hint{Issuer: "someone"}, hint)
}

func TestChain_Authenticate(t *testing.T) {
	t.Parallel()
	first := &fakeAuthenticator{issuer: "first", id: user.Admin{}}
	second := &fakeAuthenticator{issuer: "second", err: errors.New("broken")}
	chain := Chain{first, second}

	id, ok, err := chain.Authenticate(t.Context(), "", Hint{Issuer: "first"})
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, user.Admin{}, id)
	require.Equal(t, 1, first.calls)
	require.Zero(t, second.calls)

	// An error from the Authenticator that recognized the token ends the chain.
	_, ok, err = chain.Authenticate(t.Context(), "", Hint{Issuer: "second"})
	require.True(t, ok)
	require.ErrorContains(t, err, "broken")

	// A token nobody recognizes.
	id, ok, err = chain.Authenticate(t.Context(), "", Hint{Issuer: "third"})
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
