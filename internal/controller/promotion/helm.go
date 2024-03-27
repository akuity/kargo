package promotion

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v3"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
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
	selectedUpdates := make([]kargoapi.GitRepoUpdate, 0, len(updates))
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
	newFreight kargoapi.FreightReference,
	_ string, // TODO: sourceCommit would be a nice addition to the commit message
	homeDir string,
	workingDir string,
	_ git.RepoCredentials,
) ([]string, error) {
	// Image updates
	changesByFile, imageChangeSummary :=
		h.buildValuesFilesChangesFn(newFreight.Images, update.Helm.Images)
	for file, changes := range changesByFile {
		if err := h.setStringsInYAMLFileFn(
			filepath.Join(workingDir, file),
			changes,
		); err != nil {
			return nil, fmt.Errorf("error updating values in file %q: %w", file, err)
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
		return nil, fmt.Errorf("error preparing changes to affected Chart.yaml files: %w", err)
	}
	for chart, changes := range changesByChart {
		chartPath := filepath.Join(workingDir, chart)
		chartYAMLPath := filepath.Join(chartPath, "Chart.yaml")
		if err = h.setStringsInYAMLFileFn(chartYAMLPath, changes); err != nil {
			return nil, fmt.Errorf("error updating dependencies for chart %q: %w", chart, err)
		}
		if err = h.updateChartDependenciesFn(homeDir, chartPath); err != nil {
			return nil, fmt.Errorf("error updating dependencies for chart %q: :%w", chart, err)
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
	digestsByImage := make(map[string]string, len(images))
	for _, image := range images {
		tagsByImage[image.RepoURL] = image.Tag
		digestsByImage[image.RepoURL] = image.Digest
	}
	changesByFile := make(map[string]map[string]string, len(imageUpdates))
	changeSummary := make([]string, 0, len(imageUpdates))
	for _, imageUpdate := range imageUpdates {
		switch imageUpdate.Value {
		case kargoapi.ImageUpdateValueTypeImageAndTag,
			kargoapi.ImageUpdateValueTypeTag,
			kargoapi.ImageUpdateValueTypeImageAndDigest,
			kargoapi.ImageUpdateValueTypeDigest:
		default:
			// This really shouldn't happen, so we'll ignore it.
			continue
		}
		tag, tagFound := tagsByImage[imageUpdate.Image]
		digest, digestFound := digestsByImage[imageUpdate.Image]
		if !tagFound && !digestFound {
			// There's no change to make in this case.
			continue
		}
		if _, found := changesByFile[imageUpdate.ValuesFilePath]; !found {
			changesByFile[imageUpdate.ValuesFilePath] = map[string]string{}
		}

		var fqImageRef string // Fully qualified image reference
		switch imageUpdate.Value {
		case kargoapi.ImageUpdateValueTypeImageAndTag:
			changesByFile[imageUpdate.ValuesFilePath][imageUpdate.Key] =
				fmt.Sprintf("%s:%s", imageUpdate.Image, tag)
			fqImageRef = fmt.Sprintf("%s:%s", imageUpdate.Image, tag)
		case kargoapi.ImageUpdateValueTypeTag:
			changesByFile[imageUpdate.ValuesFilePath][imageUpdate.Key] = "'" + tag + "'"
			fqImageRef = fmt.Sprintf("%s:%s", imageUpdate.Image, tag)
		case kargoapi.ImageUpdateValueTypeImageAndDigest:
			changesByFile[imageUpdate.ValuesFilePath][imageUpdate.Key] =
				fmt.Sprintf("%s@%s", imageUpdate.Image, digest)
			fqImageRef = fmt.Sprintf("%s@%s", imageUpdate.Image, digest)
		case kargoapi.ImageUpdateValueTypeDigest:
			changesByFile[imageUpdate.ValuesFilePath][imageUpdate.Key] = digest
			fqImageRef = fmt.Sprintf("%s@%s", imageUpdate.Image, digest)
		}
		changeSummary = append(
			changeSummary,
			fmt.Sprintf(
				"updated %s to use image %s",
				imageUpdate.ValuesFilePath,
				fqImageRef,
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
	versionsByChart := make(map[string]string, len(charts))
	for _, chart := range charts {
		// path.Join accounts for the possibility that chart.Name is empty
		key := path.Join(chart.RepoURL, chart.Name)
		versionsByChart[key] = chart.Version
	}

	// Build a de-duped set of paths to affected Charts files
	chartPaths := make(map[string]struct{}, len(chartUpdates))
	for _, chartUpdate := range chartUpdates {
		chartPaths[chartUpdate.ChartPath] = struct{}{}
	}

	// For each chart, build the appropriate changes
	changesByFile := make(map[string]map[string]string)
	changeSummary := make([]string, 0)
	for chartPath := range chartPaths {
		absChartYAMLPath := filepath.Join(repoDir, chartPath, "Chart.yaml")
		chartYAMLBytes, err := os.ReadFile(absChartYAMLPath)
		if err != nil {
			return nil, nil, fmt.Errorf("error reading file %q: %w", absChartYAMLPath, err)
		}
		chartYAMLObj := &struct {
			Dependencies []struct {
				Repository string `json:"repository,omitempty"`
				Name       string `json:"name,omitempty"`
			} `json:"dependencies,omitempty"`
		}{}
		if err := yaml.Unmarshal(chartYAMLBytes, chartYAMLObj); err != nil {
			return nil, nil, fmt.Errorf("error unmarshaling %q: %w", absChartYAMLPath, err)
		}
		for i, dependency := range chartYAMLObj.Dependencies {
			chartKey := path.Join(dependency.Repository, dependency.Name)
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
