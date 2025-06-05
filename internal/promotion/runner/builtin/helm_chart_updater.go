package builtin

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/repo"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
	fs2 "github.com/akuity/kargo/internal/io/fs"
	"github.com/akuity/kargo/internal/logging"
	intyaml "github.com/akuity/kargo/internal/yaml"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// helmChartUpdater is an implementation of the promotion.StepRunner interface
// that updates the dependencies of a Helm chart.
type helmChartUpdater struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
}

// newHelmChartUpdater returns an implementation of the promotion.StepRunner
// interface that updates the dependencies of a Helm chart.
func newHelmChartUpdater(credsDB credentials.Database) promotion.StepRunner {
	r := &helmChartUpdater{
		credsDB: credsDB,
	}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (h *helmChartUpdater) Name() string {
	return "helm-update-chart"
}

// Run implements the promotion.StepRunner interface.
func (h *helmChartUpdater) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	failure := promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}

	if err := h.validate(stepCtx.Config); err != nil {
		return failure, err
	}

	// Convert the configuration into a typed struct
	cfg, err := promotion.ConfigToStruct[builtin.HelmUpdateChartConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", h.Name(), err)
	}

	return h.run(ctx, stepCtx, cfg)
}

// validate validates helmChartUpdater configuration against a JSON schema.
func (h *helmChartUpdater) validate(cfg promotion.Config) error {
	return validate(h.schemaLoader, gojsonschema.NewGoLoader(cfg), h.Name())
}

func (h *helmChartUpdater) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.HelmUpdateChartConfig,
) (promotion.StepResult, error) {
	absChartPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to join path %q: %w", cfg.Path, err)
	}

	chartFilePath := filepath.Join(absChartPath, "Chart.yaml")
	chartDependencies, err := helm.GetChartDependencies(chartFilePath)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusErrored,
		}, fmt.Errorf("failed to load chart dependencies from %q: %w", chartFilePath, err)
	}

	changes, err := h.processChartUpdates(cfg, chartDependencies)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	if err = intyaml.SetValuesInFile(chartFilePath, changes); err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusErrored,
		}, fmt.Errorf("failed to update chart dependencies in %q: %w", chartFilePath, err)
	}

	helmHome, err := os.MkdirTemp("", "helm-chart-update-")
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to create temporary Helm home directory: %w", err)
	}
	defer os.RemoveAll(helmHome)

	newVersions, err := h.updateDependencies(ctx, stepCtx, helmHome, absChartPath, chartDependencies)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	result := promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}
	if commitMsg := h.generateCommitMessage(cfg.Path, newVersions); commitMsg != "" {
		result.Output = map[string]any{
			"commitMessage": commitMsg,
		}
	}
	return result, nil
}

func (h *helmChartUpdater) processChartUpdates(
	cfg builtin.HelmUpdateChartConfig,
	chartDependencies []helm.ChartDependency,
) ([]intyaml.Update, error) {
	updates := make([]intyaml.Update, len(cfg.Charts))
	for i, update := range cfg.Charts {
		var updateUsed bool
		for j, dep := range chartDependencies {
			if dep.Repository == update.Repository && dep.Name == update.Name {
				updates[i] = intyaml.Update{
					Key:   fmt.Sprintf("dependencies.%d.version", j),
					Value: update.Version,
				}
				updateUsed = true
				break
			}
		}
		if !updateUsed {
			return nil, fmt.Errorf(
				"no dependency in Chart.yaml matched update with repository %s and name %q",
				update.Repository, update.Name,
			)
		}
	}
	return updates, nil
}

