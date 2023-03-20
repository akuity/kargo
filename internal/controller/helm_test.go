package controller

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/helm"
)

func TestApplyHelm(t *testing.T) {
	testCases := []struct {
		name                   string
		newState               api.EnvironmentState
		update                 api.HelmPromotionMechanism
		setStringsInYAMLFileFn func(
			string,
			map[string]string,
		) error
		buildChartDependencyChangesFn func(
			string,
			[]api.Chart,
			[]api.HelmChartDependencyUpdate,
		) (map[string]map[string]string, error)
		updateChartDependenciesFn func(homePath, chartPath string) error
		assertions                func(err error)
	}{
		{
			name: "error modifying values.yaml",
			newState: api.EnvironmentState{
				Images: []api.Image{
					{
						RepoURL: "fake-url",
						Tag:     "fake-tag",
					},
				},
			},
			update: api.HelmPromotionMechanism{
				Images: []api.HelmImageUpdate{
					{
						Image: "fake-url",
						Key:   "image",
						Value: "Image",
					},
				},
			},
			setStringsInYAMLFileFn: func(string, map[string]string) error {
				return errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error updating values in file")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "error building chart dependency changes",
			buildChartDependencyChangesFn: func(
				string,
				[]api.Chart,
				[]api.HelmChartDependencyUpdate,
			) (map[string]map[string]string, error) {
				return nil, errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error preparing changes to affected Chart.yaml files",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "error updating Chart.yaml",
			buildChartDependencyChangesFn: func(
				string,
				[]api.Chart,
				[]api.HelmChartDependencyUpdate,
			) (map[string]map[string]string, error) {
				// We only need to build enough of a change map to make sure we get into
				// the loop
				return map[string]map[string]string{
					"/fake/path/Chart.yaml": {},
				}, nil
			},
			setStringsInYAMLFileFn: func(string, map[string]string) error {
				return errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error updating dependencies for chart",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "error updating chart dependencies",
			buildChartDependencyChangesFn: func(
				string,
				[]api.Chart,
				[]api.HelmChartDependencyUpdate,
			) (map[string]map[string]string, error) {
				return map[string]map[string]string{
					// We only need to build enough of a change map to make sure we get
					// into the loop
					"/fake/path/Chart.yaml": {},
				}, nil
			},
			setStringsInYAMLFileFn: func(string, map[string]string) error {
				return nil
			},
			updateChartDependenciesFn: func(string, string) error {
				return errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error updating dependencies for chart",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "success",
			buildChartDependencyChangesFn: func(
				string,
				[]api.Chart,
				[]api.HelmChartDependencyUpdate,
			) (map[string]map[string]string, error) {
				return nil, nil
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := environmentReconciler{
				setStringsInYAMLFileFn:        testCase.setStringsInYAMLFileFn,
				buildChartDependencyChangesFn: testCase.buildChartDependencyChangesFn,
				updateChartDependenciesFn:     testCase.updateChartDependenciesFn,
			}
			testCase.assertions(
				reconciler.applyHelm(
					testCase.newState,
					testCase.update,
					"",
					"",
				),
			)
		})
	}
}

func TestGetLatestCharts(t *testing.T) {
	testCases := []struct {
		name                    string
		credentialsDB           credentialsDB
		getLatestChartVersionFn func(
			context.Context,
			string,
			string,
			string,
			*helm.Credentials,
		) (string, error)
		assertions func([]api.Chart, error)
	}{
		{
			name: "error getting registry credentials",
			credentialsDB: &fakeCredentialsDB{
				getFn: func(
					context.Context,
					string,
					credentialsType,
					string,
				) (credentials, bool, error) {
					return credentials{}, false, errors.New("something went wrong")
				},
			},
			assertions: func(_ []api.Chart, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error obtaining credentials for chart registry",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "error getting latest chart version",
			credentialsDB: &fakeCredentialsDB{
				getFn: func(
					context.Context,
					string,
					credentialsType,
					string,
				) (credentials, bool, error) {
					return credentials{}, false, nil
				},
			},
			getLatestChartVersionFn: func(
				context.Context,
				string,
				string,
				string,
				*helm.Credentials,
			) (string, error) {
				return "", errors.New("something went wrong")
			},
			assertions: func(_ []api.Chart, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error searching for latest version of chart",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "no chart found",
			credentialsDB: &fakeCredentialsDB{
				getFn: func(
					context.Context,
					string,
					credentialsType,
					string,
				) (credentials, bool, error) {
					return credentials{}, false, nil
				},
			},
			getLatestChartVersionFn: func(
				context.Context,
				string,
				string,
				string,
				*helm.Credentials,
			) (string, error) {
				return "", nil
			},
			assertions: func(_ []api.Chart, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "found no suitable version of chart")
			},
		},

		{
			name: "success",
			credentialsDB: &fakeCredentialsDB{
				getFn: func(
					context.Context,
					string,
					credentialsType,
					string,
				) (credentials, bool, error) {
					return credentials{}, false, nil
				},
			},
			getLatestChartVersionFn: func(
				context.Context,
				string,
				string,
				string,
				*helm.Credentials,
			) (string, error) {
				return "1.0.0", nil
			},
			assertions: func(charts []api.Chart, err error) {
				require.NoError(t, err)
				require.Len(t, charts, 1)
				require.Equal(
					t,
					api.Chart{
						RegistryURL: "fake-url",
						Name:        "fake-chart",
						Version:     "1.0.0",
					},
					charts[0],
				)
			},
		},
	}
	testSubs := []api.ChartSubscription{
		{
			RegistryURL: "fake-url",
			Name:        "fake-chart",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := environmentReconciler{
				credentialsDB:           testCase.credentialsDB,
				getLatestChartVersionFn: testCase.getLatestChartVersionFn,
			}
			testCase.assertions(
				reconciler.getLatestCharts(
					context.Background(),
					"fake-namespace",
					testSubs,
				),
			)
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
