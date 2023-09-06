package promotion

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
	libYAML "github.com/akuity/kargo/internal/yaml"
)

// newGenericGitMechanism returns a gitMechanism that only only selects and
// performs updates that involve Helm.
func newHelmMechanism(
	credentialsDB credentials.Database,
) Mechanism {
	return newGitMechanism(
		"Helm promotion mechanism",
		credentialsDB,
		selectHelmUpdates,
		(&helmer{
			buildValuesFilesChangesFn:     buildValuesFilesChanges,
			buildChartDependencyChangesFn: buildChartDependencyChanges,
			setStringsInYAMLFileFn:        libYAML.SetStringsInFile,
			updateChartDependenciesFn:     helm.UpdateChartDependencies,
		}).apply,
	)
}

// selectHelmUpdates returns a subset of the given updates that involve Helm.
func selectHelmUpdates(updates []kargoapi.GitRepoUpdate) []kargoapi.GitRepoUpdate {
	var selectedUpdates []kargoapi.GitRepoUpdate
	for _, update := range updates {
		if update.Helm != nil {
			selectedUpdates = append(selectedUpdates, update)
		}
	}
	return selectedUpdates
}

// helmer is a helper struct whose sole purpose is to close over several other
// functions that are used in the implementation of the apply() function.
type helmer struct {
	buildValuesFilesChangesFn func(
		[]kargoapi.Image,
		[]kargoapi.HelmImageUpdate,
	) (map[string]map[string]string, []string)
	buildChartDependencyChangesFn func(
		string,
		[]kargoapi.Chart,
		[]kargoapi.HelmChartDependencyUpdate,
	) (map[string]map[string]string, []string, error)
	setStringsInYAMLFileFn    func(file string, changes map[string]string) error
	updateChartDependenciesFn func(homeDir, chartPath string) error
}

// apply uses Helm to carry out the provided update in the specified working
// directory.
func (h *helmer) apply(
	update kargoapi.GitRepoUpdate,
	newFreight kargoapi.Freight,
	homeDir string,
	workingDir string,
) ([]string, error) {
	// Image updates
	changesByFile, imageChangeSummary :=
		h.buildValuesFilesChangesFn(newFreight.Images, update.Helm.Images)
	for file, changes := range changesByFile {
		if err := h.setStringsInYAMLFileFn(
			filepath.Join(workingDir, file),
			changes,
		); err != nil {
			return nil, errors.Wrapf(err, "error updating values in file %q", file)
		}
	}

	// Chart dependency updates
	changesByChart, subchartChangeSummary, err :=
		h.buildChartDependencyChangesFn(
			workingDir,
			newFreight.Charts,
			update.Helm.Charts,
		)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"error preparing changes to affected Chart.yaml files",
		)
	}
	for chart, changes := range changesByChart {
		chartPath := filepath.Join(workingDir, chart)
		chartYAMLPath := filepath.Join(chartPath, "Chart.yaml")
		if err = h.setStringsInYAMLFileFn(chartYAMLPath, changes); err != nil {
			return nil, errors.Wrapf(
				err,
				"error updating dependencies for chart %q",
				chart,
			)
		}
		if err = h.updateChartDependenciesFn(homeDir, chartPath); err != nil {
			return nil, errors.Wrapf(
				err,
				"error updating dependencies for chart %q",
				chart,
			)
		}
	}

	return append(imageChangeSummary, subchartChangeSummary...), nil
}

