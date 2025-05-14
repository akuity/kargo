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
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/webhook/external"
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
	require.NotNil(t, r.ensureDefaultUserRolesFn)
	require.NotNil(t, r.ensureReceiversFn)
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

func TestReconciler_reconcile(t *testing.T) {
	const testProject = "fake-project"

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name        string
		reconciler  *reconciler
		project     *kargoapi.Project
		interceptor interceptor.Funcs
		assertions  func(
			t *testing.T,
			status kargoapi.ProjectStatus,
			requeue bool,
			cl client.Client,
			err error,
		)
	}{
		{
			name:       "error migrating spec to ProjectConfig",
			reconciler: &reconciler{},
			// Requires no phase --> conditions migration.
			// Does require spec --> ProjectConfig migration.
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{Name: testProject},
				// Non-nil of spec is enough to trigger migration.
				Spec: &kargoapi.ProjectSpec{}, // nolint:staticcheck
			},
			interceptor: interceptor.Funcs{
				Update: func(
					context.Context,
					client.WithWatch,
					client.Object,
					...client.UpdateOption,
				) error {
					return fmt.Errorf("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				status kargoapi.ProjectStatus,
				requeue bool,
				_ client.Client,
				err error,
			) {
				require.ErrorContains(t, err, "something went wrong")

				// Doesn't impact conditions
				require.Len(t, status.Conditions, 0)

				// Immediate requeue should NOT be requested
				require.False(t, requeue)
			},
		},
		{
			name:       "success migrating spec to ProjectConfig",
			reconciler: &reconciler{},
			// Requires no phase --> conditions migration.
			// Does require spec --> ProjectConfig migration.
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{Name: testProject},
				// Non-nil of spec is enough to trigger migration.
				Spec: &kargoapi.ProjectSpec{}, // nolint:staticcheck
			},
			assertions: func(
				t *testing.T,
				status kargoapi.ProjectStatus,
				requeue bool,
				cl client.Client,
				err error,
			) {
				require.NoError(t, err)

				// Doesn't impact conditions
				require.Len(t, status.Conditions, 0)

				// Spec should be cleared
				project := &kargoapi.Project{}
				err = cl.Get(context.Background(), types.NamespacedName{Name: testProject}, project)
				require.NoError(t, err)
				require.Empty(t, project.Spec) // nolint:staticcheck

				// Immediate requeue should be requested
				require.True(t, requeue)
			},
		},
		{
			name: "error syncing project",
			reconciler: &reconciler{
				ensureNamespaceFn: func(context.Context, *kargoapi.Project) error {
					return fmt.Errorf("something went wrong")
				},
			},
			// Requires no phase --> conditions migration.
			// Requires no spec --> ProjectConfig migration.
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{Name: testProject},
			},
			assertions: func(
				t *testing.T,
				status kargoapi.ProjectStatus,
				requeue bool,
				_ client.Client,
				err error,
			) {
				require.ErrorContains(t, err, "something went wrong")

				// Still syncing because retry could succeed
				require.Len(t, status.Conditions, 2)
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)

				reconcilingCondition := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcilingCondition)
				require.Equal(t, metav1.ConditionTrue, reconcilingCondition.Status)
				require.Equal(t, "Syncing", reconcilingCondition.Reason)

				// Immediate requeue should NOT be requested
				require.False(t, requeue)
			},
		},
		{
			name: "error collecting Project stats",
			reconciler: &reconciler{
				ensureNamespaceFn: func(context.Context, *kargoapi.Project) error {
					return nil
				},
				ensureAPIAdminPermissionsFn: func(context.Context, *kargoapi.Project) error {
					return nil
				},
				ensureDefaultUserRolesFn: func(context.Context, *kargoapi.Project) error {
					return nil
				},
				ensureReceiversFn: func(context.Context, *kargoapi.Project) error {
					return nil
				},
			},
			// Requires no phase --> conditions migration.
			// Requires no spec --> ProjectConfig migration.
			// Is already initialized.
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{Name: testProject},
				Status: kargoapi.ProjectStatus{
					Conditions: []metav1.Condition{{
						Type:   kargoapi.ConditionTypeReady,
						Status: metav1.ConditionTrue,
					}},
				},
			},
			interceptor: interceptor.Funcs{
				// Fail to list Warehouses
				List: func(
					context.Context,
					client.WithWatch,
					client.ObjectList,
					...client.ListOption,
				) error {
					return fmt.Errorf("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				status kargoapi.ProjectStatus,
				requeue bool,
				_ client.Client,
				err error,
			) {
				require.ErrorContains(t, err, "something went wrong")

				require.Len(t, status.Conditions, 2)

				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionTrue, readyCondition.Status)

				healthCondition := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthCondition)
				require.Equal(t, metav1.ConditionFalse, healthCondition.Status)

				// Immediate requeue should NOT be requested
				require.False(t, requeue)
			},
		},
		{
			name: "success collecting Project stats",
			reconciler: &reconciler{
				ensureNamespaceFn: func(context.Context, *kargoapi.Project) error {
					return nil
				},
				ensureAPIAdminPermissionsFn: func(context.Context, *kargoapi.Project) error {
					return nil
				},
				ensureDefaultUserRolesFn: func(context.Context, *kargoapi.Project) error {
					return nil
				},
				ensureReceiversFn: func(context.Context, *kargoapi.Project) error {
					return nil
				},
			},
			// Requires no phase --> conditions migration.
			// Requires no spec --> ProjectConfig migration.
			// Is already ready.
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{Name: testProject},
				Status: kargoapi.ProjectStatus{
					Conditions: []metav1.Condition{{
						Type:   kargoapi.ConditionTypeReady,
						Status: metav1.ConditionTrue,
					}},
				},
			},
			assertions: func(
				t *testing.T,
				status kargoapi.ProjectStatus,
				requeue bool,
				_ client.Client,
				err error,
			) {
				require.NoError(t, err)

				require.Len(t, status.Conditions, 1)

				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionTrue, readyCondition.Status)

				// Immediate requeue should NOT be requested
				require.False(t, requeue)

				// Status has stats
				require.NotNil(t, status.Stats)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.reconciler.client = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.project).
				WithInterceptorFuncs(tt.interceptor).
				Build()
			status, requeue, err := tt.reconciler.reconcile(context.Background(), tt.project)
			tt.assertions(t, status, requeue, tt.reconciler.client, err)
		})
	}
}

