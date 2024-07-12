package promotion

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/freight"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
	libYAML "github.com/akuity/kargo/internal/yaml"
)

// newGenericGitMechanism returns a gitMechanism that only only selects and
// performs updates that involve Helm.
func newHelmMechanism(
	cl client.Client,
	credentialsDB credentials.Database,
) Mechanism {
	h := &helmer{
		client: cl,
	}
	h.buildValuesFilesChangesFn = h.buildValuesFilesChanges
	h.buildChartDependencyChangesFn = h.buildChartDependencyChanges
	h.setStringsInYAMLFileFn = libYAML.SetStringsInFile
	h.prepareDependencyCredentialsFn = prepareDependencyCredentialsFn(credentialsDB)
	h.updateChartDependenciesFn = helm.UpdateChartDependencies

	return newGitMechanism(
		"Helm promotion mechanism",
		cl,
		credentialsDB,
		selectHelmUpdates,
		h.apply,
	)
}

// selectHelmUpdates returns a subset of the given updates that involve Helm.
func selectHelmUpdates(updates []kargoapi.GitRepoUpdate) []*kargoapi.GitRepoUpdate {
	selectedUpdates := make([]*kargoapi.GitRepoUpdate, 0, len(updates))
	for i := range updates {
		update := &updates[i]
		if update.Helm != nil {
			selectedUpdates = append(selectedUpdates, update)
		}
	}
	return selectedUpdates
}

// helmer is a helper struct whose sole purpose is to close over several other
// functions that are used in the implementation of the apply() function.
type helmer struct {
	client                    client.Client
	buildValuesFilesChangesFn func(
		context.Context,
		*kargoapi.Stage,
		*kargoapi.HelmPromotionMechanism,
		[]kargoapi.FreightReference,
	) (map[string]map[string]string, []string, error)
	buildChartDependencyChangesFn func(
		ctx context.Context,
		stage *kargoapi.Stage,
		update *kargoapi.HelmPromotionMechanism,
		freight []kargoapi.FreightReference,
		workingDir string,
	) (map[string]map[string]string, []string, error)
	setStringsInYAMLFileFn         func(file string, changes map[string]string) error
	prepareDependencyCredentialsFn func(ctx context.Context, homePath, chartPath, namespace string) error
	updateChartDependenciesFn      func(homeDir, chartPath string) error
}

// apply uses Helm to carry out the provided update in the specified working
// directory.
func (h *helmer) apply(
	ctx context.Context,
	stage *kargoapi.Stage,
	update *kargoapi.GitRepoUpdate,
	newFreight []kargoapi.FreightReference,
	_ string, // TODO: sourceCommit would be a nice addition to the commit message
	homeDir string,
	workingDir string,
	_ git.RepoCredentials,
) ([]string, error) {
	changesByFile, imageChangeSummary, err := h.buildValuesFilesChangesFn(
		ctx,
		stage,
		update.Helm,
		newFreight,
	)
	if err != nil {
		return nil,
			fmt.Errorf("error preparing changes to affected values files: %w", err)
	}
	for file, changes := range changesByFile {
		if err = h.setStringsInYAMLFileFn(
			filepath.Join(workingDir, file),
			changes,
		); err != nil {
			return nil, fmt.Errorf("updating values in file %q: %w", file, err)
		}
	}

	// Chart dependency updates
	changesByChart, subchartChangeSummary, err :=
		h.buildChartDependencyChangesFn(
			ctx,
			stage,
			update.Helm,
			newFreight,
			workingDir,
		)
	if err != nil {
		return nil, fmt.Errorf("preparing changes to affected Chart.yaml files: %w", err)
	}
	for chart, changes := range changesByChart {
		chartPath := filepath.Join(workingDir, chart)
		chartYAMLPath := filepath.Join(chartPath, "Chart.yaml")
		if err = h.setStringsInYAMLFileFn(chartYAMLPath, changes); err != nil {
			return nil, fmt.Errorf("setting dependency versions for chart %q: %w", chart, err)
		}
		if err = h.prepareDependencyCredentialsFn(
			ctx, homeDir, chartYAMLPath, stage.Namespace,
		); err != nil {
			return nil, fmt.Errorf("preparing credentials for chart dependencies %q: :%w", chart, err)
		}
		if err = h.updateChartDependenciesFn(homeDir, chartPath); err != nil {
			return nil, fmt.Errorf("updating dependencies for chart %q: %w", chart, err)
		}
	}

	return append(imageChangeSummary, subchartChangeSummary...), nil
}

