package rbac

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

const (
	testProject       = "fake-project"
	testKargoRoleName = "fake-kargo-role"
)

var (
	scheme *runtime.Scheme
	objKey = client.ObjectKey{Namespace: testProject, Name: testKargoRoleName}
)

func init() {
	scheme = runtime.NewScheme()
	err := corev1.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}
	err = rbacv1.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}
}

func Test_rolesDatabase_Create(t *testing.T) {
	t.Run("ServiceAccount already exists", func(t *testing.T) {
		testKargoRole := &rbacapi.Role{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testKargoRoleName,
			},
		}
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		role, err := db.Create(t.Context(), testKargoRole)
		require.True(t, apierrors.IsAlreadyExists(err))
		require.Nil(t, role)
	})

	t.Run("Role already exists", func(t *testing.T) {
		testKargoRole := &rbacapi.Role{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testKargoRoleName,
			},
		}
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainRole(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		role, err := db.Create(t.Context(), testKargoRole)
		require.True(t, apierrors.IsAlreadyExists(err))
		require.Nil(t, role)
	})

	t.Run("RoleBinding already exists", func(t *testing.T) {
		testKargoRole := &rbacapi.Role{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testKargoRoleName,
			},
		}
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainRoleBinding(),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		role, err := db.Create(t.Context(), testKargoRole)
		require.True(t, apierrors.IsAlreadyExists(err))
		require.Nil(t, role)
	})

	t.Run("Success", func(t *testing.T) {
		testKargoRole := &rbacapi.Role{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testKargoRoleName,
				Annotations: map[string]string{
					kargoapi.AnnotationKeyDescription: "fake-description",
				},
			},
			Claims: []rbacapi.Claim{
				{
					Name:   "sub",
					Values: []string{"foo-sub", "bar-sub"},
				},
				{
					Name:   "email",
					Values: []string{"foo-email", "bar-email"},
				}, {
					Name:   "groups",
					Values: []string{"foo-group", "bar-group"},
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"stages", "promotions"},
					Verbs:     []string{"get", "list"},
				},
			},
		}
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		role, err := db.Create(t.Context(), testKargoRole)
		require.NoError(t, err)
		require.NotNil(t, role)

		sa := &corev1.ServiceAccount{}
		err = c.Get(
			t.Context(),
			client.ObjectKey{Namespace: testKargoRole.Namespace, Name: testKargoRole.Name},
			sa,
		)
		require.NoError(t, err)
		expected := `{"email":["bar-email","foo-email"],"groups":["bar-group","foo-group"],"sub":["bar-sub","foo-sub"]}`
		require.Equal(
			t,
			map[string]string{
				rbacapi.AnnotationKeyManaged:      rbacapi.AnnotationValueTrue,
				rbacapi.AnnotationKeyOIDCClaims:   expected,
				kargoapi.AnnotationKeyDescription: "fake-description",
			},
			sa.Annotations,
		)

		roleBinding := &rbacv1.RoleBinding{}
		err = c.Get(
			t.Context(),
			client.ObjectKey{Namespace: testKargoRole.Namespace, Name: testKargoRole.Name},
			roleBinding,
		)
		require.NoError(t, err)
		require.Equal(
			t,
			map[string]string{
				rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
			},
			roleBinding.Annotations,
		)
		require.Equal(
			t,
			[]rbacv1.Subject{{
				Kind:      rbacv1.ServiceAccountKind,
				Namespace: testKargoRole.Namespace,
				Name:      testKargoRole.Name,
			}},
			roleBinding.Subjects,
		)
		require.Equal(
			t,
			rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     testKargoRole.Name,
			},
			roleBinding.RoleRef,
		)
	})
}

func Test_rolesDatabase_Delete(t *testing.T) {
	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		err := db.Delete(t.Context(), testProject, testKargoRoleName)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("resources aren't manageable", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		err := db.Delete(t.Context(), testProject, testKargoRoleName)
		require.True(t, apierrors.IsBadRequest(err))
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(nil),
			managedRole(nil),
			managedRoleBinding(),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		err := db.Delete(t.Context(), testProject, testKargoRoleName)
		require.NoError(t, err)
		role := &rbacv1.Role{}
		err = c.Get(t.Context(), objKey, role)
		require.True(t, apierrors.IsNotFound(err))
		roleBinding := &rbacv1.RoleBinding{}
		err = c.Get(t.Context(), objKey, roleBinding)
		require.True(t, apierrors.IsNotFound(err))
		sa := &corev1.ServiceAccount{}
		err = c.Get(t.Context(), objKey, sa)
		require.True(t, apierrors.IsNotFound(err))
	})
}

func Test_rolesDatabase_Get(t *testing.T) {
	t.Run("ServiceAccount does not exist", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		kargoRole, err := db.Get(t.Context(), false, testProject, testKargoRoleName)
		require.True(t, apierrors.IsNotFound(err))
		require.Nil(t, kargoRole)
	})

	t.Run("success with non-kargo-managed role", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(map[string]string{
				rbacapi.AnnotationKeyOIDCClaim("sub"):    "foo-sub,bar-sub",
				rbacapi.AnnotationKeyOIDCClaim("email"):  "foo-email,bar-email",
				rbacapi.AnnotationKeyOIDCClaim("groups"): "foo-group,bar-group",
			}),
			plainRole([]rbacv1.PolicyRule{
				{ // This rule has groups and types that we don't recognize. Let's
					// make sure we don't choke on them. This could happen with roles
					// that aren't Kargo-managed.
					APIGroups: []string{"fake-group-1", "fake-group-2"},
					Resources: []string{"fake-type-1", "fake-type-2"},
					Verbs:     []string{"get", "list"},
				},
				{
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"stages", "promotions"},
					Verbs:     []string{"list", "get"},
				},
			}),
			plainRoleBinding(),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		kargoRole, err := db.Get(t.Context(), false, testProject, testKargoRoleName)
		require.NoError(t, err)
		// Do not factor creation timestamp into the comparison
		now := metav1.NewTime(time.Now())
		kargoRole.CreationTimestamp = now
		require.Equal(
			t,
			&rbacapi.Role{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         testProject,
					Name:              testKargoRoleName,
					CreationTimestamp: now,
				},
				KargoManaged: false,
				Claims: []rbacapi.Claim{
					{
						Name:   "email",
						Values: []string{"bar-email", "foo-email"},
					},
					{
						Name:   "groups",
						Values: []string{"bar-group", "foo-group"},
					},
					{
						Name:   "sub",
						Values: []string{"bar-sub", "foo-sub"},
					},
				},
				// There should have been no attempt to normalize these rules
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{"fake-group-1", "fake-group-2"},
						Resources: []string{"fake-type-1", "fake-type-2"},
						Verbs:     []string{"get", "list"},
					},
					{
						APIGroups: []string{kargoapi.GroupVersion.Group},
						Resources: []string{"stages", "promotions"},
						Verbs:     []string{"list", "get"},
					},
				},
			},
			kargoRole,
		)
	})

	t.Run("success with non-kargo-managed role", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(map[string]string{
				rbacapi.AnnotationKeyOIDCClaim("sub"):    "foo-sub,bar-sub",
				rbacapi.AnnotationKeyOIDCClaim("email"):  "foo-email,bar-email",
				rbacapi.AnnotationKeyOIDCClaim("groups"): "foo-group,bar-group",
			}),
			managedRole([]rbacv1.PolicyRule{{
				APIGroups: []string{kargoapi.GroupVersion.Group},
				Resources: []string{"stages", "promotions"},
				Verbs:     []string{"list", "get"},
			}}),
			managedRoleBinding(),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		kargoRole, err := db.Get(t.Context(), false, testProject, testKargoRoleName)
		require.NoError(t, err)
		// Do not factor creation timestamp into the comparison
		now := metav1.NewTime(time.Now())
		kargoRole.CreationTimestamp = now
		require.Equal(
			t,
			&rbacapi.Role{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         testProject,
					Name:              testKargoRoleName,
					CreationTimestamp: now,
				},
				KargoManaged: true,
				Claims: []rbacapi.Claim{
					{
						Name:   "email",
						Values: []string{"bar-email", "foo-email"},
					},
					{
						Name:   "groups",
						Values: []string{"bar-group", "foo-group"},
					},
					{
						Name:   "sub",
						Values: []string{"bar-sub", "foo-sub"},
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{kargoapi.GroupVersion.Group},
						Resources: []string{"promotions"},
						Verbs:     []string{"get", "list"},
					},
					{
						APIGroups: []string{kargoapi.GroupVersion.Group},
						Resources: []string{"stages"},
						Verbs:     []string{"get", "list"},
					},
				},
			},
			kargoRole,
		)
	})
}

