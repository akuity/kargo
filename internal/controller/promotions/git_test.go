package promotions

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/git"
)

func TestApplyGitRepoUpdate(t *testing.T) {
	testCases := []struct {
		name             string
		newState         api.EnvironmentState
		credentialsDB    credentials.Database
		gitApplyUpdateFn func(
			string,
			string,
			string,
			*git.Credentials,
			func(homeDir, workingDir string) (string, error),
		) (string, error)
		assertions func(inState, outState api.EnvironmentState, err error)
	}{
		{
			name: "invalid update",
			newState: api.EnvironmentState{
				Commits: []api.GitCommit{
					{
						RepoURL: "fake-url",
						Branch:  "fake-branch",
					},
				},
			},
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"invalid update specified; cannot write to branch",
				)
				require.Contains(
					t,
					err.Error(),
					"because it will form a subscription loop",
				)
			},
		},

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
			gitApplyUpdateFn: func(
				string,
				string,
				string,
				*git.Credentials,
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
			newState: api.EnvironmentState{
				Commits: []api.GitCommit{
					{
						RepoURL: "fake-url",
						// This branch deliberately doesn't match the branch we read from
						Branch: "another-fake-branch",
					},
				},
			},
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
			gitApplyUpdateFn: func(
				string,
				string,
				string,
				*git.Credentials,
				func(string, string) (string, error),
			) (string, error) {
				return "new-fake-commit", nil
			},
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.NoError(t, err)
				require.Len(t, outState.Commits, 1)
				// Check that HealthCheckCommit got set
				require.NotEmpty(t, outState.Commits[0].HealthCheckCommit)
				// Everything else should be unchanged
				outState.Commits[0].HealthCheckCommit = ""
				outState.ID = inState.ID
				require.Equal(t, inState, outState)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := reconciler{
				credentialsDB:    testCase.credentialsDB,
				gitApplyUpdateFn: testCase.gitApplyUpdateFn,
			}
			outState, err := reconciler.applyGitRepoUpdate(
				context.Background(),
				"fake-namespace",
				testCase.newState,
				api.GitRepoUpdate{
					RepoURL:     "fake-url",
					WriteBranch: "fake-branch",
				},
			)
			testCase.assertions(testCase.newState, outState, err)
		})
	}
}
