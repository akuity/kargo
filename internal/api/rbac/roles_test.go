package rbac

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

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

func TestCreate(t *testing.T) {
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
		db := NewKubernetesRolesDatabase(c)
		role, err := db.Create(context.Background(), testKargoRole)
		require.True(t, kubeerr.IsAlreadyExists(err))
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
		db := NewKubernetesRolesDatabase(c)
		role, err := db.Create(context.Background(), testKargoRole)
		require.True(t, kubeerr.IsAlreadyExists(err))
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
		db := NewKubernetesRolesDatabase(c)
		role, err := db.Create(context.Background(), testKargoRole)
		require.True(t, kubeerr.IsAlreadyExists(err))
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
		db := NewKubernetesRolesDatabase(c)
		role, err := db.Create(context.Background(), testKargoRole)
		require.NoError(t, err)
		require.NotNil(t, role)

		sa := &corev1.ServiceAccount{}
		err = c.Get(
			context.Background(),
			client.ObjectKey{Namespace: testKargoRole.Namespace, Name: testKargoRole.Name},
			sa,
		)
		require.NoError(t, err)
		require.Equal(
			t,
			map[string]string{
				rbacapi.AnnotationKeyManaged:      rbacapi.AnnotationValueTrue,
				claimAnnotationKey("sub"):         "bar-sub,foo-sub",
				claimAnnotationKey("email"):       "bar-email,foo-email",
				claimAnnotationKey("groups"):      "bar-group,foo-group",
				kargoapi.AnnotationKeyDescription: "fake-description",
			},
			sa.Annotations,
		)

		roleBinding := &rbacv1.RoleBinding{}
		err = c.Get(
			context.Background(),
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

func TestDelete(t *testing.T) {
	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		db := NewKubernetesRolesDatabase(c)
		err := db.Delete(context.Background(), testProject, testKargoRoleName)
		require.True(t, kubeerr.IsNotFound(err))
	})

	t.Run("resources aren't manageable", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		err := db.Delete(context.Background(), testProject, testKargoRoleName)
		require.True(t, kubeerr.IsBadRequest(err))
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(nil),
			managedRole(nil),
			managedRoleBinding(),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		err := db.Delete(context.Background(), testProject, testKargoRoleName)
		require.NoError(t, err)
		role := &rbacv1.Role{}
		err = c.Get(context.Background(), objKey, role)
		require.True(t, kubeerr.IsNotFound(err))
		roleBinding := &rbacv1.RoleBinding{}
		err = c.Get(context.Background(), objKey, roleBinding)
		require.True(t, kubeerr.IsNotFound(err))
		sa := &corev1.ServiceAccount{}
		err = c.Get(context.Background(), objKey, sa)
		require.True(t, kubeerr.IsNotFound(err))
	})
}

func TestGet(t *testing.T) {
	t.Run("ServiceAccount does not exist", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		db := NewKubernetesRolesDatabase(c)
		kargoRole, err := db.Get(context.Background(), testProject, testKargoRoleName)
		require.True(t, kubeerr.IsNotFound(err))
		require.Nil(t, kargoRole)
	})

	t.Run("success with non-kargo-managed role", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(map[string]string{
				claimAnnotationKey("sub"):    "foo-sub,bar-sub",
				claimAnnotationKey("email"):  "foo-email,bar-email",
				claimAnnotationKey("groups"): "foo-group,bar-group",
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
		db := NewKubernetesRolesDatabase(c)
		kargoRole, err := db.Get(context.Background(), testProject, testKargoRoleName)
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
						Values: []string{"foo-email", "bar-email"},
					},
					{
						Name:   "groups",
						Values: []string{"foo-group", "bar-group"},
					},
					{
						Name:   "sub",
						Values: []string{"foo-sub", "bar-sub"},
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
				claimAnnotationKey("sub"):    "foo-sub,bar-sub",
				claimAnnotationKey("email"):  "foo-email,bar-email",
				claimAnnotationKey("groups"): "foo-group,bar-group",
			}),
			managedRole([]rbacv1.PolicyRule{{
				APIGroups: []string{kargoapi.GroupVersion.Group},
				Resources: []string{"stages", "promotions"},
				Verbs:     []string{"list", "get"},
			}}),
			managedRoleBinding(),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		kargoRole, err := db.Get(context.Background(), testProject, testKargoRoleName)
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
						Values: []string{"foo-email", "bar-email"},
					},
					{
						Name:   "groups",
						Values: []string{"foo-group", "bar-group"},
					},
					{
						Name:   "sub",
						Values: []string{"foo-sub", "bar-sub"},
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

func TestGetAsResources(t *testing.T) {
	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		db := NewKubernetesRolesDatabase(c)
		_, _, _, err := db.GetAsResources(context.Background(), testProject, testKargoRoleName)
		require.True(t, kubeerr.IsNotFound(err))
	})

	t.Run("no RoleBindings found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		sa, roles, rbs, err := db.GetAsResources(context.Background(), testProject, testKargoRoleName)
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
		db := NewKubernetesRolesDatabase(c)
		_, _, _, err := db.GetAsResources(context.Background(), testProject, testKargoRoleName)
		require.True(t, kubeerr.IsNotFound(err))
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(nil),
			plainRole(nil),
			plainRoleBinding(),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		sa, roles, rbs, err := db.GetAsResources(context.Background(), testProject, testKargoRoleName)
		require.NoError(t, err)
		require.NotNil(t, sa)
		require.NotNil(t, roles)
		require.NotNil(t, rbs)
	})
}

