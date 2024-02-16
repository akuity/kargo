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
	r := newReconciler(fake.NewClientBuilder().Build())
	require.NotNil(t, r.client)
	require.NotNil(t, r.getProjectFn)
	require.NotNil(t, r.syncProjectFn)
	require.NotNil(t, r.patchProjectStatusFn)
	require.NotNil(t, r.getNamespaceFn)
	require.NotNil(t, r.createNamespaceFn)
	require.NotNil(t, r.updateNamespaceFn)
}

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler *reconciler
		assertions func(ctrl.Result, error)
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
			assertions: func(result ctrl.Result, err error) {
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
			assertions: func(_ ctrl.Result, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
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
			assertions: func(result ctrl.Result, err error) {
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
			assertions: func(_ ctrl.Result, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
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
			assertions: func(_ ctrl.Result, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.reconciler.Reconcile(context.Background(), ctrl.Request{}),
			)
		})
	}
}

func TestSyncProject(t *testing.T) {
	testCases := []struct {
		name       string
		project    *kargoapi.Project
		reconciler *reconciler
		assertions func(
			initialStatus kargoapi.ProjectStatus,
			newStatus kargoapi.ProjectStatus,
			err error,
		)
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
			assertions: func(initialStatus, newStatus kargoapi.ProjectStatus, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error getting namespace")
				// Status is unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},
		{
			name: "namespace exists, is not owned by project, but is labeled " +
				"as a project; error updating namespace",
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
				updateNamespaceFn: func(
					context.Context,
					client.Object,
					...client.UpdateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(initialStatus, newStatus kargoapi.ProjectStatus, err error) {
				require.Error(t, err)
				// Status should reflect the failure
				require.Contains(t, err.Error(), "error updating namespace")
				require.Contains(t, err.Error(), "something went wrong")
				// And is otherwise unchanged
				newStatus.Message = initialStatus.Message
				require.Equal(t, initialStatus, newStatus)
			},
		},
		{
			name: "namespace exists, is not owned by project, but is labeled " +
				"as a project; success updating namespace",
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
				updateNamespaceFn: func(
					context.Context,
					client.Object,
					...client.UpdateOption,
				) error {
					return nil
				},
			},
			assertions: func(initialStatus, newStatus kargoapi.ProjectStatus, err error) {
				require.NoError(t, err)
				require.Equal(t, newStatus.Phase, kargoapi.ProjectPhaseReady)
				require.Empty(t, newStatus.Message)
			},
		},
		{
			name:    "namespace exists, is not owned by project, and is not labeled as a project",
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
			assertions: func(initialStatus, newStatus kargoapi.ProjectStatus, err error) {
				require.Error(t, err)
				// Status should reflect the unrecoverable failure
				require.Equal(t, newStatus.Phase, kargoapi.ProjectPhaseInitializationFailed)
				require.Contains(t, err.Error(), "failed to initialize Project")
			},
		},
		{
			name: "namespace exists and is owned by project",
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
					ns := obj.(*corev1.Namespace) // nolint: forcetypeassert
					ns.OwnerReferences = []metav1.OwnerReference{
						{
							UID: types.UID("fake-uid"),
						},
					}
					return nil
				},
			},
			assertions: func(_, newStatus kargoapi.ProjectStatus, err error) {
				require.NoError(t, err)
				// Status should reflect that the Project is in a ready state
				require.Equal(t, newStatus.Phase, kargoapi.ProjectPhaseReady)
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
			assertions: func(initialStatus, newStatus kargoapi.ProjectStatus, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error creating namespace")
				// Status is unchanged
				require.Equal(t, initialStatus, newStatus)
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
			assertions: func(_, newStatus kargoapi.ProjectStatus, err error) {
				require.NoError(t, err)
				// Status should reflect that the Project is in a ready state
				require.Equal(t, newStatus.Phase, kargoapi.ProjectPhaseReady)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			newStatus, err :=
				testCase.reconciler.syncProject(context.Background(), testCase.project)
			testCase.assertions(testCase.project.Status, newStatus, err)
		})
	}
}
