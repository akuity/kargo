package controller

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/git"
)

func TestPromoteWithKustomize(t *testing.T) {
	testCases := []struct {
		name        string
		env         *api.Environment
		newState    api.EnvironmentState
		repoCredsFn func(context.Context, string) (git.RepoCredentials, error)
		cloneFn     func(
			context.Context,
			string,
			git.RepoCredentials,
		) (git.Repo, error)
		checkoutFn func(repo git.Repo, branch string) error
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
			name: "Kustomize is nil",
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
			name: "Kustomize promotion mechanism has len(Images) == 0",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						Git: &api.GitPromotionMechanism{
							Kustomize: &api.KustomizePromotionMechanism{},
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
			name: "new Environment state has has len(Images) == 0",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						Git: &api.GitPromotionMechanism{
							Kustomize: &api.KustomizePromotionMechanism{
								Images: []api.KustomizeImageUpdate{
									{},
								},
							},
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
			name: "Environment spec is missing Git repo details",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						Git: &api.GitPromotionMechanism{
							Kustomize: &api.KustomizePromotionMechanism{
								Images: []api.KustomizeImageUpdate{
									{},
								},
							},
						},
					},
				},
			},
			newState: api.EnvironmentState{
				Images: []api.Image{
					{},
				},
			},
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.Error(t, err)
				require.Equal(
					t,
					"cannot promote images via Kustomize because spec does not contain "+
						"git repo details",
					err.Error(),
				)
				require.Equal(t, inState, outState)
			},
		},
		{
			name: "error getting Git repo credentials",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					GitRepo: &api.GitRepo{
						URL: "fake-url",
					},
					PromotionMechanisms: &api.PromotionMechanisms{
						Git: &api.GitPromotionMechanism{
							Kustomize: &api.KustomizePromotionMechanism{
								Images: []api.KustomizeImageUpdate{
									{},
								},
							},
						},
					},
				},
			},
			newState: api.EnvironmentState{
				Images: []api.Image{
					{},
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
			name: "error cloning Git repo",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					GitRepo: &api.GitRepo{
						URL: "fake-url",
					},
					PromotionMechanisms: &api.PromotionMechanisms{
						Git: &api.GitPromotionMechanism{
							Kustomize: &api.KustomizePromotionMechanism{
								Images: []api.KustomizeImageUpdate{
									{},
								},
							},
						},
					},
				},
			},
			newState: api.EnvironmentState{
				Images: []api.Image{
					{},
				},
			},
			repoCredsFn: func(context.Context, string) (git.RepoCredentials, error) {
				return git.RepoCredentials{}, nil
			},
			cloneFn: func(
				context.Context,
				string,
				git.RepoCredentials,
			) (git.Repo, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error cloning git repo")
				require.Contains(t, err.Error(), "something went wrong")
				require.Equal(t, inState, outState)
			},
		},
		{
			name: "error checking out branch",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					GitRepo: &api.GitRepo{
						URL:    "fake-url",
						Branch: "fake-branch",
					},
					PromotionMechanisms: &api.PromotionMechanisms{
						Git: &api.GitPromotionMechanism{
							Kustomize: &api.KustomizePromotionMechanism{
								Images: []api.KustomizeImageUpdate{
									{},
								},
							},
						},
					},
				},
			},
			newState: api.EnvironmentState{
				Images: []api.Image{
					{},
				},
			},
			repoCredsFn: func(context.Context, string) (git.RepoCredentials, error) {
				return git.RepoCredentials{}, nil
			},
			cloneFn: func(
				context.Context,
				string,
				git.RepoCredentials,
			) (git.Repo, error) {
				return nil, nil
			},
			checkoutFn: func(git.Repo, string) error {
				return errors.New("something went wrong")
			},
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error checking out branch")
				require.Contains(t, err.Error(), "something went wrong")
				require.Equal(t, inState, outState)
			},
		},
		// TODO: Add more test cases. Testing beyond here is difficult because we
		// have no convenient way of mocking a Git Repo object.
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := environmentReconciler{
				logger:                  log.New(),
				getGitRepoCredentialsFn: testCase.repoCredsFn,
				gitCloneFn:              testCase.cloneFn,
				checkoutBranchFn:        testCase.checkoutFn,
			}
			reconciler.logger.SetLevel(log.ErrorLevel)
			newState, err := reconciler.promoteWithKustomize(
				context.Background(),
				testCase.env,
				testCase.newState,
			)
			testCase.assertions(testCase.newState, newState, err)
		})
	}
}
