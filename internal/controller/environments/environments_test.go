package environments

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

func TestNewEnvironmentReconciler(t *testing.T) {
	e := newReconciler(
		fake.NewClientBuilder().Build(),
		&credentials.FakeDB{},
	)
	require.NotNil(t, e.client)
	require.NotNil(t, e.credentialsDB)

	// Assert that all overridable behaviors were initialized to a default:

	// Loop guard:
	require.NotNil(t, e.hasOutstandingPromotionsFn)

	// Common:
	require.NotNil(t, e.getArgoCDAppFn)

	// Health checks:
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

func TestIndexEnvsByApp(t *testing.T) {
	testCases := []struct {
		name        string
		environment *api.Environment
		assertions  func([]string)
	}{
		{
			name: "environment has no health checks",
			environment: &api.Environment{
				Spec: &api.EnvironmentSpec{},
			},
			assertions: func(res []string) {
				require.Nil(t, res)
			},
		},
		{
			name: "environment has health checks",
			environment: &api.Environment{
				Spec: &api.EnvironmentSpec{
					HealthChecks: &api.HealthChecks{
						ArgoCDAppChecks: []api.ArgoCDAppCheck{
							{
								AppNamespace: "fake-namespace",
								AppName:      "fake-app",
							},
							{
								AppNamespace: "another-fake-namespace",
								AppName:      "another-fake-app",
							},
						},
					},
				},
			},
			assertions: func(res []string) {
				require.Equal(
					t,
					[]string{
						"fake-namespace:fake-app",
						"another-fake-namespace:another-fake-app",
					},
					res,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			indexEnvsByApp(testCase.environment)
		})
	}
}

func TestIndexOutstandingPromotionsByEnvironment(t *testing.T) {
	testCases := []struct {
		name       string
		promotion  *api.Promotion
		assertions func([]string)
	}{
		{
			name: "promotion is in terminal phase",
			promotion: &api.Promotion{
				Spec: &api.PromotionSpec{
					Environment: "fake-env",
				},
				Status: api.PromotionStatus{
					Phase: api.PromotionPhaseComplete,
				},
			},
			assertions: func(res []string) {
				require.Nil(t, res)
			},
		},
		{
			name: "promotion is in terminal phase",
			promotion: &api.Promotion{
				Spec: &api.PromotionSpec{
					Environment: "fake-env",
				},
				Status: api.PromotionStatus{
					Phase: api.PromotionPhasePending,
				},
			},
			assertions: func(res []string) {
				require.Equal(t, []string{"fake-env"}, res)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			indexOutstandingPromotionsByEnvironment(testCase.promotion)
		})
	}
}

func TestSync(t *testing.T) {
	scheme, err := api.SchemeBuilder.Build()
	require.NoError(t, err)

	noOutstandingPromotionsFn := func(
		context.Context,
		string,
		string,
	) (bool, error) {
		return false, nil
	}

	testCases := []struct {
		name                       string
		spec                       api.EnvironmentSpec
		initialStatus              api.EnvironmentStatus
		hasOutstandingPromotionsFn func(
			ctx context.Context,
			envNamespace string,
			envName string,
		) (bool, error)
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
		client     client.Client
		assertions func(
			initialStatus api.EnvironmentStatus,
			newStatus api.EnvironmentStatus,
			client client.Client,
			err error,
		)
	}{
		{
			name: "error checking for outstanding promotions",
			hasOutstandingPromotionsFn: func(
				context.Context,
				string,
				string,
			) (bool, error) {
				return false, errors.New("something went wrong")
			},
			assertions: func(
				initialStatus api.EnvironmentStatus,
				newStatus api.EnvironmentStatus,
				_ client.Client,
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "outstanding promotions found",
			hasOutstandingPromotionsFn: func(
				context.Context,
				string,
				string,
			) (bool, error) {
				return true, nil
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
			name: "no subscriptions",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{},
			},
			initialStatus:              api.EnvironmentStatus{},
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
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
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
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
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
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
				CurrentState: &api.EnvironmentState{
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
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
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
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
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
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
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
			},
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
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
			name: "no promotion policy found",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
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
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
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
				// And no Promotion should have been created
				promos := api.PromotionList{}
				err = client.List(context.Background(), &promos)
				require.NoError(t, err)
				require.Empty(t, promos.Items)
			},
		},

		{
			name: "multiple promotion policies found",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
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
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&api.PromotionPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-policy",
						Namespace: "fake-namespace",
					},
					Environment: "fake-environment",
				},
				&api.PromotionPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "another-fake-policy",
						Namespace: "fake-namespace",
					},
					Environment: "fake-environment",
				},
			).Build(),
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
				// And no Promotion should have been created
				promos := api.PromotionList{}
				err = client.List(context.Background(), &promos)
				require.NoError(t, err)
				require.Empty(t, promos.Items)
			},
		},

		{
			name: "auto-promotion not enabled",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
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
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&api.PromotionPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-policy",
						Namespace: "fake-namespace",
					},
					Environment: "fake-environment",
				},
			).Build(),
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
				// And no Promotion should have been created
				promos := api.PromotionList{}
				err = client.List(context.Background(), &promos)
				require.NoError(t, err)
				require.Empty(t, promos.Items)
			},
		},

		{
			name: "auto-promotion enabled",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
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
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&api.PromotionPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-policy",
						Namespace: "fake-namespace",
					},
					Environment:         "fake-environment",
					EnableAutoPromotion: true,
				},
			).Build(),
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
				Name:      "fake-environment",
				Namespace: "fake-namespace",
			},
			Spec:   &testCase.spec,
			Status: testCase.initialStatus,
		}
		// nolint: lll
		reconciler := &reconciler{
			client:                               testCase.client,
			hasOutstandingPromotionsFn:           testCase.hasOutstandingPromotionsFn,
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