func TestReconciler_syncProject(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		project    *kargoapi.Project
		assertions func(*testing.T, kargoapi.ProjectStatus, error,
		)
	}{
		{
			name: "error ensuring namespace",
			reconciler: &reconciler{
				ensureNamespaceFn: func(context.Context, *kargoapi.Project) error {
					return errors.New("something went wrong")
				},
			},
			project: &kargoapi.Project{},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.ErrorContains(t, err, "something went wrong")

				// Still syncing because retry could succeed
				require.Len(t, status.Conditions, 2)

				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "EnsuringNamespaceFailed", readyCondition.Reason)

				reconcilingCondition := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcilingCondition)
				require.Equal(t, metav1.ConditionTrue, reconcilingCondition.Status)
				require.Equal(t, "Syncing", reconcilingCondition.Reason)
			},
		},
		{
			name: "fatal error ensuring namespace",
			reconciler: &reconciler{
				ensureNamespaceFn: func(context.Context, *kargoapi.Project) error {
					return errProjectNamespaceExists
				},
			},
			project: &kargoapi.Project{},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.True(t, errors.Is(err, errProjectNamespaceExists))

				// Still syncing because retry could succeed
				require.Len(t, status.Conditions, 3)

				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "EnsuringNamespaceFailed", readyCondition.Reason)

				stalledCondition := conditions.Get(&status, kargoapi.ConditionTypeStalled)
				require.NotNil(t, stalledCondition)
				require.Equal(t, metav1.ConditionTrue, stalledCondition.Status)
				require.Equal(t, "ExistingNamespaceMissingLabel", stalledCondition.Reason)

				reconcilingCondition := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcilingCondition)
				require.Equal(t, metav1.ConditionTrue, reconcilingCondition.Status)
				require.Equal(t, "Syncing", reconcilingCondition.Reason)
			},
		},
		{
			name: "error ensuring api server permissions",
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
			project: &kargoapi.Project{},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.ErrorContains(t, err, "something went wrong")

				// Still syncing because retry could succeed
				require.Len(t, status.Conditions, 2)

				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "EnsuringAPIServerPermissionsFailed", readyCondition.Reason)

				reconcilingCondition := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcilingCondition)
				require.Equal(t, metav1.ConditionTrue, reconcilingCondition.Status)
				require.Equal(t, "Syncing", reconcilingCondition.Reason)
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
			project: &kargoapi.Project{},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.ErrorContains(t, err, "something went wrong")

				// Still syncing because retry could succeed
				require.Len(t, status.Conditions, 2)

				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "EnsuringControllerPermissionsFailed", readyCondition.Reason)

				reconcilingCondition := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcilingCondition)
				require.Equal(t, metav1.ConditionTrue, reconcilingCondition.Status)
				require.Equal(t, "Syncing", reconcilingCondition.Reason)
			},
		},
		{
			name: "error ensuring default user roles",
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
				ensureDefaultUserRolesFn: func(
					context.Context,
					*kargoapi.Project,
				) error {
					return errors.New("something went wrong")
				},
			},
			project: &kargoapi.Project{},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.ErrorContains(t, err, "something went wrong")

				// Still syncing because retry could succeed
				require.Len(t, status.Conditions, 2)

				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "EnsuringDefaultUserRoles", readyCondition.Reason)

				reconcilingCondition := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcilingCondition)
				require.Equal(t, metav1.ConditionTrue, reconcilingCondition.Status)
				require.Equal(t, "Syncing", reconcilingCondition.Reason)
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
				ensureDefaultUserRolesFn: func(
					context.Context,
					*kargoapi.Project,
				) error {
					return nil
				},
				ensureReceiversFn: func(
					context.Context,
					*kargoapi.Project,
				) error {
					return nil
				},
			},
			project: &kargoapi.Project{},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.NoError(t, err)

				require.Len(t, status.Conditions, 1)

				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionTrue, readyCondition.Status)
				require.Equal(t, "Synced", readyCondition.Reason)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			status, err := testCase.reconciler.syncProject(
				context.Background(),
				testCase.project,
			)
			testCase.assertions(t, status, err)
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
			name:    "namespace does not exist; success creating it",
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
				testCase.reconciler.ensureDefaultUserRoles(
					context.Background(),
					&kargoapi.Project{},
				),
			)
		})
	}
}

