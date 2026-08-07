package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	authnv1 "k8s.io/api/authentication/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/user"
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
			name: "kubernetes service account",
			user: user.Info{
				KubernetesUserInfo: &authnv1.UserInfo{
					Username: "system:serviceaccount:kargo-demo:ci-bot",
				},
			},
			expected: kargoapi.EventActorKubernetesUserPrefix + "system:serviceaccount:kargo-demo:ci-bot",
		},
		{
			name: "sub",
			user: user.Info{
				Claims: map[string]any{
					"sub": "subject",
				},
			},
			expected: kargoapi.EventActorSubjectPrefix + "subject",
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
			name:     "oidc-username",
			user:     user.Info{Username: "oidc-username"},
			expected: formatOIDCUsername(user.Info{Username: "oidc-username"}),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := FormatEventUserActor(test.user)
			require.Equal(t, test.expected, result)
		})
	}
}
