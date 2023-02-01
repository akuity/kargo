package controller

import (
	"context"
	"testing"

	api "github.com/akuityio/kargo/api/v1alpha1"
	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argoHealth "github.com/argoproj/gitops-engine/pkg/health"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestCheckHealth(t *testing.T) {
	testCases := []struct {
		name     string
		env      *api.Environment
		getAppFn func(
			context.Context,
			string,
			string,
		) (*argocd.Application, error)
		assertions func(*api.Health)
	}{
		{
			name: "healthchecks not specified",
			env:  &api.Environment{},
			assertions: func(health *api.Health) {
				require.Nil(t, health)
			},
		},
		{
			name: "healthchecks do not include any Argo CD Apps",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					HealthChecks: &api.HealthChecks{},
				},
			},
			assertions: func(health *api.Health) {
				require.Nil(t, health)
			},
		},
		{
			name: "status has no states",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					HealthChecks: &api.HealthChecks{
						ArgoCDApps: []string{"fake-app"},
					},
				},
			},
			assertions: func(health *api.Health) {
				require.Nil(t, health)
			},
		},
		{
			name: "error finding Argo CD App",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					HealthChecks: &api.HealthChecks{
						ArgoCDApps: []string{"fake-app"},
					},
				},
				Status: api.EnvironmentStatus{
					States: []api.EnvironmentState{
						{},
					},
				},
			},
			getAppFn: func(
				context.Context,
				string,
				string,
			) (*argocd.Application, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(health *api.Health) {
				require.NotNil(t, health)
				require.Equal(t, api.HealthStateUnknown, health.Status)
				require.Contains(
					t,
					health.StatusReason,
					"error finding Argo CD Application",
				)
				require.Contains(t, health.StatusReason, "something went wrong")
			},
		},
		{
			name: "Argo CD App not found",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					HealthChecks: &api.HealthChecks{
						ArgoCDApps: []string{"fake-app"},
					},
				},
				Status: api.EnvironmentStatus{
					States: []api.EnvironmentState{
						{},
					},
				},
			},
			getAppFn: func(
				context.Context,
				string,
				string,
			) (*argocd.Application, error) {
				return nil, nil
			},
			assertions: func(health *api.Health) {
				require.NotNil(t, health)
				require.Equal(t, api.HealthStateUnknown, health.Status)
				require.Contains(
					t,
					health.StatusReason,
					"unable to find Argo CD Application",
				)
			},
		},
		{
			name: "Argo CD App not synced",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					HealthChecks: &api.HealthChecks{
						ArgoCDApps: []string{"fake-app"},
					},
				},
				Status: api.EnvironmentStatus{
					States: []api.EnvironmentState{
						{
							HealthCheckCommit: "fake-commit",
						},
					},
				},
			},
			getAppFn: func(
				context.Context,
				string,
				string,
			) (*argocd.Application, error) {
				return &argocd.Application{
					Status: argocd.ApplicationStatus{
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeOutOfSync,
						},
					},
				}, nil
			},
			assertions: func(health *api.Health) {
				require.NotNil(t, health)
				require.Equal(t, api.HealthStateUnhealthy, health.Status)
				require.Contains(
					t,
					health.StatusReason,
					"is not synced to current Environment state",
				)
			},
		},
		{
			name: "Argo CD App synced but not healthy",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					HealthChecks: &api.HealthChecks{
						ArgoCDApps: []string{"fake-app"},
					},
				},
				Status: api.EnvironmentStatus{
					States: []api.EnvironmentState{
						{
							HealthCheckCommit: "fake-commit",
						},
					},
				},
			},
			getAppFn: func(
				context.Context,
				string,
				string,
			) (*argocd.Application, error) {
				return &argocd.Application{
					Status: argocd.ApplicationStatus{
						Sync: argocd.SyncStatus{
							Status:   argocd.SyncStatusCodeSynced,
							Revision: "fake-commit",
						},
						Health: argocd.HealthStatus{
							Status: argoHealth.HealthStatusDegraded,
						},
					},
				}, nil
			},
			assertions: func(health *api.Health) {
				require.NotNil(t, health)
				require.Equal(t, api.HealthStateUnhealthy, health.Status)
				require.Contains(t, health.StatusReason, "has health state")
				require.Contains(
					t,
					health.StatusReason,
					argoHealth.HealthStatusDegraded,
				)
			},
		},
		{
			name: "Argo CD App synced but not healthy",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					HealthChecks: &api.HealthChecks{
						ArgoCDApps: []string{"fake-app"},
					},
				},
				Status: api.EnvironmentStatus{
					States: []api.EnvironmentState{
						{
							HealthCheckCommit: "fake-commit",
						},
					},
				},
			},
			getAppFn: func(
				context.Context,
				string,
				string,
			) (*argocd.Application, error) {
				return &argocd.Application{
					Status: argocd.ApplicationStatus{
						Sync: argocd.SyncStatus{
							Status:   argocd.SyncStatusCodeSynced,
							Revision: "fake-commit",
						},
						Health: argocd.HealthStatus{
							Status: argoHealth.HealthStatusHealthy,
						},
					},
				}, nil
			},
			assertions: func(health *api.Health) {
				require.NotNil(t, health)
				require.Equal(t, api.HealthStateHealthy, health.Status)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := &environmentReconciler{
				logger:         log.New(),
				getArgoCDAppFn: testCase.getAppFn,
			}
			testCase.assertions(
				reconciler.checkHealth(context.Background(), testCase.env),
			)
		})
	}
}

func TestIsArgoCDAppSynced(t *testing.T) {
	testCases := []struct {
		name       string
		commit     string
		app        *argocd.Application
		assertions func(bool)
	}{
		{
			name: "App is nil",
			assertions: func(synced bool) {
				require.False(t, synced)
			},
		},
		{
			name: "App is not synced",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Status: argocd.SyncStatusCodeOutOfSync,
					},
				},
			},
			assertions: func(synced bool) {
				require.False(t, synced)
			},
		},
		{
			name:   "App is synced, but not to the correct revision",
			commit: "fake-commit",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Status:   argocd.SyncStatusCodeSynced,
						Revision: "different-fake-commit",
					},
				},
			},
			assertions: func(synced bool) {
				require.False(t, synced)
			},
		},
		{
			name:   "App is synced to the correct commit",
			commit: "fake-commit",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Status:   argocd.SyncStatusCodeSynced,
						Revision: "fake-commit",
					},
				},
			},
			assertions: func(synced bool) {
				require.True(t, synced)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := &environmentReconciler{
				logger: log.New(),
			}
			testCase.assertions(
				reconciler.isArgoCDAppSynced(testCase.app, testCase.commit),
			)
		})
	}
}

func TestBuildChangesMap(t *testing.T) {
	images := []api.Image{
		{
			RepoURL: "fake-url",
			Tag:     "fake-tag",
		},
		{
			RepoURL: "another-fake-url",
			Tag:     "another-fake-tag",
		},
	}
	imageUpdates := []api.ArgoCDHelmImageUpdate{
		{
			Image: "fake-url",
			Key:   "fake-key",
			Value: "Image",
		},
		{
			Image: "another-fake-url",
			Key:   "another-fake-key",
			Value: "Tag",
		},
		{
			Image: "image-that-is-not-in-list",
			Key:   "fake-key",
			Value: "Tag",
		},
	}
	result := buildChangesMap(images, imageUpdates)
	require.Equal(
		t,
		map[string]string{
			"fake-key":         "fake-url:fake-tag",
			"another-fake-key": "another-fake-tag",
		},
		result,
	)
}
