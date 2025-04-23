package projects

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/conditions"
)

func TestNewReconciler(t *testing.T) {
	testCfg := ReconcilerConfig{}
	r := newReconciler(fake.NewClientBuilder().Build(), testCfg)
	require.Equal(t, testCfg, r.cfg)
	require.NotNil(t, r.client)
	require.NotNil(t, r.getProjectFn)
	require.NotNil(t, r.reconcileFn)
	require.NotNil(t, r.ensureNamespaceFn)
	require.NotNil(t, r.patchProjectStatusFn)
	require.NotNil(t, r.getNamespaceFn)
	require.NotNil(t, r.createNamespaceFn)
	require.NotNil(t, r.patchOwnerReferencesFn)
	require.NotNil(t, r.ensureFinalizerFn)
	require.NotNil(t, r.ensureAPIAdminPermissionsFn)
	require.NotNil(t, r.ensureControllerPermissionsFn)
	require.NotNil(t, r.ensureDefaultProjectRolesFn)
	require.NotNil(t, r.createServiceAccountFn)
	require.NotNil(t, r.createRoleFn)
	require.NotNil(t, r.createRoleBindingFn)
}

func TestReconciler_Reconcile(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*testing.T, ctrl.Result, error)
	}{
		{
			name: "project not found",
			reconciler: &reconciler{
				getProjectFn: func(
					context.Context,
					client.Client,
					string,
				) (*kargoapi.Project, error) {
					return nil, nil
				},
			},
			assertions: func(t *testing.T, result ctrl.Result, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					ctrl.Result{
						RequeueAfter: 0,
					},
					result,
				)
			},
		},
		{
			name: "error finding project",
			reconciler: &reconciler{
				getProjectFn: func(
					context.Context,
					client.Client,
					string,
				) (*kargoapi.Project, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ ctrl.Result, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "project is being deleted",
			reconciler: &reconciler{
				getProjectFn: func(
					context.Context,
					client.Client,
					string,
				) (*kargoapi.Project, error) {
					return &kargoapi.Project{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: &metav1.Time{},
						},
					}, nil
				},
			},
			assertions: func(t *testing.T, result ctrl.Result, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					ctrl.Result{
						RequeueAfter: 0,
					},
					result,
				)
			},
		},
		{
			name: "error running internal reconcile",
			reconciler: &reconciler{
				getProjectFn: func(
					context.Context,
					client.Client,
					string,
				) (*kargoapi.Project, error) {
					return &kargoapi.Project{}, nil
				},
				reconcileFn: func(
					context.Context,
					*kargoapi.Project,
				) (kargoapi.ProjectStatus, bool, error) {
					return kargoapi.ProjectStatus{}, false, errors.New("something went wrong")
				},
				patchProjectStatusFn: func(
					_ context.Context,
					_ *kargoapi.Project,
					_ kargoapi.ProjectStatus,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, _ ctrl.Result, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				getProjectFn: func(
					context.Context,
					client.Client,
					string,
				) (*kargoapi.Project, error) {
					return &kargoapi.Project{}, nil
				},
				reconcileFn: func(
					context.Context,
					*kargoapi.Project,
				) (kargoapi.ProjectStatus, bool, error) {
					return kargoapi.ProjectStatus{}, false, nil
				},
				patchProjectStatusFn: func(
					_ context.Context,
					_ *kargoapi.Project,
					_ kargoapi.ProjectStatus,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, _ ctrl.Result, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			res, err := testCase.reconciler.Reconcile(context.Background(), ctrl.Request{})
			testCase.assertions(t, res, err)
		})
	}
}

func TestReconciler_reconcile(*testing.T) {
	// TODO: Implement this test
}

