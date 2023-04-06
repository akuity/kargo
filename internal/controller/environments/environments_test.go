package environments

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/credentials"
)

func TestNewEnvironmentReconciler(t *testing.T) {
	e, err := newReconciler(
		fake.NewClientBuilder().Build(),
		&credentials.FakeDB{},
	)
	require.NoError(t, err)
	require.NotNil(t, e.client)
	require.NotNil(t, e.credentialsDB)

	// Assert that all overridable behaviors were initialized to a default:

	// Health checks:
	require.NotNil(t, e.getArgoCDAppFn)
	require.NotNil(t, e.checkHealthFn)

	// Syncing:
	require.NotNil(t, e.getLatestStateFromReposFn)
	require.NotNil(t, e.getAvailableStatesFromUpstreamEnvsFn)
	require.NotNil(t, e.getLatestCommitsFn)
	require.NotNil(t, e.getLatestImagesFn)
	require.NotNil(t, e.getLatestTagFn)
	require.NotNil(t, e.getLatestChartsFn)
	require.NotNil(t, e.getLatestChartVersionFn)
	require.NotNil(t, e.getLatestCommitIDFn)
}

func TestSync(t *testing.T) {
	testCases := []struct {
		name          string
		spec          api.EnvironmentSpec
		initialStatus api.EnvironmentStatus
		checkHealthFn func(
			context.Context,
			api.EnvironmentState,
			*api.HealthChecks,
		) api.Health
		getLatestStateFromReposFn func(
			context.Context,
			string,
			api.RepoSubscriptions,
		) (*api.EnvironmentState, error)
		getAvailableStatesFromUpstreamEnvsFn func(
			context.Context,
			[]api.EnvironmentSubscription,
		) ([]api.EnvironmentState, error)
		assertions func(initialStatus, newStatus api.EnvironmentStatus, client client.Client, err error)
	}{
		{
			name: "no subscriptions",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{},
			},
			initialStatus: api.EnvironmentStatus{},
			assertions: func(
				initialStatus api.EnvironmentStatus,
				newStatus api.EnvironmentStatus,
				_ client.Client,
				err error,
			) {
				require.NoError(t, err)
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "error getting latest state from repos",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			getLatestStateFromReposFn: func(
				context.Context,
				string,
				api.RepoSubscriptions,
			) (*api.EnvironmentState, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(
				initialStatus api.EnvironmentStatus,
				newStatus api.EnvironmentStatus,
				_ client.Client,
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				// Status should be unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "no latest state from repos",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			getLatestStateFromReposFn: func(
				context.Context,
				string,
				api.RepoSubscriptions,
			) (*api.EnvironmentState, error) {
				return nil, nil
			},
			assertions: func(
				initialStatus api.EnvironmentStatus,
				newStatus api.EnvironmentStatus,
				_ client.Client,
				err error,
			) {
				require.NoError(t, err)
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "latest state from repos isn't new",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
				HealthChecks: &api.HealthChecks{},
			},
			initialStatus: api.EnvironmentStatus{
				AvailableStates: []api.EnvironmentState{
					{
						Commits: []api.GitCommit{
							{
								RepoURL: "fake-url",
								ID:      "fake-commit",
							},
						},
						Images: []api.Image{
							{
								RepoURL: "fake-url",
								Tag:     "fake-tag",
							},
						},
					},
				},
				History: []api.EnvironmentState{
					{
						Commits: []api.GitCommit{
							{
								RepoURL: "fake-url",
								ID:      "fake-commit",
							},
						},
						Images: []api.Image{
							{
								RepoURL: "fake-url",
								Tag:     "fake-tag",
							},
						},
						Health: &api.Health{
							Status: api.HealthStateHealthy,
						},
					},
				},
			},
			checkHealthFn: func(
				context.Context,
				api.EnvironmentState,
				*api.HealthChecks,
			) api.Health {
				return api.Health{
					Status: api.HealthStateHealthy,
				}
			},
			getLatestStateFromReposFn: func(
				context.Context,
				string,
				api.RepoSubscriptions,
			) (*api.EnvironmentState, error) {
				return &api.EnvironmentState{
					Commits: []api.GitCommit{
						{
							RepoURL: "fake-url",
							ID:      "fake-commit",
						},
					},
					Images: []api.Image{
						{
							RepoURL: "fake-url",
							Tag:     "fake-tag",
						},
					},
				}, nil
			},
			assertions: func(
				initialStatus api.EnvironmentStatus,
				newStatus api.EnvironmentStatus,
				_ client.Client,
				err error,
			) {
				require.NoError(t, err)
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "error getting available states from upstream envs",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					UpstreamEnvs: []api.EnvironmentSubscription{
						{
							Name:      "fake-name",
							Namespace: "fake-namespace",
						},
					},
				},
			},
			getAvailableStatesFromUpstreamEnvsFn: func(
				context.Context,
				[]api.EnvironmentSubscription,
			) ([]api.EnvironmentState, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(
				initialStatus api.EnvironmentStatus,
				newStatus api.EnvironmentStatus,
				_ client.Client,
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				// Status should be unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "no latest state from upstream envs",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					UpstreamEnvs: []api.EnvironmentSubscription{
						{
							Name:      "fake-name",
							Namespace: "fake-namespace",
						},
					},
				},
			},
			getAvailableStatesFromUpstreamEnvsFn: func(
				context.Context,
				[]api.EnvironmentSubscription,
			) ([]api.EnvironmentState, error) {
				return nil, nil
			},
			assertions: func(
				initialStatus api.EnvironmentStatus,
				newStatus api.EnvironmentStatus,
				_ client.Client,
				err error,
			) {
				require.NoError(t, err)
				// Status should be unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "multiple upstream envs",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					UpstreamEnvs: []api.EnvironmentSubscription{
						// Subscribing to multiple upstream environments should block
						// auto-promotion
						{
							Name:      "fake-name",
							Namespace: "fake-namespace",
						},
						{
							Name:      "another-fake-name",
							Namespace: "another-fake-namespace",
						},
					},
				},
				EnableAutoPromotion: true,
			},
			getAvailableStatesFromUpstreamEnvsFn: func(
				context.Context,
				[]api.EnvironmentSubscription,
			) ([]api.EnvironmentState, error) {
				return []api.EnvironmentState{
					{},
					{},
				}, nil
			},
			assertions: func(
				initialStatus api.EnvironmentStatus,
				newStatus api.EnvironmentStatus,
				_ client.Client,
				err error,
			) {
				require.NoError(t, err)
				// Status should have updated AvailableStates and otherwise be unchanged
				require.Equal(
					t,
					api.EnvironmentStateStack{{}, {}},
					newStatus.AvailableStates,
				)
				newStatus.AvailableStates = initialStatus.AvailableStates
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "auto-promotion not enabled",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			getLatestStateFromReposFn: func(
				context.Context,
				string,
				api.RepoSubscriptions,
			) (*api.EnvironmentState, error) {
				return &api.EnvironmentState{
					Commits: []api.GitCommit{
						{
							RepoURL: "fake-url",
							ID:      "fake-commit",
						},
					},
					Images: []api.Image{
						{
							RepoURL: "fake-url",
							Tag:     "fake-tag",
						},
					},
				}, nil
			},
			assertions: func(
				initialStatus api.EnvironmentStatus,
				newStatus api.EnvironmentStatus,
				_ client.Client,
				err error,
			) {
				require.NoError(t, err)
				// Status should have updated AvailableStates and otherwise be unchanged
				require.Equal(
					t,
					api.EnvironmentStateStack{
						{
							Commits: []api.GitCommit{
								{
									RepoURL: "fake-url",
									ID:      "fake-commit",
								},
							},
							Images: []api.Image{
								{
									RepoURL: "fake-url",
									Tag:     "fake-tag",
								},
							},
						},
					},
					newStatus.AvailableStates,
				)
				newStatus.AvailableStates = initialStatus.AvailableStates
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "successful creation of promotion resource",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
				EnableAutoPromotion: true,
			},
			getLatestStateFromReposFn: func(
				context.Context,
				string,
				api.RepoSubscriptions,
			) (*api.EnvironmentState, error) {
				return &api.EnvironmentState{
					Commits: []api.GitCommit{
						{
							RepoURL: "fake-url",
							ID:      "fake-commit",
						},
					},
					Images: []api.Image{
						{
							RepoURL: "fake-url",
							Tag:     "fake-tag",
						},
					},
				}, nil
			},
			assertions: func(
				initialStatus api.EnvironmentStatus,
				newStatus api.EnvironmentStatus,
				client client.Client,
				err error,
			) {
				require.NoError(t, err)
				// Status should have updated AvailableStates and otherwise be unchanged
				require.Equal(
					t,
					api.EnvironmentStateStack{
						{
							Commits: []api.GitCommit{
								{
									RepoURL: "fake-url",
									ID:      "fake-commit",
								},
							},
							Images: []api.Image{
								{
									RepoURL: "fake-url",
									Tag:     "fake-tag",
								},
							},
						},
					},
					newStatus.AvailableStates,
				)
				newStatus.AvailableStates = initialStatus.AvailableStates
				require.Equal(t, initialStatus, newStatus)
				// And a Promotion should have been created
				promos := api.PromotionList{}
				err = client.List(context.Background(), &promos)
				require.NoError(t, err)
				require.Len(t, promos.Items, 1)
			},
		},
	}
	for _, testCase := range testCases {
		testEnv := &api.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec:   &testCase.spec,
			Status: testCase.initialStatus,
		}
		scheme, err := api.SchemeBuilder.Build()
		require.NoError(t, err)
		// nolint: lll
		reconciler := &reconciler{
			client:                               fake.NewClientBuilder().WithScheme(scheme).Build(),
			checkHealthFn:                        testCase.checkHealthFn,
			getLatestStateFromReposFn:            testCase.getLatestStateFromReposFn,
			getAvailableStatesFromUpstreamEnvsFn: testCase.getAvailableStatesFromUpstreamEnvsFn,
		}
		t.Run(testCase.name, func(t *testing.T) {
			newStatus, err := reconciler.sync(context.Background(), testEnv)
			testCase.assertions(
				testCase.initialStatus,
				newStatus,
				reconciler.client,
				err,
			)
		})
	}
}

