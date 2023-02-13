package controller

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/git"
)

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
			name: "Helm promotion mechanism has len(Images) == 0",
			env: &api.Environment{
				Spec: api.EnvironmentSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						Git: &api.GitPromotionMechanism{
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
						Git: &api.GitPromotionMechanism{
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
						Git: &api.GitPromotionMechanism{
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
						Git: &api.GitPromotionMechanism{
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
						Git: &api.GitPromotionMechanism{
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
						Git: &api.GitPromotionMechanism{
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

func TestBuildValuesFilesChanges(t *testing.T) {
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
	result := buildValuesFilesChanges(images, imageUpdates)
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

func TestBuildChartDependencyChanges(t *testing.T) {
	// Set up a couple of fake Chart.yaml files
	testDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	testChartsDir := filepath.Join(testDir, "charts")
	err = os.Mkdir(testChartsDir, 0755)
	require.NoError(t, err)

	testFooChartDir := filepath.Join(testChartsDir, "foo")
	err = os.Mkdir(testFooChartDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(
		filepath.Join(testFooChartDir, "Chart.yaml"),
		[]byte(`dependencies:
- repository: fake-registry
  name: fake-chart
  version: placeholder
`),
		0600,
	)
	require.NoError(t, err)

	testBarChartDir := filepath.Join(testChartsDir, "bar")
	err = os.Mkdir(testBarChartDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(
		filepath.Join(testBarChartDir, "Chart.yaml"),
		// This fake chart has TWO dependencies -- one of which shouldn't be updated
		[]byte(`dependencies:
- repository: another-fake-registry
  name: another-fake-chart
  version: placeholder
- repository: yet-another-fake-registry
  name: yet-another-fake-chart
  version: placeholder
`),
		0600,
	)
	require.NoError(t, err)

	// New charts
	charts := []api.Chart{
		{
			RegistryURL: "fake-registry",
			Name:        "fake-chart",
			Version:     "fake-version",
		},
		{
			RegistryURL: "another-fake-registry",
			Name:        "another-fake-chart",
			Version:     "another-fake-version",
		},
	}

	// Instructions for how to update Chart.yaml files
	chartUpdates := []api.HelmChartDependencyUpdate{
		{
			RegistryURL: "fake-registry",
			Name:        "fake-chart",
			ChartPath:   "charts/foo",
		},
		{
			RegistryURL: "another-fake-registry",
			Name:        "another-fake-chart",
			ChartPath:   "charts/bar",
		},
		// Note there is no mention of how to update bar's second dependency, so
		// we expect it to be left alone.
	}

	result, err := buildChartDependencyChanges(testDir, charts, chartUpdates)
	require.NoError(t, err)
	require.Equal(
		t,
		map[string]map[string]string{
			"charts/foo": {
				"dependencies.0.version": "fake-version",
			},
			"charts/bar": {
				"dependencies.0.version": "another-fake-version",
			},
		},
		result,
	)
}
