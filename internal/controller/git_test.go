package controller

import (
	"context"
	"testing"

	api "github.com/akuityio/k8sta/api/v1alpha1"
	"github.com/akuityio/k8sta/internal/git"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestGetLatestCommit(t *testing.T) {
	testCases := []struct {
		name           string
		spec           api.EnvironmentSpec
		repoCredsFn    func(context.Context, string) (git.RepoCredentials, error)
		checkoutFn     func(git.Repo, string) error
		lastCommitIDFn func(git.Repo) (string, error)
		cloneFn        func(
			context.Context,
			string,
			git.RepoCredentials,
		) (git.Repo, error)
		assertions func(commit *api.GitCommit, err error)
	}{
		{
			name: "spec has no subscriptions",
			assertions: func(commit *api.GitCommit, err error) {
				require.NoError(t, err)
				require.Nil(t, commit)
			},
		},
		{
			name: "spec has no upstream repo subscriptions",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{},
			},
			assertions: func(commit *api.GitCommit, err error) {
				require.NoError(t, err)
				require.Nil(t, commit)
			},
		},
		{
			name: "spec has no git repo subscription",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{},
				},
			},
			assertions: func(commit *api.GitCommit, err error) {
				require.NoError(t, err)
				require.Nil(t, commit)
			},
		},
		{
			name: "spec has a git repo subscription, but no repo details",
			spec: api.EnvironmentSpec{
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{
						Git: true,
					},
				},
			},
			assertions: func(commit *api.GitCommit, err error) {
				require.Error(t, err)
				require.Equal(
					t,
					"environment subscribes to a git repo, but does not specify its "+
						"details",
					err.Error(),
				)
				require.Nil(t, commit)
			},
		},
		{
			name: "error getting repo credentials",
			spec: api.EnvironmentSpec{
				GitRepo: &api.GitRepo{
					URL: "fake-url",
				},
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{
						Git: true,
					},
				},
			},
			repoCredsFn: func(context.Context, string) (git.RepoCredentials, error) {
				return git.RepoCredentials{}, errors.New("something went wrong")
			},
			assertions: func(commit *api.GitCommit, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error obtaining credentials for git repo",
				)
				require.Contains(t, err.Error(), "something went wrong")
				require.Nil(t, commit)
			},
		},
		{
			name: "error cloning repo",
			spec: api.EnvironmentSpec{
				GitRepo: &api.GitRepo{
					URL: "fake-url",
				},
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{
						Git: true,
					},
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
			assertions: func(commit *api.GitCommit, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error cloning git repo")
				require.Contains(t, err.Error(), "something went wrong")
				require.Nil(t, commit)
			},
		},
		{
			name: "error checking out branch",
			spec: api.EnvironmentSpec{
				GitRepo: &api.GitRepo{
					URL:    "fake-url",
					Branch: "fake-branch",
				},
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{
						Git: true,
					},
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
			assertions: func(commit *api.GitCommit, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error checking out branch")
				require.Contains(t, err.Error(), "something went wrong")
				require.Nil(t, commit)
			},
		},
		{
			name: "error getting last commit ID from specific branch",
			spec: api.EnvironmentSpec{
				GitRepo: &api.GitRepo{
					URL:    "fake-url",
					Branch: "fake-branch",
				},
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{
						Git: true,
					},
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
				return nil
			},
			lastCommitIDFn: func(r git.Repo) (string, error) {
				return "", errors.New("something went wrong")
			},
			assertions: func(commit *api.GitCommit, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error determining last commit ID from branch",
				)
				require.Contains(t, err.Error(), "something went wrong")
				require.Nil(t, commit)
			},
		},
		{
			name: "error getting last commit ID from default branch",
			spec: api.EnvironmentSpec{
				GitRepo: &api.GitRepo{
					URL: "fake-url",
				},
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{
						Git: true,
					},
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
				return nil
			},
			lastCommitIDFn: func(r git.Repo) (string, error) {
				return "", errors.New("something went wrong")
			},
			assertions: func(commit *api.GitCommit, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error determining last commit ID from default branch",
				)
				require.Contains(t, err.Error(), "something went wrong")
				require.Nil(t, commit)
			},
		},
		{
			name: "success",
			spec: api.EnvironmentSpec{
				GitRepo: &api.GitRepo{
					URL: "fake-url",
				},
				Subscriptions: &api.Subscriptions{
					Repos: &api.RepoSubscriptions{
						Git: true,
					},
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
				return nil
			},
			lastCommitIDFn: func(r git.Repo) (string, error) {
				return "fake-commit", nil
			},
			assertions: func(commit *api.GitCommit, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					&api.GitCommit{
						RepoURL: "fake-url",
						ID:      "fake-commit",
					},
					commit,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := environmentReconciler{
				logger:                  log.New(),
				getGitRepoCredentialsFn: testCase.repoCredsFn,
				gitCloneFn:              testCase.cloneFn,
				checkoutBranchFn:        testCase.checkoutFn,
				getLastCommitIDFn:       testCase.lastCommitIDFn,
			}
			reconciler.logger.SetLevel(log.ErrorLevel)
			env := &api.Environment{
				Spec: testCase.spec,
			}
			testCase.assertions(reconciler.getLatestCommit(context.Background(), env))
		})
	}
}
