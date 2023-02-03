package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/git"
	"github.com/akuityio/kargo/internal/helm"
	libYAML "github.com/akuityio/kargo/internal/yaml"
)

// TODO: Add some logging to this function
//
// TODO: There's more than one kind of Helm promotion we might have to do.
// We could be updating images or charts -- or  both.
func (e *environmentReconciler) promoteWithHelm(
	ctx context.Context,
	env *api.Environment,
	newState api.EnvironmentState,
) (api.EnvironmentState, error) {
	if env == nil ||
		env.Spec.PromotionMechanisms == nil ||
		env.Spec.PromotionMechanisms.Git == nil ||
		env.Spec.PromotionMechanisms.Git.Helm == nil ||
		(len(env.Spec.PromotionMechanisms.Git.Helm.Images) == 0 &&
			len(env.Spec.PromotionMechanisms.Git.Helm.Charts) == 0) ||
		(len(newState.Images) == 0 && len(newState.Charts) == 0) {
		return newState, nil
	}

	if env.Spec.GitRepo == nil || env.Spec.GitRepo.URL == "" {
		return newState, errors.New(
			"cannot promote images via Helm because spec does not contain " +
				"git repo details",
		)
	}

	return e.promoteWithGit(
		ctx,
		env,
		newState,
		"updating images",
		func(repo git.Repo) error {
			// Image updates
			imgUpdates := env.Spec.PromotionMechanisms.Git.Helm.Images
			changesByFile := buildValuesFilesChanges(newState.Images, imgUpdates)
			for file, changes := range changesByFile {
				if err := libYAML.SetStringsInFile(
					filepath.Join(repo.WorkingDir(), file),
					changes,
				); err != nil {
					return errors.Wrapf(
						err,
						"error updating values in file %q",
						file,
					)
				}
			}

			// Chart dependency updates
			chartDependencyUpdates := env.Spec.PromotionMechanisms.Git.Helm.Charts
			changesByChart, err := buildChartDependencyChanges(
				repo.WorkingDir(),
				newState.Charts,
				chartDependencyUpdates,
			)
			if err != nil {
				return errors.Wrap(
					err,
					"error preparing changes to affected Chart.yaml files",
				)
			}
			for chart, changes := range changesByChart {
				chartPath := filepath.Join(repo.WorkingDir(), chart)
				chartYAMLPath := filepath.Join(chartPath, "Chart.yaml")
				if err := libYAML.SetStringsInFile(chartYAMLPath, changes); err != nil {
					return errors.Wrapf(
						err,
						"error updating dependencies for chart %q",
						chart,
					)
				}
				if err :=
					helm.UpdateChartDependencies(repo.HomeDir(), chartPath); err != nil {
					return errors.Wrapf(
						err,
						"error updating dependencies for chart %q",
						chart,
					)
				}
			}

			return nil
		},
	)
}

func (e *environmentReconciler) getChartRegistryCredentials(
	ctx context.Context,
	registryURL string,
) (*helm.RepoCredentials, error) {
	const repoTypeHelm = "helm"

	creds := helm.RepoCredentials{}

	// NB: This next call returns an empty Repository if no such Repository is
	// found, so instead of continuing to look for credentials if no Repository is
	// found, what we'll do is continue looking for credentials if the Repository
	// we get back doesn't have anything we can use, i.e. no password.
	//
	// NB: Argo CD Application resources typically reference git repositories.
	// They can also reference Helm charts, and in such cases, use the same
	// repository field references a REGISTRY URL. So it seems a bit awkward here,
	// but we're correct to call e.argoDB.GetRepository to look for REGISTRY
	// credentials.
	repo, err := e.argoDB.GetRepository(ctx, registryURL)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error getting Argo CD Repository (Secret) for Helm chart registry %q",
			registryURL,
		)
	}
	if repo.Type == repoTypeHelm {
		creds.Username = repo.Username
		creds.Password = repo.Password
	}
	if creds.Password == "" {
		// We didn't find any creds yet, so keep looking
		var repoCreds *argocd.RepoCreds
		repoCreds, err = e.argoDB.GetRepositoryCredentials(ctx, registryURL)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error getting Repository Credentials (Secret) for Helm chart "+
					"registry %q",
				registryURL,
			)
		}
		if repoCreds != nil && repoCreds.Type == repoTypeHelm {
			creds.Username = repoCreds.Username
			creds.Password = repoCreds.Password
		}
	}

	// We didn't find any creds, so we're done.
	if creds.Password == "" {
		return nil, nil
	}

	return &creds, nil
}

func (e *environmentReconciler) getLatestCharts(
	ctx context.Context,
	env *api.Environment,
) ([]api.Chart, error) {
	if env.Spec.Subscriptions == nil ||
		env.Spec.Subscriptions.Repos == nil ||
		len(env.Spec.Subscriptions.Repos.Charts) == 0 {
		return nil, nil
	}

	logger := e.logger.WithFields(log.Fields{
		"environment": env.Name,
		"namespace":   env.Namespace,
	})

	charts := make([]api.Chart, len(env.Spec.Subscriptions.Repos.Charts))

	for i, sub := range env.Spec.Subscriptions.Repos.Charts {
		imgLogger := logger.WithFields(log.Fields{
			"registry": sub.RegistryURL,
			"chart":    sub.Name,
		})

		creds, err := e.getChartRegistryCredentialsFn(ctx, sub.RegistryURL)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error getting credentials for chart registry %q",
				sub.RegistryURL,
			)
		}
		imgLogger.Debug("acquired credentials for chart registry/repository")

		vers, err := helm.GetLatestChartVersion(
			ctx,
			sub.RegistryURL,
			sub.Name,
			sub.SemverConstraint,
			creds,
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
