package builtin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// outPathIsFile returns true if the output path contains a YAML extension.
// Otherwise, the output path is considered to target a directory where the
// rendered manifest will be written to.
func outPathIsFile(cfg builtin.HelmTemplateConfig) bool {
	ext := filepath.Ext(cfg.OutPath)
	return ext == ".yaml" || ext == ".yml"
}

// helmTemplateRunner is an implementation of the promotion.StepRunner interface
// that renders a Helm chart.
type helmTemplateRunner struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
}

// newHelmTemplateRunner returns an implementation of the promotion.StepRunner
// interface that renders a Helm chart.
func newHelmTemplateRunner(credsDB credentials.Database) promotion.StepRunner {
	r := &helmTemplateRunner{
		credsDB: credsDB,
	}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (h *helmTemplateRunner) Name() string {
	return "helm-template"
}

// Run implements the promotion.StepRunner interface.
func (h *helmTemplateRunner) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	failure := promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}

	// Validate the configuration against the JSON Schema
	if err := validate(
		h.schemaLoader,
		gojsonschema.NewGoLoader(stepCtx.Config),
		h.Name(),
	); err != nil {
		return failure, err
	}

	// Convert the configuration into a typed struct
	cfg, err := promotion.ConfigToStruct[builtin.HelmTemplateConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", h.Name(), err)
	}

	return h.run(ctx, stepCtx, cfg)
}

// GetFileTreeString returns a formatted string representing the file tree under the root
func GetFileTreeString(root string) (string, error) {
	var builder strings.Builder

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// Indentation based on depth
		depth := strings.Count(relPath, string(filepath.Separator))
		indent := strings.Repeat("  ", depth)
		builder.WriteString(fmt.Sprintf("%s%s\n", indent, d.Name()))
		return nil
	})

	if err != nil {
		return "", err
	}

	return builder.String(), nil
}

func (h *helmTemplateRunner) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.HelmTemplateConfig,
) (promotion.StepResult, error) {
	composedValues, err := h.composeValues(stepCtx.WorkDir, cfg)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to compose values: %w", err)
	}

	helmHome, err := os.MkdirTemp("", "helm-template-")
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to load chart from %q: %w", cfg.Path, err)
	}

	// if err = h.checkDependencies(chartRequested); err != nil {
	// 	return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
	// 		fmt.Errorf("missing chart dependencies: %w", err)
	// }

	absOutPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to join path %q: %w", cfg.OutPath, err)
	}

	helmHome, err := os.MkdirTemp("", "helm-template-")
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("failed to create temporary Helm home directory: %w", err)
	}
	defer os.RemoveAll(helmHome)

	if err := h.buildDependencies(ctx, stepCtx, helmHome, absOutPath); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}

	helmHome, err := os.MkdirTemp("", "helm-template-")
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to initialize Helm action config: %w", err)
	}

	rls, err := install.RunWithContext(ctx, chartRequested, composedValues)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to render chart: %w", err)
	}

	if err = h.writeOutput(cfg, rls, absOutPath); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("failed to write rendered chart: %w", err)
	}
	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
	treeStr, _ := GetFileTreeString(absChartPath)

	logging.LoggerFromContext(ctx).Info(treeStr, "file tree")

	return promotion.StepResult{Status: kargoapi.PromotionPhaseSucceeded}, nil
}

