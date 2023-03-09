package controller

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/akuityio/bookkeeper"
	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/config"
)

func TestNewEnvironmentReconciler(t *testing.T) {
	testConfig := config.ControllerConfig{
		BaseConfig: config.BaseConfig{
			LogLevel: log.DebugLevel,
		},
	}
	e := newEnvironmentReconciler(
		testConfig,
		nil, // TODO: Don't know an easy way to mock this yet
		nil, // TODO: Don't know an easy way to mock this yet
		nil, // TODO: Don't know an easy way to mock this yet
		bookkeeper.NewService(nil),
	)
	require.Equal(t, testConfig, e.config)
	require.NotNil(t, e.logger)
	require.Equal(t, testConfig.LogLevel, e.logger.Level)
	// Assert that all overridable behaviors were initialized to a default:

	// Common:
	require.NotNil(t, e.getArgoCDAppFn)
	require.NotNil(t, e.gitRepoCredentialsFn)

	// Health checks:
	require.NotNil(t, e.checkHealthFn)

	// Syncing:
	require.NotNil(t, e.getLatestStateFromReposFn)
	require.NotNil(t, e.getAvailableStatesFromUpstreamEnvsFn)
	require.NotNil(t, e.getLatestCommitsFn)
	require.NotNil(t, e.getLatestImagesFn)
	require.NotNil(t, e.getLatestTagFn)
	require.NotNil(t, e.chartRegistryCredentialsFn)
	require.NotNil(t, e.getLatestChartsFn)
	require.NotNil(t, e.getLatestChartVersionFn)
	require.NotNil(t, e.getLatestCommitIDFn)

	// Promotions (general):
	require.NotNil(t, e.promoteFn)
	// Promotions via Git:
	require.NotNil(t, e.gitApplyUpdateFn)
	// Promotions via Git + Kustomize:
	require.NotNil(t, e.kustomizeSetImageFn)
	// Promotions via Git + Helm:
	require.NotNil(t, e.buildChartDependencyChangesFn)
	require.NotNil(t, e.updateChartDependenciesFn)
	require.NotNil(t, e.setStringsInYAMLFileFn)
	// Promotions via Argo CD:
	require.NotNil(t, e.applyArgoCDSourceUpdateFn)
	// TODO: Can't check this until we figure out how to mock a controller runtime
	// client
	// require.NotNil(t, e.patchFn)
}

