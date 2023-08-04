package stages

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

func TestNewStageReconciler(t *testing.T) {
	client := fake.NewClientBuilder().Build()
	e := newReconciler(
		client,
		client,
		&credentials.FakeDB{},
	)
	require.NotNil(t, e.kargoClient)
	require.NotNil(t, e.argoClient)
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
	require.NotNil(t, e.getAvailableStatesFromUpstreamStagesFn)
	require.NotNil(t, e.getLatestCommitsFn)
	require.NotNil(t, e.getLatestImagesFn)
	require.NotNil(t, e.getLatestTagFn)
	require.NotNil(t, e.getLatestChartsFn)
	require.NotNil(t, e.getLatestChartVersionFn)
	require.NotNil(t, e.getLatestCommitIDFn)
}

func TestSync(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, api.SchemeBuilder.AddToScheme(scheme))

	noOutstandingPromotionsFn := func(
		context.Context,
		string,
		string,
	) (bool, error) {
		return false, nil
	}

	testCases := []struct {
		name                       string
		spec                       api.StageSpec
		initialStatus              api.StageStatus
		hasOutstandingPromotionsFn func(
			ctx context.Context,
			stageNamespace string,
			stageName string,
		) (bool, error)
		checkHealthFn func(
			context.Context,
			api.StageState,
			[]api.ArgoCDAppUpdate,
		) api.Health
		getLatestStateFromReposFn func(
			context.Context,
			string,
			api.RepoSubscriptions,
		) (*api.StageState, error)
		getAvailableStatesFromUpstreamStagesFn func(
			ctx context.Context,
			namespace string,
			subs []api.StageSubscription,
		) ([]api.StageState, error)
		kargoClient client.Client
		assertions  func(
			initialStatus api.StageStatus,
			newStatus api.StageStatus,
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
				initialStatus api.StageStatus,
				newStatus api.StageStatus,
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
				initialStatus api.StageStatus,
				newStatus api.StageStatus,
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
			spec: api.StageSpec{
				Subscriptions: &api.Subscriptions{},
			},
			initialStatus:              api.StageStatus{},
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
			assertions: func(
				initialStatus api.StageStatus,
				newStatus api.StageStatus,
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
			spec: api.StageSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
			getLatestStateFromReposFn: func(
				context.Context,
				string,
				api.RepoSubscriptions,
			) (*api.StageState, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(
				initialStatus api.StageStatus,
				newStatus api.StageStatus,
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
			spec: api.StageSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
			getLatestStateFromReposFn: func(
				context.Context,
				string,
				api.RepoSubscriptions,
			) (*api.StageState, error) {
				return nil, nil
			},
			assertions: func(
				initialStatus api.StageStatus,
				newStatus api.StageStatus,
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
			spec: api.StageSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
				// TODO: I'm not sure about this change
				// HealthChecks: &api.HealthChecks{},
			},
			initialStatus: api.StageStatus{
				AvailableStates: []api.StageState{
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
				CurrentState: &api.StageState{
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
				History: []api.StageState{
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
				api.StageState,
				[]api.ArgoCDAppUpdate,
			) api.Health {
				return api.Health{
					Status: api.HealthStateHealthy,
				}
			},
			getLatestStateFromReposFn: func(
				context.Context,
				string,
				api.RepoSubscriptions,
			) (*api.StageState, error) {
				return &api.StageState{
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
				initialStatus api.StageStatus,
				newStatus api.StageStatus,
				_ client.Client,
				err error,
			) {
				require.NoError(t, err)
				// Status should be returned unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "error getting available states from upstream Stages",
			spec: api.StageSpec{
				Subscriptions: &api.Subscriptions{
					UpstreamStages: []api.StageSubscription{
						{
							Name: "fake-name",
						},
					},
				},
			},
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
			getAvailableStatesFromUpstreamStagesFn: func(
				context.Context,
				string,
				[]api.StageSubscription,
			) ([]api.StageState, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(
				initialStatus api.StageStatus,
				newStatus api.StageStatus,
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
			name: "no latest state from upstream Stages",
			spec: api.StageSpec{
				Subscriptions: &api.Subscriptions{
					UpstreamStages: []api.StageSubscription{
						{
							Name: "fake-name",
						},
					},
				},
			},
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
			getAvailableStatesFromUpstreamStagesFn: func(
				context.Context,
				string,
				[]api.StageSubscription,
			) ([]api.StageState, error) {
				return nil, nil
			},
			assertions: func(
				initialStatus api.StageStatus,
				newStatus api.StageStatus,
				_ client.Client,
				err error,
			) {
				require.NoError(t, err)
				// Status should be unchanged
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "multiple upstream Stages",
			spec: api.StageSpec{
				Subscriptions: &api.Subscriptions{
					UpstreamStages: []api.StageSubscription{
						// Subscribing to multiple upstream Stages should block
						// auto-promotion
						{
							Name: "fake-name",
						},
						{
							Name: "another-fake-name",
						},
					},
				},
			},
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
			getAvailableStatesFromUpstreamStagesFn: func(
				context.Context,
				string,
				[]api.StageSubscription,
			) ([]api.StageState, error) {
				return []api.StageState{
					{},
					{},
				}, nil
			},
			assertions: func(
				initialStatus api.StageStatus,
				newStatus api.StageStatus,
				_ client.Client,
				err error,
			) {
				require.NoError(t, err)
				// Status should have updated AvailableStates and otherwise be unchanged
				require.Equal(
					t,
					api.StageStateStack{{}, {}},
					newStatus.AvailableStates,
				)
				newStatus.AvailableStates = initialStatus.AvailableStates
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "no promotion policy found",
			spec: api.StageSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
			getLatestStateFromReposFn: func(
				context.Context,
				string,
				api.RepoSubscriptions,
			) (*api.StageState, error) {
				return &api.StageState{
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
			kargoClient: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(
				initialStatus api.StageStatus,
				newStatus api.StageStatus,
				client client.Client,
				err error,
			) {
				require.NoError(t, err)
				// Status should have updated AvailableStates and otherwise be unchanged
				require.Equal(
					t,
					api.StageStateStack{
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
			spec: api.StageSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
			getLatestStateFromReposFn: func(
				context.Context,
				string,
				api.RepoSubscriptions,
			) (*api.StageState, error) {
				return &api.StageState{
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
			kargoClient: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&api.PromotionPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-policy",
						Namespace: "fake-namespace",
					},
					Stage: "fake-stage",
				},
				&api.PromotionPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "another-fake-policy",
						Namespace: "fake-namespace",
					},
					Stage: "fake-stage",
				},
			).Build(),
			assertions: func(
				initialStatus api.StageStatus,
				newStatus api.StageStatus,
				client client.Client,
				err error,
			) {
				require.NoError(t, err)
				// Status should have updated AvailableStates and otherwise be unchanged
				require.Equal(
					t,
					api.StageStateStack{
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
			spec: api.StageSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
			getLatestStateFromReposFn: func(
				context.Context,
				string,
				api.RepoSubscriptions,
			) (*api.StageState, error) {
				return &api.StageState{
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
			kargoClient: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&api.PromotionPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-policy",
						Namespace: "fake-namespace",
					},
					Stage: "fake-stage",
				},
			).Build(),
			assertions: func(
				initialStatus api.StageStatus,
				newStatus api.StageStatus,
				client client.Client,
				err error,
			) {
				require.NoError(t, err)
				// Status should have updated AvailableStates and otherwise be unchanged
				require.Equal(
					t,
					api.StageStateStack{
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
			spec: api.StageSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			hasOutstandingPromotionsFn: noOutstandingPromotionsFn,
			getLatestStateFromReposFn: func(
				context.Context,
				string,
				api.RepoSubscriptions,
			) (*api.StageState, error) {
				return &api.StageState{
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
			kargoClient: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&api.PromotionPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-policy",
						Namespace: "fake-namespace",
					},
					Stage:               "fake-stage",
					EnableAutoPromotion: true,
				},
			).Build(),
			assertions: func(
				initialStatus api.StageStatus,
				newStatus api.StageStatus,
				client client.Client,
				err error,
			) {
				require.NoError(t, err)
				// Status should have updated AvailableStates and otherwise be unchanged
				require.Equal(
					t,
					api.StageStateStack{
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
		testStage := &api.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-stage",
				Namespace: "fake-namespace",
			},
			Spec:   &testCase.spec,
			Status: testCase.initialStatus,
		}
		// nolint: lll
		reconciler := &reconciler{
			kargoClient:                            testCase.kargoClient,
			hasOutstandingPromotionsFn:             testCase.hasOutstandingPromotionsFn,
			checkHealthFn:                          testCase.checkHealthFn,
			getLatestStateFromReposFn:              testCase.getLatestStateFromReposFn,
			getAvailableStatesFromUpstreamStagesFn: testCase.getAvailableStatesFromUpstreamStagesFn,
		}
		t.Run(testCase.name, func(t *testing.T) {
			newStatus, err := reconciler.sync(context.Background(), testStage)
			testCase.assertions(
				testCase.initialStatus,
				newStatus,
				testCase.kargoClient,
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
		assertions func(*api.StageState, error)
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
			assertions: func(state *api.StageState, err error) {
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
			assertions: func(state *api.StageState, err error) {
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
			assertions: func(state *api.StageState, err error) {
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
			assertions: func(state *api.StageState, err error) {
				require.NoError(t, err)
				require.NotNil(t, state)
				require.NotEmpty(t, state.ID)
				require.NotNil(t, state.FirstSeen)
				// All other fields should have a predictable value
				state.ID = ""
				state.FirstSeen = nil
				require.Equal(
					t,
					&api.StageState{
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
