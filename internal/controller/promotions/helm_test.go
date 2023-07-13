package promotions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	api "github.com/akuity/kargo/api/v1alpha1"
)

func TestApplyHelm(t *testing.T) {
	testCases := []struct {
		name                   string
		newState               api.StageState
		update                 api.HelmPromotionMechanism
		setStringsInYAMLFileFn func(
			string,
			map[string]string,
		) error
		buildChartDependencyChangesFn func(
			string,
			[]api.Chart,
			[]api.HelmChartDependencyUpdate,
		) (map[string]map[string]string, []string, error)
		updateChartDependenciesFn func(homePath, chartPath string) error
		assertions                func(changeSummary []string, err error)
	}{
		{
			name: "error modifying values.yaml",
			newState: api.StageState{
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
			assertions: func(_ []string, err error) {
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
			) (map[string]map[string]string, []string, error) {
				return nil, nil, errors.New("something went wrong")
			},
			assertions: func(_ []string, err error) {
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
			) (map[string]map[string]string, []string, error) {
				// We only need to build enough of a change map to make sure we get into
				// the loop
				return map[string]map[string]string{
					"/fake/path/Chart.yaml": {},
				}, nil, nil
			},
			setStringsInYAMLFileFn: func(string, map[string]string) error {
				return errors.New("something went wrong")
			},
			assertions: func(_ []string, err error) {
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
			) (map[string]map[string]string, []string, error) {
				return map[string]map[string]string{
					// We only need to build enough of a change map to make sure we get
					// into the loop
					"/fake/path/Chart.yaml": {},
				}, nil, nil
			},
			setStringsInYAMLFileFn: func(string, map[string]string) error {
				return nil
			},
			updateChartDependenciesFn: func(string, string) error {
				return errors.New("something went wrong")
			},
			assertions: func(_ []string, err error) {
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
			) (map[string]map[string]string, []string, error) {
				return nil, nil, nil
			},
			assertions: func(_ []string, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r := reconciler{
				setStringsInYAMLFileFn:        testCase.setStringsInYAMLFileFn,
				buildChartDependencyChangesFn: testCase.buildChartDependencyChangesFn,
				updateChartDependenciesFn:     testCase.updateChartDependenciesFn,
			}
			testCase.assertions(
				r.applyHelm(
					testCase.newState,
					testCase.update,
					"",
					"",
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
	result, changeSummary := buildValuesFilesChanges(images, imageUpdates)
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
	require.Equal(
		t,
		[]string{
			"updated fake-values.yaml to use image fake-url:fake-tag",
			"updated fake-values.yaml to use image another-fake-url:another-fake-tag",
			"updated another-fake-values.yaml to use image fake-url:fake-tag",
		},
		changeSummary,
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

	result, changeSummary, err :=
		buildChartDependencyChanges(testDir, charts, chartUpdates)
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
	require.Contains(
		t,
		changeSummary,
		"updated charts/foo/Chart.yaml to use subchart fake-chart:fake-version",
	)
	require.Contains(
		t,
		changeSummary,
		"updated charts/bar/Chart.yaml to use subchart "+
			"another-fake-chart:another-fake-version",
	)
}
