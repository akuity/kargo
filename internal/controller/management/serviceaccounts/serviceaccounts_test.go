package serviceaccounts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewReconciler(t *testing.T) {
	testCfg := ReconcilerConfig{
		KargoNamespace: "fake-ns",
	}
	kubeClient := fake.NewClientBuilder().Build()
	r := newReconciler(kubeClient, testCfg)
	require.Equal(t, testCfg, r.cfg)
	require.NotNil(t, kubeClient, r.client)

	require.NotNil(t, r.getServiceAccountFn)
	require.NotNil(t, r.updateServiceAccountFn)
	require.NotNil(t, r.createRoleBindingFn)
	require.NotNil(t, r.updateRoleBindingFn)
	require.NotNil(t, r.deleteRoleBindingFn)
	require.NotNil(t, r.listProjectFn)
	require.NotNil(t, r.ensureControllerPermissionsFn)
	require.NotNil(t, r.removeControllerPermissionsFn)
}

func TestEnsureControllerPermissions(t *testing.T) {
	saName := "test-sa"
	saNamespace := "default"
	projectName := "project-1"

	project := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: projectName,
		},
	}

	// Fake client with ServiceAccount and Project.
	scheme := runtime.NewScheme()
	err := kargoapi.AddToScheme(scheme)
	require.NoError(t, err)
	err = rbacv1.AddToScheme(scheme)
	require.NoError(t, err)

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project).
		Build()

	r := newReconciler(kubeClient, ReconcilerConfig{KargoNamespace: saNamespace})

	err = r.ensureControllerPermissions(context.Background(), types.NamespacedName{Name: saName})
	require.NoError(t, err)

	roleBinding := &rbacv1.RoleBinding{}
	err = kubeClient.Get(context.Background(), types.NamespacedName{
		Name:      saName + "-readonly-secrets",
		Namespace: projectName,
	}, roleBinding)
	require.NoError(t, err)
	require.Equal(t, saName+"-readonly-secrets", roleBinding.Name)
	require.Equal(t, "kargo-controller-secrets-readonly", roleBinding.RoleRef.Name)
	require.Equal(t, saName, roleBinding.Subjects[0].Name)
}

func TestRemoveControllerPermissions(t *testing.T) {
	saName := "test-sa"
	projectName := "project-1"
	roleBindingName := saName + "-readonly-secrets"

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: projectName,
		},
	}
	project := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: projectName,
		},
	}

	scheme := runtime.NewScheme()
	err := kargoapi.AddToScheme(scheme)
	require.NoError(t, err)
	err = rbacv1.AddToScheme(scheme)
	require.NoError(t, err)

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(roleBinding, project).
		Build()

	r := newReconciler(kubeClient, ReconcilerConfig{KargoNamespace: "fake-ns"})

	err = r.removeControllerPermissions(context.Background(), types.NamespacedName{Name: saName})
	require.NoError(t, err)

	err = kubeClient.Get(context.Background(), types.NamespacedName{
		Name:      roleBindingName,
		Namespace: projectName,
	}, roleBinding)
	require.Error(t, err)
	require.True(t, apierrors.IsNotFound(err))
}
