package controller

import (
	"context"
	"testing"

	"github.com/akuityio/bookkeeper"
	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/git"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
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
						ConfigManagement: &api.ConfigManagementPromotionMechanism{},
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
						ConfigManagement: &api.ConfigManagementPromotionMechanism{
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
						ConfigManagement: &api.ConfigManagementPromotionMechanism{
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
						ConfigManagement: &api.ConfigManagementPromotionMechanism{
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
						ConfigManagement: &api.ConfigManagementPromotionMechanism{
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
						ConfigManagement: &api.ConfigManagementPromotionMechanism{},
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
						ConfigManagement: &api.ConfigManagementPromotionMechanism{
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
						ConfigManagement: &api.ConfigManagementPromotionMechanism{
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
						ConfigManagement: &api.ConfigManagementPromotionMechanism{
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
						ConfigManagement: &api.ConfigManagementPromotionMechanism{
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
						ConfigManagement: &api.ConfigManagementPromotionMechanism{
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
						ConfigManagement: &api.ConfigManagementPromotionMechanism{
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

func TestPromoteWithHelm(t *testing.T) {
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
			name: "Helm is nil",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						ConfigManagement: &api.ConfigManagementPromotionMechanism{},
					},
				},
			},
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.NoError(t, err)
				require.Equal(t, inState, outState)
			},
		},
		{
			name: "Helm promotion mechanism has len(Images) == 0",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						ConfigManagement: &api.ConfigManagementPromotionMechanism{
							Helm: &api.HelmPromotionMechanism{},
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
						ConfigManagement: &api.ConfigManagementPromotionMechanism{
							Helm: &api.HelmPromotionMechanism{
								Images: []api.HelmImageUpdate{
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
						ConfigManagement: &api.ConfigManagementPromotionMechanism{
							Helm: &api.HelmPromotionMechanism{
								Images: []api.HelmImageUpdate{
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
					"cannot promote images via Helm because spec does not contain "+
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
						ConfigManagement: &api.ConfigManagementPromotionMechanism{
							Helm: &api.HelmPromotionMechanism{
								Images: []api.HelmImageUpdate{
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
						ConfigManagement: &api.ConfigManagementPromotionMechanism{
							Helm: &api.HelmPromotionMechanism{
								Images: []api.HelmImageUpdate{
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
						ConfigManagement: &api.ConfigManagementPromotionMechanism{
							Helm: &api.HelmPromotionMechanism{
								Images: []api.HelmImageUpdate{
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
			newState, err := reconciler.promoteWithHelm(
				context.Background(),
				testCase.env,
				testCase.newState,
			)
			testCase.assertions(testCase.newState, newState, err)
		})
	}
}

func TestBuildChangeMapsByFile(t *testing.T) {
	images := []api.Image{
		{
			RepoURL: "fake-url",
			Tag:     "fake-tag",
		},
		{
			RepoURL: "another-fake-url",
			Tag:     "another-fake-tag",
		},
	}
	imageUpdates := []api.HelmImageUpdate{
		{
			ValuesFilePath: "fake-values.yaml",
			Image:          "fake-url",
			Key:            "fake-key",
			Value:          "Image",
		},
		{
			ValuesFilePath: "fake-values.yaml",
			Image:          "another-fake-url",
			Key:            "another-fake-key",
			Value:          "Image",
		},
		{
			ValuesFilePath: "another-fake-values.yaml",
			Image:          "fake-url",
			Key:            "fake-key",
			Value:          "Tag",
		},
		{
			ValuesFilePath: "yet-another-fake-values.yaml",
			Image:          "image-that-is-not-in-list",
			Key:            "fake-key",
			Value:          "Tag",
		},
	}
	result := buildChangeMapsByFile(images, imageUpdates)
	require.Equal(
		t,
		map[string]map[string]string{
			"fake-values.yaml": {
				"fake-key":         "fake-url:fake-tag",
				"another-fake-key": "another-fake-url:another-fake-tag",
			},
			"another-fake-values.yaml": {
				"fake-key": "fake-tag",
			},
		},
		result,
	)
}

func TestPromoteWithArgoCD(t *testing.T) {
	testCases := []struct {
		name      string
		env       *api.Environment
		newState  api.EnvironmentState
		syncAppFn func(
			ctx context.Context,
			namespace string,
			name string,
		) error
		assertions func(err error)
	}{
		{
			name: "environment is nil",
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "PromotionMechanisms is nil",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{},
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "ArgoCD is nil",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{},
				},
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "ArgoCD promotion mechanism has len(AppUpdates) == 0",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						ArgoCD: &api.ArgoCDPromotionMechanism{},
					},
				},
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "error making App refresh and sync",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						ArgoCD: &api.ArgoCDPromotionMechanism{
							AppUpdates: []api.ArgoCDAppUpdate{
								{
									Name:           "fake-app",
									RefreshAndSync: true,
								},
							},
						},
					},
				},
			},
			syncAppFn: func(ctx context.Context, namespace, name string) error {
				return errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error syncing Argo CD Application ")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "success",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						ArgoCD: &api.ArgoCDPromotionMechanism{
							AppUpdates: []api.ArgoCDAppUpdate{
								{
									Name:           "fake-app",
									RefreshAndSync: true,
								},
							},
						},
					},
				},
			},
			syncAppFn: func(ctx context.Context, namespace, name string) error {
				return nil
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := environmentReconciler{
				logger:                    log.New(),
				refreshAndSyncArgoCDAppFn: testCase.syncAppFn,
			}
			reconciler.logger.SetLevel(log.ErrorLevel)
			testCase.assertions(
				reconciler.promoteWithArgoCD(
					context.Background(),
					testCase.env,
					testCase.newState,
				),
			)
		})
	}
}
