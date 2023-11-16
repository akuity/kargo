package warehouses

import (
	"context"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
)

func TestGetLatestCommits(t *testing.T) {
	testCases := []struct {
		name                  string
		credentialsDB         credentials.Database
		getLatestCommitMetaFn func(
			context.Context,
			string,
			string,
			*git.RepoCredentials,
		) (*gitMeta, error)
		assertions func(commits []kargoapi.GitCommit, err error)
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
			assertions: func(commits []kargoapi.GitCommit, err error) {
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
			getLatestCommitMetaFn: func(
				context.Context,
				string,
				string,
				*git.RepoCredentials,
			) (*gitMeta, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(commits []kargoapi.GitCommit, err error) {
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
			getLatestCommitMetaFn: func(
				context.Context,
				string,
				string,
				*git.RepoCredentials,
			) (*gitMeta, error) {
				return &gitMeta{Commit: "fake-commit", Message: "message"}, nil
			},
			assertions: func(commits []kargoapi.GitCommit, err error) {
				require.NoError(t, err)
				require.Len(t, commits, 1)
				require.Equal(
					t,
					kargoapi.GitCommit{
						RepoURL: "fake-url",
						ID:      "fake-commit",
						Message: "message",
					},
					commits[0],
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r := reconciler{
				credentialsDB:         testCase.credentialsDB,
				getLatestCommitMetaFn: testCase.getLatestCommitMetaFn,
			}
			testCase.assertions(
				r.getLatestCommits(
					context.Background(),
					"fake-namespace",
					[]kargoapi.RepoSubscription{
						{
							Git: &kargoapi.GitSubscription{
								RepoURL: "fake-url",
							},
						},
					},
				),
			)
		})
	}
}

func TestGetLatestCommitID(t *testing.T) {
	testCases := []struct {
		name       string
		repoURL    string
		branch     string
		assertions func(*gitMeta, error)
	}{
		{
			name:    "error cloning repo",
			repoURL: "fake-url", // This should force a failure
			assertions: func(_ *gitMeta, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error cloning git repo")
			},
		},
		{
			name:    "success",
			repoURL: "https://github.com/akuity/kargo.git",
			assertions: func(gm *gitMeta, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, gm.Commit)
				require.NotEmpty(t, gm.Message)
				require.Len(t, strings.Split(gm.Message, "\n"), 1)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				getLatestCommitMeta(context.TODO(), testCase.repoURL, testCase.branch, nil),
			)
		})
	}
}