func TestGrantPermissionToRole(t *testing.T) {
	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		db := NewKubernetesRolesDatabase(c)
		_, err := db.GrantPermissionsToRole(
			context.Background(),
			testProject,
			testKargoRoleName,
			&rbacapi.ResourceDetails{
				ResourceType: "fake-resource-type",
				Verbs:        []string{"get", "list"},
			},
		)
		require.True(t, kubeerr.IsNotFound(err))
	})

	t.Run("resources aren't manageable", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		_, err := db.GrantPermissionsToRole(
			context.Background(),
			testProject,
			testKargoRoleName,
			&rbacapi.ResourceDetails{
				ResourceType: "fake-resource-type",
				Verbs:        []string{"get", "list"},
			},
		)
		require.True(t, kubeerr.IsBadRequest(err))
	})

	t.Run("success with Role and RoleBinding creation", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		kargoRole, err := db.GrantPermissionsToRole(
			context.Background(),
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
		err = c.Get(context.Background(), objKey, rb)
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
		err = c.Get(context.Background(), objKey, role)
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
		db := NewKubernetesRolesDatabase(c)
		kargoRole, err := db.GrantPermissionsToRole(
			context.Background(),
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
		err = c.Get(context.Background(), objKey, role)
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

func TestGrantRoleToUsers(t *testing.T) {
	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		db := NewKubernetesRolesDatabase(c)
		_, err := db.GrantRoleToUsers(
			context.Background(),
			testProject,
			testKargoRoleName,
			[]rbacapi.Claim{
				{
					Name:   "sub",
					Values: []string{"fake-sub"},
				}},
		)
		require.True(t, kubeerr.IsNotFound(err))
	})

	t.Run("resources aren't manageable", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		_, err := db.GrantRoleToUsers(
			context.Background(),
			testProject,
			testKargoRoleName,
			[]rbacapi.Claim{
				{
					Name:   "sub",
					Values: []string{"fake-sub"},
				},
			},
		)
		require.True(t, kubeerr.IsBadRequest(err))
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(map[string]string{
				claimAnnotationKey("sub"):    "foo-sub",
				claimAnnotationKey("email"):  "foo-email",
				claimAnnotationKey("groups"): "foo-group",
			}),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		kargoRole, err := db.GrantRoleToUsers(
			context.Background(),
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
			context.Background(),
			client.ObjectKey{Namespace: testProject, Name: testKargoRoleName},
			sa,
		)
		require.NoError(t, err)
		require.Equal(
			t,
			map[string]string{
				rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				claimAnnotationKey("sub"):    "bar-sub,foo-sub",
				claimAnnotationKey("email"):  "bar-email,foo-email",
				claimAnnotationKey("groups"): "bar-group,foo-group",
			},
			sa.Annotations,
		)
	})
}