func Test_rolesDatabase_GetAsResources(t *testing.T) {
	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		_, _, _, err := db.GetAsResources(t.Context(), false, testProject, testKargoRoleName)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("no RoleBindings found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		sa, roles, rbs, err := db.GetAsResources(t.Context(), false, testProject, testKargoRoleName)
		require.NoError(t, err)
		require.NotNil(t, sa)
		require.Nil(t, roles)
		require.Nil(t, rbs)
	})

	t.Run("Role not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(nil),
			plainRoleBinding(),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		_, _, _, err := db.GetAsResources(t.Context(), false, testProject, testKargoRoleName)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(nil),
			plainRole(nil),
			plainRoleBinding(),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		sa, roles, rbs, err := db.GetAsResources(t.Context(), false, testProject, testKargoRoleName)
		require.NoError(t, err)
		require.NotNil(t, sa)
		require.NotNil(t, roles)
		require.NotNil(t, rbs)
	})
}

func Test_rolesDatabase_GrantPermissionToRole(t *testing.T) {
	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		_, err := db.GrantPermissionsToRole(
			t.Context(),
			testProject,
			testKargoRoleName,
			&rbacapi.ResourceDetails{
				ResourceType: "fake-resource-type",
				Verbs:        []string{"get", "list"},
			},
		)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("resources aren't manageable", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		_, err := db.GrantPermissionsToRole(
			t.Context(),
			testProject,
			testKargoRoleName,
			&rbacapi.ResourceDetails{
				ResourceType: "fake-resource-type",
				Verbs:        []string{"get", "list"},
			},
		)
		require.True(t, apierrors.IsBadRequest(err))
	})

	t.Run("success with Role and RoleBinding creation", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		kargoRole, err := db.GrantPermissionsToRole(
			t.Context(),
			testProject,
			testKargoRoleName,
			&rbacapi.ResourceDetails{
				ResourceType: "stages",
				Verbs:        []string{"get", "list"},
			},
		)
		require.NoError(t, err)
		require.NotNil(t, kargoRole)
		rb := &rbacv1.RoleBinding{}
		err = c.Get(t.Context(), objKey, rb)
		require.NoError(t, err)
		require.True(t, isKargoManaged(rb))
		require.Equal(
			t,
			[]rbacv1.Subject{{
				Kind:      rbacv1.ServiceAccountKind,
				Namespace: testProject,
				Name:      testKargoRoleName,
			}},
			rb.Subjects,
		)
		require.Equal(
			t,
			rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     testKargoRoleName,
			},
			rb.RoleRef,
		)
		role := &rbacv1.Role{}
		err = c.Get(t.Context(), objKey, role)
		require.NoError(t, err)
		require.True(t, isKargoManaged(rb))
		require.Equal(
			t,
			[]rbacv1.PolicyRule{{
				APIGroups: []string{kargoapi.GroupVersion.Group},
				Resources: []string{"stages"},
				Verbs:     []string{"get", "list"},
			}},
			role.Rules,
		)
	})

	t.Run("success with amended Role", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(nil),
			managedRole([]rbacv1.PolicyRule{{
				APIGroups: []string{kargoapi.GroupVersion.Group},
				Resources: []string{"promotions"},
				Verbs:     []string{"get", "list"},
			}}),
			managedRoleBinding(),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		kargoRole, err := db.GrantPermissionsToRole(
			t.Context(),
			testProject,
			testKargoRoleName,
			&rbacapi.ResourceDetails{
				ResourceType: "stages",
				Verbs:        []string{"get", "list"},
			},
		)
		require.NoError(t, err)
		require.NotNil(t, kargoRole)
		role := &rbacv1.Role{}
		err = c.Get(t.Context(), objKey, role)
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{
				{
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"promotions"},
					Verbs:     []string{"get", "list"},
				},
				{
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"stages"},
					Verbs:     []string{"get", "list"},
				},
			},
			role.Rules,
		)
	})
}

func Test_rolesDatabase_GrantRoleToUsers(t *testing.T) {
	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		_, err := db.GrantRoleToUsers(
			t.Context(),
			testProject,
			testKargoRoleName,
			[]rbacapi.Claim{
				{
					Name:   "sub",
					Values: []string{"fake-sub"},
				}},
		)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("resources aren't manageable", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		_, err := db.GrantRoleToUsers(
			t.Context(),
			testProject,
			testKargoRoleName,
			[]rbacapi.Claim{
				{
					Name:   "sub",
					Values: []string{"fake-sub"},
				},
			},
		)
		require.True(t, apierrors.IsBadRequest(err))
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(map[string]string{
				rbacapi.AnnotationKeyOIDCClaim("sub"):    "foo-sub",
				rbacapi.AnnotationKeyOIDCClaim("email"):  "foo-email",
				rbacapi.AnnotationKeyOIDCClaim("groups"): "foo-group",
			}),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		kargoRole, err := db.GrantRoleToUsers(
			t.Context(),
			testProject,
			testKargoRoleName,
			[]rbacapi.Claim{
				{
					Name:   "sub",
					Values: []string{"foo-sub", "bar-sub"},
				},
				{
					Name:   "email",
					Values: []string{"foo-email", "bar-email"},
				},
				{
					Name:   "groups",
					Values: []string{"foo-group", "bar-group"},
				},
			},
		)
		require.NoError(t, err)
		require.NotNil(t, kargoRole)
		sa := &corev1.ServiceAccount{}
		err = c.Get(
			t.Context(),
			client.ObjectKey{Namespace: testProject, Name: testKargoRoleName},
			sa,
		)
		require.NoError(t, err)
		expected := `{"email":["bar-email","foo-email"],"groups":["bar-group","foo-group"],"sub":["bar-sub","foo-sub"]}`
		require.Equal(
			t,
			map[string]string{
				rbacapi.AnnotationKeyManaged:    rbacapi.AnnotationValueTrue,
				rbacapi.AnnotationKeyOIDCClaims: expected,
			},
			sa.Annotations,
		)
	})
}