func TestReconciler_initializeProject(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(
			t *testing.T,
			status kargoapi.ProjectStatus,
			initialized bool,
			err error,
		)
	}{
		{
			name: "error ensuring namespace",
			reconciler: &reconciler{
				ensureNamespaceFn: func(context.Context, *kargoapi.Project) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, initialized bool, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.False(t, initialized)

				// Still initializing because retry could succeed
				require.Len(t, status.Conditions, 2)

				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "NamespaceInitializationFailed", readyCondition.Reason)

				reconcilingCondition := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcilingCondition)
				require.Equal(t, metav1.ConditionTrue, reconcilingCondition.Status)
				require.Equal(t, "Initializing", reconcilingCondition.Reason)
			},
		},
		{
			name: "fatal error ensuring namespace",
			reconciler: &reconciler{
				ensureNamespaceFn: func(context.Context, *kargoapi.Project) error {
					return errProjectNamespaceExists
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, initialized bool, err error) {
				require.True(t, errors.Is(err, errProjectNamespaceExists))
				require.False(t, initialized)

				// Failed because retry cannot possibly succeed
				require.Len(t, status.Conditions, 2)

				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "NamespaceInitializationFailed", readyCondition.Reason)

				stalledCondition := conditions.Get(&status, kargoapi.ConditionTypeStalled)
				require.NotNil(t, stalledCondition)
				require.Equal(t, metav1.ConditionTrue, stalledCondition.Status)
				require.Equal(t, "ExistingNamespaceMissingLabel", stalledCondition.Reason)
			},
		},
		{
			name: "error ensuring project admin permissions",
			reconciler: &reconciler{
				ensureNamespaceFn: func(context.Context, *kargoapi.Project) error {
					return nil
				},
				ensureAPIAdminPermissionsFn: func(
					context.Context,
					*kargoapi.Project,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, initialized bool, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.False(t, initialized)

				// Still initializing because retry could succeed
				require.Len(t, status.Conditions, 2)

				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "PermissionsInitializationFailed", readyCondition.Reason)

				reconcilingCondition := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcilingCondition)
				require.Equal(t, metav1.ConditionTrue, reconcilingCondition.Status)
				require.Equal(t, "Initializing", reconcilingCondition.Reason)
			},
		},
		{
			name: "error ensuring controller permissions",
			reconciler: &reconciler{
				cfg: ReconcilerConfig{
					ManageControllerRoleBindings: true,
				},
				ensureNamespaceFn: func(context.Context, *kargoapi.Project) error {
					return nil
				},
				ensureAPIAdminPermissionsFn: func(
					context.Context,
					*kargoapi.Project,
				) error {
					return nil
				},
				ensureControllerPermissionsFn: func(
					context.Context,
					*kargoapi.Project,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ kargoapi.ProjectStatus, initialized bool, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.False(t, initialized)
			},
		},
		{
			name: "error ensuring default project roles",
			reconciler: &reconciler{
				ensureNamespaceFn: func(context.Context, *kargoapi.Project) error {
					return nil
				},
				ensureAPIAdminPermissionsFn: func(
					context.Context,
					*kargoapi.Project,
				) error {
					return nil
				},
				ensureDefaultProjectRolesFn: func(
					context.Context,
					*kargoapi.Project,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, initialized bool, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.False(t, initialized)

				// Still initializing because retry could succeed
				require.Len(t, status.Conditions, 2)

				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "RolesInitializationFailed", readyCondition.Reason)

				reconcilingCondition := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcilingCondition)
				require.Equal(t, metav1.ConditionTrue, reconcilingCondition.Status)
				require.Equal(t, "Initializing", reconcilingCondition.Reason)
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				ensureNamespaceFn: func(context.Context, *kargoapi.Project) error {
					return nil
				},
				ensureAPIAdminPermissionsFn: func(
					context.Context,
					*kargoapi.Project,
				) error {
					return nil
				},
				ensureDefaultProjectRolesFn: func(
					context.Context,
					*kargoapi.Project,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, initialized bool, err error) {
				require.NoError(t, err)
				require.True(t, initialized)

				require.Len(t, status.Conditions, 1)

				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionTrue, readyCondition.Status)
				require.Equal(t, "Initialized", readyCondition.Reason)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			status, initialized, err := testCase.reconciler.initializeProject(
				context.Background(),
				&kargoapi.Project{
					Status: kargoapi.ProjectStatus{},
				},
			)
			testCase.assertions(t, status, initialized, err)
		})
	}
}

