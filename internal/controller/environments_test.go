package controller

import (
	"context"
	"testing"

	"github.com/akuityio/bookkeeper"
	api "github.com/akuityio/k8sta/api/v1alpha1"
	"github.com/akuityio/k8sta/internal/common/config"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewEnvironmentReconciler(t *testing.T) {
	testConfig := config.Config{
		LogLevel: log.DebugLevel,
	}
	e := newEnvironmentReconciler(
		testConfig,
		nil, // TODO: Don't know an easy way to mock this yet
		nil, // TODO: Don't know an easy way to mock this yet
		nil, // TODO: Don't know an easy way to mock this yet
		bookkeeper.NewService(nil),
	)
	require.Equal(t, testConfig, e.config)
	require.NotNil(t, e.bookkeeperService)
	require.NotNil(t, e.logger)
	require.Equal(t, testConfig.LogLevel, e.logger.Level)
	// Assert that all overridable behaviors were initialized to a default
	require.NotNil(t, e.getNextStateFn)
	require.NotNil(t, e.getNextStateFromUpstreamReposFn)
	require.NotNil(t, e.getLatestCommitFn)
	require.NotNil(t, e.getGitRepoCredentialsFn)
	require.NotNil(t, e.gitCloneFn)
	require.NotNil(t, e.checkoutBranchFn)
	require.NotNil(t, e.getLastCommitIDFn)
	require.NotNil(t, e.getLatestImagesFn)
	require.NotNil(t, e.getImageRepoCredentialsFn)
	require.NotNil(t, e.getImageTagsFn)
	require.NotNil(t, e.getNewestImageTagFn)
	require.NotNil(t, e.promoteFn)
}

func TestSync(t *testing.T) {
	testCases := []struct {
		name          string
		initialStatus api.EnvironmentStatus
		nextStateFn   func(
			context.Context,
			*api.Environment,
		) (*api.EnvironmentState, error)
		promoteFn func(
			ctx context.Context,
			env *api.Environment,
			newState api.EnvironmentState,
		) (api.EnvironmentState, error)
		assertions func(initialStatus, newStatus api.EnvironmentStatus)
	}{
		{
			name: "error getting next state",
			// Status should be returned unchanged -- except for Error field
			nextStateFn: func(
				context.Context,
				*api.Environment,
			) (*api.EnvironmentState, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(initialStatus, newStatus api.EnvironmentStatus) {
				require.Equal(t, "something went wrong", newStatus.Error)
				newStatus.Error = ""
				require.Equal(t, initialStatus, newStatus)
			},
		},
		{
			name: "next state is nil",
			// Status should be returned unchanged
			nextStateFn: func(
				context.Context,
				*api.Environment,
			) (*api.EnvironmentState, error) {
				return nil, nil
			},
			assertions: func(initialStatus, newStatus api.EnvironmentStatus) {
				require.Equal(t, initialStatus, newStatus)
			},
		},
		{
			name: "next state isn't different from existing state",
			// Status should be returned unchanged
			initialStatus: api.EnvironmentStatus{
				States: []api.EnvironmentState{
					{
						GitCommit: &api.GitCommit{
							RepoURL: "fake-url",
							ID:      "fake-commit",
						},
						Images: []api.Image{
							{
								RepoURL: "fake-url",
								Tag:     "fake-tag",
							},
						},
					},
				},
			},
			nextStateFn: func(
				context.Context,
				*api.Environment,
			) (*api.EnvironmentState, error) {
				return &api.EnvironmentState{
					GitCommit: &api.GitCommit{
						RepoURL: "fake-url",
						ID:      "fake-commit",
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
				require.Equal(t, initialStatus, newStatus)
			},
		},
		{
			name: "error executing promotion",
			// Status should be returned unchanged -- except for Error field
			nextStateFn: func(
				context.Context,
				*api.Environment,
			) (*api.EnvironmentState, error) {
				return &api.EnvironmentState{}, nil
			},
			promoteFn: func(
				_ context.Context,
				_ *api.Environment,
				newState api.EnvironmentState,
			) (api.EnvironmentState, error) {
				return newState, errors.New("something went wrong")
			},
			assertions: func(initialStatus, newStatus api.EnvironmentStatus) {
				require.Equal(t, "something went wrong", newStatus.Error)
				newStatus.Error = ""
				require.Equal(t, initialStatus, newStatus)
			},
		},
		{
			name: "successful promotion",
			// Status should reflect the next state
			nextStateFn: func(
				context.Context,
				*api.Environment,
			) (*api.EnvironmentState, error) {
				return &api.EnvironmentState{
					GitCommit: &api.GitCommit{
						RepoURL: "fake-url",
						ID:      "fake-commit",
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
				_ *api.Environment,
				newState api.EnvironmentState,
			) (api.EnvironmentState, error) {
				return newState, nil
			},
			assertions: func(_, newStatus api.EnvironmentStatus) {
				require.Empty(t, newStatus.Error)
				require.Len(t, newStatus.States, 1)
			},
		},
		{
			name: "state history exceeds max",
			// Status should reflect the next state
			initialStatus: api.EnvironmentStatus{
				States: []api.EnvironmentState{
					{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, // Ten
				},
			},
			nextStateFn: func(
				context.Context,
				*api.Environment,
			) (*api.EnvironmentState, error) {
				return &api.EnvironmentState{
					GitCommit: &api.GitCommit{
						RepoURL: "fake-url",
						ID:      "fake-commit",
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
				_ *api.Environment,
				newState api.EnvironmentState,
			) (api.EnvironmentState, error) {
				return newState, nil
			},
			assertions: func(_, newStatus api.EnvironmentStatus) {
				require.Empty(t, newStatus.Error)
				require.Len(t, newStatus.States, 10) // Verify cap was respected
				// Verify the states behave as a stack with FILO semantics
				require.Equal(
					t,
					api.EnvironmentState{
						GitCommit: &api.GitCommit{
							RepoURL: "fake-url",
							ID:      "fake-commit",
						},
						Images: []api.Image{
							{
								RepoURL: "fake-url",
								Tag:     "fake-tag",
							},
						},
					},
					newStatus.States[0],
				)
			},
		},
	}
	for _, testCase := range testCases {
		testEnv := &api.Environment{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Status: testCase.initialStatus,
		}
		testReconciler := &environmentReconciler{
			logger:         log.New(),
			getNextStateFn: testCase.nextStateFn,
			promoteFn:      testCase.promoteFn,
		}
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.initialStatus,
				testReconciler.sync(context.Background(), testEnv),
			)
		})
	}
}

func TestGetNextState(t *testing.T) {
	testCases := []struct {
		name            string
		spec            api.EnvironmentSpec
		upstreamReposFn func(
			context.Context,
			*api.Environment,
		) (*api.EnvironmentState, error)
		assertions func(*api.EnvironmentState, error)
	}{
		{
			name: "spec has no subscriptions",
			assertions: func(state *api.EnvironmentState, err error) {
				require.NoError(t, err)
				require.Nil(t, state)
			},
		},
		{
			name: "spec has no upstream repo subscriptions",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{},
			},
			assertions: func(state *api.EnvironmentState, err error) {
				require.NoError(t, err)
				require.Nil(t, state)
			},
		},
		{
			name: "error getting next state from upstream repos",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			upstreamReposFn: func(
				context.Context,
				*api.Environment,
			) (*api.EnvironmentState, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(state *api.EnvironmentState, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error getting next state")
				require.Contains(t, err.Error(), "something went wrong")
				require.Nil(t, state)
			},
		},
		{
			name: "success getting next state from upstream repos",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			upstreamReposFn: func(
				context.Context,
				*api.Environment,
			) (*api.EnvironmentState, error) {
				return &api.EnvironmentState{
					GitCommit: &api.GitCommit{
						RepoURL: "fake-url",
						ID:      "fake-commit",
					},
					Images: []api.Image{
						{
							RepoURL: "fake-url",
							Tag:     "fake-tag",
						},
					},
				}, nil
			},
			assertions: func(state *api.EnvironmentState, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					&api.EnvironmentState{
						GitCommit: &api.GitCommit{
							RepoURL: "fake-url",
							ID:      "fake-commit",
						},
						Images: []api.Image{
							{
								RepoURL: "fake-url",
								Tag:     "fake-tag",
							},
						},
					},
					state,
				)
			},
		},
	}
	for _, testCase := range testCases {
		testEnv := &api.Environment{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: testCase.spec,
		}
		testReconciler := &environmentReconciler{
			logger:                          log.New(),
			getNextStateFromUpstreamReposFn: testCase.upstreamReposFn,
		}
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testReconciler.getNextState(context.Background(), testEnv),
			)
		})
	}
}

func TestGetNextStateFromUpstreamRepos(t *testing.T) {
	testCases := []struct {
		name       string
		spec       api.EnvironmentSpec
		gitFn      func(context.Context, *api.Environment) (*api.GitCommit, error)
		imagesFn   func(context.Context, *api.Environment) ([]api.Image, error)
		assertions func(*api.EnvironmentState, error)
	}{
		{
			name: "spec has no subscriptions",
			assertions: func(state *api.EnvironmentState, err error) {
				require.NoError(t, err)
				require.Nil(t, state)
			},
		},
		{
			name: "spec has no upstream repo subscriptions",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{},
			},
			assertions: func(state *api.EnvironmentState, err error) {
				require.NoError(t, err)
				require.Nil(t, state)
			},
		},
		{
			name: "error getting latest git commit",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			gitFn: func(context.Context, *api.Environment) (*api.GitCommit, error) {
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
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			gitFn: func(context.Context, *api.Environment) (*api.GitCommit, error) {
				return nil, nil
			},
			imagesFn: func(context.Context, *api.Environment) ([]api.Image, error) {
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
			name: "success",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			gitFn: func(context.Context, *api.Environment) (*api.GitCommit, error) {
				return &api.GitCommit{
					RepoURL: "fake-url",
					ID:      "fake-commit",
				}, nil
			},
			imagesFn: func(context.Context, *api.Environment) ([]api.Image, error) {
				return []api.Image{
					{
						RepoURL: "fake-url",
						Tag:     "fake-tag",
					},
				}, nil
			},
			assertions: func(state *api.EnvironmentState, err error) {
				require.NoError(t, err)
				require.NotNil(t, state)
				require.NotEmpty(t, state.ID)
				require.Equal(
					t,
					&api.GitCommit{
						RepoURL: "fake-url",
						ID:      "fake-commit",
					},
					state.GitCommit,
				)
				require.Equal(
					t,
					[]api.Image{
						{
							RepoURL: "fake-url",
							Tag:     "fake-tag",
						},
					},
					state.Images,
				)
				require.Nil(t, state.Health)
			},
		},
	}
	for _, testCase := range testCases {
		testEnv := &api.Environment{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: testCase.spec,
		}
		testReconciler := &environmentReconciler{
			logger:            log.New(),
			getLatestCommitFn: testCase.gitFn,
			getLatestImagesFn: testCase.imagesFn,
		}
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testReconciler.getNextStateFromUpstreamRepos(
					context.Background(),
					testEnv,
				),
			)
		})
	}
}