func (h *helmTemplateRunner) setupDependencyRepositories(
	ctx context.Context,
	credentialsDB credentials.Database,
	registryClient *registry.Client,
	repositoryFile *repo.File,
	project string,
	dependencies []chartDependency,
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

func (h *helmTemplateRunner) downloadRepositoryIndexes(
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

func (h *helmTemplateRunner) validateFileDependency(workDir, chartPath, dependencyPath string) error {
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

// buildDependencies builds the dependencies for the given chart
func (h *helmTemplateRunner) buildDependencies(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	helmHome, chartPath string,
) error {
	registryClient, err := helm.NewRegistryClient(helmHome)
	if err != nil {
		return fmt.Errorf("failed to create Helm registry client: %w", err)
	}

	chartFilePath := filepath.Join(chartPath, "Chart.yaml")
	chartDependencies, err := readChartDependencies(chartFilePath)
	if err != nil {
		return fmt.Errorf("failed to load chart dependencies from %q: %w", chartFilePath, err)
	}

	repositoryFile := repo.NewFile()

	for _, dep := range chartDependencies {
		if strings.HasPrefix(dep.Repository, "file://") {
			depPath := filepath.FromSlash(strings.TrimPrefix(dep.Repository, "file://"))
			if err = h.validateFileDependency(stepCtx.WorkDir, chartPath, depPath); err != nil {
				return fmt.Errorf("invalid dependency %q: %w", dep.Repository, err)
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
		return err
	}

	repositoryConfig := filepath.Join(helmHome, "repositories.yaml")
	if err = repositoryFile.WriteFile(repositoryConfig, 0o600); err != nil {
		return fmt.Errorf("failed to write Helm repositories file: %w", err)
	}

	// Check if Chart.lock exists and create a backup if it does
	lockFile := filepath.Join(chartPath, "Chart.lock")
	bakLockFile := fmt.Sprintf("%s.%s.bak", lockFile, time.Now().Format("20060102150405"))
	lockFileExists := false // Track whether the original Chart.lock file exists

	if _, err = os.Lstat(lockFile); err == nil {
		lockFileExists = true // Mark that the Chart.lock file exists
		if err = backupFile(lockFile, bakLockFile); err != nil {
			return fmt.Errorf("failed to backup Chart.lock: %w", err)
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
		return fmt.Errorf("failed to check Chart.lock: %w", err)
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
		return err
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
	if err = manager.Build(); err != nil {
		return fmt.Errorf("failed to build chart dependencies: %w", err)
	}

	// Read versions from both Chart.lock files after the update.
	//
	// NB: We rely on the lock file to determine the version changes because
	// the dependency update process may change the version of a dependency
	// without updating the Chart.yaml. For example, because a new version is
	// available in the repository for a dependency that has a version range
	// specified in the Chart.yaml.
	initialVersions := map[string]string{}
	if lockFileExists { // Only read the original Chart.lock if it existed
		initialVersions, err = readChartLock(bakLockFile)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf(
				"failed to read original chart lock file: %w", sanitizePathError(err, stepCtx.WorkDir),
			)
		}
	}

	updatedVersions, err := readChartLock(lockFile)
	if err != nil {
		return fmt.Errorf(
			"failed to read updated chart lock file: %w",
			sanitizePathError(err, stepCtx.WorkDir),
		)
	}

	// Compare the versions to determine if any changes occurred
	changes := compareChartVersions(initialVersions, updatedVersions)

	// If no versions changed, restore the original Chart.lock (only if it existed)
	if len(changes) == 0 && lockFileExists {
		if err = os.Rename(bakLockFile, lockFile); err != nil {
			return fmt.Errorf(
				"failed to restore original Chart.lock: %w", sanitizePathError(err, stepCtx.WorkDir),
			)
		}
	}
	return nil
}

// composeValues composes the values from the given values files and set values.
// It merges the value files in the order they are provided. Set values are merged
// into last as overrides
func (h *helmTemplateRunner) composeValues(
	workDir string,
	cfg builtin.HelmTemplateConfig,
) (map[string]any, error) {
	valueOpts := &values.Options{}
	for _, p := range cfg.ValuesFiles {
		absValuesPath, err := securejoin.SecureJoin(workDir, p)
		if err != nil {
			return nil, fmt.Errorf("failed to join path %q: %w", p, err)
		}
		valueOpts.ValueFiles = append(valueOpts.ValueFiles, absValuesPath)
	}
	setValueStrVal := make([]string, len(cfg.SetValues))
	for i, setValue := range cfg.SetValues {
		setValueStrVal[i] = fmt.Sprintf("%s=%s", setValue.Key, setValue.Value)
	}
	valueOpts.Values = append(valueOpts.Values, setValueStrVal...)
	return valueOpts.MergeValues(nil)
}

// newInstallAction creates a new Helm install action with the given
// configuration. It sets the action to dry-run mode and client-only mode,
// meaning that it will not install the chart, but only render the manifest.
func (h *helmTemplateRunner) newInstallAction(
	cfg builtin.HelmTemplateConfig,
	project, absOutPath string,
) (*action.Install, error) {
	client := action.NewInstall(&action.Configuration{})

	client.DryRun = true
	client.DryRunOption = "client"
	client.Replace = true
	client.ClientOnly = true
	client.ReleaseName = defaultValue(cfg.ReleaseName, "release-name")
	client.UseReleaseName = cfg.UseReleaseName
	client.Namespace = defaultValue(cfg.Namespace, project)
	client.IncludeCRDs = cfg.IncludeCRDs
	client.APIVersions = cfg.APIVersions
	client.DisableHooks = cfg.DisableHooks

	// If the output path does not have a YAML extension, it is considered a
	// directory where the manifest will be written to.
	if !outPathIsFile(cfg) {
		client.OutputDir = absOutPath
	}

	if cfg.KubeVersion != "" {
		kubeVersion, err := chartutil.ParseKubeVersion(cfg.KubeVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Kubernetes version %q: %w", cfg.KubeVersion, err)
		}
		client.KubeVersion = kubeVersion
	}

	return client, nil
}

// loadChart loads the chart from the given path.
func (h *helmTemplateRunner) loadChart(workDir, relPath string) (*chart.Chart, error) {
	absChartPath, err := securejoin.SecureJoin(workDir, relPath)
	if err != nil {
		return nil, fmt.Errorf("failed to join relPath %q: %w", relPath, err)
	}
	return loader.Load(absChartPath)
}

// checkDependencies checks if the chart has all its dependencies.
func (h *helmTemplateRunner) checkDependencies(chartRequested *chart.Chart) error {
	if req := chartRequested.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			return err
		}
	}
	return nil
}

// writeOutput writes the rendered manifest to the output path.
func (h *helmTemplateRunner) writeOutput(cfg builtin.HelmTemplateConfig, rls *release.Release, outPath string) error {
	var (
		manifests     bytes.Buffer
		outPathIsFile = outPathIsFile(cfg)
	)

	if outPathIsFile {
		_, _ = fmt.Fprintln(&manifests, strings.TrimSpace(rls.Manifest))
	}

	if !cfg.DisableHooks {
		for _, h := range rls.Hooks {
			if cfg.SkipTests && isTestHook(h) {
				continue
			}

			if outPathIsFile {
				_, _ = fmt.Fprintf(&manifests, "---\n# Source: %s\n%s\n", h.Path, h.Manifest)
				continue
			}

			exists := true
			outDir := outPath
			if cfg.UseReleaseName {
				outDir = filepath.Join(outDir, cfg.ReleaseName)
			}
			if _, err := os.Stat(filepath.Join(outDir, h.Path)); err != nil {
				if !os.IsNotExist(err) {
					return fmt.Errorf("failed to check if file %q exists: %w", h.Path, err)
				}
				exists = false
			}

			if err := writeToHelmFile(outPath, h.Path, h.Manifest, exists); err != nil {
				return fmt.Errorf("failed to write hook %q: %w", h.Path, err)
			}
		}
	}

	if !outPathIsFile {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o700); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", cfg.OutPath, err)
	}
	return os.WriteFile(outPath, manifests.Bytes(), 0o600)
}

// defaultValue returns the value if it is not zero or empty, otherwise it
// returns the default value.
func defaultValue[T any](value, defaultValue T) T {
	if v := reflect.ValueOf(value); !v.IsValid() || v.IsZero() || (v.Kind() == reflect.Slice && v.Len() == 0) {
		return defaultValue
	}
	return value
}

// isTestHook returns true if the hook is a test hook.
func isTestHook(h *release.Hook) bool {
	for _, e := range h.Events {
		if e == release.HookTest {
			return true
		}
	}
	return false
}

// The logic below is directly derived from the Helm source code:
// https://github.com/helm/helm/blob/b2286c4caabdfdcf2baaecb42819db9d38c04597/cmd/helm/template.go#L222
// Licensed under the Apache License 2.0.

// writeToHelmFile writes the given data to the output directory with the given
// name. If the appendMode flag is set to true, the data is appended to the file.
func writeToHelmFile(outputDir string, name string, data string, appendMode bool) (err error) {
	outfileName := strings.Join([]string{outputDir, name}, string(filepath.Separator))

	if err = ensureDirectoryForHelmFile(outfileName); err != nil {
		return err
	}

	f, err := createOrOpenHelmFile(outfileName, appendMode)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if _, err = f.WriteString(fmt.Sprintf("---\n# Source: %s\n%s\n", name, data)); err != nil {
		return err
	}

	return nil
}

// createOrOpenHelmFile creates or opens the file with the given name. If the
// append flag is set to true, the file is opened in append mode.
func createOrOpenHelmFile(filename string, appendMode bool) (*os.File, error) {
	if appendMode {
		return os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0o600)
	}
	return os.Create(filename)
}

// ensureDirectoryForHelmFile ensures that the directory for the given file
// exists. If the directory does not exist, it is created with the default
// permissions.
func ensureDirectoryForHelmFile(file string) error {
	baseDir := path.Dir(file)
	if _, err := os.Stat(baseDir); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.MkdirAll(baseDir, 0o755)
}
