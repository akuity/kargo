package stages

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
)

func TestCheckHealth(t *testing.T) {
	testCases := []struct {
		name             string
		freight          kargoapi.FreightReference
		argoCDAppUpdates []kargoapi.ArgoCDAppUpdate
		reconciler       *reconciler
		assertions       func(*kargoapi.Health)
	}{
		{
			name:       "no argoCDAppUpdates are defined",
			reconciler: &reconciler{},
			assertions: func(health *kargoapi.Health) {
				require.Nil(t, health)
			},
		},
		{
			name:             "argo cd integration is not enabled",
			argoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{{}},
			reconciler:       &reconciler{},
			assertions: func(health *kargoapi.Health) {
				require.NotNil(t, health)
				require.Equal(t, kargoapi.HealthStateUnknown, health.Status)
			},
		},
		{
			name: "error finding Argo CD App",
			argoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
				{
					AppName:      "fake-app",
					AppNamespace: "fake-namespace",
				},
			},
			reconciler: &reconciler{
				argocdClient: fake.NewClientBuilder().Build(),
				getArgoCDAppFn: func(
					context.Context,
					client.Client,
					string,
					string,
				) (*argocd.Application, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(health *kargoapi.Health) {
				require.Equal(t, kargoapi.HealthStateUnknown, health.Status)
				require.Equal(
					t,
					[]kargoapi.ArgoCDAppStatus{
						{
							Namespace: "fake-namespace",
							Name:      "fake-app",
							HealthStatus: kargoapi.ArgoCDAppHealthStatus{
								Status: kargoapi.ArgoCDAppHealthStateUnknown,
							},
							SyncStatus: kargoapi.ArgoCDAppSyncStatus{
								Status: kargoapi.ArgoCDAppSyncStateUnknown,
							},
						},
					},
					health.ArgoCDApps,
				)
				require.Len(t, health.Issues, 1)
				require.Contains(
					t,
					health.Issues[0],
					"error finding Argo CD Application",
				)
				require.Contains(
					t,
					health.Issues[0],
					"something went wrong",
				)
			},
		},

		{
			name: "Argo CD App not found",
			argoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
				{
					AppName:      "fake-app",
					AppNamespace: "fake-namespace",
				},
			},
			reconciler: &reconciler{
				argocdClient: fake.NewClientBuilder().Build(),
				getArgoCDAppFn: func(
					context.Context,
					client.Client,
					string,
					string,
				) (*argocd.Application, error) {
					return nil, nil
				},
			},
			assertions: func(health *kargoapi.Health) {
				require.Equal(t, kargoapi.HealthStateUnknown, health.Status)
				require.Equal(
					t,
					[]kargoapi.ArgoCDAppStatus{
						{
							Namespace: "fake-namespace",
							Name:      "fake-app",
							HealthStatus: kargoapi.ArgoCDAppHealthStatus{
								Status: kargoapi.ArgoCDAppHealthStateUnknown,
							},
							SyncStatus: kargoapi.ArgoCDAppSyncStatus{
								Status: kargoapi.ArgoCDAppSyncStateUnknown,
							},
						},
					},
					health.ArgoCDApps,
				)
				require.Len(t, health.Issues, 1)
				require.Contains(
					t,
					health.Issues[0],
					"unable to find Argo CD Application",
				)
			},
		},

		{
			name: "Argo CD App is multi-source",
			// This doesn't require there to actually BE multiple sources. Simply
			// using the sources field instead of the source fields should be enough
			// to trigger this case.
			argoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
				{
					AppName:      "fake-app",
					AppNamespace: "fake-namespace",
				},
			},
			reconciler: &reconciler{
				argocdClient: fake.NewClientBuilder().Build(),
				getArgoCDAppFn: func(
					context.Context,
					client.Client,
					string,
					string,
				) (*argocd.Application, error) {
					return &argocd.Application{
						Spec: argocd.ApplicationSpec{
							Sources: argocd.ApplicationSources{
								{},
							},
						},
						Status: argocd.ApplicationStatus{
							Health: argocd.HealthStatus{
								Status: argocd.HealthStatusHealthy,
							},
							Sync: argocd.SyncStatus{
								Status: argocd.SyncStatusCodeSynced,
							},
						},
					}, nil
				},
			},
			assertions: func(health *kargoapi.Health) {
				require.Equal(t, kargoapi.HealthStateUnknown, health.Status)
				require.Equal(
					t,
					[]kargoapi.ArgoCDAppStatus{
						{
							Namespace: "fake-namespace",
							Name:      "fake-app",
							HealthStatus: kargoapi.ArgoCDAppHealthStatus{
								Status: kargoapi.ArgoCDAppHealthStateHealthy,
							},
							SyncStatus: kargoapi.ArgoCDAppSyncStatus{
								Status: kargoapi.ArgoCDAppSyncStateSynced,
							},
						},
					},
					health.ArgoCDApps,
				)
				require.Len(t, health.Issues, 1)
				require.Contains(
					t,
					health.Issues[0],
					"bugs in Argo CD currently prevent a comprehensive assessment of "+
						"the health of multi-source Application",
				)
			},
		},

		{
			name: "Argo CD App is not healthy",
			argoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
				{
					AppName:      "fake-app",
					AppNamespace: "fake-namespace",
				},
			},
			reconciler: &reconciler{
				argocdClient: fake.NewClientBuilder().Build(),
				getArgoCDAppFn: func(
					context.Context,
					client.Client,
					string,
					string,
				) (*argocd.Application, error) {
					return &argocd.Application{
						Status: argocd.ApplicationStatus{
							Health: argocd.HealthStatus{
								Status: argocd.HealthStatusDegraded,
							},
							Sync: argocd.SyncStatus{
								Status: argocd.SyncStatusCodeSynced,
							},
						},
					}, nil
				},
			},
			assertions: func(health *kargoapi.Health) {
				require.Equal(t, kargoapi.HealthStateUnhealthy, health.Status)
				require.Equal(
					t,
					[]kargoapi.ArgoCDAppStatus{
						{
							Namespace: "fake-namespace",
							Name:      "fake-app",
							HealthStatus: kargoapi.ArgoCDAppHealthStatus{
								Status: kargoapi.ArgoCDAppHealthStateDegraded,
							},
							SyncStatus: kargoapi.ArgoCDAppSyncStatus{
								Status: kargoapi.ArgoCDAppSyncStateSynced,
							},
						},
					},
					health.ArgoCDApps,
				)
				require.Len(t, health.Issues, 1)
				require.Contains(t, health.Issues[0], "has health state")
				require.Contains(t, health.Issues[0], argocd.HealthStatusDegraded)
			},
		},

		{
			name: "Argo CD App not synced",
			freight: kargoapi.FreightReference{
				Commits: []kargoapi.GitCommit{
					{
						RepoURL: "fake-url",
						ID:      "fake-commit",
					},
				},
			},
			argoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
				{
					AppName:      "fake-app",
					AppNamespace: "fake-namespace",
				},
			},
			reconciler: &reconciler{
				argocdClient: fake.NewClientBuilder().Build(),
				getArgoCDAppFn: func(
					context.Context,
					client.Client,
					string,
					string,
				) (*argocd.Application, error) {
					return &argocd.Application{
						Spec: argocd.ApplicationSpec{
							Source: &argocd.ApplicationSource{
								RepoURL: "fake-url",
							},
						},
						Status: argocd.ApplicationStatus{
							Health: argocd.HealthStatus{
								Status: argocd.HealthStatusHealthy,
							},
							Sync: argocd.SyncStatus{
								Status:   argocd.SyncStatusCodeSynced,
								Revision: "not-the-right-commit",
							},
						},
					}, nil
				},
			},
			assertions: func(health *kargoapi.Health) {
				require.Equal(t, kargoapi.HealthStateUnhealthy, health.Status)
				require.Equal(
					t,
					[]kargoapi.ArgoCDAppStatus{
						{
							Namespace: "fake-namespace",
							Name:      "fake-app",
							HealthStatus: kargoapi.ArgoCDAppHealthStatus{
								Status: kargoapi.ArgoCDAppHealthStateHealthy,
							},
							SyncStatus: kargoapi.ArgoCDAppSyncStatus{
								Status:   kargoapi.ArgoCDAppSyncStateSynced,
								Revision: "not-the-right-commit",
							},
						},
					},
					health.ArgoCDApps,
				)
				require.Len(t, health.Issues, 1)
				require.Contains(t, health.Issues[0], "is not synced to revision")
			},
		},

		{
			name: "Argo CD App healthy and synced",
			freight: kargoapi.FreightReference{
				Commits: []kargoapi.GitCommit{
					{
						RepoURL: "fake-url",
						ID:      "fake-commit",
					},
				},
			},
			argoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
				{
					AppName:      "fake-app",
					AppNamespace: "fake-namespace",
				},
			},
			reconciler: &reconciler{
				argocdClient: fake.NewClientBuilder().Build(),
				getArgoCDAppFn: func(
					context.Context,
					client.Client,
					string,
					string,
				) (*argocd.Application, error) {
					return &argocd.Application{
						Spec: argocd.ApplicationSpec{
							Source: &argocd.ApplicationSource{
								RepoURL: "fake-url",
							},
						},
						Status: argocd.ApplicationStatus{
							Health: argocd.HealthStatus{
								Status: argocd.HealthStatusHealthy,
							},
							Sync: argocd.SyncStatus{
								Status:   argocd.SyncStatusCodeSynced,
								Revision: "fake-commit",
							},
						},
					}, nil
				},
			},
			assertions: func(health *kargoapi.Health) {
				require.Equal(t, kargoapi.HealthStateHealthy, health.Status)
				require.Equal(
					t,
					[]kargoapi.ArgoCDAppStatus{
						{
							Namespace: "fake-namespace",
							Name:      "fake-app",
							HealthStatus: kargoapi.ArgoCDAppHealthStatus{
								Status: kargoapi.ArgoCDAppHealthStateHealthy,
							},
							SyncStatus: kargoapi.ArgoCDAppSyncStatus{
								Status:   kargoapi.ArgoCDAppSyncStateSynced,
								Revision: "fake-commit",
							},
						},
					},
					health.ArgoCDApps,
				)
				require.Empty(t, health.Issues)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.reconciler.checkHealth(
					context.Background(),
					testCase.freight,
					testCase.argoCDAppUpdates,
				),
			)
		})
	}
}

