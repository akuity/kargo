package argocd

import (
	"context"
	"testing"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/akuityio/kargo/internal/git"
	"github.com/akuityio/kargo/internal/helm"
)

func TestGitRepoCredentials(t *testing.T) {
	testCases := []struct {
		name       string
		repoURL    string
		argoDB     DB
		assertions func(*git.RepoCredentials, error)
	}{
		{
			name: "error calling argoDB.GetRepository",
			argoDB: &fakeDB{
				getRepositoryFn: func(
					context.Context,
					string,
				) (*argocd.Repository, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_ *git.RepoCredentials, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error getting Repository (Secret) for repo",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "argoDB.GetRepository finds usable creds",
			argoDB: &fakeDB{
				getRepositoryFn: func(
					context.Context,
					string,
				) (*argocd.Repository, error) {
					return &argocd.Repository{
						Username: "fake-user",
						Password: "fake-password",
					}, nil
				},
			},
			assertions: func(creds *git.RepoCredentials, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					&git.RepoCredentials{
						Username: "fake-user",
						Password: "fake-password",
					},
					creds,
				)
			},
		},

		{
			name: "error calling argoDB.GetRepositoryCredentials",
			argoDB: &fakeDB{
				getRepositoryFn: func(
					context.Context,
					string,
				) (*argocd.Repository, error) {
					return nil, nil
				},
				getRepositoryCredentialsFn: func(
					context.Context,
					string,
				) (*argocd.RepoCreds, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_ *git.RepoCredentials, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error getting Repository Credentials (Secret) for repo",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "argoDB.GetRepositoryCredentials finds usable creds",
			argoDB: &fakeDB{
				getRepositoryFn: func(
					context.Context,
					string,
				) (*argocd.Repository, error) {
					return nil, nil
				},
				getRepositoryCredentialsFn: func(
					context.Context,
					string,
				) (*argocd.RepoCreds, error) {
					return &argocd.RepoCreds{
						Username: "fake-user",
						Password: "fake-password",
					}, nil
				},
			},
			assertions: func(creds *git.RepoCredentials, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					&git.RepoCredentials{
						Username: "fake-user",
						Password: "fake-password",
					},
					creds,
				)
			},
		},

		{
			name: "no usable creds found",
			argoDB: &fakeDB{
				getRepositoryFn: func(
					context.Context,
					string,
				) (*argocd.Repository, error) {
					return nil, nil
				},
				getRepositoryCredentialsFn: func(
					context.Context,
					string,
				) (*argocd.RepoCreds, error) {
					return nil, nil
				},
			},
			assertions: func(creds *git.RepoCredentials, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				GetGitRepoCredentials(
					context.Background(),
					testCase.argoDB,
					testCase.repoURL,
				),
			)
		})
	}
}

func TestChartRegistryCredentials(t *testing.T) {
	testCases := []struct {
		name       string
		argoDB     DB
		assertions func(*helm.RegistryCredentials, error)
	}{
		{
			name: "error calling argoDB.GetRepository",
			argoDB: &fakeDB{
				getRepositoryFn: func(
					context.Context,
					string,
				) (*argocd.Repository, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_ *helm.RegistryCredentials, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error getting Argo CD Repository (Secret) for Helm chart registry",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "argoDB.GetRepository finds usable creds",
			argoDB: &fakeDB{
				getRepositoryFn: func(
					context.Context,
					string,
				) (*argocd.Repository, error) {
					return &argocd.Repository{
						Type:     "helm",
						Username: "fake-user",
						Password: "fake-password",
					}, nil
				},
			},
			assertions: func(creds *helm.RegistryCredentials, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					&helm.RegistryCredentials{
						Username: "fake-user",
						Password: "fake-password",
					},
					creds,
				)
			},
		},

		{
			name: "error calling argoDB.GetRepositoryCredentials",
			argoDB: &fakeDB{
				getRepositoryFn: func(
					context.Context,
					string,
				) (*argocd.Repository, error) {
					return nil, nil
				},
				getRepositoryCredentialsFn: func(
					context.Context,
					string,
				) (*argocd.RepoCreds, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(creds *helm.RegistryCredentials, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error getting Argo CD Repository Credentials (Secret) for Helm "+
						"chart registry",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "argoDB.GetRepositoryCredentials finds usable creds",
			argoDB: &fakeDB{
				getRepositoryFn: func(
					context.Context,
					string,
				) (*argocd.Repository, error) {
					return nil, nil
				},
				getRepositoryCredentialsFn: func(
					context.Context,
					string,
				) (*argocd.RepoCreds, error) {
					return &argocd.RepoCreds{
						Type:     "helm",
						Username: "fake-username",
						Password: "fake-password",
					}, nil
				},
			},
			assertions: func(creds *helm.RegistryCredentials, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					&helm.RegistryCredentials{
						Username: "fake-username",
						Password: "fake-password",
					},
					creds,
				)
			},
		},

		{
			name: "no usable creds found",
			argoDB: &fakeDB{
				getRepositoryFn: func(
					context.Context,
					string,
				) (*argocd.Repository, error) {
					return nil, nil
				},
				getRepositoryCredentialsFn: func(
					context.Context,
					string,
				) (*argocd.RepoCreds, error) {
					return nil, nil
				},
			},
			assertions: func(creds *helm.RegistryCredentials, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				GetChartRegistryCredentials(
					context.Background(),
					testCase.argoDB,
					"fake-url",
				),
			)
		})
	}
}