func TestSync(t *testing.T) {
	testCases := []struct {
		name          string
		spec          api.EnvironmentSpec
		initialStatus api.EnvironmentStatus
		checkHealthFn func(
			context.Context,
			api.EnvironmentState,
			api.HealthChecks,
		) api.Health
		getLatestStateFromReposFn func(
			context.Context,
			api.RepoSubscriptions,
		) (*api.EnvironmentState, error)
		getAvailableStatesFromUpstreamEnvsFn func(
			context.Context,
			[]api.EnvironmentSubscription,
		) ([]api.EnvironmentState, error)
		promoteFn func(
			context.Context,
			api.PromotionMechanisms,
			api.EnvironmentState,
		) (api.EnvironmentState, error)
		assertions func(initialStatus, newStatus api.EnvironmentStatus)
	}{
		{
			name: "no subscriptions",
			spec: api.EnvironmentSpec{
				Subscriptions:       &api.Subscriptions{},
				PromotionMechanisms: &api.PromotionMechanisms{},
				HealthChecks:        &api.HealthChecks{},
			},
			initialStatus: api.EnvironmentStatus{},
			assertions: func(initialStatus, newStatus api.EnvironmentStatus) {
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
				PromotionMechanisms: &api.PromotionMechanisms{},
				HealthChecks:        &api.HealthChecks{},
			},
			getLatestStateFromReposFn: func(
				context.Context,
				api.RepoSubscriptions,
			) (*api.EnvironmentState, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(initialStatus, newStatus api.EnvironmentStatus) {
				// Status should be returned unchanged -- except for Error field
				require.Equal(t, "something went wrong", newStatus.Error)
				newStatus.Error = ""
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "no latest state from repos",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
				PromotionMechanisms: &api.PromotionMechanisms{},
				HealthChecks:        &api.HealthChecks{},
			},
			getLatestStateFromReposFn: func(
				context.Context,
				api.RepoSubscriptions,
			) (*api.EnvironmentState, error) {
				return nil, nil
			},
			assertions: func(initialStatus, newStatus api.EnvironmentStatus) {
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
				PromotionMechanisms: &api.PromotionMechanisms{},
				HealthChecks:        &api.HealthChecks{},
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
				States: []api.EnvironmentState{
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
				api.HealthChecks,
			) api.Health {
				return api.Health{
					Status: api.HealthStateHealthy,
				}
			},
			getLatestStateFromReposFn: func(
				context.Context,
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
			assertions: func(initialStatus, newStatus api.EnvironmentStatus) {
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
				PromotionMechanisms: &api.PromotionMechanisms{},
				HealthChecks:        &api.HealthChecks{},
			},
			getAvailableStatesFromUpstreamEnvsFn: func(
				context.Context,
				[]api.EnvironmentSubscription,
			) ([]api.EnvironmentState, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(initialStatus, newStatus api.EnvironmentStatus) {
				// Status should be returned unchanged -- except for Error field
				require.Equal(t, "something went wrong", newStatus.Error)
				newStatus.Error = ""
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "not auto-promotion eligible",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					UpstreamEnvs: []api.EnvironmentSubscription{
						{
							Name:      "fake-name",
							Namespace: "fake-namespace",
						},
						{
							Name:      "another-fake-name",
							Namespace: "fake-namespace",
						},
					},
				},
				PromotionMechanisms: &api.PromotionMechanisms{},
				HealthChecks:        &api.HealthChecks{},
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
			assertions: func(initialStatus, newStatus api.EnvironmentStatus) {
				// Status should have updated AvailableStates updated and no Error
				require.Empty(t, newStatus.Error)
				require.Equal(
					t,
					api.EnvironmentStateStack{{}, {}},
					newStatus.AvailableStates,
				)
			},
		},

		{
			name: "error executing promotion",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
				PromotionMechanisms: &api.PromotionMechanisms{},
				HealthChecks:        &api.HealthChecks{},
			},
			getLatestStateFromReposFn: func(
				context.Context,
				api.RepoSubscriptions,
			) (*api.EnvironmentState, error) {
				return &api.EnvironmentState{}, nil
			},
			promoteFn: func(
				_ context.Context,
				_ api.PromotionMechanisms,
				newState api.EnvironmentState,
			) (api.EnvironmentState, error) {
				return newState, errors.New("something went wrong")
			},
			assertions: func(initialStatus, newStatus api.EnvironmentStatus) {
				// Status should be returned unchanged -- except for AvailableStates and
				// Error fields
				require.Equal(t, "something went wrong", newStatus.Error)
				require.NotEmpty(t, newStatus.AvailableStates)
				newStatus.AvailableStates = nil
				newStatus.Error = ""
				require.Equal(t, initialStatus, newStatus)
			},
		},

		{
			name: "successful promotion",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
				PromotionMechanisms: &api.PromotionMechanisms{},
				HealthChecks:        &api.HealthChecks{},
			},
			getLatestStateFromReposFn: func(
				context.Context,
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
			promoteFn: func(
				_ context.Context,
				_ api.PromotionMechanisms,
				newState api.EnvironmentState,
			) (api.EnvironmentState, error) {
				return newState, nil
			},
			assertions: func(_, newStatus api.EnvironmentStatus) {
				// Status should reflect the new state
				require.Empty(t, newStatus.Error)
				require.Len(t, newStatus.AvailableStates, 1)
				require.Len(t, newStatus.States, 1)
			},
		},
	}
	for _, testCase := range testCases {
		testEnv := &api.Environment{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec:   &testCase.spec,
			Status: testCase.initialStatus,
		}
		reconciler := &environmentReconciler{
			logger:                               log.New(),
			checkHealthFn:                        testCase.checkHealthFn,
			getLatestStateFromReposFn:            testCase.getLatestStateFromReposFn,
			getAvailableStatesFromUpstreamEnvsFn: testCase.getAvailableStatesFromUpstreamEnvsFn, // nolint: lll
			promoteFn:                            testCase.promoteFn,
		}
		reconciler.logger.SetLevel(log.ErrorLevel)
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.initialStatus,
				reconciler.sync(context.Background(), testEnv),
			)
		})
	}
}

func TestGetLatestStateFromRepos(t *testing.T) {
	testCases := []struct {
		name               string
		getLatestCommitsFn func(
			context.Context,
			[]api.GitSubscription,
		) ([]api.GitCommit, error)
		getLatestImagesFn func(
			context.Context,
			[]api.ImageSubscription,
		) ([]api.Image, error)
		getLatestChartsFn func(
			context.Context,
			[]api.ChartSubscription,
		) ([]api.Chart, error)
		assertions func(*api.EnvironmentState, error)
	}{
		{
			name: "error getting latest git commit",
			getLatestCommitsFn: func(
				context.Context,
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
				[]api.GitSubscription,
			) ([]api.GitCommit, error) {
				return nil, nil
			},
			getLatestImagesFn: func(
				context.Context,
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
				[]api.GitSubscription,
			) ([]api.GitCommit, error) {
				return nil, nil
			},
			getLatestImagesFn: func(
				context.Context,
				[]api.ImageSubscription,
			) ([]api.Image, error) {
				return nil, nil
			},
			getLatestChartsFn: func(
				context.Context,
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
		testReconciler := &environmentReconciler{
			logger:             log.New(),
			getLatestCommitsFn: testCase.getLatestCommitsFn,
			getLatestImagesFn:  testCase.getLatestImagesFn,
			getLatestChartsFn:  testCase.getLatestChartsFn,
		}
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testReconciler.getLatestStateFromRepos(
					context.Background(),
					api.RepoSubscriptions{},
				),
			)
		})
	}
}
