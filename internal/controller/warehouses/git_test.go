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
		name       string
		reconciler *reconciler
		assertions func(commits []kargoapi.GitCommit, err error)
	}{
		{
			name: "error getting repo credentials",
			reconciler: &reconciler{
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
			reconciler: &reconciler{
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
					kargoapi.GitSubscription,
					*git.RepoCredentials,
				) (*gitMeta, error) {
					return nil, errors.New("something went wrong")
				},
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
			reconciler: &reconciler{
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
					kargoapi.GitSubscription,
					*git.RepoCredentials,
				) (*gitMeta, error) {
					return &gitMeta{Commit: "fake-commit", Message: "message"}, nil
				},
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
			testCase.assertions(
				testCase.reconciler.getLatestCommits(
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

func TestGetLatestCommitMeta(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.GitSubscription
		assertions func(*gitMeta, error)
	}{
		{
			name: "error cloning repo",
			sub: kargoapi.GitSubscription{
				RepoURL: "fake-url", // This should force a failure
			},
			assertions: func(_ *gitMeta, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error cloning git repo")
			},
		},
		{
			name: "success",
			sub: kargoapi.GitSubscription{
				RepoURL: "https://github.com/akuity/kargo.git",
			},
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
				getLatestCommitMeta(context.TODO(), testCase.sub, nil),
			)
		})
	}
}
