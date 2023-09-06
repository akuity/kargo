package stages

import (
	"context"
	"testing"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argoHealth "github.com/argoproj/gitops-engine/pkg/health"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestCheckHealth(t *testing.T) {
	testCases := []struct {
		name             string
		freight          kargoapi.Freight
		argoCDAppUpdates []kargoapi.ArgoCDAppUpdate
		getArgoCDAppFn   func(
			context.Context,
			client.Client,
			string,
			string,
		) (*argocd.Application, error)
		assertions func(kargoapi.Health)
	}{
		{
			name: "no argoCDAppUpdates are defined",
			assertions: func(health kargoapi.Health) {
				require.Equal(t,
					kargoapi.Health{
						Status: kargoapi.HealthStateNotApplicable,
						Issues: []string{
							"no spec.promotionMechanisms.argoCDAppUpdates are defined",
						},
					},
					health,
				)
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
			getArgoCDAppFn: func(
				context.Context,
				client.Client,
				string,
				string,
			) (*argocd.Application, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(health kargoapi.Health) {
				require.Equal(t, kargoapi.HealthStateUnknown, health.Status)
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
			getArgoCDAppFn: func(
				context.Context,
				client.Client,
				string,
				string,
			) (*argocd.Application, error) {
				return nil, nil
			},
			assertions: func(health kargoapi.Health) {
				require.Equal(t, kargoapi.HealthStateUnknown, health.Status)
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
				}, nil
			},
			assertions: func(health kargoapi.Health) {
				require.Equal(t, kargoapi.HealthStateNotApplicable, health.Status)
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
			getArgoCDAppFn: func(
				context.Context,
				client.Client,
				string,
				string,
			) (*argocd.Application, error) {
				return &argocd.Application{
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argoHealth.HealthStatusDegraded,
						},
					},
				}, nil
			},
			assertions: func(health kargoapi.Health) {
				require.Equal(t, kargoapi.HealthStateUnhealthy, health.Status)
				require.Len(t, health.Issues, 1)
				require.Contains(t, health.Issues[0], "has health state")
				require.Contains(t, health.Issues[0], argoHealth.HealthStatusDegraded)
			},
		},

		{
			name: "Argo CD App not synced",
			argoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
				{
					AppName:      "fake-app",
					AppNamespace: "fake-namespace",
				},
			},
			getArgoCDAppFn: func(
				context.Context,
				client.Client,
				string,
				string,
			) (*argocd.Application, error) {
				return &argocd.Application{
					Spec: argocd.ApplicationSpec{
						Source: &argocd.ApplicationSource{},
					},
					Status: argocd.ApplicationStatus{
						Health: argocd.HealthStatus{
							Status: argoHealth.HealthStatusHealthy,
						},
						Sync: argocd.SyncStatus{
							Status: argocd.SyncStatusCodeOutOfSync,
						},
					},
				}, nil
			},
			assertions: func(health kargoapi.Health) {
				require.Equal(t, kargoapi.HealthStateUnhealthy, health.Status)
				require.Len(t, health.Issues, 1)
				require.Contains(t, health.Issues[0], "is not synced to revision")
			},
		},

		{
			name: "Argo CD App healthy and synced",
			freight: kargoapi.Freight{
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
							Status: argoHealth.HealthStatusHealthy,
						},
						Sync: argocd.SyncStatus{
							Status:   argocd.SyncStatusCodeSynced,
							Revision: "fake-commit",
						},
					},
				}, nil
			},
			assertions: func(health kargoapi.Health) {
				require.Equal(t, kargoapi.HealthStateHealthy, health.Status)
				require.Empty(t, health.Issues)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := &reconciler{
				getArgoCDAppFn: testCase.getArgoCDAppFn,
			}
			testCase.assertions(
				reconciler.checkHealth(
					context.Background(),
					testCase.freight,
					testCase.argoCDAppUpdates,
				),
			)
		})
	}
}
