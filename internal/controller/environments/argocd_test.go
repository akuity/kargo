package environments

import (
	"context"
	"testing"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argoHealth "github.com/argoproj/gitops-engine/pkg/health"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/akuityio/kargo/api/v1alpha1"
)

func TestCheckHealth(t *testing.T) {
	testCases := []struct {
		name           string
		state          api.EnvironmentState
		healthChecks   *api.HealthChecks
		getArgoCDAppFn func(
			context.Context,
			client.Client,
			string,
			string,
		) (*argocd.Application, error)
		assertions func(api.Health)
	}{
		{
			name: "healthchecks are nil",
			assertions: func(health api.Health) {
				require.Equal(t,
					api.Health{
						Status:       api.HealthStateUnknown,
						StatusReason: "spec.healthChecks is undefined",
					},
					health,
				)
			},
		},

		{
			name:         "healthchecks do not include any Argo CD Apps",
			healthChecks: &api.HealthChecks{},
			assertions: func(health api.Health) {
				require.Equal(t,
					api.Health{
						Status: api.HealthStateUnknown,
						StatusReason: "spec.healthChecks contains insufficient " +
							"instructions to assess Environment health",
					},
					health,
				)
			},
		},

		{
			name: "error finding Argo CD App",
			healthChecks: &api.HealthChecks{
				ArgoCDAppChecks: []api.ArgoCDAppCheck{
					{
						AppName:      "fake-app",
						AppNamespace: "fake-namespace",
					},
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
			assertions: func(health api.Health) {
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
			healthChecks: &api.HealthChecks{
				ArgoCDAppChecks: []api.ArgoCDAppCheck{
					{
						AppName:      "fake-app",
						AppNamespace: "fake-namespace",
					},
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
			assertions: func(health api.Health) {
				require.Equal(t, api.HealthStateUnknown, health.Status)
				require.Contains(
					t,
					health.StatusReason,
					"unable to find Argo CD Application",
				)
			},
		},

		{
			name: "Argo CD App is multi-source",
			// This doesn't require there to actually BE multiple sources. Simply
			// using the sources field instead of the source fields should be enough
			// to trigger this case.
			healthChecks: &api.HealthChecks{
				ArgoCDAppChecks: []api.ArgoCDAppCheck{
					{
						AppName:      "fake-app",
						AppNamespace: "fake-namespace",
					},
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
			assertions: func(health api.Health) {
				require.Equal(t, api.HealthStateUnknown, health.Status)
				require.Contains(
					t,
					health.StatusReason,
					"bugs in Argo CD currently prevent a comprehensive assessment of "+
						"the health of multi-source Application",
				)
			},
		},

		{
			name: "Argo CD App is not healthy",
			healthChecks: &api.HealthChecks{
				ArgoCDAppChecks: []api.ArgoCDAppCheck{
					{
						AppName:      "fake-app",
						AppNamespace: "fake-namespace",
					},
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
			assertions: func(health api.Health) {
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
			name: "Argo CD App not synced",
			healthChecks: &api.HealthChecks{
				ArgoCDAppChecks: []api.ArgoCDAppCheck{
					{
						AppName:      "fake-app",
						AppNamespace: "fake-namespace",
					},
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
			assertions: func(health api.Health) {
				require.Equal(t, api.HealthStateUnhealthy, health.Status)
				require.Contains(
					t,
					health.StatusReason,
					"is not synced to revision",
				)
			},
		},

		{
			name: "Argo CD App healthy and synced",
			state: api.EnvironmentState{
				Commits: []api.GitCommit{
					{
						RepoURL: "fake-url",
						ID:      "fake-commit",
					},
				},
			},
			healthChecks: &api.HealthChecks{
				ArgoCDAppChecks: []api.ArgoCDAppCheck{
					{
						AppName:      "fake-app",
						AppNamespace: "fake-namespace",
					},
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
			assertions: func(health api.Health) {
				require.Equal(t, api.HealthStateHealthy, health.Status)
				require.Empty(t, health.StatusReason)
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
					testCase.state,
					testCase.healthChecks,
				),
			)
		})
	}
}
