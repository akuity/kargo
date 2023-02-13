package controller

import (
	"context"
	"testing"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argoHealth "github.com/argoproj/gitops-engine/pkg/health"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	api "github.com/akuityio/kargo/api/v1alpha1"
)

func TestPromoteWithArgoCD(t *testing.T) {
	testCases := []struct {
		name        string
		env         *api.Environment
		newState    api.EnvironmentState
		updateAppFn func(
			ctx context.Context,
			env *api.Environment,
			newState api.EnvironmentState,
			appUpdate api.ArgoCDAppUpdate,
		) error
		assertions func(err error)
	}{
		{
			name: "environment is nil",
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "PromotionMechanisms is nil",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{},
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "ArgoCD is nil",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{},
				},
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "ArgoCD promotion mechanism has len(AppUpdates) == 0",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						ArgoCD: &api.ArgoCDPromotionMechanism{},
					},
				},
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "error making App refresh and sync",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						ArgoCD: &api.ArgoCDPromotionMechanism{
							AppUpdates: []api.ArgoCDAppUpdate{
								{
									Name:           "fake-app",
									RefreshAndSync: true,
								},
							},
						},
					},
				},
			},
			updateAppFn: func(
				context.Context,
				*api.Environment,
				api.EnvironmentState,
				api.ArgoCDAppUpdate,
			) error {
				return errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error updating Argo CD Application")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "success",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						ArgoCD: &api.ArgoCDPromotionMechanism{
							AppUpdates: []api.ArgoCDAppUpdate{
								{
									Name:           "fake-app",
									RefreshAndSync: true,
								},
							},
						},
					},
				},
			},
			updateAppFn: func(
				context.Context,
				*api.Environment,
				api.EnvironmentState,
				api.ArgoCDAppUpdate,
			) error {
				return nil
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := environmentReconciler{
				logger:            log.New(),
				updateArgoCDAppFn: testCase.updateAppFn,
			}
			reconciler.logger.SetLevel(log.ErrorLevel)
			testCase.assertions(
				reconciler.promoteWithArgoCD(
					context.Background(),
					testCase.env,
					testCase.newState,
				),
			)
		})
	}
}

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

func TestBuildKustomizeImagesForArgoCDApp(t *testing.T) {
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
	imageUpdates := []string{
		"fake-url",
		"another-fake-url",
		"image-that-is-not-in-list",
	}
	result := buildKustomizeImagesForArgoCDApp(images, imageUpdates)
	require.Equal(
		t,
		argocd.KustomizeImages{
			"fake-url=fake-url:fake-tag",
			"another-fake-url=another-fake-url:another-fake-tag",
		},
		result,
	)
}

func TestBuildHelmParamChangesForArgoCDApp(t *testing.T) {
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
	result := buildHelmParamChangesForArgoCDApp(images, imageUpdates)
	require.Equal(
		t,
		map[string]string{
			"fake-key":         "fake-url:fake-tag",
			"another-fake-key": "another-fake-tag",
		},
		result,
	)
}
