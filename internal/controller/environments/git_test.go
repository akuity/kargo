package environments

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/git"
)

func TestGetLatestCommits(t *testing.T) {
	testCases := []struct {
		name                string
		credentialsDB       credentials.Database
		getLatestCommitIDFn func(
			string,
			string,
			*git.Credentials,
		) (string, error)
		assertions func(commits []api.GitCommit, err error)
	}{
		{
			name: "error getting repo credentials",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false,
						errors.New("something went wrong")
				},
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
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false, nil
				},
			},
			getLatestCommitIDFn: func(
				string,
				string,
				*git.Credentials,
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
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false, nil
				},
			},
			getLatestCommitIDFn: func(
				string,
				string,
				*git.Credentials,
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
			reconciler := reconciler{
				credentialsDB:       testCase.credentialsDB,
				getLatestCommitIDFn: testCase.getLatestCommitIDFn,
			}
			testCase.assertions(
				reconciler.getLatestCommits(
					context.Background(),
					"fake-namespace",
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
