package projects

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewReconciler(t *testing.T) {
	testCfg := ReconcilerConfig{}
	r := newReconciler(fake.NewClientBuilder().Build(), testCfg)
	require.Equal(t, testCfg, r.cfg)
	require.NotNil(t, r.client)
	require.NotNil(t, r.getProjectFn)
	require.NotNil(t, r.syncProjectFn)
	require.NotNil(t, r.ensureNamespaceFn)
	require.NotNil(t, r.patchProjectStatusFn)
	require.NotNil(t, r.getNamespaceFn)
	require.NotNil(t, r.createNamespaceFn)
	require.NotNil(t, r.updateNamespaceFn)
	require.NotNil(t, r.ensureProjectAdminPermissionsFn)
	require.NotNil(t, r.createRoleBindingFn)
}

func TestReconcile(t *testing.T) {
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
			name: "error syncing project",
			reconciler: &reconciler{
				getProjectFn: func(
					context.Context,
					client.Client,
					string,
				) (*kargoapi.Project, error) {
					return &kargoapi.Project{}, nil
				},
				syncProjectFn: func(
					context.Context,
					*kargoapi.Project,
				) (kargoapi.ProjectStatus, error) {
					return kargoapi.ProjectStatus{}, errors.New("something went wrong")
				},
				patchProjectStatusFn: func(
					_ context.Context,
					_ *kargoapi.Project,
					status kargoapi.ProjectStatus,
				) error {
					require.Equal(t, "something went wrong", status.Message)
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
				syncProjectFn: func(
					context.Context,
					*kargoapi.Project,
				) (kargoapi.ProjectStatus, error) {
					return kargoapi.ProjectStatus{}, nil
				},
				patchProjectStatusFn: func(
					_ context.Context,
					_ *kargoapi.Project,
					status kargoapi.ProjectStatus,
				) error {
					require.Empty(t, status.Message)
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

func TestSyncProject(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(*testing.T, kargoapi.ProjectStatus, error)
	}{
		{
			name: "error ensuring namespace",
			reconciler: &reconciler{
				ensureNamespaceFn: func(
					_ context.Context,
					project *kargoapi.Project,
				) (kargoapi.ProjectStatus, error) {
					return *project.Status.DeepCopy(), errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.ErrorContains(t, err, "something went wrong")
				// Still initializing because retry could succeed
				require.Equal(t, kargoapi.ProjectPhaseInitializing, status.Phase)
			},
		},
		{
			name: "fatal error ensuring namespace",
			reconciler: &reconciler{
				ensureNamespaceFn: func(
					_ context.Context,
					project *kargoapi.Project,
				) (kargoapi.ProjectStatus, error) {
					status := *project.Status.DeepCopy()
					status.Phase = kargoapi.ProjectPhaseInitializationFailed
					return status, errors.New("something went very wrong")
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.ErrorContains(t, err, "something went very wrong")
				// Failed because retry cannot possibly succeed
				require.Equal(
					t,
					kargoapi.ProjectPhaseInitializationFailed,
					status.Phase,
				)
			},
		},
		{
			name: "error ensuring project admin permissions",
			reconciler: &reconciler{
				ensureNamespaceFn: func(
					_ context.Context,
					project *kargoapi.Project,
				) (kargoapi.ProjectStatus, error) {
					return *project.Status.DeepCopy(), nil
				},
				ensureProjectAdminPermissionsFn: func(
					context.Context,
					*kargoapi.Project,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.ErrorContains(t, err, "something went wrong")
				// Still initializing because retry could succeed
				require.Equal(t, kargoapi.ProjectPhaseInitializing, status.Phase)
			},
		},
		{
			name: "success",
			reconciler: &reconciler{
				ensureNamespaceFn: func(
					_ context.Context,
					project *kargoapi.Project,
				) (kargoapi.ProjectStatus, error) {
					return *project.Status.DeepCopy(), nil
				},
				ensureProjectAdminPermissionsFn: func(
					context.Context,
					*kargoapi.Project,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.NoError(t, err)
				// Success == ready
				require.Equal(t, kargoapi.ProjectPhaseReady, status.Phase)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			status, err := testCase.reconciler.syncProject(
				context.Background(),
				&kargoapi.Project{
					Status: kargoapi.ProjectStatus{
						Phase: kargoapi.ProjectPhaseInitializing,
					},
				},
			)
			testCase.assertions(t, status, err)
		})
	}
}

func TestEnsureNamespace(t *testing.T) {
	testCases := []struct {
		name       string
		project    *kargoapi.Project
		reconciler *reconciler
		assertions func(*testing.T, kargoapi.ProjectStatus, error)
	}{
		{
			name: "error getting namespace",
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
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error getting namespace")
				// Phase wasn't changed
				require.Equal(t, kargoapi.ProjectPhaseInitializing, status.Phase)
			},
		},
		{
			name: "namespace exists, is not owned by project, but is labeled " +
				"as a project; error updating namespace",
			project: &kargoapi.Project{
				Status: kargoapi.ProjectStatus{
					Phase: kargoapi.ProjectPhaseInitializing,
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
					return nil
				},
				updateNamespaceFn: func(
					context.Context,
					client.Object,
					...client.UpdateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.ErrorContains(t, err, "error updating namespace")
				require.ErrorContains(t, err, "something went wrong")
				// Phase wasn't changed
				require.Equal(t, kargoapi.ProjectPhaseInitializing, status.Phase)
			},
		},
		{
			name: "namespace exists, is not owned by project, but is labeled " +
				"as a project; success updating namespace",
			project: &kargoapi.Project{
				Status: kargoapi.ProjectStatus{
					Phase: kargoapi.ProjectPhaseInitializing,
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
					return nil
				},
				updateNamespaceFn: func(
					context.Context,
					client.Object,
					...client.UpdateOption,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.NoError(t, err)
				// Phase wasn't changed
				require.Equal(t, kargoapi.ProjectPhaseInitializing, status.Phase)
			},
		},
		{
			name: "namespace exists, is not owned by project, and is not labeled as a project",
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
					return nil
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.ErrorContains(t, err, "failed to initialize Project")
				// Phase reflects the unrecoverable failure
				require.Equal(t, status.Phase, kargoapi.ProjectPhaseInitializationFailed)
			},
		},
		{
			name: "namespace exists and is owned by project",
			project: &kargoapi.Project{
				ObjectMeta: metav1.ObjectMeta{
					UID: types.UID("fake-uid"),
				},
				Status: kargoapi.ProjectStatus{
					Phase: kargoapi.ProjectPhaseInitializing,
				},
			},
			reconciler: &reconciler{
				getNamespaceFn: func(
					_ context.Context,
					_ types.NamespacedName,
					obj client.Object,
					_ ...client.GetOption,
				) error {
					ns := obj.(*corev1.Namespace) // nolint: forcetypeassert
					ns.OwnerReferences = []metav1.OwnerReference{
						{
							UID: types.UID("fake-uid"),
						},
					}
					return nil
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.NoError(t, err)
				// Phase wasn't changed
				require.Equal(t, kargoapi.ProjectPhaseInitializing, status.Phase)
			},
		},
		{
			name: "namespace does not exist; error creating it",
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
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error creating namespace")
				// Phase wasn't changed
				require.Equal(t, kargoapi.ProjectPhaseInitializing, status.Phase)
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
			assertions: func(t *testing.T, status kargoapi.ProjectStatus, err error) {
				require.NoError(t, err)
				// Phase wasn't changed
				require.Equal(t, kargoapi.ProjectPhaseInitializing, status.Phase)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			res, err := testCase.reconciler.ensureNamespace(
				context.Background(),
				testCase.project,
			)
			testCase.assertions(t, res, err)
		})
	}
}

func TestEnsureProjectAdminPermissions(t *testing.T) {
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
				require.ErrorContains(t, err, "error creating role binding")
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
				testCase.reconciler.ensureProjectAdminPermissions(
					context.Background(),
					&kargoapi.Project{},
				),
			)
		})
	}
}