func (h *helmChartUpdater) updateDependencies(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	helmHome, chartPath string,
	chartDependencies []helm.ChartDependency,
) (map[string]string, error) {
	registryClient, err := helm.NewRegistryClient(helmHome)
	if err != nil {
		return nil, fmt.Errorf("failed to create Helm registry client: %w", err)
	}

	repositoryFile := repo.NewFile()

	for _, dep := range chartDependencies {
		if strings.HasPrefix(dep.Repository, "file://") {
			depPath := filepath.FromSlash(strings.TrimPrefix(dep.Repository, "file://"))
			if err = h.validateFileDependency(stepCtx.WorkDir, chartPath, depPath); err != nil {
				return nil, fmt.Errorf("invalid dependency %q: %w", dep.Repository, err)
			}
		}
	}

	if err = h.setupDependencyRepositories(
		ctx,
		h.credsDB,
		registryClient,
		repositoryFile,
		stepCtx.Project,
		chartDependencies,
	); err != nil {
		return nil, err
	}

	repositoryConfig := filepath.Join(helmHome, "repositories.yaml")
	if err = repositoryFile.WriteFile(repositoryConfig, 0o600); err != nil {
		return nil, fmt.Errorf("failed to write Helm repositories file: %w", err)
	}

	// Check if Chart.lock exists and create a backup if it does
	lockFile := filepath.Join(chartPath, "Chart.lock")
	bakLockFile := fmt.Sprintf("%s.%s.bak", lockFile, time.Now().Format("20060102150405"))
	if _, err = os.Lstat(lockFile); err == nil {
		if err = fs2.CopyFile(lockFile, bakLockFile); err != nil {
			return nil, fmt.Errorf("failed to backup Chart.lock: %w", err)
		}

		// Ensure backup file is deleted at the end
		defer func() {
			if removeErr := os.Remove(bakLockFile); removeErr != nil && !os.IsNotExist(removeErr) {
				logging.LoggerFromContext(ctx).Error(
					fs2.SanitizePathError(removeErr, stepCtx.WorkDir),
					"failed to remove backup of Chart.lock",
				)
			}
		}()
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to check Chart.lock: %w", err)
	}

	// Prepare the environment settings for Helm
	env := &cli.EnvSettings{
		RepositoryConfig: repositoryConfig,
		RepositoryCache:  filepath.Join(helmHome, "cache"),
	}

	// Download the repository indexes. This is necessary to ensure that the
	// cache is properly populated, as otherwise the download manager will
	// attempt to download the repository indexes to the default cache path
	// instead of using the cache path set in the environment settings.
	if err = h.downloadRepositoryIndexes(repositoryFile.Repositories, env); err != nil {
		return nil, err
	}

	// Run the dependency update
	manager := downloader.Manager{
		Out:              io.Discard,
		ChartPath:        chartPath,
		Verify:           downloader.VerifyNever,
		SkipUpdate:       false,
		Getters:          getter.All(env),
		RegistryClient:   registryClient,
		RepositoryConfig: env.RepositoryConfig,
		RepositoryCache:  env.RepositoryCache,
	}
	if err = manager.Update(); err != nil {
		return nil, fmt.Errorf("failed to update chart dependencies: %w", err)
	}

	// Read versions from both Chart.lock files after the update.
	//
	// NB: We rely on the lock file to determine the version changes because
	// the dependency update process may change the version of a dependency
	// without updating the Chart.yaml. For example, because a new version is
	// available in the repository for a dependency that has a version range
	// specified in the Chart.yaml.
	initialVersions, err := helm.GetChartDependencies(bakLockFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf(
			"failed to read original chart lock file: %w", fs2.SanitizePathError(err, stepCtx.WorkDir),
		)
	}
	updatedVersions, err := helm.GetChartDependencies(lockFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf(
			"failed to read updated chart lock file: %w",
			fs2.SanitizePathError(err, stepCtx.WorkDir),
		)
	}

	// Compare the versions to determine if any changes occurred
	changes := compareChartVersions(initialVersions, updatedVersions)

	// If no versions changed, restore the original Chart.lock
	if len(changes) == 0 {
		if err = os.Rename(bakLockFile, lockFile); err != nil {
			return nil, fmt.Errorf(
				"failed to restore original Chart.lock: %w", fs2.SanitizePathError(err, stepCtx.WorkDir),
			)
		}
	}
	return changes, nil
}

func (h *helmChartUpdater) validateFileDependency(workDir, chartPath, dependencyPath string) error {
	if filepath.IsAbs(dependencyPath) {
		return errors.New("dependency path must be relative")
	}

	// Resolve the dependency path relative to the chart directory
	dependencyPath = filepath.Join(chartPath, dependencyPath)

	// Check if the resolved dependency path is within the work directory
	resolvedDependencyPath, err := filepath.EvalSymlinks(dependencyPath)
	if err != nil {
		return fmt.Errorf("failed to resolve dependency path: %w", fs2.SanitizePathError(err, workDir))
	}
	if !fs2.IsSubPath(workDir, resolvedDependencyPath) {
		return errors.New("dependency path is outside of the work directory")
	}

	// Recursively check for symlinks that go outside the work directory,
	// as Helm follows symlinks when packaging charts
	return fs2.ValidateSymlinks(workDir, dependencyPath, 100)
}