func TestReconciler_ensureReceivers(t *testing.T) {
	for _, test := range []struct {
		name       string
		reconciler func() *reconciler
		project    *kargoapi.Project
		assertions func(*testing.T, *kargoapi.Project, error)
	}{
		{
			name: "project config not found",
			reconciler: func() *reconciler {
				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))
				require.NoError(t, kargoapi.AddToScheme(scheme))
				return newReconciler(
					fake.NewClientBuilder().WithScheme(scheme).Build(),
					ReconcilerConfig{
						KargoNamespace: "fake-namespace",
					},
				)
			},
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-project",
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Project, err error) {
				require.ErrorContains(t, err, "error getting ProjectConfig")
			},
		},
		{
			name: "secret-ref not found",
			reconciler: func() *reconciler {
				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))
				require.NoError(t, kargoapi.AddToScheme(scheme))
				return newReconciler(
					fake.NewClientBuilder().
						WithScheme(scheme).
						WithObjects(
							&kargoapi.ProjectConfig{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "fake-project",
									Namespace: "fake-namespace",
								},
								Spec: kargoapi.ProjectConfigSpec{
									WebhookReceiverConfigs: []kargoapi.WebhookReceiverConfig{
										{
											Type:      kargoapi.WebhookReceiverTypeGitHub,
											SecretRef: "secret-ref-that-does-not-exist",
										},
									},
								},
							},
						).
						Build(),
					ReconcilerConfig{
						KargoNamespace: "fake-namespace",
					},
				)
			},
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-project",
				},
				Spec: &kargoapi.ProjectSpec{ // nolint:staticcheck
					WebhookReceiverConfigs: []kargoapi.WebhookReceiverConfig{
						{
							Type:      kargoapi.WebhookReceiverTypeGitHub,
							SecretRef: "secret-ref-that-does-not-exist",
						},
					},
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Project, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},
		{
			name: "success",
			reconciler: func() *reconciler {
				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))
				require.NoError(t, kargoapi.AddToScheme(scheme))
				return newReconciler(
					fake.NewClientBuilder().
						WithScheme(scheme).
						WithObjects(
							&kargoapi.Project{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "fake-name",
									Namespace: "fake-namespace",
								},
								Spec: &kargoapi.ProjectConfigSpec{},
							},
							&kargoapi.ProjectConfig{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "fake-project",
									Namespace: "fake-namespace",
								},
								Spec: kargoapi.ProjectConfigSpec{
									WebhookReceiverConfigs: []kargoapi.WebhookReceiverConfig{
										{
											Type:      kargoapi.WebhookReceiverTypeGitHub,
											SecretRef: "secret-that-exists",
										},
									},
								},
							},
							&corev1.Secret{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "secret-that-exists",
									Namespace: "fake-namespace",
								},
								Data: map[string][]byte{
									"seed": []byte("fake-secret-data"),
								},
							},
						).
						Build(),
					ReconcilerConfig{
						KargoNamespace: "fake-namespace",
					},
				)
			},
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-project",
				},
				Spec: &kargoapi.ProjectSpec{ // nolint:staticcheck
					WebhookReceiverConfigs: []kargoapi.WebhookReceiverConfig{
						{
							Type:      kargoapi.WebhookReceiverTypeGitHub,
							SecretRef: "secret-that-exists",
						},
					},
				},
			},
			assertions: func(t *testing.T, p *kargoapi.Project, err error) {
				require.NoError(t, err)
				require.Len(t, p.Status.WebhookReceivers, 1)
				require.Equal(t,
					kargoapi.WebhookReceiverTypeGitHub,
					p.Spec.WebhookReceiverConfigs[0].Type, // nolint: staticcheck
				)
				require.Equal(t,
					external.GenerateWebhookPath(
						p.Name,
						kargoapi.WebhookReceiverTypeGitHub,
						"fake-secret-data",
					),
					p.Status.WebhookReceivers[0].Path,
				)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := test.reconciler()
			l := logging.NewLogger(logging.DebugLevel)
			ctx := logging.ContextWithLogger(t.Context(), l)
			err := r.ensureReceiversFn(ctx, test.project)
			test.assertions(t, test.project, err)
		})
	}
}

