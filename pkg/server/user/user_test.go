package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	authnv1 "k8s.io/api/authentication/v1"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestAdmin(t *testing.T) {
	t.Parallel()
	require.True(t, Admin{}.IsAdmin())
	require.Empty(t, Admin{}.Subjects(authv1.ResourceAttributes{Namespace: "demo"}))
	require.Equal(t, kargoapi.EventActorAdmin, Admin{}.Actor())
	require.False(t, Admin{}.MemberOf("demo"))
}

func TestOIDCUser_Subjects(t *testing.T) {
	t.Parallel()
	sa := func(namespace, name string) types.NamespacedName {
		return types.NamespacedName{Namespace: namespace, Name: name}
	}
	u := OIDCUser{
		ServiceAccountsByNamespace: map[string]map[types.NamespacedName]struct{}{
			"demo":   {sa("demo", "viewer"): {}},
			"global": {sa("global", "ops"): {}},
			"other":  {sa("other", "admin"): {}},
		},
		GlobalServiceAccountNamespaces: []string{"global"},
	}
	usernames := func(subjects []Subject) []string {
		out := make([]string, len(subjects))
		for i, s := range subjects {
			out[i] = s.Username
		}
		return out
	}
	testCases := []struct {
		name   string
		ra     authv1.ResourceAttributes
		expect []string
	}{
		{
			name: "namespaced resource: its namespace first, then the global ones",
			ra:   authv1.ResourceAttributes{Namespace: "demo", Resource: "stages"},
			expect: []string{
				"system:serviceaccount:demo:viewer",
				"system:serviceaccount:global:ops",
			},
		},
		{
			name: "a Project: its own namespace counts as its namespace",
			ra: authv1.ResourceAttributes{
				Group:    kargoapi.GroupVersion.Group,
				Resource: "projects",
				Name:     "demo",
			},
			expect: []string{
				"system:serviceaccount:demo:viewer",
				"system:serviceaccount:global:ops",
			},
		},
		{
			name:   "another cluster-scoped resource: only the global namespaces",
			ra:     authv1.ResourceAttributes{Resource: "clusterconfigs", Name: "cluster"},
			expect: []string{"system:serviceaccount:global:ops"},
		},
		{
			name:   "a namespace the user is not mapped in",
			ra:     authv1.ResourceAttributes{Namespace: "elsewhere", Resource: "stages"},
			expect: []string{"system:serviceaccount:global:ops"},
		},
		{
			name:   "no ServiceAccounts at all",
			ra:     authv1.ResourceAttributes{Namespace: "demo"},
			expect: nil,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			subject := u
			if testCase.expect == nil {
				subject = OIDCUser{}
				require.Empty(t, subject.Subjects(testCase.ra))
				return
			}
			require.Equal(t, testCase.expect, usernames(subject.Subjects(testCase.ra)))
		})
	}
	require.False(t, u.IsAdmin())
	require.True(t, u.MemberOf("demo"))
	require.False(t, u.MemberOf("elsewhere"))
}

func TestOIDCUser_Actor(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		user   OIDCUser
		expect string
	}{
		{
			name:   "username claim",
			user:   OIDCUser{UsernameClaim: "preferred_username", Username: "tony"},
			expect: "preferred_username:tony",
		},
		{
			name:   "email",
			user:   OIDCUser{Claims: map[string]any{"email": "tony@stark.io", "sub": "ironman"}},
			expect: kargoapi.EventActorEmailPrefix + "tony@stark.io",
		},
		{
			name:   "subject",
			user:   OIDCUser{Claims: map[string]any{"sub": "ironman"}},
			expect: kargoapi.EventActorSubjectPrefix + "ironman",
		},
		{
			name:   "nothing usable",
			user:   OIDCUser{Claims: map[string]any{"email": 42}},
			expect: kargoapi.EventActorUnknown,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.expect, testCase.user.Actor())
		})
	}
}

func TestKubernetesUser(t *testing.T) {
	t.Parallel()
	u := KubernetesUser{UserInfo: authnv1.UserInfo{
		Username: "system:serviceaccount:kargo-demo:ci-bot",
		UID:      "abc-123",
		Groups:   []string{"system:serviceaccounts", "system:authenticated"},
		Extra:    map[string]authnv1.ExtraValue{"authentication.kubernetes.io/pod-name": {"fake-pod"}},
	}}
	require.False(t, u.IsAdmin())
	require.Equal(t, kargoapi.EventActorKubernetesUserPrefix+u.Username, u.Actor())
	require.False(t, u.MemberOf("kargo-demo"))
	subjects := u.Subjects(authv1.ResourceAttributes{Namespace: "demo"})
	require.Equal(t, []Subject{{
		Username: u.Username,
		UID:      u.UID,
		Groups:   u.Groups,
		Extra:    map[string]authv1.ExtraValue{"authentication.kubernetes.io/pod-name": {"fake-pod"}},
	}}, subjects)
	// Extras are carried across only when present, not as an empty map.
	require.Nil(t, KubernetesUser{}.Subjects(authv1.ResourceAttributes{})[0].Extra)
}

func TestContext(t *testing.T) {
	t.Parallel()
	_, ok := IdentityFromContext(context.Background())
	require.False(t, ok)
	ctx := ContextWithIdentity(context.Background(), Admin{})
	id, ok := IdentityFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, Admin{}, id)
	// A nil Identity is no identity.
	_, ok = IdentityFromContext(ContextWithIdentity(context.Background(), nil))
	require.False(t, ok)
}