func (h *helmChartUpdater) setupDependencyRepositories(
	ctx context.Context,
	credentialsDB credentials.Database,
	registryClient *registry.Client,
	repositoryFile *repo.File,
	project string,
	dependencies []helm.ChartDependency,
) error {
	for _, dep := range dependencies {
		switch {
		case strings.HasPrefix(dep.Repository, "file://"):
			continue
		case strings.HasPrefix(dep.Repository, "http://"):
			entry := &repo.Entry{
				Name: nameForRepositoryURL(dep.Repository),
				URL:  dep.Repository,
			}
			repositoryFile.Update(entry)
		case strings.HasPrefix(dep.Repository, "https://"):
			entry := &repo.Entry{
				Name: nameForRepositoryURL(dep.Repository),
				URL:  dep.Repository,
			}

			creds, err := credentialsDB.Get(ctx, project, credentials.TypeHelm, dep.Repository)
			if err != nil {
				return fmt.Errorf("failed to obtain credentials for chart repository %q: %w", dep.Repository, err)
			}
			if creds != nil {
				entry.Username = creds.Username
				entry.Password = creds.Password
			}

			repositoryFile.Update(entry)
		case strings.HasPrefix(dep.Repository, "oci://"):
			credURL := "oci://" + path.Join(helm.NormalizeChartRepositoryURL(dep.Repository), dep.Name)
			creds, err := credentialsDB.Get(ctx, project, credentials.TypeHelm, credURL)
			if err != nil {
				return fmt.Errorf("failed to obtain credentials for chart repository %q: %w", dep.Repository, err)
			}
			if creds != nil {
				if err = registryClient.Login(
					strings.TrimPrefix(dep.Repository, "oci://"),
					registry.LoginOptBasicAuth(creds.Username, creds.Password),
				); err != nil {
					return fmt.Errorf("failed to authenticate with chart repository %q: %w", dep.Repository, err)
				}
			}
		}
	}
	return nil
}

func (h *helmChartUpdater) downloadRepositoryIndexes(
	repositories []*repo.Entry,
	env *cli.EnvSettings,
) error {
	for _, entry := range repositories {
		cr, err := repo.NewChartRepository(entry, getter.All(env))
		if err != nil {
			return fmt.Errorf("failed to create chart repository for %q: %w", entry.URL, err)
		}

		// NB: Explicitly overwrite the cache path to avoid using the default
		// cache path from the environment variables. Without this, the download
		// manager will not find the repository index files in the cache, and
		// will attempt to download them again (to the default cache path).
		// I.e. without this, the download manager will not use the isolated
		// cache.
		cr.CachePath = env.RepositoryCache

		if _, err = cr.DownloadIndexFile(); err != nil {
			return fmt.Errorf("failed to download repository index for %q: %w", entry.URL, err)
		}
	}
	return nil
}

func (h *helmChartUpdater) generateCommitMessage(chartPath string, newVersions map[string]string) string {
	if len(newVersions) == 0 {
		return ""
	}

	var commitMsg strings.Builder
	_, _ = commitMsg.WriteString("Updated chart dependencies for ")
	_, _ = commitMsg.WriteString(chartPath)
	_, _ = commitMsg.WriteString("\n")
	for name, change := range newVersions {
		if change == "" {
			change = "removed"
		}
		_, _ = commitMsg.WriteString(fmt.Sprintf("\n- %s: %s", name, change))
	}
	return commitMsg.String()
}

func normalizeChartReference(repoURL, chartName string) (string, string) {
	if strings.HasPrefix(repoURL, "oci://") {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(repoURL, "/"), chartName), ""
	}
	return repoURL, chartName
}

func compareChartVersions(before, after []helm.ChartDependency) map[string]string {
	beforeMap := make(map[string]string, len(before))
	for _, dep := range before {
		beforeMap[dep.Name] = dep.Version
	}

	changes := make(map[string]string)
	for _, dep := range after {
		if oldVersion, exists := beforeMap[dep.Name]; exists {
			if oldVersion != dep.Version {
				changes[dep.Name] = oldVersion + " -> " + dep.Version
			}
			// Remove the dependency from before map to track allow remaining
			// items to be counted as removed
			delete(beforeMap, dep.Name)
		} else {
			changes[dep.Name] = dep.Version
		}
	}

	// Handle any removed dependencies which are still listed in before map
	for name := range beforeMap {
		changes[name] = ""
	}
	return changes
}

// nameForRepositoryURL generates an SHA-256 hash of the repository URL to use
// as the name for the repository in the Helm repository cache.
//
// The repository URL is normalized before hashing using the same logic as
// urlutil.Equal from Helm, which is used to compare repository URLs in the
// download manager when looking at cached repository indexes to find the
// correct chart URL.
func nameForRepositoryURL(repoURL string) string {
	u, err := url.Parse(repoURL)
	if err != nil {
		repoURL = filepath.Clean(repoURL)
	}

	if u != nil {
		if u.Path == "" {
			u.Path = "/"
		}
		u.Path = filepath.Clean(u.Path)
		repoURL = u.String()
	}

	return fmt.Sprintf("%x", sha256.Sum256([]byte(repoURL)))
}
