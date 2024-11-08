package directives

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	yaml "sigs.k8s.io/yaml/goyaml.v3"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/freight"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/logging"
	intyaml "github.com/akuity/kargo/internal/yaml"
)

func init() {
	builtins.RegisterPromotionStepRunner(
		newHelmChartUpdater(),
		&StepRunnerPermissions{
			AllowKargoClient:   true,
			AllowCredentialsDB: true,
		},
	)
}

// helmChartUpdater is an implementation of the PromotionStepRunner interface
// that updates the dependencies of a Helm chart.
type helmChartUpdater struct {
	schemaLoader gojsonschema.JSONLoader
}

// newHelmChartUpdater returns an implementation of the PromotionStepRunner
// interface that updates the dependencies of a Helm chart.
func newHelmChartUpdater() PromotionStepRunner {
	r := &helmChartUpdater{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the PromotionStepRunner interface.
func (h *helmChartUpdater) Name() string {
	return "helm-update-chart"
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (h *helmChartUpdater) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	failure := PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}

	// Validate the configuration against the JSON Schema
	if err := validate(
		h.schemaLoader,
		gojsonschema.NewGoLoader(stepCtx.Config),
		h.Name(),
	); err != nil {
		return failure, err
	}

	// Convert the configuration into a typed struct
	cfg, err := ConfigToStruct[HelmUpdateChartConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", h.Name(), err)
	}

	return h.runPromotionStep(ctx, stepCtx, cfg)
}

func (h *helmChartUpdater) runPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	cfg HelmUpdateChartConfig,
) (PromotionStepResult, error) {
	absChartPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("failed to join path %q: %w", cfg.Path, err)
	}

	chartFilePath := filepath.Join(absChartPath, "Chart.yaml")
	chartDependencies, err := readChartDependencies(chartFilePath)
	if err != nil {
		return PromotionStepResult{
			Status: kargoapi.PromotionPhaseErrored,
		}, fmt.Errorf("failed to load chart dependencies from %q: %w", chartFilePath, err)
	}

	changes, err := h.processChartUpdates(ctx, stepCtx, cfg, chartDependencies)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}

	if err = intyaml.SetStringsInFile(chartFilePath, changes); err != nil {
		return PromotionStepResult{
			Status: kargoapi.PromotionPhaseErrored,
		}, fmt.Errorf("failed to update chart dependencies in %q: %w", chartFilePath, err)
	}

	helmHome, err := os.MkdirTemp("", "helm-chart-update-")
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("failed to create temporary Helm home directory: %w", err)
	}
	defer os.RemoveAll(helmHome)

	newVersions, err := h.updateDependencies(ctx, stepCtx, helmHome, absChartPath, chartDependencies)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}

	result := PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}
	if commitMsg := h.generateCommitMessage(cfg.Path, newVersions); commitMsg != "" {
		result.Output = map[string]any{
			"commitMessage": commitMsg,
		}
	}
	return result, nil
}

