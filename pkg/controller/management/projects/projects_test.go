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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/kubernetes"
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
	require.NotNil(t, r.removeFinalizerFn)
	require.NotNil(t, r.ensureSystemPermissionsFn)
	require.NotNil(t, r.ensureControllerPermissionsFn)
	require.NotNil(t, r.ensureDefaultUserRolesFn)
	require.NotNil(t, r.ensureExtendedPermissionsFn)
	require.NotNil(t, r.createServiceAccountFn)
	require.NotNil(t, r.createRoleFn)
	require.NotNil(t, r.createRoleBindingFn)
	require.NotNil(t, r.createClusterRoleFn)
	require.NotNil(t, r.createClusterRoleBindingFn)
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
							Name:              "test-project",
							DeletionTimestamp: &metav1.Time{},
							Finalizers:        []string{kargoapi.FinalizerName},
						},
					}, nil
				},
				ensureFinalizerFn: func(
					context.Context,
					client.Client,
					client.Object,
				) (bool, error) {
					return false, nil
				},
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns, ok := obj.(*corev1.Namespace)
					require.True(t, ok)
					ns.Labels = map[string]string{
						kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
					}
					ns.Finalizers = []string{kargoapi.FinalizerName}
					return nil
				},
				deleteClusterRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				deleteClusterRoleFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				removeFinalizerFn: func(
					_ context.Context,
					_ client.Client,
					_ client.Object,
				) error {
					return nil
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
				) (kargoapi.ProjectStatus, error) {
					return kargoapi.ProjectStatus{}, errors.New("something went wrong")
				},
				ensureFinalizerFn: func(
					context.Context,
					client.Client,
					client.Object,
				) (bool, error) {
					return false, nil
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
				ensureFinalizerFn: func(
					context.Context,
					client.Client,
					client.Object,
				) (bool, error) {
					return false, nil
				},
				reconcileFn: func(
					context.Context,
					*kargoapi.Project,
				) (kargoapi.ProjectStatus, error) {
					return kargoapi.ProjectStatus{}, nil
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
			cl client.Client,
			err error,
		)
	}{
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
			},
		},
		{
			name: "error collecting Project stats",
			reconciler: &reconciler{
				ensureNamespaceFn: func(context.Context, *kargoapi.Project) error {
					return nil
				},
				ensureSystemPermissionsFn: func(context.Context, *kargoapi.Project) error {
					return nil
				},
				ensureDefaultUserRolesFn: func(context.Context, *kargoapi.Project) error {
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
			},
		},
		{
			name: "success collecting Project stats",
			reconciler: &reconciler{
				ensureNamespaceFn: func(context.Context, *kargoapi.Project) error {
					return nil
				},
				ensureSystemPermissionsFn: func(context.Context, *kargoapi.Project) error {
					return nil
				},
				ensureDefaultUserRolesFn: func(context.Context, *kargoapi.Project) error {
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
				_ client.Client,
				err error,
			) {
				require.NoError(t, err)

				require.Len(t, status.Conditions, 1)

				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionTrue, readyCondition.Status)

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
			status, err := tt.reconciler.reconcile(context.Background(), tt.project)
			tt.assertions(t, status, tt.reconciler.client, err)
		})
	}
}

