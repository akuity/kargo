package promotion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

func TestNewHelmMechanism(t *testing.T) {
	pm := newHelmMechanism(&credentials.FakeDB{})
	hpm, ok := pm.(*gitMechanism)
	require.True(t, ok)
	require.NotNil(t, hpm.selectUpdatesFn)
	require.NotNil(t, hpm.applyConfigManagementFn)
}

func TestSelectHelmUpdates(t *testing.T) {
	testCases := []struct {
		name       string
		updates    []kargoapi.GitRepoUpdate
		assertions func(selectedUpdates []kargoapi.GitRepoUpdate)
	}{
		{
			name: "no updates",
			assertions: func(selectedUpdates []kargoapi.GitRepoUpdate) {
				require.Empty(t, selectedUpdates)
			},
		},
		{
			name: "no helm updates",
			updates: []kargoapi.GitRepoUpdate{
				{
					RepoURL: "fake-url",
				},
			},
			assertions: func(selectedUpdates []kargoapi.GitRepoUpdate) {
				require.Empty(t, selectedUpdates)
			},
		},
		{
			name: "some helm updates",
			updates: []kargoapi.GitRepoUpdate{
				{
					RepoURL:   "fake-url",
					Kustomize: &kargoapi.KustomizePromotionMechanism{},
				},
				{
					RepoURL: "fake-url",
					Helm:    &kargoapi.HelmPromotionMechanism{},
				},
				{
					RepoURL: "fake-url",
				},
			},
			assertions: func(selectedUpdates []kargoapi.GitRepoUpdate) {
				require.Len(t, selectedUpdates, 1)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(selectHelmUpdates(testCase.updates))
		})
	}
}

