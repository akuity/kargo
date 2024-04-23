package rbac

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
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
		testKargoRole := &svcv1alpha1.Role{
			Project: testProject,
			Name:    testKargoRoleName,
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
		testKargoRole := &svcv1alpha1.Role{
			Project: testProject,
			Name:    testKargoRoleName,
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
		testKargoRole := &svcv1alpha1.Role{
			Project: testProject,
			Name:    testKargoRoleName,
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
		testKargoRole := &svcv1alpha1.Role{
			Project: testProject,
			Name:    testKargoRoleName,
			Subs:    []string{"foo-sub", "bar-sub"},
			Emails:  []string{"foo-email", "bar-email"},
			Groups:  []string{"foo-group", "bar-group"},
			Rules: []*rbacv1.PolicyRule{
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
			client.ObjectKey{Namespace: testKargoRole.Project, Name: testKargoRole.Name},
			sa,
		)
		require.NoError(t, err)
		require.Equal(
			t,
			map[string]string{
				kargoapi.AnnotationKeyManaged:      kargoapi.AnnotationValueTrue,
				kargoapi.AnnotationKeyOIDCSubjects: "bar-sub,foo-sub",
				kargoapi.AnnotationKeyOIDCEmails:   "bar-email,foo-email",
				kargoapi.AnnotationKeyOIDCGroups:   "bar-group,foo-group",
			},
			sa.Annotations,
		)

		roleBinding := &rbacv1.RoleBinding{}
		err = c.Get(
			context.Background(),
			client.ObjectKey{Namespace: testKargoRole.Project, Name: testKargoRole.Name},
			roleBinding,
		)
		require.NoError(t, err)
		require.Equal(
			t,
			map[string]string{
				kargoapi.AnnotationKeyManaged: kargoapi.AnnotationValueTrue,
			},
			roleBinding.Annotations,
		)
		require.Equal(
			t,
			[]rbacv1.Subject{{
				Kind:      rbacv1.ServiceAccountKind,
				Namespace: testKargoRole.Project,
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
		require.True(t, kubeerr.IsConflict(err))
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

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			plainServiceAccount(map[string]string{
				kargoapi.AnnotationKeyOIDCSubjects: "foo-sub,bar-sub",
				kargoapi.AnnotationKeyOIDCEmails:   "foo-email,bar-email",
				kargoapi.AnnotationKeyOIDCGroups:   "foo-group,bar-group",
			}),
			plainRole([]rbacv1.PolicyRule{{
				APIGroups: []string{kargoapi.GroupVersion.Group},
				Resources: []string{"stages", "promotions"},
				Verbs:     []string{"list", "get"},
			}}),
			plainRoleBinding(),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		kargoRole, err := db.Get(context.Background(), testProject, testKargoRoleName)
		require.NoError(t, err)
		// Do not factor creation timestamp into the comparison
		kargoRole.CreationTimestamp = nil
		require.Equal(
			t,
			&svcv1alpha1.Role{
				Project: testProject,
				Name:    testKargoRoleName,
				Subs:    []string{"bar-sub", "foo-sub"},
				Emails:  []string{"bar-email", "foo-email"},
				Groups:  []string{"bar-group", "foo-group"},
				Rules: []*rbacv1.PolicyRule{
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
			&svcv1alpha1.ResourceDetails{
				ResourceGroup: "fake-group",
				ResourceType:  "fake-resource-type",
				Verbs:         []string{"get", "list"},
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
			&svcv1alpha1.ResourceDetails{
				ResourceGroup: "fake-group",
				ResourceType:  "fake-resource-type",
				Verbs:         []string{"get", "list"},
			},
		)
		require.True(t, kubeerr.IsConflict(err))
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
			&svcv1alpha1.ResourceDetails{
				ResourceGroup: kargoapi.GroupVersion.Group,
				ResourceType:  "stages",
				Verbs:         []string{"get", "list"},
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
			&svcv1alpha1.ResourceDetails{
				ResourceGroup: kargoapi.GroupVersion.Group,
				ResourceType:  "stages",
				Verbs:         []string{"get", "list"},
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
			&svcv1alpha1.UserClaims{
				Subs: []string{"fake-sub"},
			},
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
			&svcv1alpha1.UserClaims{
				Subs: []string{"fake-sub"},
			},
		)
		require.True(t, kubeerr.IsConflict(err))
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(map[string]string{
				kargoapi.AnnotationKeyOIDCSubjects: "foo-sub",
				kargoapi.AnnotationKeyOIDCEmails:   "foo-email",
				kargoapi.AnnotationKeyOIDCGroups:   "foo-group",
			}),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		kargoRole, err := db.GrantRoleToUsers(
			context.Background(),
			testProject,
			testKargoRoleName,
			&svcv1alpha1.UserClaims{
				Subs:   []string{"bar-sub"},
				Emails: []string{"bar-email"},
				Groups: []string{"bar-group"},
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
				kargoapi.AnnotationKeyManaged:      kargoapi.AnnotationValueTrue,
				kargoapi.AnnotationKeyOIDCSubjects: "bar-sub,foo-sub",
				kargoapi.AnnotationKeyOIDCEmails:   "bar-email,foo-email",
				kargoapi.AnnotationKeyOIDCGroups:   "bar-group,foo-group",
			},
			sa.Annotations,
		)
	})
}

func TestList(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		managedServiceAccount(map[string]string{
			kargoapi.AnnotationKeyOIDCSubjects: "foo-sub,bar-sub",
			kargoapi.AnnotationKeyOIDCEmails:   "foo-email,bar-email",
			kargoapi.AnnotationKeyOIDCGroups:   "foo-group,bar-group",
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
	for _, kargoRole := range kargoRoles {
		kargoRole.CreationTimestamp = nil
	}
	require.Equal(
		t,
		[]*svcv1alpha1.Role{{
			Project: testProject,
			Name:    testKargoRoleName,
			Subs:    []string{"bar-sub", "foo-sub"},
			Emails:  []string{"bar-email", "foo-email"},
			Groups:  []string{"bar-group", "foo-group"},
			Rules: []*rbacv1.PolicyRule{
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
}

func TestRevokePermissionsFromRole(t *testing.T) {
	t.Run("ServiceAccount not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		db := NewKubernetesRolesDatabase(c)
		_, err := db.RevokePermissionsFromRole(
			context.Background(),
			testProject,
			testKargoRoleName,
			&svcv1alpha1.ResourceDetails{
				ResourceGroup: "fake-group",
				ResourceType:  "fake-resource-type",
				Verbs:         []string{"get", "list"},
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
			&svcv1alpha1.ResourceDetails{
				ResourceGroup: "fake-group",
				ResourceType:  "fake-resource-type",
				Verbs:         []string{"get", "list"},
			},
		)
		require.True(t, kubeerr.IsConflict(err))
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
			&svcv1alpha1.ResourceDetails{
				ResourceGroup: "fake-group",
				ResourceType:  "fake-resource-type",
				Verbs:         []string{"get", "list"},
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
			&svcv1alpha1.ResourceDetails{
				ResourceGroup: kargoapi.GroupVersion.Group,
				ResourceType:  "stages",
				Verbs:         []string{"get", "list"},
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
			&svcv1alpha1.UserClaims{
				Subs: []string{"fake-sub"},
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
			&svcv1alpha1.UserClaims{
				Subs: []string{"fake-sub"},
			},
		)
		require.True(t, kubeerr.IsConflict(err))
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(map[string]string{
				kargoapi.AnnotationKeyOIDCSubjects: "bar-sub,foo-sub",
				kargoapi.AnnotationKeyOIDCEmails:   "bar-email,foo-email",
				kargoapi.AnnotationKeyOIDCGroups:   "bar-group,foo-group",
			}),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		kargoRole, err := db.RevokeRoleFromUsers(
			context.Background(),
			testProject,
			testKargoRoleName,
			&svcv1alpha1.UserClaims{
				Subs:   []string{"bar-sub"},
				Emails: []string{"foo-email", "bar-email"},
				Groups: []string{"foo-group", "bar-group"},
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
				kargoapi.AnnotationKeyManaged:      kargoapi.AnnotationValueTrue,
				kargoapi.AnnotationKeyOIDCSubjects: "foo-sub",
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
			&svcv1alpha1.Role{
				Project: testProject,
				Name:    testKargoRoleName,
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
			&svcv1alpha1.Role{
				Project: testProject,
				Name:    testKargoRoleName,
			},
		)
		require.True(t, kubeerr.IsConflict(err))
	})

	t.Run("success with Role and RoleBinding creation", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			managedServiceAccount(nil),
		).Build()
		db := NewKubernetesRolesDatabase(c)
		kargoRole, err := db.Update(
			context.Background(),
			&svcv1alpha1.Role{
				Project: testProject,
				Name:    testKargoRoleName,
				Subs:    []string{"foo-sub", "bar-sub"},
				Emails:  []string{"foo-email", "bar-email"},
				Groups:  []string{"foo-group", "bar-group"},
				Rules: []*rbacv1.PolicyRule{{
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
				kargoapi.AnnotationKeyManaged:      kargoapi.AnnotationValueTrue,
				kargoapi.AnnotationKeyOIDCSubjects: "bar-sub,foo-sub",
				kargoapi.AnnotationKeyOIDCEmails:   "bar-email,foo-email",
				kargoapi.AnnotationKeyOIDCGroups:   "bar-group,foo-group",
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
			managedServiceAccount(nil),
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
			&svcv1alpha1.Role{
				Project: testProject,
				Name:    testKargoRoleName,
				Subs:    []string{"foo-sub", "bar-sub"},
				Emails:  []string{"foo-email", "bar-email"},
				Groups:  []string{"foo-group", "bar-group"},
				Rules: []*rbacv1.PolicyRule{{
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
				kargoapi.AnnotationKeyManaged:      kargoapi.AnnotationValueTrue,
				kargoapi.AnnotationKeyOIDCSubjects: "bar-sub,foo-sub",
				kargoapi.AnnotationKeyOIDCEmails:   "bar-email,foo-email",
				kargoapi.AnnotationKeyOIDCGroups:   "bar-group,foo-group",
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

func TestDedupeStringSlice(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		require.Equal(
			t,
			[]string{},
			dedupeStringSlice([]string{}),
		)
	})

	t.Run("no duplicates", func(t *testing.T) {
		require.Equal(
			t,
			[]string{"foo", "bar"},
			dedupeStringSlice([]string{"foo", "bar"}),
		)
	})

	t.Run("with duplicates", func(t *testing.T) {
		require.Equal(
			t,
			[]string{"foo", "bar"},
			dedupeStringSlice([]string{"foo", "bar", "foo", "bar"}),
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
		require.True(t, kubeerr.IsConflict(err))
	})

	t.Run("multiple Roles", func(t *testing.T) {
		_, _, err := manageableResources(
			*managedServiceAccount(nil),
			[]rbacv1.Role{{}, {}},
			nil,
		)
		require.True(t, kubeerr.IsConflict(err))
	})

	t.Run("single Role not annotated correctly", func(t *testing.T) {
		_, _, err := manageableResources(
			*managedServiceAccount(nil),
			[]rbacv1.Role{*plainRole(nil)},
			nil,
		)
		require.True(t, kubeerr.IsConflict(err))
	})

	t.Run("multiple RoleBindings", func(t *testing.T) {
		_, _, err := manageableResources(
			*managedServiceAccount(nil),
			[]rbacv1.Role{*managedRole(nil)},
			[]rbacv1.RoleBinding{{}, {}},
		)
		require.True(t, kubeerr.IsConflict(err))
	})

	t.Run("single RoleBinding is not annotated correctly", func(t *testing.T) {
		_, _, err := manageableResources(
			*managedServiceAccount(nil),
			[]rbacv1.Role{*managedRole(nil)},
			[]rbacv1.RoleBinding{*plainRoleBinding()},
		)
		require.True(t, kubeerr.IsConflict(err))
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
	annotations[kargoapi.AnnotationKeyManaged] = kargoapi.AnnotationValueTrue

	return metav1.ObjectMeta{
		Namespace:   testProject,
		Name:        testKargoRoleName,
		Annotations: annotations,
	}
}