func Test_rolesDatabase_List(t *testing.T) {
	t.Run("with only kargo-managed roles", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(map[string]string{
				rbacapi.AnnotationKeyOIDCClaim("sub"):    "foo-sub,bar-sub",
				rbacapi.AnnotationKeyOIDCClaim("email"):  "foo-email,bar-email",
				rbacapi.AnnotationKeyOIDCClaim("groups"): "foo-group,bar-group",
			}),
			managedRole([]rbacv1.PolicyRule{
				{
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"stages", "promotions"},
					Verbs:     []string{"list", "get"},
				},
			}),
			managedRoleBinding(),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		kargoRoles, err := db.List(t.Context(), false, testProject)
		require.NoError(t, err)
		// Do not factor creation timestamp into the comparison
		now := metav1.NewTime(time.Now())
		for _, kargoRole := range kargoRoles {
			kargoRole.CreationTimestamp = now
		}
		require.Equal(
			t,
			[]*rbacapi.Role{{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         testProject,
					Name:              testKargoRoleName,
					CreationTimestamp: now,
				},
				KargoManaged: true,
				Claims: []rbacapi.Claim{
					{
						Name:   "email",
						Values: []string{"bar-email", "foo-email"},
					},
					{
						Name:   "groups",
						Values: []string{"bar-group", "foo-group"},
					},
					{
						Name:   "sub",
						Values: []string{"bar-sub", "foo-sub"},
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{kargoapi.GroupVersion.Group},
						Resources: []string{"promotions"},
						Verbs:     []string{"get", "list"},
					},
					{
						APIGroups: []string{kargoapi.GroupVersion.Group},
						Resources: []string{"stages"},
						Verbs:     []string{"get", "list"},
					},
				},
			}},
			kargoRoles,
		)
	})

	t.Run("with a non-kargo-managed role", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(map[string]string{
				rbacapi.AnnotationKeyOIDCClaim("sub"):    "foo-sub,bar-sub",
				rbacapi.AnnotationKeyOIDCClaim("email"):  "foo-email,bar-email",
				rbacapi.AnnotationKeyOIDCClaim("groups"): "foo-group,bar-group",
			}),
			plainRole([]rbacv1.PolicyRule{
				{ // This rule has groups and types that we don't recognize. Let's
					// make sure we don't choke on them. This could happen with roles
					// that aren't Kargo-managed.
					APIGroups: []string{"fake-group-1", "fake-group-2"},
					Resources: []string{"fake-type-1", "fake-type-2"},
					Verbs:     []string{"get", "list"},
				},
				{
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"stages", "promotions"},
					Verbs:     []string{"list", "get"},
				},
			}),
			plainRoleBinding(),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		kargoRoles, err := db.List(t.Context(), false, testProject)
		require.NoError(t, err)
		// Do not factor creation timestamp into the comparison
		now := metav1.NewTime(time.Now())
		for _, kargoRole := range kargoRoles {
			kargoRole.CreationTimestamp = now
		}
		require.Equal(
			t,
			[]*rbacapi.Role{{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         testProject,
					Name:              testKargoRoleName,
					CreationTimestamp: now,
				},
				KargoManaged: false,
				Claims: []rbacapi.Claim{
					{
						Name:   "email",
						Values: []string{"bar-email", "foo-email"},
					},
					{
						Name:   "groups",
						Values: []string{"bar-group", "foo-group"},
					},
					{
						Name:   "sub",
						Values: []string{"bar-sub", "foo-sub"},
					},
				},
				// There should have been no attempt to normalize these rules
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{"fake-group-1", "fake-group-2"},
						Resources: []string{"fake-type-1", "fake-type-2"},
						Verbs:     []string{"get", "list"},
					},
					{
						APIGroups: []string{kargoapi.GroupVersion.Group},
						Resources: []string{"stages", "promotions"},
						Verbs:     []string{"list", "get"},
					},
				},
			}},
			kargoRoles,
		)
	})
}

func Test_rolesDatabase_RevokePermissionsFromRole(t *testing.T) {
	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		_, err := db.RevokePermissionsFromRole(
			t.Context(),
			testProject,
			testKargoRoleName,
			&rbacapi.ResourceDetails{
				ResourceType: "fake-resource-type",
				Verbs:        []string{"get", "list"},
			},
		)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("resources aren't manageable", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		_, err := db.RevokePermissionsFromRole(
			t.Context(),
			testProject,
			testKargoRoleName,
			&rbacapi.ResourceDetails{
				ResourceType: "fake-resource-type",
				Verbs:        []string{"get", "list"},
			},
		)
		require.True(t, apierrors.IsBadRequest(err))
	})

	t.Run("success with no action required", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		kargoRole, err := db.RevokePermissionsFromRole(
			t.Context(),
			testProject,
			testKargoRoleName,
			&rbacapi.ResourceDetails{
				ResourceType: "fake-resource-type",
				Verbs:        []string{"get", "list"},
			},
		)
		require.NoError(t, err)
		require.NotNil(t, kargoRole)
	})

	t.Run("success with rule changes", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(nil),
			managedRole([]rbacv1.PolicyRule{{
				APIGroups: []string{kargoapi.GroupVersion.Group},
				Resources: []string{"stages", "promotions"},
				Verbs:     []string{"get", "list"},
			}}),
			managedRoleBinding(),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		kargoRole, err := db.RevokePermissionsFromRole(
			t.Context(),
			testProject,
			testKargoRoleName,
			&rbacapi.ResourceDetails{
				ResourceType: "stages",
				Verbs:        []string{"get", "list"},
			},
		)
		require.NoError(t, err)
		require.NotNil(t, kargoRole)
		role := &rbacv1.Role{}
		err = c.Get(t.Context(), objKey, role)
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{{
				APIGroups: []string{kargoapi.GroupVersion.Group},
				Resources: []string{"promotions"},
				Verbs:     []string{"get", "list"},
			}},
			role.Rules,
		)
	})
}