func TestHelmerApply(t *testing.T) {
	const testChartDir = "fake-chart-dir"
	testValuesFile := filepath.Join(testChartDir, "values.yaml")
	testChartFile := filepath.Join(testChartDir, "Chart.yaml")
	const testKey = "fake-key"
	const testValue = "fake-value"
	testCases := []struct {
		name       string
		helmer     *helmer
		assertions func(changes []string, err error)
	}{
		{
			name: "error updating values file",
			helmer: &helmer{
				buildValuesFilesChangesFn: func(
					[]kargoapi.Image,
					[]kargoapi.HelmImageUpdate,
				) (map[string]map[string]string, []string) {
					return map[string]map[string]string{
						testValuesFile: {
							testKey: testValue,
						},
					}, nil
				},
				setStringsInYAMLFileFn: func(string, map[string]string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(_ []string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error updating values in file")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "error building chart dependency changes",
			helmer: &helmer{
				buildValuesFilesChangesFn: func(
					[]kargoapi.Image,
					[]kargoapi.HelmImageUpdate,
				) (map[string]map[string]string, []string) {
					// This returns nothing so that the only calls to
					// setStringsInYAMLFileFn will be for updating subcharts in
					// Charts.yaml.
					return nil, nil
				},
				buildChartDependencyChangesFn: func(
					string,
					[]kargoapi.Chart,
					[]kargoapi.HelmChartDependencyUpdate,
				) (map[string]map[string]string, []string, error) {
					return nil, nil, errors.New("something went wrong")
				},
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
			helmer: &helmer{
				buildValuesFilesChangesFn: func(
					[]kargoapi.Image,
					[]kargoapi.HelmImageUpdate,
				) (map[string]map[string]string, []string) {
					// This returns nothing so that the only calls to
					// setStringsInYAMLFileFn will be for updating subcharts in
					// Charts.yaml.
					return nil, nil
				},
				buildChartDependencyChangesFn: func(
					string,
					[]kargoapi.Chart,
					[]kargoapi.HelmChartDependencyUpdate,
				) (map[string]map[string]string, []string, error) {
					return map[string]map[string]string{
						testChartFile: {
							testKey: testValue,
						},
					}, nil, nil
				},
				setStringsInYAMLFileFn: func(string, map[string]string) error {
					return errors.New("something went wrong")
				},
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
			name: "error running helm chart dep up",
			helmer: &helmer{
				buildValuesFilesChangesFn: func(
					[]kargoapi.Image,
					[]kargoapi.HelmImageUpdate,
				) (map[string]map[string]string, []string) {
					return nil, nil
				},
				buildChartDependencyChangesFn: func(
					string,
					[]kargoapi.Chart,
					[]kargoapi.HelmChartDependencyUpdate,
				) (map[string]map[string]string, []string, error) {
					return map[string]map[string]string{
						testChartFile: {
							testKey: testValue,
						},
					}, nil, nil
				},
				setStringsInYAMLFileFn: func(string, map[string]string) error {
					return nil
				},
				updateChartDependenciesFn: func(string, string) error {
					return errors.New("something went wrong")
				},
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
			helmer: &helmer{
				buildValuesFilesChangesFn: func(
					[]kargoapi.Image,
					[]kargoapi.HelmImageUpdate,
				) (map[string]map[string]string, []string) {
					return map[string]map[string]string{
						testValuesFile: {
							testKey: testValue,
						},
					}, []string{"fake-image-update"}
				},
				buildChartDependencyChangesFn: func(
					string,
					[]kargoapi.Chart,
					[]kargoapi.HelmChartDependencyUpdate,
				) (map[string]map[string]string, []string, error) {
					return map[string]map[string]string{
						testChartFile: {
							testKey: testValue,
						},
					}, []string{"fake-chart-update"}, nil
				},
				setStringsInYAMLFileFn: func(string, map[string]string) error {
					return nil
				},
				updateChartDependenciesFn: func(string, string) error {
					return nil
				},
			},
			assertions: func(changes []string, err error) {
				require.NoError(t, err)
				require.Len(t, changes, 2)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.helmer.apply(
					kargoapi.GitRepoUpdate{
						Helm: &kargoapi.HelmPromotionMechanism{},
					},
					kargoapi.FreightReference{}, // The way the tests are structured, this value doesn't matter
					"",
					"",
				),
			)
		})
	}
}

func TestBuildValuesFilesChanges(t *testing.T) {
	images := []kargoapi.Image{
		{
			RepoURL: "fake-url",
			Tag:     "fake-tag",
			Digest:  "fake-digest",
		},
		{
			RepoURL: "second-fake-url",
			Tag:     "second-fake-tag",
			Digest:  "second-fake-digest",
		},
		{
			RepoURL: "third-fake-url",
			Tag:     "third-fake-tag",
			Digest:  "third-fake-digest",
		},
		{
			RepoURL: "fourth-fake-url",
			Tag:     "fourth-fake-tag",
			Digest:  "fourth-fake-digest",
		},
	}
	imageUpdates := []kargoapi.HelmImageUpdate{
		{
			ValuesFilePath: "fake-values.yaml",
			Image:          "fake-url",
			Key:            "fake-key",
			Value:          kargoapi.ImageUpdateValueTypeImageAndTag,
		},
		{
			ValuesFilePath: "fake-values.yaml",
			Image:          "second-fake-url",
			Key:            "second-fake-key",
			Value:          kargoapi.ImageUpdateValueTypeTag,
		},
		{
			ValuesFilePath: "another-fake-values.yaml",
			Image:          "third-fake-url",
			Key:            "third-fake-key",
			Value:          kargoapi.ImageUpdateValueTypeImageAndDigest,
		},
		{
			ValuesFilePath: "another-fake-values.yaml",
			Image:          "fourth-fake-url",
			Key:            "fourth-fake-key",
			Value:          kargoapi.ImageUpdateValueTypeDigest,
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
				"fake-key":        "fake-url:fake-tag",
				"second-fake-key": "'second-fake-tag'",
			},
			"another-fake-values.yaml": {
				"third-fake-key":  "third-fake-url@third-fake-digest",
				"fourth-fake-key": "fourth-fake-digest",
			},
		},
		result,
	)
	require.Equal(
		t,
		[]string{
			"updated fake-values.yaml to use image fake-url:fake-tag",
			"updated fake-values.yaml to use image second-fake-url:second-fake-tag",
			"updated another-fake-values.yaml to use image third-fake-url@third-fake-digest",
			"updated another-fake-values.yaml to use image fourth-fake-url@fourth-fake-digest",
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
	charts := []kargoapi.Chart{
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
	chartUpdates := []kargoapi.HelmChartDependencyUpdate{
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