func TestGetLatestStateFromRepos(t *testing.T) {
	testCases := []struct {
		name               string
		getLatestCommitsFn func(
			context.Context,
			string,
			[]api.GitSubscription,
		) ([]api.GitCommit, error)
		getLatestImagesFn func(
			context.Context,
			string,
			[]api.ImageSubscription,
		) ([]api.Image, error)
		getLatestChartsFn func(
			context.Context,
			string,
			[]api.ChartSubscription,
		) ([]api.Chart, error)
		assertions func(*api.EnvironmentState, error)
	}{
		{
			name: "error getting latest git commit",
			getLatestCommitsFn: func(
				context.Context,
				string,
				[]api.GitSubscription,
			) ([]api.GitCommit, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(state *api.EnvironmentState, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error syncing git repo subscription")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "error getting latest images",
			getLatestCommitsFn: func(
				context.Context,
				string,
				[]api.GitSubscription,
			) ([]api.GitCommit, error) {
				return nil, nil
			},
			getLatestImagesFn: func(
				context.Context,
				string,
				[]api.ImageSubscription,
			) ([]api.Image, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(state *api.EnvironmentState, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error syncing image repo subscriptions",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "error getting latest charts",
			getLatestCommitsFn: func(
				context.Context,
				string,
				[]api.GitSubscription,
			) ([]api.GitCommit, error) {
				return nil, nil
			},
			getLatestImagesFn: func(
				context.Context,
				string,
				[]api.ImageSubscription,
			) ([]api.Image, error) {
				return nil, nil
			},
			getLatestChartsFn: func(
				context.Context,
				string,
				[]api.ChartSubscription,
			) ([]api.Chart, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(state *api.EnvironmentState, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error syncing chart repo subscriptions",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "success",
			getLatestCommitsFn: func(
				context.Context,
				string,
				[]api.GitSubscription,
			) ([]api.GitCommit, error) {
				return []api.GitCommit{
					{
						RepoURL: "fake-url",
						ID:      "fake-commit",
					},
				}, nil
			},
			getLatestImagesFn: func(
				context.Context,
				string,
				[]api.ImageSubscription,
			) ([]api.Image, error) {
				return []api.Image{
					{
						RepoURL: "fake-url",
						Tag:     "fake-tag",
					},
				}, nil
			},
			getLatestChartsFn: func(
				context.Context,
				string,
				[]api.ChartSubscription,
			) ([]api.Chart, error) {
				return []api.Chart{
					{
						RegistryURL: "fake-registry",
						Name:        "fake-chart",
						Version:     "fake-version",
					},
				}, nil
			},
			assertions: func(state *api.EnvironmentState, err error) {
				require.NoError(t, err)
				require.NotNil(t, state)
				require.NotEmpty(t, state.ID)
				require.NotNil(t, state.FirstSeen)
				// All other fields should have a predictable value
				state.ID = ""
				state.FirstSeen = nil
				require.Equal(
					t,
					&api.EnvironmentState{
						Commits: []api.GitCommit{
							{
								RepoURL: "fake-url",
								ID:      "fake-commit",
							},
						},
						Images: []api.Image{
							{
								RepoURL: "fake-url",
								Tag:     "fake-tag",
							},
						},
						Charts: []api.Chart{
							{
								RegistryURL: "fake-registry",
								Name:        "fake-chart",
								Version:     "fake-version",
							},
						},
					},
					state,
				)
			},
		},
	}
	for _, testCase := range testCases {
		testReconciler := &reconciler{
			getLatestCommitsFn: testCase.getLatestCommitsFn,
			getLatestImagesFn:  testCase.getLatestImagesFn,
			getLatestChartsFn:  testCase.getLatestChartsFn,
		}
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testReconciler.getLatestStateFromRepos(
					context.Background(),
					"fake-namespace",
					api.RepoSubscriptions{},
				),
			)
		})
	}
}
