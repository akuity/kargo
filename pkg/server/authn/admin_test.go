package authn

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/user"
)

func TestAdmin_Authenticate(t *testing.T) {
	t.Parallel()
	const issuer = "kargo"
	signingKey := []byte("iwishtowashmyirishwristwatch")
	signed := func(t *testing.T, key []byte, expiresIn time.Duration) string {
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
			Issuer:    issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		}).SignedString(key)
		require.NoError(t, err)
		return token
	}
	authenticator := NewAdmin(&config.AdminConfig{
		TokenIssuer:     issuer,
		TokenSigningKey: signingKey,
	})
	testCases := []struct {
		name   string
		token  string
		hint   Hint
		assert func(*testing.T, user.Identity, bool, error)
	}{
		{
			name: "another issuer is not this kind of token",
			hint: Hint{Issuer: "someone-else"},
			assert: func(t *testing.T, id user.Identity, ok bool, err error) {
				require.NoError(t, err)
				require.False(t, ok)
				require.Nil(t, id)
			},
		},
		{
			name:  "signed with the wrong key",
			token: signed(t, []byte("wrong key"), time.Hour),
			hint:  Hint{Issuer: issuer},
			assert: func(t *testing.T, _ user.Identity, ok bool, err error) {
				require.True(t, ok)
				require.ErrorIs(t, err, ErrInvalidToken)
			},
		},
		{
			name:  "expired",
			token: signed(t, signingKey, -time.Hour),
			hint:  Hint{Issuer: issuer},
			assert: func(t *testing.T, _ user.Identity, ok bool, err error) {
				require.True(t, ok)
				require.ErrorIs(t, err, ErrInvalidToken)
			},
		},
		{
			name:  "not a JWT",
			token: "some-token",
			hint:  Hint{Issuer: issuer},
			assert: func(t *testing.T, _ user.Identity, ok bool, err error) {
				require.True(t, ok)
				require.ErrorIs(t, err, ErrInvalidToken)
			},
		},
		{
			name:  "valid",
			token: signed(t, signingKey, time.Hour),
			hint:  Hint{Issuer: issuer},
			assert: func(t *testing.T, id user.Identity, ok bool, err error) {
				require.NoError(t, err)
				require.True(t, ok)
				require.Equal(t, user.Admin{}, id)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			id, ok, err := authenticator.Authenticate(t.Context(), testCase.token, testCase.hint)
			testCase.assert(t, id, ok, err)
		})
	}
}