func (h *helmChartUpdater) processChartUpdates(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	cfg HelmUpdateChartConfig,
	chartDependencies []chartDependency,
) (map[string]string, error) {
	changes := make(map[string]string)
	for _, update := range cfg.Charts {
		repoURL, chartName := normalizeChartReference(update.Repository, update.Name)

		var desiredOrigin *kargoapi.FreightOrigin
		if update.FromOrigin != nil {
			desiredOrigin = &kargoapi.FreightOrigin{
				Kind: kargoapi.FreightOriginKind(update.FromOrigin.Kind),
				Name: update.FromOrigin.Name,
			}
		}

		chart, err := freight.FindChart(
			ctx,
			stepCtx.KargoClient,
			stepCtx.Project,
			stepCtx.FreightRequests,
			desiredOrigin,
			stepCtx.Freight.References(),
			repoURL,
			chartName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to find chart: %w", err)
		}

		var updateUsed bool
		for i, dep := range chartDependencies {
			if dep.Repository == update.Repository && dep.Name == update.Name {
				changes[fmt.Sprintf("dependencies.%d.version", i)] = chart.Version
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
	return changes, nil
}

func (h *helmChartUpdater) updateDependencies(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	helmHome, chartPath string,
	chartDependencies []chartDependency,
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

	if err = h.loadDependencyCredentials(
		ctx,
		stepCtx.CredentialsDB,
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
		if err = backupFile(lockFile, bakLockFile); err != nil {
			return nil, fmt.Errorf("failed to backup Chart.lock: %w", err)
		}

		// Ensure backup file is deleted at the end
		defer func() {
			if removeErr := os.Remove(bakLockFile); removeErr != nil && !os.IsNotExist(removeErr) {
				logging.LoggerFromContext(ctx).Error(
					sanitizePathError(removeErr, stepCtx.WorkDir),
					"failed to remove backup of Chart.lock",
				)
			}
		}()
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to check Chart.lock: %w", err)
	}

	// Run the dependency update
	env := &cli.EnvSettings{
		RepositoryConfig: repositoryConfig,
		RepositoryCache:  filepath.Join(helmHome, "cache"),
	}
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
	initialVersions, err := readChartLock(bakLockFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf(
			"failed to read original chart lock file: %w", sanitizePathError(err, stepCtx.WorkDir),
		)
	}
	updatedVersions, err := readChartLock(lockFile)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read updated chart lock file: %w",
			sanitizePathError(err, stepCtx.WorkDir),
		)
	}

	// Compare the versions to determine if any changes occurred
	changes := compareChartVersions(initialVersions, updatedVersions)

	// If no versions changed, restore the original Chart.lock
	if len(changes) == 0 {
		if err = os.Rename(bakLockFile, lockFile); err != nil {
			return nil, fmt.Errorf(
				"failed to restore original Chart.lock: %w", sanitizePathError(err, stepCtx.WorkDir),
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
		return fmt.Errorf("failed to resolve dependency path: %w", sanitizePathError(err, workDir))
	}
	if !isSubPath(workDir, resolvedDependencyPath) {
		return errors.New("dependency path is outside of the work directory")
	}

	// Recursively check for symlinks that go outside the work directory,
	// as Helm follows symlinks when packaging charts
	visited := make(map[string]struct{})
	return checkSymlinks(workDir, dependencyPath, visited, 0, 100)
}

func (h *helmChartUpdater) loadDependencyCredentials(
	ctx context.Context,
	credentialsDB credentials.Database,
	registryClient *registry.Client,
	repositoryFile *repo.File,
	project string,
	dependencies []chartDependency,
) error {
	for _, dep := range dependencies {
		var credType credentials.Type
		var credURL string

		switch {
		case strings.HasPrefix(dep.Repository, "https://"):
			credType = credentials.TypeHelm
			credURL = dep.Repository
		case strings.HasPrefix(dep.Repository, "oci://"):
			credType = credentials.TypeHelm
			credURL = "oci://" + path.Join(helm.NormalizeChartRepositoryURL(dep.Repository), dep.Name)
		default:
			continue
		}

		creds, ok, err := credentialsDB.Get(ctx, project, credType, credURL)
		if err != nil {
			return fmt.Errorf("failed to obtain credentials for chart repository %q: %w", dep.Repository, err)
		}
		if !ok {
			continue
		}

		if strings.HasPrefix(dep.Repository, "https://") {
			repositoryFile.Add(&repo.Entry{
				Name:     dep.Name,
				URL:      dep.Repository,
				Username: creds.Username,
				Password: creds.Password,
			})
		} else {
			if err = registryClient.Login(
				strings.TrimPrefix(dep.Repository, "oci://"),
				registry.LoginOptBasicAuth(creds.Username, creds.Password),
			); err != nil {
				return fmt.Errorf("failed to authenticate with chart repository %q: %w", dep.Repository, err)
			}
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

type chartDependency struct {
	Repository string `json:"repository,omitempty"`
	Name       string `json:"name,omitempty"`
	Version    string `json:"version,omitempty"`
}

func readChartDependencies(chartFilePath string) ([]chartDependency, error) {
	b, err := os.ReadFile(chartFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", chartFilePath, err)
	}

	var chartMeta struct {
		Dependencies []chartDependency `json:"dependencies,omitempty"`
	}
	if err := yaml.Unmarshal(b, &chartMeta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %q: %w", chartFilePath, err)
	}

	return chartMeta.Dependencies, nil
}

func readChartLock(src string) (map[string]string, error) {
	data, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("failed to read Chart.lock: %w", err)
	}

	var lockContent struct {
		Dependencies []chartDependency `yaml:"dependencies"`
	}
	if err = yaml.Unmarshal(data, &lockContent); err != nil {
		return nil, fmt.Errorf("failed to parse Chart.lock: %w", err)
	}

	versions := make(map[string]string)
	for _, dep := range lockContent.Dependencies {
		versions[dep.Name] = dep.Version
	}
	return versions, nil
}

func compareChartVersions(before, after map[string]string) map[string]string {
	changes := make(map[string]string)

	for name, newVersion := range after {
		if oldVersion, ok := before[name]; !ok || oldVersion != newVersion {
			if oldVersion == "" {
				changes[name] = newVersion
			} else {
				changes[name] = fmt.Sprintf("%s -> %s", oldVersion, newVersion)
			}
		}
	}

	for name := range before {
		if _, ok := after[name]; !ok {
			changes[name] = ""
		}
	}

	return changes
}

// checkSymlinks recursively checks for symlinks that point outside the root path
// and avoids infinite recursion by using a single map of visited directories
// (absolute paths). The depth parameter is used to limit the recursion depth,
// with a value of -1 indicating no limit.
func checkSymlinks(root, dir string, visited map[string]struct{}, depth, maxDepth int) error {
	// Get the absolute path of the current directory
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for dir: %v", err)
	}

	// Check if we've already visited this directory
	if _, ok := visited[absDir]; ok {
		// Skip it to avoid infinite recursion or redundant visits
		return nil
	}

	// Mark this directory as visited only when starting to process it
	visited[absDir] = struct{}{}

	// Check if the recursion depth is within the limit
	if maxDepth >= 0 && depth >= maxDepth {
		return fmt.Errorf("maximum recursion depth exceeded")
	}

	// Open the directory
	dirEntries, err := os.ReadDir(absDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", sanitizePathError(err, root))
	}

	// Process each entry in the directory
	for _, entry := range dirEntries {
		entryPath := filepath.Join(dir, entry.Name())

		// If the entry is a symlink, resolve it
		if entry.Type()&os.ModeSymlink != 0 {
			// Resolve the symlink to its target
			target, pathErr := filepath.EvalSymlinks(entryPath)
			if pathErr != nil {
				return fmt.Errorf("failed to resolve symlink: %w", sanitizePathError(pathErr, root))
			}

			// Convert the target path to its absolute form
			absTarget, pathErr := filepath.Abs(target)
			if pathErr != nil {
				return pathErr
			}

			// Ensure the target is within the root directory
			if !isSubPath(root, absTarget) {
				return fmt.Errorf("symlink at %s points outside the path boundary", relativePath(root, entryPath))
			}

			// Recursively check the symlinked directory or file if not visited yet
			if _, ok := visited[absTarget]; !ok {
				// Check if the symlink target is a directory
				targetInfo, pathErr := os.Stat(absTarget)
				if pathErr != nil {
					return fmt.Errorf(
						"failed to stat symlink target of %s: %w",
						relativePath(root, entryPath),
						sanitizePathError(pathErr, root),
					)
				}

				if targetInfo.IsDir() {
					// Recursively call the function for the symlinked directory
					if err = checkSymlinks(root, absTarget, visited, depth+1, maxDepth); err != nil {
						return err
					}
				}

				// It's a file, no further need for recursion here
				// We still add it to the visited map to avoid redundant checks
				visited[absTarget] = struct{}{}
			}
		} else if entry.IsDir() {
			// If it's a directory, manually recurse into it
			if err = checkSymlinks(root, entryPath, visited, depth+1, maxDepth); err != nil {
				return err
			}
		}
	}

	return nil
}

// isSubPath checks if the child path is a subpath of the parent path.
func isSubPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != ".."
}

// backupFile creates a backup of the source file at the destination path.
func backupFile(src, dst string) (err error) {
	// Open the source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := srcFile.Close()
		if err == nil {
			err = closeErr
		}
	}()

	// Get file info to retrieve permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create the destination file with the same permissions
	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer func() {
		closeErr := dstFile.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(dst)
		}
	}()

	// Copy the contents
	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return nil
}
