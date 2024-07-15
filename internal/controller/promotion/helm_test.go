package promotion

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
)

func TestNewHelmMechanism(t *testing.T) {
	pm := newHelmMechanism(
		fake.NewFakeClient(),
		&credentials.FakeDB{},
	)
	hpm, ok := pm.(*gitMechanism)
	require.True(t, ok)
	require.Equal(t, "Helm promotion mechanism", hpm.name)
	require.NotNil(t, hpm.client)
	require.NotNil(t, hpm.selectUpdatesFn)
	require.NotNil(t, hpm.applyConfigManagementFn)
}

func TestSelectHelmUpdates(t *testing.T) {
	testCases := []struct {
		name       string
		updates    []kargoapi.GitRepoUpdate
		assertions func(t *testing.T, selectedUpdates []*kargoapi.GitRepoUpdate)
	}{
		{
			name: "no updates",
			assertions: func(t *testing.T, selectedUpdates []*kargoapi.GitRepoUpdate) {
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
			assertions: func(t *testing.T, selectedUpdates []*kargoapi.GitRepoUpdate) {
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
			assertions: func(t *testing.T, selectedUpdates []*kargoapi.GitRepoUpdate) {
				require.Len(t, selectedUpdates, 1)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(t, selectHelmUpdates(testCase.updates))
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
		assertions func(t *testing.T, changes []string, err error)
	}{
		{
			name: "error updating values file",
			helmer: &helmer{
				buildValuesFilesChangesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.HelmPromotionMechanism,
					[]kargoapi.FreightReference,
				) (map[string]map[string]string, []string, error) {
					return map[string]map[string]string{
						testValuesFile: {
							testKey: testValue,
						},
					}, nil, nil
				},
				setStringsInYAMLFileFn: func(string, map[string]string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []string, err error) {
				require.ErrorContains(t, err, "updating values in file")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error building chart dependency changes",
			helmer: &helmer{
				buildValuesFilesChangesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.HelmPromotionMechanism,
					[]kargoapi.FreightReference,
				) (map[string]map[string]string, []string, error) {
					// This returns nothing so that the only calls to
					// setStringsInYAMLFileFn will be for updating subcharts in
					// Charts.yaml.
					return nil, nil, nil
				},
				buildChartDependencyChangesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.HelmPromotionMechanism,
					[]kargoapi.FreightReference,
					string,
				) (map[string]map[string]string, []string, error) {
					return nil, nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []string, err error) {
				require.ErrorContains(t, err, "preparing changes to affected Chart.yaml files")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error updating Chart.yaml",
			helmer: &helmer{
				buildValuesFilesChangesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.HelmPromotionMechanism,
					[]kargoapi.FreightReference,
				) (map[string]map[string]string, []string, error) {
					// This returns nothing so that the only calls to
					// setStringsInYAMLFileFn will be for updating subcharts in
					// Charts.yaml.
					return nil, nil, nil
				},
				buildChartDependencyChangesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.HelmPromotionMechanism,
					[]kargoapi.FreightReference,
					string,
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
			assertions: func(t *testing.T, _ []string, err error) {
				require.ErrorContains(t, err, "setting dependency versions for chart")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error preparing dependency credentials",
			helmer: &helmer{
				buildValuesFilesChangesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.HelmPromotionMechanism,
					[]kargoapi.FreightReference,
				) (map[string]map[string]string, []string, error) {
					return nil, nil, nil
				},
				prepareDependencyCredentialsFn: func(context.Context, string, string, string) error {
					return fmt.Errorf("something went wrong")
				},
				buildChartDependencyChangesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.HelmPromotionMechanism,
					[]kargoapi.FreightReference,
					string,
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
			},
			assertions: func(t *testing.T, _ []string, err error) {
				require.ErrorContains(t, err, "preparing credentials for chart dependencies")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error running helm chart dep up",
			helmer: &helmer{
				buildValuesFilesChangesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.HelmPromotionMechanism,
					[]kargoapi.FreightReference,
				) (map[string]map[string]string, []string, error) {
					return nil, nil, nil
				},
				prepareDependencyCredentialsFn: func(context.Context, string, string, string) error {
					return nil
				},
				buildChartDependencyChangesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.HelmPromotionMechanism,
					[]kargoapi.FreightReference,
					string,
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
			assertions: func(t *testing.T, _ []string, err error) {
				require.ErrorContains(t, err, "updating dependencies for chart")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success",
			helmer: &helmer{
				buildValuesFilesChangesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.HelmPromotionMechanism,
					[]kargoapi.FreightReference,
				) (map[string]map[string]string, []string, error) {
					return map[string]map[string]string{
						testValuesFile: {
							testKey: testValue,
						},
					}, []string{"fake-image-update"}, nil
				},
				buildChartDependencyChangesFn: func(
					context.Context,
					*kargoapi.Stage,
					*kargoapi.HelmPromotionMechanism,
					[]kargoapi.FreightReference,
					string,
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
				prepareDependencyCredentialsFn: func(context.Context, string, string, string) error {
					return nil
				},
				updateChartDependenciesFn: func(string, string) error {
					return nil
				},
			},
			assertions: func(t *testing.T, changes []string, err error) {
				require.NoError(t, err)
				require.Len(t, changes, 2)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			stage := &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						GitRepoUpdates: []kargoapi.GitRepoUpdate{{
							Helm: &kargoapi.HelmPromotionMechanism{},
						}},
					},
				},
			}
			changes, err := testCase.helmer.apply(
				context.Background(),
				stage,
				&stage.Spec.PromotionMechanisms.GitRepoUpdates[0],
				[]kargoapi.FreightReference{}, // The way the tests are structured, this value doesn't matter
				"",
				"",
				"",
				git.RepoCredentials{},
			)
			testCase.assertions(t, changes, err)
		})
	}
}

func TestBuildValuesFilesChanges(t *testing.T) {
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	stage := &kargoapi.Stage{
		Spec: kargoapi.StageSpec{
			PromotionMechanisms: &kargoapi.PromotionMechanisms{
				GitRepoUpdates: []kargoapi.GitRepoUpdate{{
					Helm: &kargoapi.HelmPromotionMechanism{
						Origin: &testOrigin,
						Images: []kargoapi.HelmImageUpdate{
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
						},
					},
				}},
			},
		},
	}
	h := &helmer{}
	result, changeSummary, err := h.buildValuesFilesChanges(
		context.Background(),
		stage,
		stage.Spec.PromotionMechanisms.GitRepoUpdates[0].Helm,
		[]kargoapi.FreightReference{{
			Origin: testOrigin,
			Images: []kargoapi.Image{
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
			},
		}},
	)
	require.NoError(t, err)
	require.Equal(
		t,
		map[string]map[string]string{
			"fake-values.yaml": {
				"fake-key":        "fake-url:fake-tag",
				"second-fake-key": "'second-fake-tag'",
			},
			"another-fake-values.yaml": {
				"third-fake-key":  "third-fake-url@third-fake-digest",
				"fourth-fake-key": "'fourth-fake-digest'",
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
	testDir := t.TempDir()

	testChartsDir := filepath.Join(testDir, "charts")
	err := os.Mkdir(testChartsDir, 0755)
	require.NoError(t, err)

	testFooChartDir := filepath.Join(testChartsDir, "foo")
	err = os.Mkdir(testFooChartDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(
		filepath.Join(testFooChartDir, "Chart.yaml"),
		[]byte(`dependencies:
- repository: fake-repo
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
- repository: another-fake-repo
  name: another-fake-chart
  version: placeholder
- repository: yet-another-fake-repo
  name: yet-another-fake-chart
  version: placeholder
`),
		0600,
	)
	require.NoError(t, err)

	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}

	// New charts
	freight := []kargoapi.FreightReference{{
		Origin: testOrigin,
		Charts: []kargoapi.Chart{
			{
				RepoURL: "fake-repo",
				Name:    "fake-chart",
				Version: "fake-version",
			},
			{
				RepoURL: "another-fake-repo",
				Name:    "another-fake-chart",
				Version: "another-fake-version",
			},
		},
	}}

	stage := &kargoapi.Stage{
		Spec: kargoapi.StageSpec{
			PromotionMechanisms: &kargoapi.PromotionMechanisms{
				GitRepoUpdates: []kargoapi.GitRepoUpdate{{
					Helm: &kargoapi.HelmPromotionMechanism{
						Origin: &testOrigin,
						// Instructions for how to update Chart.yaml files
						Charts: []kargoapi.HelmChartDependencyUpdate{
							{
								Repository: "fake-repo",
								Name:       "fake-chart",
								ChartPath:  "charts/foo",
							},
							{
								Repository: "another-fake-repo",
								Name:       "another-fake-chart",
								ChartPath:  "charts/bar",
							},
							// Note there is no mention of how to update bar's second dependency, so
							// we expect it to be left alone.
						},
					},
				}},
			},
		},
	}

	h := &helmer{}
	result, changeSummary, err := h.buildChartDependencyChanges(
		context.Background(),
		stage,
		stage.Spec.PromotionMechanisms.GitRepoUpdates[0].Helm,
		freight,
		testDir,
	)
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
