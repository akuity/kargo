package rbac

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// fakeAuthorizer is a test Authorizer. When allow is nil, everything is
// permitted; otherwise allow decides. Every call is recorded.
type fakeAuthorizer struct {
	allow func(verb string, gvr schema.GroupVersionResource, subresource string) bool
	calls []string
}

func (f *fakeAuthorizer) Authorize(
	_ context.Context,
	verb string,
	gvr schema.GroupVersionResource,
	subresource string,
	key client.ObjectKey,
) error {
	res := gvr.Resource
	if subresource != "" {
		res += "/" + subresource
	}
	f.calls = append(f.calls, verb+" "+gvr.Group+"/"+res)
	if f.allow == nil || f.allow(verb, gvr, subresource) {
		return nil
	}
	return apierrors.NewForbidden(gvr.GroupResource(), key.Name, errors.New("not allowed"))
}

// authorizingFakeClient is both a controller-runtime client and an Authorizer,
// so NewKubernetesRolesDatabase wires the escalation check to it.
type authorizingFakeClient struct {
	client.Client
	authz *fakeAuthorizer
}

func (a *authorizingFakeClient) Authorize(
	ctx context.Context,
	verb string,
	gvr schema.GroupVersionResource,
	subresource string,
	key client.ObjectKey,
) error {
	return a.authz.Authorize(ctx, verb, gvr, subresource, key)
}

func TestSplitResourceType(t *testing.T) {
	r, s := splitResourceType("secrets")
	require.Equal(t, "secrets", r)
	require.Empty(t, s)

	r, s = splitResourceType("freights/status")
	require.Equal(t, "freights", r)
	require.Equal(t, "status", s)
}

func TestVerifyRulesNotEscalating(t *testing.T) {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{kargoapi.GroupVersion.Group},
			Resources: []string{"stages"},
			Verbs:     []string{"get", "promote"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"get"},
		},
	}

	t.Run("permitted when user holds every rule", func(t *testing.T) {
		authz := &fakeAuthorizer{}
		_, err := verifyRulesNotEscalating(t.Context(), authz, testProject, rules)
		require.NoError(t, err)
		require.ElementsMatch(
			t,
			[]string{
				"get " + kargoapi.GroupVersion.Group + "/stages",
				"promote " + kargoapi.GroupVersion.Group + "/stages",
				"get /secrets",
			},
			authz.calls,
		)
	})

	t.Run("forbidden when user lacks one rule", func(t *testing.T) {
		authz := &fakeAuthorizer{
			allow: func(_ string, gvr schema.GroupVersionResource, _ string) bool {
				return gvr.Resource != "secrets" // user has no secrets access
			},
		}
		_, err := verifyRulesNotEscalating(t.Context(), authz, testProject, rules)
		require.Error(t, err)
		require.True(t, apierrors.IsForbidden(err))
		require.ErrorContains(t, err, `resource "secrets"`)
	})
}

func TestNewKubernetesRolesDatabaseWiresAuthorizer(t *testing.T) {
	plain := fake.NewClientBuilder().WithScheme(scheme).Build()

	db, ok := NewKubernetesRolesDatabase(plain, RolesDatabaseConfig{}).(*rolesDatabase)
	require.True(t, ok)
	require.Nil(t, db.authorizer, "a plain client must leave the authorizer nil")

	authorizing := &authorizingFakeClient{Client: plain, authz: &fakeAuthorizer{}}
	db, ok = NewKubernetesRolesDatabase(authorizing, RolesDatabaseConfig{}).(*rolesDatabase)
	require.True(t, ok)
	require.NotNil(t, db.authorizer, "a client implementing Authorizer must populate the authorizer")
}

func TestVerifyBindingNotEscalating(t *testing.T) {
	roles := []rbacv1.Role{{
		Rules: []rbacv1.PolicyRule{
			{APIGroups: []string{""}, Resources: []string{"configmaps", "secrets"}, Verbs: []string{"*"}},
		},
	}}

	t.Run("nil authorizer disables the check", func(t *testing.T) {
		require.NoError(t, verifyBindingNotEscalating(t.Context(), nil, testProject, roles))
	})

	t.Run("no bound roles means nothing to escalate", func(t *testing.T) {
		require.NoError(t, verifyBindingNotEscalating(t.Context(), &fakeAuthorizer{}, testProject, nil))
	})

	t.Run("permitted when requester holds the role's permissions", func(t *testing.T) {
		require.NoError(t, verifyBindingNotEscalating(t.Context(), &fakeAuthorizer{}, testProject, roles))
	})

	t.Run("forbidden when requester lacks a permission in the bound role", func(t *testing.T) {
		authz := &fakeAuthorizer{
			allow: func(_ string, gvr schema.GroupVersionResource, _ string) bool {
				return gvr.Resource != "secrets"
			},
		}
		err := verifyBindingNotEscalating(t.Context(), authz, testProject, roles)
		require.Error(t, err)
		require.True(t, apierrors.IsForbidden(err))
		require.ErrorContains(t, err, `resource "secrets"`)
	})
}

