package serviceaccounts

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
}

func TestEnsureControllerPermissions(t *testing.T) {
	testCases := []struct {
		name               string
		serviceAccountName string
		project            *kargoapi.Project
		existingRB         *rbacv1.RoleBinding
		expectedRBName     string
	}{
		{
			name:               "Create RoleBinding when it does not exist",
			serviceAccountName: "test-sa",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "project-1",
				},
			},
			expectedRBName: "test-sa-readonly-secrets",
		},
		{
			name:               "Update RoleBinding when it already exists",
			serviceAccountName: "test-sa",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "project-1",
				},
			},
			existingRB: &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-sa-readonly-secrets",
					Namespace: "project-1",
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: rbacv1.GroupName,
					Kind:     "ClusterRole",
					Name:     "kargo-controller-secrets-readonly",
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      "test-sa",
						Namespace: "default",
					},
				},
			},
			expectedRBName: "test-sa-readonly-secrets",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			scheme := runtime.NewScheme()
			err := kargoapi.AddToScheme(scheme)
			require.NoError(t, err)
			err = rbacv1.AddToScheme(scheme)
			require.NoError(t, err)

			kubeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tc.project).
				Build()

			r := newReconciler(kubeClient, ReconcilerConfig{KargoNamespace: "test-ns"})

			if tc.existingRB != nil {
				err = r.createRoleBindingFn(context.Background(), tc.existingRB)
				require.NoError(t, err)
			}

			err = r.ensureControllerPermissions(context.Background(), types.NamespacedName{Name: tc.serviceAccountName})
			require.NoError(t, err)

			roleBinding := &rbacv1.RoleBinding{}
			err = kubeClient.Get(context.Background(), types.NamespacedName{
				Name:      tc.expectedRBName,
				Namespace: tc.project.Name,
			}, roleBinding)

			require.NoError(t, err)
			require.Equal(t, tc.expectedRBName, roleBinding.Name)
			require.Equal(t, "kargo-controller-secrets-readonly", roleBinding.RoleRef.Name)
			require.Equal(t, tc.serviceAccountName, roleBinding.Subjects[0].Name)
		})
	}
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

	// Test case 1: Successful deletion of RoleBinding
	t.Run("Successful deletion of RoleBinding", func(t *testing.T) {
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
	})

	// Test case 2: RoleBinding does not exist
	t.Run("RoleBinding does not exist", func(t *testing.T) {
		kubeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(project).
			Build()

		r := newReconciler(kubeClient, ReconcilerConfig{KargoNamespace: "fake-ns"})

		err = r.removeControllerPermissions(context.Background(), types.NamespacedName{Name: saName})
		require.NoError(t, err)
	})

	// Test case 3: Error during deletion
	t.Run("Error during deletion of RoleBinding", func(t *testing.T) {
		kubeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(roleBinding, project).
			Build()

		r := newReconciler(kubeClient, ReconcilerConfig{KargoNamespace: "fake-ns"})
		r.deleteRoleBindingFn = func(_ context.Context, _ client.Object, _ ...client.DeleteOption) error {
			return fmt.Errorf("simulated error deleting RoleBinding")
		}

		err = r.removeControllerPermissions(context.Background(), types.NamespacedName{Name: saName})
		require.Error(t, err)
		require.Contains(t, err.Error(), "simulated error deleting RoleBinding")
	})
}
