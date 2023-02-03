package controller

import (
	"context"
	"testing"

	"github.com/akuityio/bookkeeper"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/git"
)

func TestPromoteWithBookkeeper(t *testing.T) {
	testCases := []struct {
		name        string
		env         *api.Environment
		newState    api.EnvironmentState
		repoCredsFn func(
			context.Context,
			string,
		) (git.RepoCredentials, error)
		bookkeeperRenderFn func(
			context.Context,
			bookkeeper.RenderRequest,
		) (bookkeeper.RenderResponse, error)
		assertions func(inState, outState api.EnvironmentState, err error)
	}{
		{
			name: "environment is nil",
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.NoError(t, err)
				require.Equal(t, inState, outState)
			},
		},
		{
			name: "PromotionMechanisms is nil",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{},
			},
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.NoError(t, err)
				require.Equal(t, inState, outState)
			},
		},
		{
			name: "ConfigManagement is nil",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{},
				},
			},
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.NoError(t, err)
				require.Equal(t, inState, outState)
			},
		},
		{
			name: "Bookkeeper is nil",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						Git: &api.GitPromotionMechanism{},
					},
				},
			},
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.NoError(t, err)
				require.Equal(t, inState, outState)
			},
		},
		{
			name: "target branch is unspecified",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						Git: &api.GitPromotionMechanism{
							Bookkeeper: &api.BookkeeperPromotionMechanism{},
						},
					},
				},
			},
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.NoError(t, err)
				require.Equal(t, inState, outState)
			},
		},
		{
			name: "error getting Git repo credentials",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						Git: &api.GitPromotionMechanism{
							Bookkeeper: &api.BookkeeperPromotionMechanism{
								TargetBranch: "env/fake-branch",
							},
						},
					},
				},
			},
			newState: api.EnvironmentState{
				GitCommit: &api.GitCommit{
					RepoURL: "fake-url",
				},
				Images: []api.Image{
					{
						RepoURL: "fake-url",
						Tag:     "fake-tag",
					},
				},
			},
			repoCredsFn: func(context.Context, string) (git.RepoCredentials, error) {
				return git.RepoCredentials{}, errors.New("something went wrong")
			},
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error obtaining credentials for git repo",
				)
				require.Contains(t, err.Error(), "something went wrong")
				require.Equal(t, inState, outState)
			},
		},
		{
			name: "error rendering manifests",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						Git: &api.GitPromotionMechanism{
							Bookkeeper: &api.BookkeeperPromotionMechanism{
								TargetBranch: "env/fake-branch",
							},
						},
					},
				},
			},
			newState: api.EnvironmentState{
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
			repoCredsFn: func(context.Context, string) (git.RepoCredentials, error) {
				return git.RepoCredentials{}, nil
			},
			bookkeeperRenderFn: func(
				context.Context,
				bookkeeper.RenderRequest) (bookkeeper.RenderResponse, error) {
				return bookkeeper.RenderResponse{}, errors.New("something went wrong")
			},
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error rendering manifests via Bookkeeper",
				)
				require.Contains(t, err.Error(), "something went wrong")
				require.Equal(t, inState, outState)
			},
		},
		{
			name: "success",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						Git: &api.GitPromotionMechanism{
							Bookkeeper: &api.BookkeeperPromotionMechanism{
								TargetBranch: "env/fake-branch",
							},
						},
					},
				},
			},
			newState: api.EnvironmentState{
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
			repoCredsFn: func(context.Context, string) (git.RepoCredentials, error) {
				return git.RepoCredentials{}, nil
			},
			bookkeeperRenderFn: func(
				context.Context,
				bookkeeper.RenderRequest) (bookkeeper.RenderResponse, error) {
				return bookkeeper.RenderResponse{
					ActionTaken: bookkeeper.ActionTakenPushedDirectly,
					CommitID:    "new-fake-commit",
				}, nil
			},
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.NoError(t, err)
				require.Equal(t, "new-fake-commit", outState.HealthCheckCommit)
				outState.HealthCheckCommit = ""
				require.Equal(t, inState, outState)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := environmentReconciler{
				logger:                          log.New(),
				getGitRepoCredentialsFn:         testCase.repoCredsFn,
				renderManifestsWithBookkeeperFn: testCase.bookkeeperRenderFn,
			}
			reconciler.logger.SetLevel(log.ErrorLevel)
			newState, err := reconciler.promoteWithBookkeeper(
				context.Background(),
				testCase.env,
				testCase.newState,
			)
			testCase.assertions(testCase.newState, newState, err)
		})
	}
}
