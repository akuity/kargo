package git

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetLatestCommitID(t *testing.T) {
	testCases := []struct {
		name       string
		repoURL    string
		branch     string
		assertions func(string, error)
	}{
		{
			name:    "error cloning repo",
			repoURL: "fake-url", // This should force a failure
			assertions: func(_ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error cloning git repo")
			},
		},

		{
			name:    "error checking out branch",
			repoURL: "https://github.com/argoproj/argo-cd.git",
			branch:  "bogus", // This should force a failure
			assertions: func(_ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error checking out branch")
			},
		},

		{
			name:    "success",
			repoURL: "https://github.com/argoproj/argo-cd.git",
			assertions: func(commit string, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, commit)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				GetLatestCommitID(testCase.repoURL, testCase.branch, nil),
			)
		})
	}
}

func TestApplyUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		repoURL    string
		branch     string
		updateFn   func(string, string) (string, error)
		assertions func(string, error)
	}{
		{
			name:    "error cloning repo",
			repoURL: "fake-url", // This should force a failure
			assertions: func(_ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error cloning git repo")
			},
		},

		{
			name:    "error checking out branch",
			repoURL: "https://github.com/argoproj/argo-cd.git",
			branch:  "bogus", // This should force a failure
			assertions: func(_ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error checking out branch")
			},
		},

		{
			name:    "error running update function",
			repoURL: "https://github.com/argoproj/argo-cd.git",
			updateFn: func(string, string) (string, error) {
				return "", errors.New("something went wrong")
			},
			assertions: func(_ string, err error) {
				require.Error(t, err)
				require.Equal(t, err.Error(), "something went wrong")
			},
		},

		{
			name:    "no diffs after update",
			repoURL: "https://github.com/argoproj/argo-cd.git",
			updateFn: func(string, string) (string, error) {
				return "", nil
			},
			assertions: func(commitID string, err error) {
				require.NoError(t, err)
				require.Empty(t, commitID)
			},
		},

		// TODO: Hard to test success case without actually pushing to a remote
		// repository.
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				ApplyUpdate(testCase.repoURL, testCase.branch, nil, testCase.updateFn),
			)
		})
	}
}
