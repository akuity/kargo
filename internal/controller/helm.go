package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/helm"
)

func (e *environmentReconciler) applyHelm(
	newState api.EnvironmentState,
	update api.HelmPromotionMechanism,
	homeDir string,
	repoDir string,
) error {
	// Image updates
	changesByFile := buildValuesFilesChanges(newState.Images, update.Images)
	for file, changes := range changesByFile {
		if err := e.setStringsInYAMLFileFn(
			filepath.Join(repoDir, file),
			changes,
		); err != nil {
			return errors.Wrapf(err, "error updating values in file %q", file)
		}
	}

	// Chart dependency updates
	changesByChart, err := e.buildChartDependencyChangesFn(
		repoDir,
		newState.Charts,
		update.Charts,
	)
	if err != nil {
		return errors.Wrap(
			err,
			"error preparing changes to affected Chart.yaml files",
		)
	}
	for chart, changes := range changesByChart {
		chartPath := filepath.Join(repoDir, chart)
		chartYAMLPath := filepath.Join(chartPath, "Chart.yaml")
		if err := e.setStringsInYAMLFileFn(chartYAMLPath, changes); err != nil {
			return errors.Wrapf(
				err,
				"error updating dependencies for chart %q",
				chart,
			)
		}
		if err :=
			e.updateChartDependenciesFn(homeDir, chartPath); err != nil {
			return errors.Wrapf(
				err,
				"error updating dependencies for chart %q",
				chart,
			)
		}
	}

	return nil
}

func (e *environmentReconciler) getLatestCharts(
	ctx context.Context,
	namespace string,
	subs []api.ChartSubscription,
) ([]api.Chart, error) {
	charts := make([]api.Chart, len(subs))

	for i, sub := range subs {
		imgLogger := e.logger.WithFields(log.Fields{
			"registry": sub.RegistryURL,
			"chart":    sub.Name,
		})

		creds, ok, err :=
			e.credentialsDB.get(ctx, namespace, credentialsTypeHelm, sub.RegistryURL)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error obtaining credentials for chart registry %q",
				sub.RegistryURL,
			)
		}
		imgLogger.Debug("acquired credentials for chart registry/repository")

		var helmCreds *helm.Credentials
		if ok {
			helmCreds = &helm.Credentials{
				Username: creds.Username,
				Password: creds.Password,
			}
		}

		vers, err := e.getLatestChartVersionFn(
			ctx,
			sub.RegistryURL,
			sub.Name,
			sub.SemverConstraint,
			helmCreds,
		)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error searching for latest version of chart %q in registry %q",
				sub.Name,
				sub.RegistryURL,
			)
		}

		if vers != "" {
			imgLogger.WithField("version", vers).
				Debug("found latest suitable chart version")
		} else {
			imgLogger.Error("found no suitable chart version")
			return nil, errors.Errorf(
				"found no suitable version of chart %q in registry %q",
				sub.Name,
				sub.RegistryURL,
			)
		}

		charts[i] = api.Chart{
			RegistryURL: sub.RegistryURL,
			Name:        sub.Name,
			Version:     vers,
		}
	}

	return charts, nil
}

// buildValuesFilesChanges takes a list of images and a list of instructions
// about changes that should be made to various YAML files and distills them
// into a map of maps that indexes new values for each YAML file by file name
// and key.
func buildValuesFilesChanges(
	images []api.Image,
	imageUpdates []api.HelmImageUpdate,
) map[string]map[string]string {
	tagsByImage := map[string]string{}
	for _, image := range images {
		tagsByImage[image.RepoURL] = image.Tag
	}

	changesByFile := map[string]map[string]string{}
	for _, imageUpdate := range imageUpdates {
		if imageUpdate.Value != api.ImageUpdateValueTypeImage &&
			imageUpdate.Value != api.ImageUpdateValueTypeTag {
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
		if imageUpdate.Value == api.ImageUpdateValueTypeImage {
			changesByFile[imageUpdate.ValuesFilePath][imageUpdate.Key] =
				fmt.Sprintf("%s:%s", imageUpdate.Image, tag)
		} else {
			changesByFile[imageUpdate.ValuesFilePath][imageUpdate.Key] = tag
		}
	}

	return changesByFile
}

// buildChartDependencyChanges takes a list of charts and a list of instructions
// about changes that should be made to various Chart.yaml files and distills
// them into a map of maps that indexes new values for each Chart.yaml file by
// file name and key.
func buildChartDependencyChanges(
	repoDir string,
	charts []api.Chart,
	chartUpdates []api.HelmChartDependencyUpdate,
) (map[string]map[string]string, error) {
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
	changesByChart := map[string]map[string]string{}
	for chartPath := range chartPaths {
		absChartYAMLPath := filepath.Join(repoDir, chartPath, "Chart.yaml")
		chartYAMLBytes, err := os.ReadFile(absChartYAMLPath)
		if err != nil {
			return nil, errors.Wrapf(err, "error reading file %q", absChartYAMLPath)
		}
		chartYAMLObj := &struct {
			Dependencies []struct {
				Repository string `json:"repository,omitempty"`
				Name       string `json:"name,omitempty"`
			} `json:"dependencies,omitempty"`
		}{}
		if err := yaml.Unmarshal(chartYAMLBytes, chartYAMLObj); err != nil {
			return nil, errors.Wrapf(err, "error unmarshaling %q", absChartYAMLPath)
		}
		for i, dependency := range chartYAMLObj.Dependencies {
			chartKey := fmt.Sprintf("%s:%s", dependency.Repository, dependency.Name)
			version, found := versionsByChart[chartKey]
			if !found {
				continue
			}
			if found {
				if _, found = changesByChart[chartPath]; !found {
					changesByChart[chartPath] = map[string]string{}
				}
			}
			versionKey := fmt.Sprintf("dependencies.%d.version", i)
			changesByChart[chartPath][versionKey] = version
		}
	}

	return changesByChart, nil
}
