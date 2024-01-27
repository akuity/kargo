package projects

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			name: "success",
			project: &kargoapi.Project{
				Status: kargoapi.ProjectStatus{
					Phase: kargoapi.ProjectPhaseInitializing,
				},
			},
			reconciler: &reconciler{},
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