func Test_rolesDatabase_RevokeRoleFromUsers(t *testing.T) {
	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		_, err := db.RevokeRoleFromUsers(
			t.Context(),
			testProject,
			testKargoRoleName,
			[]rbacapi.Claim{
				{
					Name:   "sub",
					Values: []string{"fake-sub"},
				},
			},
		)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("resources aren't manageable", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		_, err := db.RevokeRoleFromUsers(
			t.Context(),
			testProject,
			testKargoRoleName,
			[]rbacapi.Claim{
				{
					Name:   "sub",
					Values: []string{"fake-sub"},
				},
			},
		)
		require.True(t, apierrors.IsBadRequest(err))
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(map[string]string{
				rbacapi.AnnotationKeyOIDCClaim("sub"):    "bar-sub,foo-sub",
				rbacapi.AnnotationKeyOIDCClaim("email"):  "bar-email,foo-email",
				rbacapi.AnnotationKeyOIDCClaim("groups"): "bar-group,foo-group",
			}),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		kargoRole, err := db.RevokeRoleFromUsers(
			t.Context(),
			testProject,
			testKargoRoleName,
			[]rbacapi.Claim{
				{
					Name:   "sub",
					Values: []string{"bar-sub"},
				},
				{
					Name:   "email",
					Values: []string{"foo-email", "bar-email"},
				},
				{
					Name:   "groups",
					Values: []string{"foo-group", "bar-group"},
				},
			},
		)
		require.NoError(t, err)
		require.NotNil(t, kargoRole)
		sa := &corev1.ServiceAccount{}
		err = c.Get(
			t.Context(),
			client.ObjectKey{Namespace: testProject, Name: testKargoRoleName},
			sa,
		)
		require.NoError(t, err)
		require.Equal(
			t,
			map[string]string{
				rbacapi.AnnotationKeyManaged:    rbacapi.AnnotationValueTrue,
				rbacapi.AnnotationKeyOIDCClaims: `{"sub":["foo-sub"]}`,
			},
			sa.Annotations,
		)
	})
}

func Test_rolesDatabase_Update(t *testing.T) {
	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		_, err := db.Update(
			t.Context(),
			&rbacapi.Role{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      testKargoRoleName,
				},
			},
		)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("resources aren't manageable", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		_, err := db.Update(
			t.Context(),
			&rbacapi.Role{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      testKargoRoleName,
				},
			},
		)
		require.True(t, apierrors.IsBadRequest(err))
	})

	t.Run("success with Role and RoleBinding creation", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		kargoRole, err := db.Update(
			t.Context(),
			&rbacapi.Role{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      testKargoRoleName,
				},
				Claims: []rbacapi.Claim{
					{
						Name:   "sub",
						Values: []string{"foo-sub", "bar-sub"},
					},
					{
						Name:   "email",
						Values: []string{"foo-email", "bar-email"},
					}, {
						Name:   "groups",
						Values: []string{"foo-group", "bar-group"},
					},
				},
				Rules: []rbacv1.PolicyRule{{
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"stages", "promotions"},
					Verbs:     []string{"get", "list"},
				}},
			},
		)
		require.NoError(t, err)
		require.NotNil(t, kargoRole)
		sa := &corev1.ServiceAccount{}
		err = c.Get(t.Context(), objKey, sa)
		require.NoError(t, err)
		expected := `{"email":["bar-email","foo-email"],"groups":["bar-group","foo-group"],"sub":["bar-sub","foo-sub"]}`
		require.Equal(
			t,
			map[string]string{
				rbacapi.AnnotationKeyManaged:    rbacapi.AnnotationValueTrue,
				rbacapi.AnnotationKeyOIDCClaims: expected,
			},
			sa.Annotations,
		)
		rb := &rbacv1.RoleBinding{}
		err = c.Get(t.Context(), objKey, rb)
		require.NoError(t, err)
		require.True(t, isKargoManaged(rb))
		require.Equal(
			t,
			[]rbacv1.Subject{{
				Kind:      rbacv1.ServiceAccountKind,
				Namespace: testProject,
				Name:      testKargoRoleName,
			}},
			rb.Subjects,
		)
		require.Equal(
			t,
			rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     testKargoRoleName,
			},
			rb.RoleRef,
		)
		role := &rbacv1.Role{}
		err = c.Get(t.Context(), objKey, role)
		require.NoError(t, err)
		require.True(t, isKargoManaged(role))
		require.Equal(
			t,
			[]rbacv1.PolicyRule{
				{
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"promotions"},
					Verbs:     []string{"get", "list"},
				},
				{
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"stages"},
					Verbs:     []string{"get", "list"},
				},
			},
			role.Rules,
		)
	})

	t.Run("success with updated ServiceAccount and Role", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(map[string]string{
				rbacapi.AnnotationKeyOIDCClaim("sub"):    "foo-sub,bar-sub",
				rbacapi.AnnotationKeyOIDCClaim("email"):  "foo-email,bar-email",
				rbacapi.AnnotationKeyOIDCClaim("groups"): "foo-group,bar-group",
			}),
			managedRole([]rbacv1.PolicyRule{{
				APIGroups: []string{kargoapi.GroupVersion.Group},
				Resources: []string{"promotions"},
				Verbs:     []string{"get", "list"},
			}}),
			managedRoleBinding(),
		).Build()
		db := NewKubernetesRolesDatabase(c, RolesDatabaseConfig{})
		kargoRole, err := db.Update(
			t.Context(),
			&rbacapi.Role{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      testKargoRoleName,
					Annotations: map[string]string{
						kargoapi.AnnotationKeyDescription: "foo-description",
					},
				},
				Claims: []rbacapi.Claim{
					{
						Name:   "sub",
						Values: []string{"foo-sub"},
					},
					{
						Name:   "email",
						Values: []string{"foo-email"},
					}, {
						Name:   "groups",
						Values: []string{"foo-group"},
					},
				},
				Rules: []rbacv1.PolicyRule{{
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"stages", "promotions"},
					Verbs:     []string{"get", "list"},
				}},
			},
		)
		require.NoError(t, err)
		require.NotNil(t, kargoRole)
		sa := &corev1.ServiceAccount{}
		err = c.Get(t.Context(), objKey, sa)
		require.NoError(t, err)
		require.Equal(
			t,
			map[string]string{
				rbacapi.AnnotationKeyManaged:      rbacapi.AnnotationValueTrue,
				rbacapi.AnnotationKeyOIDCClaims:   `{"email":["foo-email"],"groups":["foo-group"],"sub":["foo-sub"]}`,
				kargoapi.AnnotationKeyDescription: "foo-description",
			},
			sa.Annotations,
		)
		role := &rbacv1.Role{}
		err = c.Get(t.Context(), objKey, role)
		require.NoError(t, err)
		require.Equal(
			t,
			[]rbacv1.PolicyRule{
				{
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"promotions"},
					Verbs:     []string{"get", "list"},
				},
				{
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"stages"},
					Verbs:     []string{"get", "list"},
				},
			},
			role.Rules,
		)
	})
}