func TestList(t *testing.T) {
	t.Run("with only kargo-managed roles", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(map[string]string{
				claimAnnotationKey("sub"):    "foo-sub,bar-sub",
				claimAnnotationKey("email"):  "foo-email,bar-email",
				claimAnnotationKey("groups"): "foo-group,bar-group",
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
		db := NewKubernetesRolesDatabase(c)
		kargoRoles, err := db.List(context.Background(), testProject)
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
						Values: []string{"foo-email", "bar-email"},
					},
					{
						Name:   "groups",
						Values: []string{"foo-group", "bar-group"},
					},
					{
						Name:   "sub",
						Values: []string{"foo-sub", "bar-sub"},
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
				claimAnnotationKey("sub"):    "foo-sub,bar-sub",
				claimAnnotationKey("email"):  "foo-email,bar-email",
				claimAnnotationKey("groups"): "foo-group,bar-group",
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
		db := NewKubernetesRolesDatabase(c)
		kargoRoles, err := db.List(context.Background(), testProject)
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
						Values: []string{"foo-email", "bar-email"},
					},
					{
						Name:   "groups",
						Values: []string{"foo-group", "bar-group"},
					},
					{
						Name:   "sub",
						Values: []string{"foo-sub", "bar-sub"},
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

func TestRevokePermissionsFromRole(t *testing.T) {
	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		db := NewKubernetesRolesDatabase(c)
		_, err := db.RevokePermissionsFromRole(
			context.Background(),
			testProject,
			testKargoRoleName,
			&rbacapi.ResourceDetails{
				ResourceType: "fake-resource-type",
				Verbs:        []string{"get", "list"},
			},
		)
		require.True(t, kubeerr.IsNotFound(err))
	})

	t.Run("resources aren't manageable", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		_, err := db.RevokePermissionsFromRole(
			context.Background(),
			testProject,
			testKargoRoleName,
			&rbacapi.ResourceDetails{
				ResourceType: "fake-resource-type",
				Verbs:        []string{"get", "list"},
			},
		)
		require.True(t, kubeerr.IsBadRequest(err))
	})

	t.Run("success with no action required", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		kargoRole, err := db.RevokePermissionsFromRole(
			context.Background(),
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
		db := NewKubernetesRolesDatabase(c)
		kargoRole, err := db.RevokePermissionsFromRole(
			context.Background(),
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
		err = c.Get(context.Background(), objKey, role)
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

func TestRevokeRoleFromUsers(t *testing.T) {
	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		db := NewKubernetesRolesDatabase(c)
		_, err := db.RevokeRoleFromUsers(
			context.Background(),
			testProject,
			testKargoRoleName,
			[]rbacapi.Claim{
				{
					Name:   "sub",
					Values: []string{"fake-sub"},
				},
			},
		)
		require.True(t, kubeerr.IsNotFound(err))
	})

	t.Run("resources aren't manageable", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		_, err := db.RevokeRoleFromUsers(
			context.Background(),
			testProject,
			testKargoRoleName,
			[]rbacapi.Claim{
				{
					Name:   "sub",
					Values: []string{"fake-sub"},
				},
			},
		)
		require.True(t, kubeerr.IsBadRequest(err))
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(map[string]string{
				claimAnnotationKey("sub"):    "bar-sub,foo-sub",
				claimAnnotationKey("email"):  "bar-email,foo-email",
				claimAnnotationKey("groups"): "bar-group,foo-group",
			}),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		kargoRole, err := db.RevokeRoleFromUsers(
			context.Background(),
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
			context.Background(),
			client.ObjectKey{Namespace: testProject, Name: testKargoRoleName},
			sa,
		)
		require.NoError(t, err)
		require.Equal(
			t,
			map[string]string{
				rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				claimAnnotationKey("sub"):    "foo-sub",
			},
			sa.Annotations,
		)
	})
}

func TestUpdate(t *testing.T) {
	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		db := NewKubernetesRolesDatabase(c)
		_, err := db.Update(
			context.Background(),
			&rbacapi.Role{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      testKargoRoleName,
				},
			},
		)
		require.True(t, kubeerr.IsNotFound(err))
	})

	t.Run("resources aren't manageable", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		_, err := db.Update(
			context.Background(),
			&rbacapi.Role{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      testKargoRoleName,
				},
			},
		)
		require.True(t, kubeerr.IsBadRequest(err))
	})

	t.Run("success with Role and RoleBinding creation", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		kargoRole, err := db.Update(
			context.Background(),
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
		err = c.Get(context.Background(), objKey, sa)
		require.NoError(t, err)
		require.Equal(
			t,
			map[string]string{
				rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
				claimAnnotationKey("sub"):    "bar-sub,foo-sub",
				claimAnnotationKey("email"):  "bar-email,foo-email",
				claimAnnotationKey("groups"): "bar-group,foo-group",
			},
			sa.Annotations,
		)
		rb := &rbacv1.RoleBinding{}
		err = c.Get(context.Background(), objKey, rb)
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
		err = c.Get(context.Background(), objKey, role)
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
				claimAnnotationKey("sub"):    "foo-sub,bar-sub",
				claimAnnotationKey("email"):  "foo-email,bar-email",
				claimAnnotationKey("groups"): "foo-group,bar-group",
			}),
			managedRole([]rbacv1.PolicyRule{{
				APIGroups: []string{kargoapi.GroupVersion.Group},
				Resources: []string{"promotions"},
				Verbs:     []string{"get", "list"},
			}}),
			managedRoleBinding(),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		kargoRole, err := db.Update(
			context.Background(),
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
		err = c.Get(context.Background(), objKey, sa)
		require.NoError(t, err)
		require.Equal(
			t,
			map[string]string{
				rbacapi.AnnotationKeyManaged:      rbacapi.AnnotationValueTrue,
				claimAnnotationKey("sub"):         "foo-sub",
				claimAnnotationKey("email"):       "foo-email",
				claimAnnotationKey("groups"):      "foo-group",
				kargoapi.AnnotationKeyDescription: "foo-description",
			},
			sa.Annotations,
		)
		role := &rbacv1.Role{}
		err = c.Get(context.Background(), objKey, role)
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

func TestRemoveFromStringSlice(t *testing.T) {
	t.Run("nil slice", func(t *testing.T) {
		require.Equal(
			t,
			[]string{},
			removeFromStringSlice(nil, []string{"foo"}),
		)
	})

	t.Run("empty slice", func(t *testing.T) {
		require.Equal(
			t,
			[]string{},
			removeFromStringSlice([]string{}, []string{"foo"}),
		)
	})

	t.Run("no match", func(t *testing.T) {
		require.Equal(
			t,
			[]string{"foo", "bar"},
			removeFromStringSlice([]string{"foo", "bar"}, []string{"baz"}),
		)
	})

	t.Run("match", func(t *testing.T) {
		require.Equal(
			t,
			[]string{"foo", "bar"},
			removeFromStringSlice([]string{"foo", "bar", "baz"}, []string{"baz"}),
		)
	})
}

func TestManageableResources(t *testing.T) {
	t.Run("ServiceAccount is not annotated correctly", func(t *testing.T) {
		_, _, err := manageableResources(
			*plainServiceAccount(nil),
			nil,
			nil,
		)
		require.True(t, kubeerr.IsBadRequest(err))
	})

	t.Run("multiple Roles", func(t *testing.T) {
		_, _, err := manageableResources(
			*managedServiceAccount(nil),
			[]rbacv1.Role{{}, {}},
			nil,
		)
		require.True(t, kubeerr.IsBadRequest(err))
	})

	t.Run("single Role not annotated correctly", func(t *testing.T) {
		_, _, err := manageableResources(
			*managedServiceAccount(nil),
			[]rbacv1.Role{*plainRole(nil)},
			nil,
		)
		require.True(t, kubeerr.IsBadRequest(err))
	})

	t.Run("multiple RoleBindings", func(t *testing.T) {
		_, _, err := manageableResources(
			*managedServiceAccount(nil),
			[]rbacv1.Role{*managedRole(nil)},
			[]rbacv1.RoleBinding{{}, {}},
		)
		require.True(t, kubeerr.IsBadRequest(err))
	})

	t.Run("single RoleBinding is not annotated correctly", func(t *testing.T) {
		_, _, err := manageableResources(
			*managedServiceAccount(nil),
			[]rbacv1.Role{*managedRole(nil)},
			[]rbacv1.RoleBinding{*plainRoleBinding()},
		)
		require.True(t, kubeerr.IsBadRequest(err))
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

func claimAnnotationKey(name string) string {
	return rbacapi.AnnotationKeyOIDCClaimNamePrefix + name
}