func TestReconciler_ensureNamespace(t *testing.T) {
	testCases := []struct {
		name       string
		project    *kargoapi.Project
		reconciler *reconciler
		assertions func(*testing.T, error)
	}{
		{
			name:    "error getting namespace",
			project: &kargoapi.Project{},
			reconciler: &reconciler{
				getNamespaceFn: func(
					context.Context,
					types.NamespacedName,
					client.Object,
					...client.GetOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error getting namespace")
			},
		},
		{
			name:    "namespace exists and is not labeled as a project namespace",
			project: &kargoapi.Project{},
			reconciler: &reconciler{
				getNamespaceFn: func(
					context.Context,
					types.NamespacedName,
					client.Object,
					...client.GetOption,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, errProjectNamespaceExists))
			},
		},
		{
			name: "namespace exists, is labeled as a project namespace, and is " +
				"already owned by the project",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					UID: types.UID("fake-uid"),
				},
			},
			reconciler: &reconciler{
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns, ok := obj.(*corev1.Namespace)
					require.True(t, ok)
					ns.Labels = map[string]string{
						kargoapi.ProjectLabelKey: kargoapi.LabelTrueValue,
					}
					ns.OwnerReferences = []metav1.OwnerReference{{
						UID: "fake-uid",
					}}
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "namespace exists, is labeled as a project namespace, and is " +
				"NOT already owned by the project; error ensuring finalizer",
			project: &kargoapi.Project{},
			reconciler: &reconciler{
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns, ok := obj.(*corev1.Namespace)
					require.True(t, ok)
					ns.Labels = map[string]string{
						kargoapi.ProjectLabelKey: kargoapi.LabelTrueValue,
					}
					return nil
				},
				ensureFinalizerFn: func(
					context.Context,
					client.Client,
					client.Object,
				) (bool, error) {
					return false, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error ensuring finalizer on namespace")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "namespace exists, is labeled as a project namespace, and is " +
				"NOT already owned by the project; error patching it",
			project: &kargoapi.Project{},
			reconciler: &reconciler{
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns, ok := obj.(*corev1.Namespace)
					require.True(t, ok)
					ns.Labels = map[string]string{
						kargoapi.ProjectLabelKey: kargoapi.LabelTrueValue,
					}
					return nil
				},
				ensureFinalizerFn: func(
					context.Context,
					client.Client,
					client.Object,
				) (bool, error) {
					return false, nil
				},
				patchOwnerReferencesFn: func(
					context.Context,
					client.Client,
					client.Object,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error patching namespace")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "namespace exists, is labeled as a project namespace, and is " +
				"NOT already owned by the project; success",
			project: &kargoapi.Project{},
			reconciler: &reconciler{
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns, ok := obj.(*corev1.Namespace)
					require.True(t, ok)
					ns.Labels = map[string]string{
						kargoapi.ProjectLabelKey: kargoapi.LabelTrueValue,
					}
					return nil
				},
				ensureFinalizerFn: func(
					context.Context,
					client.Client,
					client.Object,
				) (bool, error) {
					return false, nil
				},
				patchOwnerReferencesFn: func(
					context.Context,
					client.Client,
					client.Object,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:    "namespace does not exist; error creating it",
			project: &kargoapi.Project{},
			reconciler: &reconciler{
				getNamespaceFn: func(
					context.Context,
					types.NamespacedName,
					client.Object,
					...client.GetOption,
				) error {
					return apierrors.NewNotFound(schema.GroupResource{}, "")
				},
				createNamespaceFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error creating namespace")
			},
		},
		{
			name: "namespace does not exist; success creating it",
			project: &kargoapi.Project{
				Status: kargoapi.ProjectStatus{
					Phase: kargoapi.ProjectPhaseInitializing,
				},
			},
			reconciler: &reconciler{
				getNamespaceFn: func(
					context.Context,
					types.NamespacedName,
					client.Object,
					...client.GetOption,
				) error {
					return apierrors.NewNotFound(schema.GroupResource{}, "")
				},
				createNamespaceFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.reconciler.ensureNamespace(context.Background(), testCase.project),
			)
		})
	}
}

func TestReconciler_ensureAPIAdminPermissions(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*testing.T, error)
	}{
		{
			name: "error creating role binding",
			reconciler: &reconciler{
				createRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error creating RoleBinding")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "role binding already exists",
			reconciler: &reconciler{
				createRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return apierrors.NewAlreadyExists(schema.GroupResource{}, "")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "success creating role binding",
			reconciler: &reconciler{
				createRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.reconciler.ensureAPIAdminPermissions(
					context.Background(),
					&kargoapi.Project{},
				),
			)
		})
	}
}