// buildValuesFilesChanges takes a list of images and a list of instructions
// about changes that should be made to various YAML files and distills them
// into a map of maps that indexes new values for each YAML file by file name
// and key.
func buildValuesFilesChanges(
	images []kargoapi.Image,
	imageUpdates []kargoapi.HelmImageUpdate,
) (map[string]map[string]string, []string) {
	tagsByImage := map[string]string{}
	for _, image := range images {
		tagsByImage[image.RepoURL] = image.Tag
	}

	changesByFile := map[string]map[string]string{}
	changeSummary := []string{}
	for _, imageUpdate := range imageUpdates {
		if imageUpdate.Value != kargoapi.ImageUpdateValueTypeImage &&
			imageUpdate.Value != kargoapi.ImageUpdateValueTypeTag {
			// This really shouldn't happen, so we'll ignore it.
			continue
		}
		tag, found := tagsByImage[imageUpdate.Image]
		if !found {
			// There's no change to make in this case.
			continue
		}
		if _, found = changesByFile[imageUpdate.ValuesFilePath]; !found {
			changesByFile[imageUpdate.ValuesFilePath] = map[string]string{}
		}
		if imageUpdate.Value == kargoapi.ImageUpdateValueTypeImage {
			changesByFile[imageUpdate.ValuesFilePath][imageUpdate.Key] =
				fmt.Sprintf("%s:%s", imageUpdate.Image, tag)
		} else {
			changesByFile[imageUpdate.ValuesFilePath][imageUpdate.Key] = tag
		}
		changeSummary = append(
			changeSummary,
			fmt.Sprintf(
				"updated %s to use image %s:%s",
				imageUpdate.ValuesFilePath,
				imageUpdate.Image,
				tag,
			),
		)
	}

	return changesByFile, changeSummary
}

// buildChartDependencyChanges takes a list of charts and a list of instructions
// about changes that should be made to various Chart.yaml files and distills
// them into a map of maps that indexes new values for each Chart.yaml file by
// file name and key.
func buildChartDependencyChanges(
	repoDir string,
	charts []kargoapi.Chart,
	chartUpdates []kargoapi.HelmChartDependencyUpdate,
) (map[string]map[string]string, []string, error) {
	// Build a table of charts --> versions
	versionsByChart := map[string]string{}
	for _, chart := range charts {
		key := fmt.Sprintf("%s:%s", chart.RegistryURL, chart.Name)
		versionsByChart[key] = chart.Version
	}

	// Build a de-duped set of paths to affected Charts files
	chartPaths := map[string]struct{}{}
	for _, chartUpdate := range chartUpdates {
		chartPaths[chartUpdate.ChartPath] = struct{}{}
	}

	// For each chart, build the appropriate changes
	changesByFile := map[string]map[string]string{}
	changeSummary := []string{}
	for chartPath := range chartPaths {
		absChartYAMLPath := filepath.Join(repoDir, chartPath, "Chart.yaml")
		chartYAMLBytes, err := os.ReadFile(absChartYAMLPath)
		if err != nil {
			return nil, nil,
				errors.Wrapf(err, "error reading file %q", absChartYAMLPath)
		}
		chartYAMLObj := &struct {
			Dependencies []struct {
				Repository string `json:"repository,omitempty"`
				Name       string `json:"name,omitempty"`
			} `json:"dependencies,omitempty"`
		}{}
		if err := yaml.Unmarshal(chartYAMLBytes, chartYAMLObj); err != nil {
			return nil, nil,
				errors.Wrapf(err, "error unmarshaling %q", absChartYAMLPath)
		}
		for i, dependency := range chartYAMLObj.Dependencies {
			chartKey := fmt.Sprintf("%s:%s", dependency.Repository, dependency.Name)
			version, found := versionsByChart[chartKey]
			if !found {
				continue
			}
			if found {
				if _, found = changesByFile[chartPath]; !found {
					changesByFile[chartPath] = map[string]string{}
				}
			}
			versionKey := fmt.Sprintf("dependencies.%d.version", i)
			changesByFile[chartPath][versionKey] = version
			changeSummary = append(
				changeSummary,
				fmt.Sprintf(
					"updated %s/Chart.yaml to use subchart %s:%s",
					chartPath,
					dependency.Name,
					version,
				),
			)
		}
	}

	return changesByFile, changeSummary, nil
}