// buildValuesFilesChanges takes a list of images and a list of instructions
// about changes that should be made to various YAML files and distills them
// into a map of maps that indexes new values for each YAML file by file name
// and key.
func (h *helmer) buildValuesFilesChanges(
	ctx context.Context,
	stage *kargoapi.Stage,
	update *kargoapi.HelmPromotionMechanism,
	newFreight []kargoapi.FreightReference,
) (map[string]map[string]string, []string, error) {
	changesByFile := make(map[string]map[string]string, len(update.Images))
	changeSummary := make([]string, 0, len(update.Images))
	for i := range update.Images {
		imageUpdate := &update.Images[i]
		switch imageUpdate.Value {
		case kargoapi.ImageUpdateValueTypeImageAndTag,
			kargoapi.ImageUpdateValueTypeTag,
			kargoapi.ImageUpdateValueTypeImageAndDigest,
			kargoapi.ImageUpdateValueTypeDigest:
		default:
			// This really shouldn't happen, so we'll ignore it.
			continue
		}
		desiredOrigin := freight.GetDesiredOrigin(stage, imageUpdate)
		image, err := freight.FindImage(ctx, h.client, stage, desiredOrigin, newFreight, imageUpdate.Image)
		if err != nil {
			return nil, nil,
				fmt.Errorf("error finding image from repo %q: %w", imageUpdate.Image, err)
		}
		if image == nil {
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
				fmt.Sprintf("%s:%s", imageUpdate.Image, image.Tag)
			fqImageRef = fmt.Sprintf("%s:%s", imageUpdate.Image, image.Tag)
		case kargoapi.ImageUpdateValueTypeTag:
			changesByFile[imageUpdate.ValuesFilePath][imageUpdate.Key] =
				fmt.Sprintf("'%s'", image.Tag)
			fqImageRef = fmt.Sprintf("%s:%s", imageUpdate.Image, image.Tag)
		case kargoapi.ImageUpdateValueTypeImageAndDigest:
			changesByFile[imageUpdate.ValuesFilePath][imageUpdate.Key] =
				fmt.Sprintf("%s@%s", imageUpdate.Image, image.Digest)
			fqImageRef = fmt.Sprintf("%s@%s", imageUpdate.Image, image.Digest)
		case kargoapi.ImageUpdateValueTypeDigest:
			changesByFile[imageUpdate.ValuesFilePath][imageUpdate.Key] =
				fmt.Sprintf("'%s'", image.Digest)
			fqImageRef = fmt.Sprintf("%s@%s", imageUpdate.Image, image.Digest)
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
	return changesByFile, changeSummary, nil
}

// buildChartDependencyChanges takes a list of charts and a list of instructions
// about changes that should be made to various Chart.yaml files and distills
// them into a map of maps that indexes new values for each Chart.yaml file by
// file name and key.
func (h *helmer) buildChartDependencyChanges(
	ctx context.Context,
	stage *kargoapi.Stage,
	update *kargoapi.HelmPromotionMechanism,
	newFreight []kargoapi.FreightReference,
	repoDir string,
) (map[string]map[string]string, []string, error) {
	// Build a map of updates by chart
	updatesByChartPath := map[string][]*kargoapi.HelmChartDependencyUpdate{}
	for i := range update.Charts {
		chartUpdate := &update.Charts[i]
		if updates, found := updatesByChartPath[chartUpdate.ChartPath]; !found {
			updates = []*kargoapi.HelmChartDependencyUpdate{chartUpdate}
			updatesByChartPath[chartUpdate.ChartPath] = updates
		} else {
			updatesByChartPath[chartUpdate.ChartPath] = append(updates, chartUpdate)
		}
	}
	changesByChart := make(map[string]map[string]string)
	changeSummary := make([]string, 0)
	for chartPath, updates := range updatesByChartPath {
		absChartYAMLPath := filepath.Join(repoDir, chartPath, "Chart.yaml")
		chartDependencies, err := loadChartDependencies(absChartYAMLPath)
		if err != nil {
			return nil, nil, fmt.Errorf("loading dependencies for chart: %w", err)
		}
		for _, update := range updates {
			desiredOrigin := freight.GetDesiredOrigin(stage, update)
			chart, err := freight.FindChart(
				ctx,
				h.client,
				stage,
				desiredOrigin,
				newFreight,
				update.Repository,
				update.Name,
			)
			if err != nil {
				return nil, nil,
					fmt.Errorf("error finding chart from repo %q: %w", update.Repository, err)
			}
			if chart == nil {
				// There's no change to make in this case.
				continue
			}
			for i, dependency := range chartDependencies {
				if update.Repository != dependency.Repository || update.Name != dependency.Name {
					continue
				}
				key := fmt.Sprintf("dependencies.%d.version", i)
				if _, found := changesByChart[chartPath]; !found {
					changesByChart[chartPath] = map[string]string{
						key: chart.Version,
					}
				} else {
					changesByChart[chartPath][key] = chart.Version
				}
				changeSummary = append(
					changeSummary,
					fmt.Sprintf(
						"updated %s/Chart.yaml to use subchart %s:%s",
						chartPath,
						dependency.Name,
						chart.Version,
					),
				)
			}
		}

	}
	return changesByChart, changeSummary, nil
}

// prepareDependencyCredentialsFn returns a function that prepares the necessary
// credentials for the dependencies of a Helm chart. The returned function is
// intended to be called once per chart.
func prepareDependencyCredentialsFn(
	db credentials.Database,
) func(ctx context.Context, homePath, chartPath, namespace string) error {
	return func(ctx context.Context, homePath, chartPath, namespace string) error {
		dependencies, err := loadChartDependencies(chartPath)
		if err != nil {
			return fmt.Errorf("loading dependencies to resolve credentials for: %w", err)
		}

		for _, dependency := range dependencies {
			var creds credentials.Credentials
			var ok bool
			var repository string

			switch {
			case strings.HasPrefix(dependency.Repository, "https://"):
				repository = dependency.Repository
				if creds, ok, err = db.Get(ctx, namespace, credentials.TypeHelm, repository); err != nil {
					return fmt.Errorf(
						"obtaining credentials for chart repository %q: %w",
						dependency.Repository,
						err,
					)
				}
			case strings.HasPrefix(dependency.Repository, "oci://"):
				// NB: We log in to the OCI registry using the repository URL,
				// and not the full chart reference.
				repository = dependency.Repository
				if creds, ok, err = db.Get(
					ctx,
					namespace,
					credentials.TypeHelm,
					"oci://"+path.Join(helm.NormalizeChartRepositoryURL(repository), dependency.Name),
				); err != nil {
					return fmt.Errorf(
						"obtaining credentials for chart repository %q: %w",
						repository,
						err,
					)
				}
			}

			if !ok {
				continue
			}

			if err := helm.Login(homePath, repository, helm.Credentials{
				Username: creds.Username,
				Password: creds.Password,
			}); err != nil {
				return fmt.Errorf("login to chart repository %q: %w", repository, err)
			}
		}

		return nil
	}
}

// chartDependency is a struct that represents a dependency listed in a
// Chart.yaml. It only includes the fields that are relevant to this package.
type chartDependency struct {
	Repository string `json:"repository,omitempty"`
	Name       string `json:"name,omitempty"`
}

// loadChartDependencies reads the Chart.yaml file at the given path and returns
// the dependencies listed in it.
func loadChartDependencies(chartPath string) ([]chartDependency, error) {
	b, err := os.ReadFile(chartPath)
	if err != nil {
		return nil, fmt.Errorf("reading file %q: %w", chartPath, err)
	}

	chartObj := &struct {
		Dependencies []chartDependency `json:"dependencies,omitempty"`
	}{}
	if err := yaml.Unmarshal(b, chartObj); err != nil {
		return nil, fmt.Errorf("unmarshaling %q: %w", chartPath, err)
	}

	return chartObj.Dependencies, nil
}