func TestReconciler_ensureControllerPermissions(t *testing.T) {
	cfg := ReconcilerConfigFromEnv()

	testControllerSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-controller",
			Namespace: cfg.KargoNamespace,
			Labels: map[string]string{
				controllerServiceAccountLabelKey: controllerServiceAccountLabelValue,
			},
			Finalizers: []string{kargoapi.FinalizerName},
		},
	}

	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-project",
		},
	}

	scheme := runtime.NewScheme()
	err := corev1.AddToScheme(scheme)
	require.NoError(t, err)
	err = rbacv1.AddToScheme(scheme)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, client.Client, error)
	}{
		{
			name: "error listing ServiceAccounts",
			client: fake.NewClientBuilder().WithScheme(scheme).
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
				require.ErrorContains(t, err, "error listing controller ServiceAccounts")
				require.ErrorContains(t, err, "something went wrong")
			},
		},

		{
			name: "error adding finalizer",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(
					&corev1.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-controller",
							Namespace: cfg.KargoNamespace,
							Labels: map[string]string{
								controllerServiceAccountLabelKey: controllerServiceAccountLabelValue,
							},
							// Lacks an existing finalizer
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
				require.ErrorContains(t, err, "error adding finalizer to controller ServiceAccount")
			},
		},
		{
			name: "finalizer is added when it does not exist",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(
					&corev1.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-controller",
							Namespace: cfg.KargoNamespace,
							Labels: map[string]string{
								controllerServiceAccountLabelKey: controllerServiceAccountLabelValue,
							},
							// Lacks an existing finalizer
						},
					},
				).Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)
				sa := &corev1.ServiceAccount{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      "fake-controller",
						Namespace: cfg.KargoNamespace,
					},
					sa,
				)
				require.NoError(t, err)
				require.Contains(t, sa.Finalizers, kargoapi.FinalizerName)
			},
		},
		{
			name: "error creating RoleBinding",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(testControllerSA).
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
				WithObjects(testControllerSA).
				Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)
				rb := &rbacv1.RoleBinding{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      getRoleBindingName(testControllerSA.Name),
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
						Name:      testControllerSA.Name,
						Namespace: testControllerSA.Namespace,
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
					testControllerSA,
					&rbacv1.RoleBinding{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      getRoleBindingName(testControllerSA.Name),
						},
					},
				).Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)
				rb := &rbacv1.RoleBinding{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      getRoleBindingName(testControllerSA.Name),
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
						Name:      testControllerSA.Name,
						Namespace: testControllerSA.Namespace,
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
					testControllerSA,
					&rbacv1.RoleBinding{
						ObjectMeta: metav1.ObjectMeta{
							Name:      getRoleBindingName(testControllerSA.Name),
							Namespace: testProject.Name,
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
			err = r.ensureControllerPermissions(context.Background(), testProject)
			testCase.assertions(t, testCase.client, err)
		})
	}
}

