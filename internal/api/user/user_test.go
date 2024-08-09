package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContextWithUserInfo(t *testing.T) {
	testUserInfo := Info{
		Claims: map[string]any{
			"subs": "hansolo",
		},
	}
	ctx := ContextWithInfo(context.Background(), testUserInfo)
	require.Equal(t, testUserInfo, ctx.Value(userInfoKey{}))
}

func TestUserInfoFromContext(t *testing.T) {
	_, ok := InfoFromContext(context.Background())
	require.False(t, ok)
	testUserInfo := Info{
		Claims: map[string]any{
			"subs": "hansolo",
		},
	}
	ctx := context.WithValue(context.Background(), userInfoKey{}, testUserInfo)
	u, ok := InfoFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, testUserInfo, u)
}