func TestReconciler_cleanupProject(t *testing.T) {
	testCases := []struct {
		name       string
		project    *kargoapi.Project
		reconciler *reconciler
		assertions func(*testing.T, error)
	}{
		{
			name: "error deleting cluster role binding",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-project",
				},
			},
			reconciler: &reconciler{
				deleteClusterRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error deleting ClusterRoleBinding")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error deleting cluster role",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-project",
				},
			},
			reconciler: &reconciler{
				deleteClusterRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				deleteClusterRoleFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error deleting ClusterRole")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "handles not found errors gracefully",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-project",
				},
			},
			reconciler: &reconciler{
				deleteClusterRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return apierrors.NewNotFound(schema.GroupResource{}, "test")
				},
				deleteClusterRoleFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return apierrors.NewNotFound(schema.GroupResource{}, "test")
				},
				getNamespaceFn: func(
					context.Context,
					types.NamespacedName,
					client.Object,
					...client.GetOption,
				) error {
					return apierrors.NewNotFound(schema.GroupResource{}, "test-project")
				},
				removeFinalizerFn: func(
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
			name: "deletes promotion ArgoCD cluster role binding",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-project",
				},
			},
			reconciler: &reconciler{
				deleteClusterRoleBindingFn: func(
					_ context.Context,
					obj client.Object,
					_ ...client.DeleteOption,
				) error {
					// Verify both ClusterRoleBindings are deleted
					name := obj.GetName()
					if name != kubernetes.ShortenResourceName("kargo-project-admin-test-project") &&
						name != kubernetes.ShortenResourceName("kargo-argocd-test-project") {
						return fmt.Errorf("unexpected ClusterRoleBinding name: %s", name)
					}
					return nil
				},
				deleteClusterRoleFn: func(
					_ context.Context,
					obj client.Object,
					_ ...client.DeleteOption,
				) error {
					// Only the admin ClusterRole should be deleted
					name := obj.GetName()
					if name != kubernetes.ShortenResourceName("kargo-project-admin-test-project") {
						return fmt.Errorf("unexpected ClusterRole name: %s", name)
					}
					return nil
				},
				getNamespaceFn: func(
					context.Context,
					types.NamespacedName,
					client.Object,
					...client.GetOption,
				) error {
					return apierrors.NewNotFound(schema.GroupResource{}, "test-project")
				},
				removeFinalizerFn: func(
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
			name: "error getting namespace",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-project",
				},
			},
			reconciler: &reconciler{
				deleteClusterRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				deleteClusterRoleFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
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
				require.ErrorContains(t, err, "error getting namespace")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "namespace not found - success",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-project",
				},
			},
			reconciler: &reconciler{
				deleteClusterRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				deleteClusterRoleFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				getNamespaceFn: func(
					context.Context,
					types.NamespacedName,
					client.Object,
					...client.GetOption,
				) error {
					return apierrors.NewNotFound(schema.GroupResource{}, "test-project")
				},
				removeFinalizerFn: func(
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
			name: "namespace not found - error removing project finalizer",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-project",
				},
			},
			reconciler: &reconciler{
				deleteClusterRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				deleteClusterRoleFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				getNamespaceFn: func(
					context.Context,
					types.NamespacedName,
					client.Object,
					...client.GetOption,
				) error {
					return apierrors.NewNotFound(schema.GroupResource{}, "test-project")
				},
				removeFinalizerFn: func(
					context.Context,
					client.Client,
					client.Object,
				) error {
					return errors.New("finalizer removal failed")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "failed to remove finalizer from project")
				require.ErrorContains(t, err, "finalizer removal failed")
			},
		},
		{
			name: "keep namespace - success",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-project",
					UID:  "project-uid",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyKeepNamespace: kargoapi.AnnotationValueTrue,
					},
				},
			},
			reconciler: &reconciler{
				deleteClusterRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				deleteClusterRoleFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns, ok := obj.(*corev1.Namespace)
					require.True(t, ok)
					ns.Name = "test-project"
					ns.OwnerReferences = []metav1.OwnerReference{
						{UID: "project-uid"},
						{UID: "other-uid"},
					}
					return nil
				},
				patchOwnerReferencesFn: func(
					context.Context,
					client.Client,
					client.Object,
				) error {
					return nil
				},
				removeFinalizerFn: func(
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
			name: "keep namespace - error patching owner references",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-project",
					UID:  "project-uid",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyKeepNamespace: kargoapi.AnnotationValueTrue,
					},
				},
			},
			reconciler: &reconciler{
				deleteClusterRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				deleteClusterRoleFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns, ok := obj.(*corev1.Namespace)
					require.True(t, ok)
					ns.Name = "test-project"
					ns.OwnerReferences = []metav1.OwnerReference{
						{UID: "project-uid"},
					}
					return nil
				},
				patchOwnerReferencesFn: func(
					context.Context,
					client.Client,
					client.Object,
				) error {
					return errors.New("patch failed")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "failed to patch namespace")
				require.ErrorContains(t, err, "patch failed")
			},
		},
		{
			name: "keep namespace - no owner reference to remove",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-project",
					UID:  "project-uid",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyKeepNamespace: kargoapi.AnnotationValueTrue,
					},
				},
			},
			reconciler: &reconciler{
				deleteClusterRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				deleteClusterRoleFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns, ok := obj.(*corev1.Namespace)
					require.True(t, ok)
					ns.Name = "test-project"
					ns.OwnerReferences = []metav1.OwnerReference{
						{UID: "other-uid"},
					}
					return nil
				},
				removeFinalizerFn: func(
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
			name: "delete namespace - success",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-project",
					Annotations: map[string]string{},
				},
			},
			reconciler: &reconciler{
				deleteClusterRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				deleteClusterRoleFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns, ok := obj.(*corev1.Namespace)
					require.True(t, ok)
					ns.Name = "test-project"
					return nil
				},
				removeFinalizerFn: func(
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
			name: "delete namespace - namespace already deleted",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-project",
					Annotations: map[string]string{},
				},
			},
			reconciler: &reconciler{
				deleteClusterRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				deleteClusterRoleFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns, ok := obj.(*corev1.Namespace)
					require.True(t, ok)
					ns.Name = "test-project"
					return nil
				},
				removeFinalizerFn: func(
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
			name: "error removing namespace finalizer",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-project",
					Annotations: map[string]string{},
				},
			},
			reconciler: &reconciler{
				deleteClusterRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				deleteClusterRoleFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns, ok := obj.(*corev1.Namespace)
					require.True(t, ok)
					ns.Name = "test-project"
					return nil
				},
				removeFinalizerFn: func(
					_ context.Context,
					_ client.Client,
					_ client.Object,
				) error {
					return fmt.Errorf("finalizer removal failed")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "failed to remove finalizer from namespace")
				require.ErrorContains(t, err, "finalizer removal failed")
			},
		},
		{
			name: "error removing project finalizer",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-project",
					Annotations: map[string]string{},
				},
			},
			reconciler: &reconciler{
				deleteClusterRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				deleteClusterRoleFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns, ok := obj.(*corev1.Namespace)
					require.True(t, ok)
					ns.Name = "test-project"
					return nil
				},
				removeFinalizerFn: func() func(context.Context, client.Client, client.Object) error {
					var count int
					return func(
						context.Context,
						client.Client,
						client.Object,
					) error {
						if count == 0 {
							count++
							return nil // First call succeeds
						}
						return fmt.Errorf("finalizer removal failed")
					}
				}(),
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "failed to remove finalizer from project")
				require.ErrorContains(t, err, "finalizer removal failed")
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.reconciler.cleanupProject(context.Background(), testCase.project)
			testCase.assertions(t, err)
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
				ensureSystemPermissionsFn: func(
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
				require.Equal(t, "EnsuringSystemPermissionsFailed", readyCondition.Reason)

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
				ensureSystemPermissionsFn: func(
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
				ensureSystemPermissionsFn: func(
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
				ensureSystemPermissionsFn: func(
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
				ensureFinalizerFn: func(
					context.Context,
					client.Client,
					client.Object,
				) (bool, error) {
					return false, nil
				},
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns, ok := obj.(*corev1.Namespace)
					require.True(t, ok)
					ns.Labels = map[string]string{
						kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
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
						kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
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
						kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
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
						kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
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

func TestReconciler_ensureSystemPermissions(t *testing.T) {
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
			name: "error updating existing role binding",
			reconciler: &reconciler{
				client: fake.NewClientBuilder().WithInterceptorFuncs(interceptor.Funcs{
					Update: func(
						context.Context,
						client.WithWatch,
						client.Object,
						...client.UpdateOption,
					) error {
						return errors.New("something went wrong")
					},
				}).Build(),
				createRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return apierrors.NewAlreadyExists(schema.GroupResource{}, "")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error updating existing RoleBinding")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success updating existing role binding",
			reconciler: &reconciler{
				client: fake.NewClientBuilder().WithInterceptorFuncs(interceptor.Funcs{
					Update: func(
						context.Context,
						client.WithWatch,
						client.Object,
						...client.UpdateOption,
					) error {
						return nil
					},
				}).Build(),
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
				testCase.reconciler.ensureSystemPermissions(
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

func TestReconciler_ensureDefaultUserRoles(t *testing.T) {
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
			name: "error creating ClusterRole",
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
				createClusterRoleFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error creating ClusterRole")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error creating ClusterRoleBinding",
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
				createClusterRoleFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
				createClusterRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error creating ClusterRoleBinding")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				createClusterRoleFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
				createClusterRoleBindingFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
				createServiceAccountFn: func(
					_ context.Context,
					obj client.Object,
					_ ...client.CreateOption,
				) error {
					sa, ok := obj.(*corev1.ServiceAccount)
					require.True(t, ok)
					require.Equal(
						t,
						`{"email":["tony@stark.io"]}`,
						sa.Annotations[rbacapi.AnnotationKeyOIDCClaims],
					)
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
			p := &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyCreateActor: "email:tony@stark.io",
					},
				},
			}
			testCase.assertions(
				t,
				testCase.reconciler.ensureDefaultUserRoles(context.Background(), p),
			)
		})
	}
}

func TestReconciler_ensureExtendedPermissions(t *testing.T) {
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
		cfg        ReconcilerConfig
		client     client.Client
		assertions func(*testing.T, client.Client, error)
	}{
		{
			name: "error creating ServiceAccount",
			cfg: ReconcilerConfig{
				ControlPlaneServiceAccountName: "test-control-plane",
			},
			client: fake.NewClientBuilder().WithScheme(scheme).
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
				require.ErrorContains(t, err, "error creating ServiceAccount")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "ServiceAccounts already exist",
			cfg: ReconcilerConfig{
				ControlPlaneServiceAccountName: "test-control-plane",
			},
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(
					&corev1.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-control-plane",
							Namespace: testProject.Name,
							Annotations: map[string]string{
								rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
							},
						},
					},
				).Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)
				// Verify ServiceAccount still exists
				sa := &corev1.ServiceAccount{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      "test-control-plane",
						Namespace: testProject.Name,
					},
					sa,
				)
				require.NoError(t, err)
			},
		},
		{
			name: "creates control plane ServiceAccount and RoleBinding",
			cfg: ReconcilerConfig{
				ControlPlaneServiceAccountName: "test-control-plane",
				ControlPlaneClusterRoleName:    "test-control-plane-role",
			},
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)

				// Verify ServiceAccount was created
				sa := &corev1.ServiceAccount{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      "test-control-plane",
						Namespace: testProject.Name,
					},
					sa,
				)
				require.NoError(t, err)
				require.Equal(t, rbacapi.AnnotationValueTrue, sa.Annotations[rbacapi.AnnotationKeyManaged])

				// Verify RoleBinding was created
				rb := &rbacv1.RoleBinding{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      "test-control-plane",
						Namespace: testProject.Name,
					},
					rb,
				)
				require.NoError(t, err)
				require.Equal(t, "test-control-plane-role", rb.RoleRef.Name)
				require.Equal(t, "ClusterRole", rb.RoleRef.Kind)
				require.Equal(t, rbacapi.AnnotationValueTrue, rb.Annotations[rbacapi.AnnotationKeyManaged])
			},
		},
		{
			name: "ArgoCD configured - creates ArgoCD ServiceAccount",
			cfg: ReconcilerConfig{
				ArgoCDServiceAccountName: "kargo-argocd-service-account",
			},
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)

				// Verify ArgoCD ServiceAccount was created
				sa := &corev1.ServiceAccount{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      "kargo-argocd-service-account",
						Namespace: testProject.Name,
					},
					sa,
				)
				require.NoError(t, err)
				require.Equal(t, rbacapi.AnnotationValueTrue, sa.Annotations[rbacapi.AnnotationKeyManaged])
			},
		},
		{
			name: "ArgoCD configured and not watching namespace only - creates ClusterRoleBinding",
			cfg: ReconcilerConfig{
				ArgoCDServiceAccountName: "kargo-argocd-service-account",
				ArgoCDClusterRoleName:    "kargo-argocd",
				ArgoCDWatchNamespaceOnly: false,
			},
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)

				// Verify ClusterRoleBinding was created
				crb := &rbacv1.ClusterRoleBinding{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name: kubernetes.ShortenResourceName(fmt.Sprintf("kargo-argocd-%s", testProject.Name)),
					},
					crb,
				)
				require.NoError(t, err)
				require.Equal(t, "kargo-argocd", crb.RoleRef.Name)
				require.Equal(t, rbacapi.AnnotationValueTrue, crb.Annotations[rbacapi.AnnotationKeyManaged])
				require.Len(t, crb.Subjects, 1)
				require.Equal(t, "kargo-argocd-service-account", crb.Subjects[0].Name)
				require.Equal(t, testProject.Name, crb.Subjects[0].Namespace)
			},
		},
		{
			name: "ArgoCD configured and watching namespace only - creates RoleBinding",
			cfg: ReconcilerConfig{
				ArgoCDServiceAccountName: "kargo-argocd-service-account",
				ArgoCDRoleName:           "kargo-argocd",
				ArgoCDNamespace:          "argocd",
				ArgoCDWatchNamespaceOnly: true,
			},
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)

				// Verify ArgoCD RoleBinding was created
				rb := &rbacv1.RoleBinding{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      kubernetes.ShortenResourceName(fmt.Sprintf("kargo-argocd-%s", testProject.Name)),
						Namespace: "argocd",
					},
					rb,
				)
				require.NoError(t, err)
				require.Equal(t, "kargo-argocd", rb.RoleRef.Name)
				require.Equal(t, "Role", rb.RoleRef.Kind)
				require.Len(t, rb.Subjects, 1)
				require.Equal(t, "kargo-argocd-service-account", rb.Subjects[0].Name)
			},
		},
		{
			name: "manage orchestrator enabled without dedicated namespace",
			cfg: ReconcilerConfig{
				ManageOrchestrator:             true,
				OrchestratorServiceAccountName: "test-orchestrator",
				OrchestratorClusterRoleName:    "test-orchestrator-role",
				TokenManagerClusterRoleName:    "test-token-manager",
				ControlPlaneServiceAccountName: "test-control-plane",
				ControlPlaneClusterRoleName:    "test-control-plane-role",
				ManagedResourceNamespace:       "", // Empty means use project namespace
			},
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)

				// Verify orchestrator ServiceAccount was created
				sa := &corev1.ServiceAccount{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      "test-orchestrator",
						Namespace: testProject.Name,
					},
					sa,
				)
				require.NoError(t, err)

				// Verify control plane ServiceAccount was created
				sa = &corev1.ServiceAccount{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      "test-control-plane",
						Namespace: testProject.Name,
					},
					sa,
				)
				require.NoError(t, err)

				// Verify orchestrator RoleBindings were created
				roleBindings := []string{
					"test-orchestrator",
					"test-token-manager",
					"test-control-plane",
					kubernetes.ShortenResourceName(fmt.Sprintf("%s-secrets-reader", "test-orchestrator")),
				}

				for _, rbName := range roleBindings {
					rb := &rbacv1.RoleBinding{}
					err = cl.Get(
						context.Background(),
						types.NamespacedName{
							Name:      rbName,
							Namespace: testProject.Name,
						},
						rb,
					)
					require.NoError(t, err, "RoleBinding %s should exist", rbName)
				}
			},
		},
		{
			name: "manage orchestrator enabled with dedicated namespace",
			cfg: ReconcilerConfig{
				ManageOrchestrator:             true,
				OrchestratorServiceAccountName: "test-orchestrator",
				ManagedResourceNamespace:       "kargo-resources",
			},
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)

				// Verify orchestrator ServiceAccount was NOT created in project namespace
				sa := &corev1.ServiceAccount{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      "test-orchestrator",
						Namespace: testProject.Name,
					},
					sa,
				)
				require.True(t, apierrors.IsNotFound(err))
			},
		},
		{
			name: "manage resource manager role without dedicated namespace",
			cfg: ReconcilerConfig{
				ManagerServiceAccountName: "manager-sa",
				ManagerClusterRoleName:    "kargo-manager",
				ManagedResourceNamespace:  "", // Empty means use project namespace
				KargoNamespace:            "kargo-system",
			},
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)

				// Verify manager RoleBinding was created
				rb := &rbacv1.RoleBinding{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      "kargo-manager",
						Namespace: testProject.Name,
					},
					rb,
				)
				require.NoError(t, err)
				require.Equal(t, "kargo-manager", rb.RoleRef.Name)
				require.Len(t, rb.Subjects, 1)
				require.Equal(t, "manager-sa", rb.Subjects[0].Name)
				require.Equal(t, "kargo-system", rb.Subjects[0].Namespace)
			},
		},
		{
			name: "do not manage resource manager role with dedicated namespace",
			cfg: ReconcilerConfig{
				ManagerServiceAccountName: "manager-sa",
				ManagedResourceNamespace:  "kargo-resources",
			},
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)

				// Verify manager RoleBinding was NOT created
				rb := &rbacv1.RoleBinding{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name:      "kargo-manager",
						Namespace: testProject.Name,
					},
					rb,
				)
				require.True(t, apierrors.IsNotFound(err))
			},
		},
		{
			name: "error creating RoleBinding",
			cfg: ReconcilerConfig{
				ControlPlaneServiceAccountName: "test-control-plane",
				ControlPlaneClusterRoleName:    "test-control-plane-role",
			},
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					Create: func(
						_ context.Context,
						_ client.WithWatch,
						obj client.Object,
						_ ...client.CreateOption,
					) error {
						if _, ok := obj.(*rbacv1.RoleBinding); ok {
							return fmt.Errorf("something went wrong")
						}
						return nil
					},
				}).Build(),
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "error creating RoleBinding")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error creating ClusterRoleBinding",
			cfg: ReconcilerConfig{
				ArgoCDServiceAccountName: "kargo-argocd-service-account",
				ArgoCDClusterRoleName:    "kargo-argocd-role",
			},
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					Create: func(
						_ context.Context,
						_ client.WithWatch,
						obj client.Object,
						_ ...client.CreateOption,
					) error {
						if _, ok := obj.(*rbacv1.ClusterRoleBinding); ok {
							return fmt.Errorf("something went wrong")
						}
						return nil
					},
				}).Build(),
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "error creating ClusterRoleBinding")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "ClusterRoleBinding already exists",
			cfg: ReconcilerConfig{
				ArgoCDServiceAccountName: "kargo-argocd-service-account",
				ArgoCDClusterRoleName:    "kargo-argocd-role",
			},
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(
					&rbacv1.ClusterRoleBinding{
						ObjectMeta: metav1.ObjectMeta{
							Name: kubernetes.ShortenResourceName(fmt.Sprintf("kargo-argocd-role-%s", testProject.Name)),
						},
					},
				).Build(),
			assertions: func(t *testing.T, cl client.Client, err error) {
				require.NoError(t, err)
				// Verify it still exists
				crb := &rbacv1.ClusterRoleBinding{}
				err = cl.Get(
					context.Background(),
					types.NamespacedName{
						Name: kubernetes.ShortenResourceName(fmt.Sprintf("kargo-argocd-role-%s", testProject.Name)),
					},
					crb,
				)
				require.NoError(t, err)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r := newReconciler(testCase.client, testCase.cfg)
			err := r.ensureExtendedPermissions(context.Background(), testProject)
			testCase.assertions(t, testCase.client, err)
		})
	}
}

func Test_shouldKeepNamespace(t *testing.T) {
	testCases := []struct {
		name      string
		project   *kargoapi.Project
		namespace *corev1.Namespace
		expected  bool
	}{
		{
			name: "no keep annotation on either",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			expected: false,
		},
		{
			name: "keep annotation on project",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyKeepNamespace: kargoapi.AnnotationValueTrue,
					},
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			expected: true,
		},
		{
			name: "keep annotation on namespace",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyKeepNamespace: kargoapi.AnnotationValueTrue,
					},
				},
			},
			expected: true,
		},
		{
			name: "keep annotation on both",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyKeepNamespace: kargoapi.AnnotationValueTrue,
					},
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyKeepNamespace: kargoapi.AnnotationValueTrue,
					},
				},
			},
			expected: true,
		},
		{
			name: "false value on project",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyKeepNamespace: "false",
					},
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			expected: false,
		},
		{
			name: "nil annotations on project",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: nil,
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			expected: false,
		},
		{
			name: "nil annotations on namespace",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: nil,
				},
			},
			expected: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := shouldKeepNamespace(testCase.project, testCase.namespace)
			require.Equal(t, testCase.expected, result)
		})
	}
}
