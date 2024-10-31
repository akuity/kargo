package serviceaccounts

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

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

func TestReconcile(t *testing.T) {
	const testProjectName = "fake-project"

	cfg := ReconcilerConfigFromEnv()

	testControllerSARef := types.NamespacedName{
		Name:      "fake-controller",
		Namespace: cfg.KargoNamespace,
	}

	scheme := runtime.NewScheme()
	err := corev1.AddToScheme(scheme)
	require.NoError(t, err)
	err = rbacv1.AddToScheme(scheme)
	require.NoError(t, err)
	err = kargoapi.AddToScheme(scheme)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, client.Client, error)
	}{
		{
			name: "error getting ServiceAccount",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					Get: func(
						context.Context,
						client.WithWatch,
						client.ObjectKey,
						client.Object,
						...client.GetOption,
					) error {
						return fmt.Errorf("something went wrong")
					},
				}).Build(),
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "error getting ServiceAccount")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name:   "ServiceAccount not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "ServiceAccount is being deleted",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testControllerSARef.Name,
						Namespace: testControllerSARef.Namespace,
						Labels: map[string]string{
							controllerServiceAccountLabelKey: controllerServiceAccountLabelValue,
						},
						DeletionTimestamp: &metav1.Time{
							Time: time.Now(),
						},
						Finalizers: []string{
							kargoapi.FinalizerName,
							// This is here just to prevent the SA from being deleted
							// immediately after we remove the Kargo finalizer.
							"fake-finalizer",
						},
					},
				},
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: testProjectName,
					},
				},
				&rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      getRoleBindingName(testControllerSARef.Name),
					},
				},
			).Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)

				// RoleBinding should be deleted
				rb := &rbacv1.RoleBinding{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      getRoleBindingName(testControllerSARef.Name),
						Namespace: testProjectName,
					},
					rb,
				)
				require.Error(t, err)
				require.True(t, apierrors.IsNotFound(err))

				// Finalizer should be removed
				sa := &corev1.ServiceAccount{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      testControllerSARef.Name,
						Namespace: testControllerSARef.Namespace,
					},
					sa,
				)
				require.NoError(t, err)
				require.NotContains(t, sa.Finalizers, kargoapi.FinalizerName)
			},
		},
		{
			name: "ServiceAccount lost controller label",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:       testControllerSARef.Name,
						Namespace:  testControllerSARef.Namespace,
						Finalizers: []string{kargoapi.FinalizerName},
					},
				},
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: testProjectName,
					},
				},
				&rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProjectName,
						Name:      getRoleBindingName(testControllerSARef.Name),
					},
				},
			).Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)

				// RoleBinding should be deleted
				rb := &rbacv1.RoleBinding{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      getRoleBindingName(testControllerSARef.Name),
						Namespace: testProjectName,
					},
					rb,
				)
				require.Error(t, err)
				require.True(t, apierrors.IsNotFound(err))

				// Finalizer should be removed
				sa := &corev1.ServiceAccount{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      testControllerSARef.Name,
						Namespace: testControllerSARef.Namespace,
					},
					sa,
				)
				require.NoError(t, err)
				require.NotContains(t, sa.Finalizers, kargoapi.FinalizerName)
			},
		},
		{
			name: "error removing finalizer",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testControllerSARef.Name,
						Namespace: testControllerSARef.Namespace,
						Labels: map[string]string{
							controllerServiceAccountLabelKey: controllerServiceAccountLabelValue,
						},
						DeletionTimestamp: &metav1.Time{
							Time: time.Now(),
						},
						Finalizers: []string{
							kargoapi.FinalizerName,
						},
					},
				},
				// Deliberately not adding any Projects so that the update that is
				// intercepted will be the SA update that removes the finalizer.
			).WithInterceptorFuncs(interceptor.Funcs{
				Update: func(
					context.Context,
					client.WithWatch,
					client.Object,
					...client.UpdateOption,
				) error {
					return fmt.Errorf("something went wrong")
				},
			}).Build(),
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "error removing finalizer")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error adding finalizer",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testControllerSARef.Name,
						Namespace: testControllerSARef.Namespace,
						Labels: map[string]string{
							controllerServiceAccountLabelKey: controllerServiceAccountLabelValue,
						},
					},
				},
			).WithInterceptorFuncs(interceptor.Funcs{
				Update: func(
					context.Context,
					client.WithWatch,
					client.Object,
					...client.UpdateOption,
				) error {
					return fmt.Errorf("something went wrong")
				},
			}).Build(),
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "error adding finalizer")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "permissions added",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testControllerSARef.Name,
						Namespace: testControllerSARef.Namespace,
						Labels: map[string]string{
							controllerServiceAccountLabelKey: controllerServiceAccountLabelValue,
						},
					},
				},
				&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: testProjectName,
					},
				},
			).Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)

				// Finalizer should be added
				sa := &corev1.ServiceAccount{}
				err = cl.Get(
					context.Background(),
					testControllerSARef,
					sa,
				)
				require.NoError(t, err)
				require.Contains(t, sa.Finalizers, kargoapi.FinalizerName)

				// RoleBinding should be created
				rb := &rbacv1.RoleBinding{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      getRoleBindingName(testControllerSARef.Name),
						Namespace: testProjectName,
					},
					rb,
				)
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r := newReconciler(testCase.client, cfg)
			_, err := r.Reconcile(
				context.Background(),
				ctrl.Request{
					NamespacedName: testControllerSARef,
				},
			)
			testCase.assertions(t, testCase.client, err)
		})
	}
}

