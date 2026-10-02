package authn

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/server/user"
)

// countingAuthenticator accepts every token as the same identity, or answers
// with a fixed result, and counts how often it is asked.
type countingAuthenticator struct {
	id    user.Identity
	ok    bool
	err   error
	calls int
}

func (c *countingAuthenticator) Authenticate(
	context.Context,
	string,
) (user.Identity, bool, error) {
	c.calls++
	return c.id, c.ok, c.err
}

type fakeClock struct{ now time.Time }

func (f *fakeClock) Now() time.Time { return f.now }

func TestWithTokenCache(t *testing.T) {
	t.Parallel()
	next := &countingAuthenticator{}
	require.Same(t, next, WithTokenCache(next, 0))
	require.NotSame(t, next, WithTokenCache(next, time.Minute))
}

func TestTokenCache_Authenticate(t *testing.T) {
	t.Parallel()
	const ttl = 2 * time.Minute
	start := time.Now()
	expiringAt := func(t *testing.T, exp time.Time) string {
		raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
		}).SignedString([]byte("any key"))
		require.NoError(t, err)
		return raw
	}
	kubeUser := user.KubernetesUser{}
	testCases := []struct {
		name   string
		next   *countingAuthenticator
		token  string
		assert func(*testing.T, *countingAuthenticator, *tokenCache, *fakeClock, string)
	}{
		{
			name:  "accepted token is remembered",
			next:  &countingAuthenticator{id: kubeUser, ok: true},
			token: tokenFrom(t, "kubernetes"),
			assert: func(t *testing.T, next *countingAuthenticator, c *tokenCache, _ *fakeClock, token string) {
				for range 3 {
					id, ok, err := c.Authenticate(t.Context(), token)
					require.NoError(t, err)
					require.True(t, ok)
					require.Equal(t, kubeUser, id)
				}
				require.Equal(t, 1, next.calls)
			},
		},
		{
			name:  "another token is verified on its own",
			next:  &countingAuthenticator{id: kubeUser, ok: true},
			token: tokenFrom(t, "kubernetes"),
			assert: func(t *testing.T, next *countingAuthenticator, c *tokenCache, _ *fakeClock, token string) {
				_, _, _ = c.Authenticate(t.Context(), token)
				_, _, _ = c.Authenticate(t.Context(), tokenFrom(t, "other"))
				require.Equal(t, 2, next.calls)
			},
		},
		{
			name:  "entry expires after the TTL",
			next:  &countingAuthenticator{id: kubeUser, ok: true},
			token: tokenFrom(t, "kubernetes"),
			assert: func(t *testing.T, next *countingAuthenticator, c *tokenCache, clock *fakeClock, token string) {
				_, _, _ = c.Authenticate(t.Context(), token)
				clock.now = start.Add(ttl + time.Second)
				_, _, _ = c.Authenticate(t.Context(), token)
				require.Equal(t, 2, next.calls)
			},
		},
		{
			name:  "entry does not outlive the token",
			next:  &countingAuthenticator{id: kubeUser, ok: true},
			token: expiringAt(t, start.Add(30*time.Second)),
			assert: func(t *testing.T, next *countingAuthenticator, c *tokenCache, clock *fakeClock, token string) {
				_, _, _ = c.Authenticate(t.Context(), token)
				clock.now = start.Add(20 * time.Second)
				_, _, _ = c.Authenticate(t.Context(), token)
				require.Equal(t, 1, next.calls)
				clock.now = start.Add(31 * time.Second)
				_, _, _ = c.Authenticate(t.Context(), token)
				require.Equal(t, 2, next.calls)
			},
		},
		{
			name:  "already expired token is not remembered",
			next:  &countingAuthenticator{id: kubeUser, ok: true},
			token: expiringAt(t, start.Add(-time.Second)),
			assert: func(t *testing.T, next *countingAuthenticator, c *tokenCache, _ *fakeClock, token string) {
				_, _, _ = c.Authenticate(t.Context(), token)
				_, _, _ = c.Authenticate(t.Context(), token)
				require.Equal(t, 2, next.calls)
			},
		},
		{
			name:  "rejection is not remembered",
			next:  &countingAuthenticator{ok: true, err: ErrInvalidToken},
			token: tokenFrom(t, "kubernetes"),
			assert: func(t *testing.T, next *countingAuthenticator, c *tokenCache, _ *fakeClock, token string) {
				for range 2 {
					_, ok, err := c.Authenticate(t.Context(), token)
					require.True(t, ok)
					require.ErrorIs(t, err, ErrInvalidToken)
				}
				require.Equal(t, 2, next.calls)
			},
		},
		{
			name:  "failure to check is not remembered",
			next:  &countingAuthenticator{ok: true, err: errors.New("connection refused")},
			token: tokenFrom(t, "kubernetes"),
			assert: func(t *testing.T, next *countingAuthenticator, c *tokenCache, _ *fakeClock, token string) {
				_, _, _ = c.Authenticate(t.Context(), token)
				_, _, err := c.Authenticate(t.Context(), token)
				require.ErrorContains(t, err, "connection refused")
				require.Equal(t, 2, next.calls)
			},
		},
		{
			name:  "unrecognized token is not remembered",
			next:  &countingAuthenticator{},
			token: tokenFrom(t, "nobody"),
			assert: func(t *testing.T, next *countingAuthenticator, c *tokenCache, _ *fakeClock, token string) {
				for range 2 {
					id, ok, err := c.Authenticate(t.Context(), token)
					require.NoError(t, err)
					require.False(t, ok)
					require.Nil(t, id)
				}
				require.Equal(t, 2, next.calls)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			clock := &fakeClock{now: start}
			c := newTokenCache(testCase.next, ttl, clock)
			testCase.assert(t, testCase.next, c, clock, testCase.token)
		})
	}
}