func TestMigrateSpecToProjectConfig(t *testing.T) {
	const testProject = "fake-project"
	testScheme := runtime.NewScheme()
	err := kargoapi.AddToScheme(testScheme)
	require.NoError(t, err)

	tests := []struct {
		name        string
		project     *kargoapi.Project
		interceptor interceptor.Funcs
		assertions  func(t *testing.T, migrated bool, cl client.Client, err error)
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
		{
			name: "error creating ProjectConfig",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{Name: testProject},
				Spec: &kargoapi.ProjectSpec{ // nolint:staticcheck
					PromotionPolicies: []kargoapi.PromotionPolicy{
						{Stage: "policy-1"}, // nolint:staticcheck
					},
				},
			},
			interceptor: interceptor.Funcs{
				Create: func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, migrated bool, _ client.Client, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.False(t, migrated)
			},
		},
		{
			name: "success with promotion policies",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{Name: testProject},
				Spec: &kargoapi.ProjectSpec{ // nolint:staticcheck
					PromotionPolicies: []kargoapi.PromotionPolicy{
						{Stage: "policy-1"}, // nolint:staticcheck
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
			cl := fake.NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(tt.project).
				WithInterceptorFuncs(tt.interceptor).
				Build()
			r := &reconciler{client: cl}
			migrated, err := r.migrateSpecToProjectConfig(context.Background(), tt.project)
			tt.assertions(t, migrated, cl, err)
		})
	}
}