func Test_manageableResources(t *testing.T) {
	t.Run("ServiceAccount is not annotated correctly", func(t *testing.T) {
		_, _, err := manageableResources(
			*plainServiceAccount(nil),
			nil,
			nil,
		)
		require.True(t, apierrors.IsBadRequest(err))
	})

	t.Run("multiple Roles", func(t *testing.T) {
		_, _, err := manageableResources(
			*managedServiceAccount(nil),
			[]rbacv1.Role{{}, {}},
			nil,
		)
		require.True(t, apierrors.IsBadRequest(err))
	})

	t.Run("single Role not annotated correctly", func(t *testing.T) {
		_, _, err := manageableResources(
			*managedServiceAccount(nil),
			[]rbacv1.Role{*plainRole(nil)},
			nil,
		)
		require.True(t, apierrors.IsBadRequest(err))
	})

	t.Run("multiple RoleBindings", func(t *testing.T) {
		_, _, err := manageableResources(
			*managedServiceAccount(nil),
			[]rbacv1.Role{*managedRole(nil)},
			[]rbacv1.RoleBinding{{}, {}},
		)
		require.True(t, apierrors.IsBadRequest(err))
	})

	t.Run("single RoleBinding is not annotated correctly", func(t *testing.T) {
		_, _, err := manageableResources(
			*managedServiceAccount(nil),
			[]rbacv1.Role{*managedRole(nil)},
			[]rbacv1.RoleBinding{*plainRoleBinding()},
		)
		require.True(t, apierrors.IsBadRequest(err))
	})

	t.Run("success", func(t *testing.T) {
		role, rb, err := manageableResources(
			*managedServiceAccount(nil),
			[]rbacv1.Role{*managedRole(nil)},
			[]rbacv1.RoleBinding{*managedRoleBinding()},
		)
		require.NoError(t, err)
		require.NotNil(t, role)
		require.NotNil(t, rb)
	})
}

func Test_amendClaimAnnotations(t *testing.T) {
	testCases := []struct {
		name                string
		sa                  *corev1.ServiceAccount
		claimsToAmend       map[string][]string
		expectedAnnotations map[string]string
	}{
		{
			name: "amend simple",
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						rbacapi.AnnotationKeyOIDCClaims: `{"email":["bar@foo.com"],"sub":["foo","bar"]}`,
					},
				},
			},
			claimsToAmend: map[string][]string{
				"email": {"foo@bar.com"},
				"sub":   {"baz"},
			},
			expectedAnnotations: map[string]string{
				rbacapi.AnnotationKeyOIDCClaims: `{"email":["bar@foo.com","foo@bar.com"],"sub":["bar","baz","foo"]}`,
			},
		},
		{
			name: "amend from old sa claim annotations and amend with new rbac.kargo.akuity.io/claims format",
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						rbacapi.AnnotationKeyOIDCClaim("email"): "foo@bar.com",
						rbacapi.AnnotationKeyOIDCClaim("sub"):   "foo,bar",
					},
				},
			},
			claimsToAmend: map[string][]string{
				"email": {"bar@foo.com"},
				"sub":   {"baz"},
			},
			expectedAnnotations: map[string]string{
				rbacapi.AnnotationKeyOIDCClaims: `{"email":["bar@foo.com","foo@bar.com"],"sub":["bar","baz","foo"]}`,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := amendClaimAnnotations(tc.sa, tc.claimsToAmend)
			require.NoError(t, err)
			isEqual := maps.Equal(tc.expectedAnnotations, tc.sa.Annotations)
			require.True(t, isEqual, "expected:\n%+v\n, got:\n%+v\n", tc.expectedAnnotations, tc.sa.Annotations)
		})
	}
}

func Test_dropClaimAnnotations(t *testing.T) {
	testCases := []struct {
		name                string
		sa                  *corev1.ServiceAccount
		claimsToDrop        map[string][]string
		expectedAnnotations map[string]string
		assertions          func(t *testing.T, expectedClaims []string, saAnnotations map[string]string)
	}{
		{
			name: "drop simple",
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						rbacapi.AnnotationKeyOIDCClaims: `{"email":["foo@bar.com"],"sub":["foo","bar"]}`,
					},
				},
			},
			claimsToDrop: map[string][]string{
				"email": {"foo@bar.com"},
				"sub":   {"foo"},
			},
			expectedAnnotations: map[string]string{
				rbacapi.AnnotationKeyOIDCClaims: `{"sub":["bar"]}`,
			},
		},
		{
			name: "drop from old sa claim annotations and convert to rbac.kargo.akuity.io/claims format",
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						rbacapi.AnnotationKeyOIDCClaim("email"): "foo@bar.com",
						rbacapi.AnnotationKeyOIDCClaim("sub"):   "foo,bar",
					},
				},
			},
			claimsToDrop: map[string][]string{
				"email": {"foo@bar.com"},
				"sub":   {"bar"},
			},
			expectedAnnotations: map[string]string{
				rbacapi.AnnotationKeyOIDCClaims: `{"sub":["foo"]}`,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := dropFromClaimAnnotations(tc.sa, tc.claimsToDrop)
			require.NoError(t, err)
			isEqual := maps.Equal(tc.expectedAnnotations, tc.sa.Annotations)
			require.True(t, isEqual, "expected:\n%+v\n, got:\n%+v\n", tc.expectedAnnotations, tc.sa.Annotations)
		})
	}
}

func plainServiceAccount(annotations map[string]string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{ObjectMeta: plainObjectMeta(annotations)}
}

func managedServiceAccount(annotations map[string]string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{ObjectMeta: managedObjectMeta(annotations)}
}

func plainRole(rules []rbacv1.PolicyRule) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: plainObjectMeta(nil),
		Rules:      rules,
	}
}

func managedRole(rules []rbacv1.PolicyRule) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: managedObjectMeta(nil),
		Rules:      rules,
	}
}

func plainRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: plainObjectMeta(nil),
		Subjects: []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Namespace: testProject,
			Name:      testKargoRoleName,
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     testKargoRoleName,
		},
	}
}

func managedRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: managedObjectMeta(nil),
		Subjects: []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Namespace: testProject,
			Name:      testKargoRoleName,
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     testKargoRoleName,
		},
	}
}

func plainObjectMeta(annotations map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace:   testProject,
		Name:        testKargoRoleName,
		Annotations: annotations,
	}
}

func managedObjectMeta(annotations map[string]string) metav1.ObjectMeta {
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[rbacapi.AnnotationKeyManaged] = rbacapi.AnnotationValueTrue

	return metav1.ObjectMeta{
		Namespace:   testProject,
		Name:        testKargoRoleName,
		Annotations: annotations,
	}
}