func TestReconciler_ensureDefaultProjectRoles(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*testing.T, error)
	}{
		{
			name: "error creating ServiceAccount",
			reconciler: &reconciler{
				createServiceAccountFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error creating ServiceAccount")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error creating Role",
			reconciler: &reconciler{
				createServiceAccountFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return apierrors.NewAlreadyExists(schema.GroupResource{}, "")
				},
				createRoleFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error creating Role")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error creating RoleBinding",
			reconciler: &reconciler{
				createServiceAccountFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return apierrors.NewAlreadyExists(schema.GroupResource{}, "")
				},
				createRoleFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return apierrors.NewAlreadyExists(schema.GroupResource{}, "")
				},
				createRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error creating RoleBinding")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				createServiceAccountFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
				createRoleFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
				createRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.reconciler.ensureDefaultProjectRoles(
					context.Background(),
					&kargoapi.Project{},
				),
			)
		})
	}
}

func TestReconciler_migratePhaseToConditions(t *testing.T) {
	tests := []struct {
		name       string
		project    *kargoapi.Project
		assertions func(t *testing.T, status kargoapi.ProjectStatus, ok bool)
	}{
		{
			name: "Empty Phase",
			project: &kargoapi.Project{
				Status: kargoapi.ProjectStatus{
					Phase:   "",
					Message: "Some message",
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, ok bool) {
				require.True(t, ok)
				require.Empty(t, status.Phase)   // nolint:staticcheck
				require.Empty(t, status.Message) // nolint:staticcheck
				require.Empty(t, status.Conditions)
			},
		},
		{
			name: "Initializing Phase",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Status: kargoapi.ProjectStatus{
					Phase: kargoapi.ProjectPhaseInitializing,
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, ok bool) {
				require.True(t, ok)
				require.Empty(t, status.Phase)   // nolint:staticcheck
				require.Empty(t, status.Message) // nolint:staticcheck
				require.Len(t, status.Conditions, 2)

				reconcilingCondition := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcilingCondition)
				require.Equal(t, metav1.ConditionTrue, reconcilingCondition.Status)
				require.Equal(t, "Initializing", reconcilingCondition.Reason)
				require.Equal(
					t,
					"Creating project namespace and initializing permissions",
					reconcilingCondition.Message,
				)
				require.Equal(t, int64(1), reconcilingCondition.ObservedGeneration)

				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "Initializing", readyCondition.Reason)
				require.Equal(
					t,
					"Creating project namespace and initializing permissions",
					readyCondition.Message,
				)
				require.Equal(t, int64(1), readyCondition.ObservedGeneration)
			},
		},
		{
			name: "InitializationFailed Phase",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-project",
					Generation: 2,
				},
				Status: kargoapi.ProjectStatus{
					Phase: kargoapi.ProjectPhaseInitializationFailed,
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, ok bool) {
				require.True(t, ok)
				require.Empty(t, status.Phase)   // nolint:staticcheck
				require.Empty(t, status.Message) // nolint:staticcheck
				require.Len(t, status.Conditions, 2)

				stalledCondition := conditions.Get(&status, kargoapi.ConditionTypeStalled)
				require.NotNil(t, stalledCondition)
				require.Equal(t, metav1.ConditionTrue, stalledCondition.Status)
				require.Equal(t, "ExistingNamespaceMissingLabel", stalledCondition.Reason)
				require.Contains(
					t,
					stalledCondition.Message,
					"already exists but is not labeled as a Project namespace using label",
				)
				require.Equal(t, int64(2), stalledCondition.ObservedGeneration)

				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "NamespaceInitializationFailed", readyCondition.Reason)
				require.Contains(
					t,
					readyCondition.Message,
					"namespace already exists and is not labeled as a Project namespace",
				)
				require.Equal(t, int64(2), readyCondition.ObservedGeneration)
			},
		},
		{
			name: "Ready Phase",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 3,
				},
				Status: kargoapi.ProjectStatus{
					Phase: kargoapi.ProjectPhaseReady,
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, ok bool) {
				require.True(t, ok)
				require.Empty(t, status.Phase)   // nolint:staticcheck
				require.Empty(t, status.Message) // nolint:staticcheck
				require.Len(t, status.Conditions, 1)

				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionTrue, readyCondition.Status)
				require.Equal(t, "Initialized", readyCondition.Reason)
				require.Equal(t, "Project is initialized and ready for use", readyCondition.Message)
				require.Equal(t, int64(3), readyCondition.ObservedGeneration)
			},
		},
		{
			name: "Unknown Phase",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 4,
				},
				Status: kargoapi.ProjectStatus{
					Phase: "UnknownPhase",
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, ok bool) {
				require.True(t, ok)
				require.Empty(t, status.Phase)   // nolint:staticcheck
				require.Empty(t, status.Message) // nolint:staticcheck
				require.Empty(t, status.Conditions)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, ok := migratePhaseToConditions(tt.project)
			tt.assertions(t, status, ok)
		})
	}
}