// TestGrantRoleToUsersBlocksEscalation is the end-to-end guard: a user who
// cannot read secrets must not be able to bind an identity onto a Role that can.
func TestGrantRoleToUsersBlocksEscalation(t *testing.T) {
	secretsRole := []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"secrets"},
		Verbs:     []string{"*"},
	}}
	claims := []rbacapi.Claim{{Name: "email", Values: []string{"attacker@evil.example"}}}

	newDB := func(authz *fakeAuthorizer) (RolesDatabase, client.Client) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(nil),
			managedRole(secretsRole),
			managedRoleBinding(),
		).Build()
		return NewKubernetesRolesDatabase(
			&authorizingFakeClient{Client: c, authz: authz}, RolesDatabaseConfig{},
		), c
	}

	t.Run("blocked when requester lacks the role's permissions", func(t *testing.T) {
		db, c := newDB(&fakeAuthorizer{
			allow: func(_ string, gvr schema.GroupVersionResource, _ string) bool {
				return gvr.Resource != "secrets"
			},
		})
		_, err := db.GrantRoleToUsers(t.Context(), testProject, testKargoRoleName, claims)
		require.Error(t, err)
		require.True(t, apierrors.IsForbidden(err))

		// The claim must not have been written to the ServiceAccount.
		sa := &corev1.ServiceAccount{}
		require.NoError(t, c.Get(t.Context(), objKey, sa))
		got, err := rbacapi.OIDCClaimsFromAnnotationValues(sa.Annotations)
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("allowed when requester holds the role's permissions", func(t *testing.T) {
		db, c := newDB(&fakeAuthorizer{}) // permits everything
		_, err := db.GrantRoleToUsers(t.Context(), testProject, testKargoRoleName, claims)
		require.NoError(t, err)

		sa := &corev1.ServiceAccount{}
		require.NoError(t, c.Get(t.Context(), objKey, sa))
		got, err := rbacapi.OIDCClaimsFromAnnotationValues(sa.Annotations)
		require.NoError(t, err)
		require.Equal(t, []string{"attacker@evil.example"}, got["email"])
	})
}

// TestGrantPermissionsToRoleBlocksEscalation is the end-to-end guard: a user who
// cannot read secrets must not be able to grant secrets access via a Role edit.
func TestGrantPermissionsToRoleBlocksEscalation(t *testing.T) {
	newDB := func(authz *fakeAuthorizer) (RolesDatabase, client.Client) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(
			&authorizingFakeClient{Client: c, authz: authz}, RolesDatabaseConfig{},
		)
		// A fake client has no discovery, so stub group resolution.
		rdb, ok := db.(*rolesDatabase)
		require.True(t, ok)
		rdb.resolveGroup = fakeGroupResolver
		return db, c
	}

	t.Run("blocked when requester lacks secrets", func(t *testing.T) {
		db, c := newDB(&fakeAuthorizer{
			allow: func(_ string, gvr schema.GroupVersionResource, _ string) bool {
				return gvr.Resource != "secrets"
			},
		})
		_, err := db.GrantPermissionsToRole(
			t.Context(), testProject, testKargoRoleName,
			&rbacapi.ResourceDetails{ResourceType: "secrets", Verbs: []string{"get", "list"}},
		)
		require.Error(t, err)
		require.True(t, apierrors.IsForbidden(err))
		require.ErrorContains(t, err, "may not grant permissions it does not hold")

		// The Role must not have been created as a side effect.
		err = c.Get(t.Context(), objKey, &rbacv1.Role{})
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("allowed when requester holds the permission", func(t *testing.T) {
		db, c := newDB(&fakeAuthorizer{}) // permits everything
		_, err := db.GrantPermissionsToRole(
			t.Context(), testProject, testKargoRoleName,
			&rbacapi.ResourceDetails{ResourceType: "secrets", Verbs: []string{"get", "list"}},
		)
		require.NoError(t, err)

		role := &rbacv1.Role{}
		require.NoError(t, c.Get(t.Context(), objKey, role))
		require.Equal(
			t,
			[]rbacv1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list"},
			}},
			role.Rules,
		)
	})
}