func TestResourcesToRole(t *testing.T) {
	testCases := []struct {
		name           string
		sa             *corev1.ServiceAccount
		roles          []rbacv1.Role
		roleBindings   []rbacv1.RoleBinding
		expectedClaims []rbacapi.Claim
		assertions     func(t *testing.T, role *rbacapi.Role, err error)
	}{
		{
			name: "nil service account",
			sa:   nil,
			assertions: func(t *testing.T, role *rbacapi.Role, err error) {
				require.Nil(t, role)
				require.Nil(t, err)
			},
		},
		{
			name: "no resources",
			sa:   new(corev1.ServiceAccount),
			assertions: func(t *testing.T, role *rbacapi.Role, err error) {
				require.NoError(t, err)
				require.Empty(t, role.Claims)
				require.Empty(t, role.Rules)
			},
		},
		{
			name: "kargo-managed",
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
					},
				},
			},
			assertions: func(t *testing.T, role *rbacapi.Role, err error) {
				require.NoError(t, err)
				require.True(t, role.KargoManaged)
				require.Empty(t, role.Claims)
				require.Empty(t, role.Rules)
			},
		},
		{
			name: "with old annotations",
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						rbacapi.AnnotationKeyOIDCClaim("groups"): "foo:bar",
					},
				},
			},
			assertions: func(t *testing.T, role *rbacapi.Role, err error) {
				require.NoError(t, err)
				require.Empty(t, role.Rules)
				require.Len(t, role.Claims, 1)
				require.Len(t, role.Claims[0].Values, 1)
				require.Equal(t, "groups", role.Claims[0].Name)
				require.Equal(t, "foo:bar", role.Claims[0].Values[0])
			},
		},
		{
			name: "with new annotations",
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						rbacapi.AnnotationKeyOIDCClaims: `{"groups":["foo:bar"]}`,
					},
				},
			},
			assertions: func(t *testing.T, role *rbacapi.Role, err error) {
				require.NoError(t, err)
				require.Empty(t, role.Rules)
				require.Len(t, role.Claims, 1)
				require.Len(t, role.Claims[0].Values, 1)
				require.Equal(t, "groups", role.Claims[0].Name)
				require.Equal(t, "foo:bar", role.Claims[0].Values[0])
			},
		},
		{
			name: "with both old and new annotations",
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						rbacapi.AnnotationKeyOIDCClaim("sub"): "foo-sub",
						rbacapi.AnnotationKeyOIDCClaims: `
						{
							"email":["email@inbox.com"],
							"sub":["another-sub"],
							"groups":["another-group"]
						}`,
						rbacapi.AnnotationKeyOIDCClaim("groups"): "foo:bar",
					},
				},
			},
			assertions: func(t *testing.T, role *rbacapi.Role, err error) {
				require.NoError(t, err)
				require.Empty(t, role.Rules)
				require.Len(t, role.Claims, 3)

				claimsMap := map[string]rbacapi.Claim{}
				for _, claim := range role.Claims {
					claimsMap[claim.Name] = claim
				}

				emailClaim, ok := claimsMap["email"]
				require.True(t, ok)
				require.Len(t, emailClaim.Values, 1)
				require.Equal(t, "email@inbox.com", emailClaim.Values[0])

				groupsClaim, ok := claimsMap["groups"]
				require.True(t, ok)
				require.Len(t, groupsClaim.Values, 2)
				require.Equal(t, "another-group", groupsClaim.Values[0])
				require.Equal(t, "foo:bar", groupsClaim.Values[1])

				subClaim, ok := claimsMap["sub"]
				require.True(t, ok)
				require.Len(t, subClaim.Values, 2)
				require.Equal(t, "another-sub", subClaim.Values[0])
				require.Equal(t, "foo-sub", subClaim.Values[1])
			},
		},
		{
			name: "policy rules",
			sa:   new(corev1.ServiceAccount),
			roles: []rbacv1.Role{
				*managedRole([]rbacv1.PolicyRule{{
					APIGroups: []string{kargoapi.GroupVersion.Group},
					Resources: []string{"stages", "promotions"},
					Verbs:     []string{"list", "get"},
				}}),
			},
			assertions: func(t *testing.T, role *rbacapi.Role, err error) {
				require.NoError(t, err)
				require.Empty(t, role.Claims)
				require.Len(t, role.Rules, 1)
				require.Equal(t, []string{kargoapi.GroupVersion.Group}, role.Rules[0].APIGroups)
				require.Equal(t, []string{"stages", "promotions"}, role.Rules[0].Resources)
				require.Equal(t, []string{"list", "get"}, role.Rules[0].Verbs)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			role, err := ResourcesToRole(tc.sa, tc.roles, tc.roleBindings)
			tc.assertions(t, role, err)
		})
	}
}

const testRoleName = "fake-role"

func Test_rolesDatabase_CreateAPIToken(t *testing.T) {
	const testTokenName = "fake-token"

	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		_, err := NewKubernetesRolesDatabase(c, RolesDatabaseConfigFromEnv()).
			CreateAPIToken(t.Context(), false, testProject, testRoleName, testTokenName)
		require.Error(t, err)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Secret with token name already exists", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testRoleName,
				Annotations: map[string]string{
					rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				},
			}},
			&corev1.Secret{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testTokenName,
			}},
		).Build()
		_, err := NewKubernetesRolesDatabase(c, RolesDatabaseConfigFromEnv()).
			CreateAPIToken(t.Context(), false, testProject, testRoleName, testTokenName)
		require.Error(t, err)
		require.True(t, apierrors.IsAlreadyExists(err))
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).
			WithObjects(&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      testRoleName,
					Annotations: map[string]string{
						rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
					},
				},
			}).
			WithInterceptorFuncs(interceptor.Funcs{
				// The method under test has a simple retry loop that waits for the
				// new Secret's data to be populated. We need populate the Secret's data
				// ourselves because the fake client doesn't do it.
				Get: func(
					ctx context.Context,
					client client.WithWatch,
					key client.ObjectKey,
					obj client.Object,
					opts ...client.GetOption,
				) error {
					if s, ok := obj.(*corev1.Secret); ok {
						newS := &corev1.Secret{}
						if err := client.Get(ctx, key, newS); err != nil {
							return err
						}
						newS.Data = map[string][]byte{
							"token": []byte("fake-token-value"),
						}
						*s = *newS
						return nil
					}
					return client.Get(ctx, key, obj, opts...)
				},
			}).
			Build()
		tokenSecret, err := NewKubernetesRolesDatabase(c, RolesDatabaseConfigFromEnv()).
			CreateAPIToken(t.Context(), false, testProject, testRoleName, testTokenName)
		require.NoError(t, err)
		require.NotNil(t, tokenSecret)
		tokenSecret = &corev1.Secret{}
		err = c.Get(
			t.Context(),
			client.ObjectKey{
				Namespace: testProject,
				Name:      testTokenName,
			},
			tokenSecret,
		)
		require.NoError(t, err)
		require.Equal(t, corev1.SecretTypeServiceAccountToken, tokenSecret.Type)
		require.Equal(
			t,
			testRoleName,
			tokenSecret.Annotations["kubernetes.io/service-account.name"],
		)
	})
}

