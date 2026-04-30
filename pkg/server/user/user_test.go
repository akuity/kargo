package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContextWithUserInfo(t *testing.T) {
	testUserInfo := Info{
		Claims: map[string]any{"sub": "hansolo"},
	}
	ctx := ContextWithInfo(t.Context(), testUserInfo)
	require.Equal(t, testUserInfo, ctx.Value(userInfoKey{}))
}

func TestUserInfoFromContext(t *testing.T) {
	_, ok := InfoFromContext(t.Context())
	require.False(t, ok)
	testUserInfo := Info{
		Claims: map[string]any{"sub": "hansolo"},
	}
	ctx := context.WithValue(t.Context(), userInfoKey{}, testUserInfo)
	u, ok := InfoFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, testUserInfo, u)
}