func TestMigrateSpecToProjectConfig(t *testing.T) {
	const testProject = "fake-project"
	testScheme := runtime.NewScheme()
	err := kargoapi.AddToScheme(testScheme)
	require.NoError(t, err)

	tests := []struct {
		name       string
		project    *kargoapi.Project
		assertions func(t *testing.T, migrated bool, cl client.Client, err error)
	}{
		{
			name: "nil spec",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{Name: testProject},
				Spec:       nil,
			},
			assertions: func(t *testing.T, migrated bool, cl client.Client, err error) {
				require.NoError(t, err)
				require.False(t, migrated)
				projCfg := &kargoapi.ProjectConfig{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      testProject,
						Namespace: testProject,
					},
					projCfg,
				)
				require.True(t, kubeerr.IsNotFound(err))
			},
		},
		{
			name: "empty promotion policies",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{Name: testProject},
				Spec: &kargoapi.ProjectSpec{ // nolint:staticcheck
					PromotionPolicies: []kargoapi.PromotionPolicy{},
				},
			},
			assertions: func(t *testing.T, migrated bool, cl client.Client, err error) {
				require.NoError(t, err)
				require.True(t, migrated)
				project := &kargoapi.Project{}
				err = cl.Get(context.Background(), types.NamespacedName{Name: testProject}, project)
				require.NoError(t, err)
				require.Nil(t, project.Spec) // nolint:staticcheck
				projCfg := &kargoapi.ProjectConfig{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      testProject,
						Namespace: testProject,
					},
					projCfg,
				)
				require.True(t, kubeerr.IsNotFound(err))
			},
		},
		// TODO: Add error cases
		{
			name: "success with promotion policies",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{Name: testProject},
				Spec: &kargoapi.ProjectSpec{ // nolint:staticcheck
					PromotionPolicies: []kargoapi.PromotionPolicy{
						{Stage: "policy-1"},
					},
				},
			},
			assertions: func(t *testing.T, migrated bool, cl client.Client, err error) {
				require.NoError(t, err)
				require.True(t, migrated)
				project := &kargoapi.Project{}
				err = cl.Get(context.Background(), types.NamespacedName{Name: testProject}, project)
				require.NoError(t, err)
				require.Nil(t, project.Spec) // nolint:staticcheck
				projCfg := &kargoapi.ProjectConfig{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      testProject,
						Namespace: testProject,
					},
					projCfg,
				)
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(tt.project).Build()
			r := &reconciler{client: cl}
			migrated, err := r.migrateSpecToProjectConfig(context.Background(), tt.project)
			tt.assertions(t, migrated, cl, err)
		})
	}
}