func Test_rolesDatabase_GetAPIToken(t *testing.T) {
	t.Run("token Secret not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		_, err := NewKubernetesRolesDatabase(c, RolesDatabaseConfigFromEnv()).
			GetAPIToken(t.Context(), false, testProject, "non-existent-token")
		require.Error(t, err)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Secret is not a Kargo API token", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "regular-secret",
				},
				Type: corev1.SecretTypeOpaque,
			},
		).Build()
		_, err := NewKubernetesRolesDatabase(c, RolesDatabaseConfigFromEnv()).
			GetAPIToken(t.Context(), false, testProject, "regular-secret")
		require.Error(t, err)
		require.True(t, apierrors.IsBadRequest(err))
		require.Contains(t, err.Error(), "not labeled as a Kargo API token")
	})

	t.Run("success", func(t *testing.T) {
		tokenName := "test-token"
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      tokenName,
					Labels: map[string]string{
						rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
					},
					Annotations: map[string]string{
						"kubernetes.io/service-account.name": testRoleName,
						rbacapi.AnnotationKeyManaged:         rbacapi.AnnotationValueTrue,
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
				Data: map[string][]byte{
					"token": []byte("test-token-data"),
				},
			},
		).Build()
		secret, err := NewKubernetesRolesDatabase(c, RolesDatabaseConfigFromEnv()).
			GetAPIToken(t.Context(), false, testProject, tokenName)
		require.NoError(t, err)
		require.NotNil(t, secret)
		require.Equal(t, testProject, secret.Namespace)
		require.Equal(t, tokenName, secret.Name)
		require.Equal(t, corev1.SecretTypeServiceAccountToken, secret.Type)
		require.Equal(
			t,
			testRoleName,
			secret.Annotations["kubernetes.io/service-account.name"],
		)
		// Token data should be redacted
		require.Equal(t, []byte("*** REDACTED ***"), secret.Data["token"])
	})
}

func Test_rolesDatabase_DeleteAPIToken(t *testing.T) {
	t.Run("token Secret not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testRoleName,
				Annotations: map[string]string{
					rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				},
			}},
		).Build()
		err := NewKubernetesRolesDatabase(c, RolesDatabaseConfigFromEnv()).
			DeleteAPIToken(t.Context(), false, testProject, "non-existent-token")
		require.Error(t, err)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("token Secret not labeled as Kargo API token", func(t *testing.T) {
		tokenName := "test-token"
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testRoleName,
				Annotations: map[string]string{
					rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				},
			}},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      tokenName,
					Annotations: map[string]string{
						"kubernetes.io/service-account.name": testRoleName,
						rbacapi.AnnotationKeyManaged:         rbacapi.AnnotationValueTrue,
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
			},
		).Build()
		err := NewKubernetesRolesDatabase(c, RolesDatabaseConfigFromEnv()).
			DeleteAPIToken(t.Context(), false, testProject, tokenName)
		require.Error(t, err)
		require.True(t, apierrors.IsConflict(err))
		require.Contains(t, err.Error(), "not labeled as a Kargo API token")
	})

	t.Run("token Secret not annotated as Kargo-managed", func(t *testing.T) {
		tokenName := "test-token"
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testRoleName,
				Annotations: map[string]string{
					rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				},
			}},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      tokenName,
					Labels: map[string]string{
						rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
					},
					Annotations: map[string]string{
						"kubernetes.io/service-account.name": testRoleName,
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
			},
		).Build()
		err := NewKubernetesRolesDatabase(c, RolesDatabaseConfigFromEnv()).
			DeleteAPIToken(t.Context(), false, testProject, tokenName)
		require.Error(t, err)
		require.True(t, apierrors.IsConflict(err))
		require.Contains(t, err.Error(), "not annotated as Kargo-managed")
	})

	t.Run("success", func(t *testing.T) {
		tokenName := "test-token"
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testRoleName,
				Annotations: map[string]string{
					rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				},
			}},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      tokenName,
					Labels: map[string]string{
						rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
					},
					Annotations: map[string]string{
						"kubernetes.io/service-account.name": testRoleName,
						rbacapi.AnnotationKeyManaged:         rbacapi.AnnotationValueTrue,
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
			},
		).Build()
		err := NewKubernetesRolesDatabase(c, RolesDatabaseConfigFromEnv()).
			DeleteAPIToken(t.Context(), false, testProject, tokenName)
		require.NoError(t, err)
		// Verify the token Secret was deleted
		secret := &corev1.Secret{}
		err = c.Get(
			t.Context(),
			client.ObjectKey{
				Namespace: testProject,
				Name:      tokenName,
			},
			secret,
		)
		require.Error(t, err)
		require.True(t, apierrors.IsNotFound(err))
	})
}

func Test_rolesDatabase_ListAPITokens(t *testing.T) {
	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		_, err := NewKubernetesRolesDatabase(c, RolesDatabaseConfigFromEnv()).
			ListAPITokens(t.Context(), false, testProject, testRoleName)
		require.Error(t, err)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("no tokens", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      testRoleName,
				Annotations: map[string]string{
					rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				},
			}},
		).Build()
		tokens, err := NewKubernetesRolesDatabase(c, RolesDatabaseConfigFromEnv()).
			ListAPITokens(t.Context(), false, testProject, testRoleName)
		require.NoError(t, err)
		require.Empty(t, tokens)
	})

	t.Run("with tokens for different ServiceAccounts", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      testRoleName,
					Annotations: map[string]string{
						rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "token-1",
					Labels: map[string]string{
						rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
					},
					Annotations: map[string]string{
						"kubernetes.io/service-account.name": testRoleName,
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
				Data: map[string][]byte{"token": []byte("token-1-data")},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "token-2",
					Labels: map[string]string{
						rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
					},
					Annotations: map[string]string{
						"kubernetes.io/service-account.name": testRoleName,
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
				Data: map[string][]byte{"token": []byte("token-2-data")},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "other-token",
					Labels: map[string]string{
						rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
					},
					Annotations: map[string]string{
						"kubernetes.io/service-account.name": "other-sa",
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
				Data: map[string][]byte{"token": []byte("other-token-data")},
			},
		).Build()
		tokens, err := NewKubernetesRolesDatabase(c, RolesDatabaseConfigFromEnv()).
			ListAPITokens(t.Context(), false, testProject, testRoleName)
		require.NoError(t, err)
		require.Len(t, tokens, 2)

		names := []string{tokens[0].Name, tokens[1].Name}
		require.Contains(t, names, "token-1")
		require.Contains(t, names, "token-2")
		require.NotContains(t, names, "other-token")
	})
}