func TestStageHealthForAppSync(t *testing.T) {
	testCases := []struct {
		name       string
		app        *argocd.Application
		revision   string
		assertions func(kargoapi.HealthState)
	}{
		{
			name: "revision is empty",
			assertions: func(health kargoapi.HealthState) {
				require.Equal(t, kargoapi.HealthStateHealthy, health)
			},
		},
		{
			name: "revision is specified; does not match app; still syncing",
			app: &argocd.Application{
				Operation: &argocd.Operation{
					Sync: &argocd.SyncOperation{},
				},
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revision: "not-the-right-commit",
					},
				},
			},
			revision: "fake-commit",
			assertions: func(health kargoapi.HealthState) {
				require.Equal(t, kargoapi.HealthStateProgressing, health)
			},
		},
		{
			name: "revision is specified; does not match app; done syncing",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revision: "not-the-right-commit",
					},
				},
			},
			revision: "fake-commit",
			assertions: func(health kargoapi.HealthState) {
				require.Equal(t, kargoapi.HealthStateUnhealthy, health)
			},
		},
		{
			name: "revision is specified; matches app; still syncing",
			app: &argocd.Application{
				Operation: &argocd.Operation{
					Sync: &argocd.SyncOperation{},
				},
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revision: "fake-commit",
					},
				},
			},
			revision: "fake-commit",
			assertions: func(health kargoapi.HealthState) {
				require.Equal(t, kargoapi.HealthStateProgressing, health)
			},
		},
		{
			name: "revision is specified; matches app; done syncing",
			app: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revision: "fake-commit",
					},
				},
			},
			revision: "fake-commit",
			assertions: func(health kargoapi.HealthState) {
				require.Equal(t, kargoapi.HealthStateHealthy, health)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			health, _ := stageHealthForAppSync(
				testCase.app,
				testCase.revision,
			)
			testCase.assertions(health)
		})
	}
}
