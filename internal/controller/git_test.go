package controller

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	api "github.com/akuityio/kargo/api/v1alpha1"
	libArgoCD "github.com/akuityio/kargo/internal/argocd"
	"github.com/akuityio/kargo/internal/git"
)

func TestApplyGitRepoUpdate(t *testing.T) {
	testCases := []struct {
		name           string
		gitRepoCredsFn func(
			context.Context,
			libArgoCD.DB,
			string,
		) (*git.RepoCredentials, error)
		gitApplyUpdateFn func(
			string,
			string,
			*git.RepoCredentials,
			func(homeDir, workingDir string) (string, error),
		) (string, error)
		assertions func(inState, outState api.EnvironmentState, err error)
	}{
		{
			name: "error getting repo credentials",
			gitRepoCredsFn: func(
				context.Context,
				libArgoCD.DB,
				string,
			) (*git.RepoCredentials, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(_, _ api.EnvironmentState, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error obtaining credentials for git repo",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "error applying updates",
			gitRepoCredsFn: func(
				context.Context,
				libArgoCD.DB,
				string,
			) (*git.RepoCredentials, error) {
				return nil, nil
			},
			gitApplyUpdateFn: func(
				string,
				string,
				*git.RepoCredentials,
				func(string, string) (string, error),
			) (string, error) {
				return "", errors.New("something went wrong")
			},
			assertions: func(_, _ api.EnvironmentState, err error) {
				require.Error(t, err)
				require.Equal(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "success",
			gitRepoCredsFn: func(
				context.Context,
				libArgoCD.DB,
				string,
			) (*git.RepoCredentials, error) {
				return nil, nil
			},
			gitApplyUpdateFn: func(
				string,
				string,
				*git.RepoCredentials,
				func(string, string) (string, error),
			) (string, error) {
				return "new-fake-commit", nil
			},
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.NoError(t, err)
				require.Len(t, outState.Commits, 1)
				// Check that the commit ID in the state was updated
				require.Equal(t, "new-fake-commit", outState.Commits[0].ID)
				// Everything else should be unchanged
				outState.Commits[0].ID = "fake-commit"
				require.Equal(t, inState, outState)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := environmentReconciler{
				logger:               log.New(),
				gitRepoCredentialsFn: testCase.gitRepoCredsFn,
				gitApplyUpdateFn:     testCase.gitApplyUpdateFn,
			}
			reconciler.logger.SetLevel(log.ErrorLevel)
			newState := api.EnvironmentState{
				Commits: []api.GitCommit{
					{
						RepoURL: "fake-url",
						ID:      "fake-commit",
					},
				},
			}
			outState, err := reconciler.applyGitRepoUpdate(
				context.Background(),
				newState,
				api.GitRepoUpdate{
					RepoURL: "fake-url",
				},
			)
			testCase.assertions(newState, outState, err)
		})
	}
}

func TestGetLatestCommits(t *testing.T) {
	testCases := []struct {
		name           string
		gitRepoCredsFn func(
			context.Context,
			libArgoCD.DB,
			string,
		) (*git.RepoCredentials, error)
		getLatestCommitIDFn func(
			string,
			string,
			*git.RepoCredentials,
		) (string, error)
		assertions func(commits []api.GitCommit, err error)
	}{
		{
			name: "error getting repo credentials",
			gitRepoCredsFn: func(
				context.Context,
				libArgoCD.DB,
				string,
			) (*git.RepoCredentials, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(commits []api.GitCommit, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error obtaining credentials for git repo",
				)
				require.Contains(t, err.Error(), "something went wrong")
				require.Empty(t, commits)
			},
		},

		{
			name: "error getting last commit ID",
			gitRepoCredsFn: func(
				context.Context,
				libArgoCD.DB,
				string,
			) (*git.RepoCredentials, error) {
				return nil, nil
			},
			getLatestCommitIDFn: func(
				string,
				string,
				*git.RepoCredentials,
			) (string, error) {
				return "", errors.New("something went wrong")
			},
			assertions: func(commits []api.GitCommit, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error determining latest commit ID of git repo",
				)
				require.Contains(t, err.Error(), "something went wrong")
				require.Empty(t, commits)
			},
		},

		{
			name: "success",
			gitRepoCredsFn: func(
				context.Context,
				libArgoCD.DB,
				string,
			) (*git.RepoCredentials, error) {
				return nil, nil
			},
			getLatestCommitIDFn: func(
				string,
				string,
				*git.RepoCredentials,
			) (string, error) {
				return "fake-commit", nil
			},
			assertions: func(commits []api.GitCommit, err error) {
				require.NoError(t, err)
				require.Len(t, commits, 1)
				require.Equal(
					t,
					api.GitCommit{
						RepoURL: "fake-url",
						ID:      "fake-commit",
					},
					commits[0],
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := environmentReconciler{
				logger:               log.New(),
				gitRepoCredentialsFn: testCase.gitRepoCredsFn,
				getLatestCommitIDFn:  testCase.getLatestCommitIDFn,
			}
			reconciler.logger.SetLevel(log.ErrorLevel)
			testCase.assertions(
				reconciler.getLatestCommits(
					context.Background(),
					[]api.GitSubscription{
						{
							RepoURL: "fake-url",
						},
					},
				),
			)
		})
	}
}