func Test_isKargoAPIToken(t *testing.T) {
	t.Run("wrong secret type", func(t *testing.T) {
		require.False(t, isKargoAPIToken(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "test-secret",
					Labels: map[string]string{
						rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
					},
				},
				Type: corev1.SecretTypeOpaque,
			},
		))
	})

	t.Run("not labeled as a Kargo API token", func(t *testing.T) {
		require.False(t, isKargoAPIToken(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "test-secret",
				},
				Type: corev1.SecretTypeServiceAccountToken,
			},
		))
	})

	t.Run("has label but wrong value", func(t *testing.T) {
		require.False(t, isKargoAPIToken(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "test-secret",
					Labels: map[string]string{
						rbacapi.LabelKeyAPIToken: "false",
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
			},
		))
	})

	t.Run("is a Kargo API token", func(t *testing.T) {
		require.True(t, isKargoAPIToken(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "test-token",
					Labels: map[string]string{
						rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
					},
				},
				Type: corev1.SecretTypeServiceAccountToken,
			},
		))
	})
}

func Test_redactTokenData(t *testing.T) {
	t.Run("no token data", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      "test-token",
			},
			Type: corev1.SecretTypeServiceAccountToken,
			Data: map[string][]byte{
				"ca.crt": []byte("cert-data"),
			},
		}
		redactTokenData(secret)
		require.Equal(t, []byte("cert-data"), secret.Data["ca.crt"])
		require.NotContains(t, secret.Data, "token")
	})

	t.Run("with token data", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      "test-token",
			},
			Type: corev1.SecretTypeServiceAccountToken,
			Data: map[string][]byte{
				"token":  []byte("sensitive-token-value"),
				"ca.crt": []byte("cert-data"),
			},
		}
		redactTokenData(secret)
		require.Equal(t, []byte("*** REDACTED ***"), secret.Data["token"])
		require.Equal(t, []byte("cert-data"), secret.Data["ca.crt"])
	})

	t.Run("with empty token data", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      "test-token",
			},
			Type: corev1.SecretTypeServiceAccountToken,
			Data: map[string][]byte{
				"token": []byte(""),
			},
		}
		redactTokenData(secret)
		require.Equal(t, []byte("*** REDACTED ***"), secret.Data["token"])
	})

	t.Run("with nil data map", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject,
				Name:      "test-token",
			},
			Type: corev1.SecretTypeServiceAccountToken,
		}
		// Should not panic
		redactTokenData(secret)
		require.Nil(t, secret.Data)
	})
}

func Test_rolesDatabase_waitForTokenData(t *testing.T) {
	const testTokenName = "test-token"

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, *corev1.Secret, error)
	}{
		{
			name: "non-retriable error",
			client: fake.NewClientBuilder().WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					Get: func(
						context.Context,
						client.WithWatch,
						client.ObjectKey,
						client.Object,
						...client.GetOption,
					) error {
						// Return a NotFound error - should not retry
						return apierrors.NewNotFound(corev1.Resource("secrets"), "")
					},
				}).
				Build(),
			assertions: func(t *testing.T, secret *corev1.Secret, err error) {
				require.Error(t, err)
				require.Nil(t, secret)
				require.Contains(t, err.Error(), "error while waiting for token Secret")
				require.True(t, apierrors.IsNotFound(err))
			},
		},
		{
			name: "token data not yet populated; all attempts fail",
			client: fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      testTokenName,
					},
					Type: corev1.SecretTypeServiceAccountToken,
					// No data
				}).Build(),
			assertions: func(t *testing.T, secret *corev1.Secret, err error) {
				require.Error(t, err)
				require.Nil(t, secret)
				require.Contains(t, err.Error(), "error while waiting for token Secret")
			},
		},
		{
			name: "other retriable error; all attempts fail",
			client: fake.NewClientBuilder().WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					Get: func(
						context.Context,
						client.WithWatch,
						client.ObjectKey,
						client.Object,
						...client.GetOption,
					) error {
						// Always return a retriable error
						return apierrors.NewServiceUnavailable("service unavailable")
					},
				}).
				Build(),
			assertions: func(t *testing.T, secret *corev1.Secret, err error) {
				require.Error(t, err)
				require.Nil(t, secret)
				require.Contains(t, err.Error(), "error while waiting for token Secret")
				require.True(t, apierrors.IsServiceUnavailable(err))
			},
		},
		{
			name: "token data not yet populated; second attempt succeeds",
			client: func() client.Client {
				var attemptCount int
				return fake.NewClientBuilder().WithScheme(scheme).
					WithInterceptorFuncs(interceptor.Funcs{
						Get: func(
							_ context.Context,
							_ client.WithWatch,
							_ client.ObjectKey,
							obj client.Object,
							_ ...client.GetOption,
						) error {
							attemptCount++
							if attemptCount == 1 {
								return nil
							}
							// All subsequent attempts: populate token data
							s, ok := obj.(*corev1.Secret)
							require.True(t, ok)
							s.Data = map[string][]byte{"token": []byte("fake-token-value")}
							return nil
						},
					}).
					Build()
			}(),
			assertions: func(t *testing.T, secret *corev1.Secret, err error) {
				require.NoError(t, err)
				require.NotNil(t, secret)
				require.Equal(t, []byte("fake-token-value"), secret.Data["token"])
			},
		},
		{
			name: "other retriable error; second attempt succeeds",
			client: func() client.Client {
				var attemptCount int
				return fake.NewClientBuilder().WithScheme(scheme).
					WithInterceptorFuncs(interceptor.Funcs{
						Get: func(
							_ context.Context,
							_ client.WithWatch,
							_ client.ObjectKey,
							obj client.Object,
							_ ...client.GetOption,
						) error {
							attemptCount++
							if attemptCount == 1 {
								// First attempt: return retriable error
								return apierrors.NewServerTimeout(
									corev1.Resource("secrets"),
									"get",
									5,
								)
							}
							// All subsequent attempts: populate token data
							s, ok := obj.(*corev1.Secret)
							require.True(t, ok)
							s.Data = map[string][]byte{
								"token": []byte("fake-token-value"),
							}
							return nil
						},
					}).
					Build()
			}(),
			assertions: func(t *testing.T, secret *corev1.Secret, err error) {
				require.NoError(t, err)
				require.NotNil(t, secret)
				require.Equal(t, []byte("fake-token-value"), secret.Data["token"])
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s := &rolesDatabase{client: testCase.client}
			secret, err := s.waitForTokenData(
				context.Background(),
				testProject,
				testTokenName,
				2, // Only two attempts so that backoffs are minimal during tests
			)
			testCase.assertions(t, secret, err)
		})
	}
}