func TestEnsureControllerPermissions(t *testing.T) {
	cfg := ReconcilerConfigFromEnv()

	testControllerSARef := types.NamespacedName{
		Name:      "fake-controller",
		Namespace: cfg.KargoNamespace,
	}

	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-project",
		},
	}

	scheme := runtime.NewScheme()
	err := rbacv1.AddToScheme(scheme)
	require.NoError(t, err)
	err = kargoapi.AddToScheme(scheme)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, client.Client, error)
	}{
		{
			name: "error listing Projects",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					List: func(
						context.Context,
						client.WithWatch,
						client.ObjectList,
						...client.ListOption,
					) error {
						return fmt.Errorf("something went wrong")
					},
				}).Build(),
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "error listing Projects")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error creating RoleBinding",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(testProject).
				WithInterceptorFuncs(interceptor.Funcs{
					Create: func(
						context.Context,
						client.WithWatch,
						client.Object,
						...client.CreateOption,
					) error {
						return fmt.Errorf("something went wrong")
					},
				}).Build(),
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "error creating RoleBinding")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "RoleBinding is created when it does not exist",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(testProject).
				Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)
				rb := &rbacv1.RoleBinding{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      getRoleBindingName(testControllerSARef.Name),
						Namespace: testProject.Name,
					},
					rb,
				)
				require.NoError(t, err)
				require.Len(t, rb.Subjects, 1)
				require.Equal(
					t,
					rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     controllerReadSecretsClusterRoleName,
					},
					rb.RoleRef,
				)
				require.Equal(
					t,
					rbacv1.Subject{
						Kind:      "ServiceAccount",
						Name:      testControllerSARef.Name,
						Namespace: testControllerSARef.Namespace,
					},
					rb.Subjects[0],
				)
			},
		},
		{
			name: "RoleBinding is updated when it already exists",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(
					testProject,
					&rbacv1.RoleBinding{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      getRoleBindingName(testControllerSARef.Name),
						},
					},
				).Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)
				rb := &rbacv1.RoleBinding{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      getRoleBindingName(testControllerSARef.Name),
						Namespace: testProject.Name,
					},
					rb,
				)
				require.NoError(t, err)
				require.Len(t, rb.Subjects, 1)
				require.Equal(
					t,
					rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     controllerReadSecretsClusterRoleName,
					},
					rb.RoleRef,
				)
				require.Equal(
					t,
					rbacv1.Subject{
						Kind:      "ServiceAccount",
						Name:      testControllerSARef.Name,
						Namespace: testControllerSARef.Namespace,
					},
					rb.Subjects[0],
				)
			},
		},
		{
			name: "error updating existing RoleBinding",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(
					testProject,
					&rbacv1.RoleBinding{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      getRoleBindingName(testControllerSARef.Name),
						},
					},
				).
				WithInterceptorFuncs(interceptor.Funcs{
					Update: func(
						context.Context,
						client.WithWatch,
						client.Object,
						...client.UpdateOption,
					) error {
						return fmt.Errorf("something went wrong")
					},
				}).Build(),
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "error updating existing RoleBinding")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r := newReconciler(testCase.client, cfg)
			err = r.ensureControllerPermissions(context.Background(), testControllerSARef)
			testCase.assertions(t, testCase.client, err)
		})
	}
}

func TestRemoveControllerPermissions(t *testing.T) {
	cfg := ReconcilerConfigFromEnv()

	testControllerSARef := types.NamespacedName{
		Name:      "fake-controller",
		Namespace: cfg.KargoNamespace,
	}

	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-project",
		},
	}

	scheme := runtime.NewScheme()
	err := rbacv1.AddToScheme(scheme)
	require.NoError(t, err)
	err = kargoapi.AddToScheme(scheme)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, client.Client, error)
	}{
		{
			name: "error listing Projects",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					List: func(
						context.Context,
						client.WithWatch,
						client.ObjectList,
						...client.ListOption,
					) error {
						return fmt.Errorf("something went wrong")
					},
				}).Build(),
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "error listing Projects")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error deleting RoleBinding",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(testProject).
				WithInterceptorFuncs(interceptor.Funcs{
					Delete: func(
						context.Context,
						client.WithWatch,
						client.Object,
						...client.DeleteOption,
					) error {
						return fmt.Errorf("something went wrong")
					},
				}).Build(),
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "error deleting RoleBinding")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "RoleBinding is deleted when it exists",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(
					testProject,
					&rbacv1.RoleBinding{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      getRoleBindingName(testControllerSARef.Name),
						},
					},
				).
				Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)
				rb := &rbacv1.RoleBinding{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      getRoleBindingName(testControllerSARef.Name),
						Namespace: testProject.Name,
					},
					rb,
				)
				require.Error(t, err)
				require.True(t, apierrors.IsNotFound(err))
			},
		},
		{
			name: "no error when RoleBinding does not exist",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(testProject).
				Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)
				rb := &rbacv1.RoleBinding{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      getRoleBindingName(testControllerSARef.Name),
						Namespace: testProject.Name,
					},
					rb,
				)
				require.Error(t, err)
				require.True(t, apierrors.IsNotFound(err))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r := newReconciler(testCase.client, cfg)
			err = r.removeControllerPermissions(context.Background(), testControllerSARef)
			testCase.assertions(t, testCase.client, err)
		})
	}
}
