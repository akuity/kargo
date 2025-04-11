package api

import (
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/server/user"
)

func TestFormatEventUserActor(t *testing.T) {
	for _, test := range []struct {
		name     string
		user     user.Info
		expected string
	}{
		{
			name:     "admin",
			user:     user.Info{IsAdmin: true},
			expected: kargoapi.EventActorAdmin,
		},
		{
			name: "email",
			user: user.Info{
				Claims: map[string]any{
					"email": "email@inbox.com",
				},
			},
			expected: kargoapi.EventActorEmailPrefix + "email@inbox.com",
		},
		{
			name: "sub",
			user: user.Info{
				Claims: map[string]any{
					"email": "email@inbox.com",
				},
			},
			expected: kargoapi.EventActorEmailPrefix + "email@inbox.com",
		},
		{
			name: "oidc-username",
			user: user.Info{
				Username: "oidc-username",
			},
			expected: kargoapi.EventActorOIDCClaimPrefix + "oidc-username",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := FormatEventUserActor(test.user)
			require.Equal(t, test.expected, result)
		})
	}
}
