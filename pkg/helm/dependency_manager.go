package helm

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
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/io/fs"
	"github.com/akuity/kargo/pkg/urls"
	intyaml "github.com/akuity/kargo/pkg/yaml"
)

// EphemeralDependencyManager is a Helm dependency manager that uses an
// ephemeral Helm home directory for managing chart dependencies. This manager
// is designed for temporary operations where you do not want to store any
// Helm-related data permanently, such as when working with multiple tenants in
// a single process.
type EphemeralDependencyManager struct {
	credsDB    credentials.Database
	authorizer *EphemeralAuthorizer

	helmHome string
	workDir  string
	project  string
}

// NewEphemeralDependencyManager creates a new EphemeralDependencyManager
// that uses an ephemeral Helm home directory for managing chart dependencies.
// This manager is suitable for temporary operations where you do not want to
// store any Helm-related data permanently, such as when working with multiple
// tenants in a single process.
//
// When using this manager, it is important to call the Teardown method
// to clean up the Helm home directory after you are done with it. This will
// remove the Helm home directory and all its contents, ensuring that no
// temporary data is left behind.
func NewEphemeralDependencyManager(
	credsDB credentials.Database,
	project, workDir string,
) (*EphemeralDependencyManager, error) {

	home, err := os.MkdirTemp("", "helm-home-*")
	if err != nil {
		return nil, err
	}

	return &EphemeralDependencyManager{
		credsDB:    credsDB,
		authorizer: NewEphemeralAuthorizer(),
		helmHome:   home,
		project:    project,
		workDir:    workDir,
	}, nil
}

// Teardown cleans up the EphemeralDependencyManager by removing the Helm home
// directory that was created during its initialization. This is useful for
// cleaning up resources after the dependency manager is no longer needed.
func (em *EphemeralDependencyManager) Teardown() error {
	if em.helmHome != "" {
		if err := os.RemoveAll(em.helmHome); err != nil {
			return err
		}
	}
	return nil
}

// Update updates the chart dependencies for the given chart path. It reads the
// Chart.yaml file to get the list of dependencies, validates them, and then
// processes any updates specified in the updates slice.
//
// It returns a map of changes where the keys are the names of the dependencies
// and the values are the new versions. If a dependency was removed, the value
// will be an empty string. If a dependency was updated, the value will be a
// string indicating the old and new versions in the format
// "oldVersion -> newVersion".
func (em *EphemeralDependencyManager) Update(
	ctx context.Context,
	chartPath string,
	updates ...ChartDependency,
) (map[string]string, error) {
	absChartPath, err := securejoin.SecureJoin(em.workDir, chartPath)
	if err != nil {
		return nil, err
	}

	absChartFilePath := filepath.Join(absChartPath, "Chart.yaml")
	dependencies, err := GetChartDependencies(absChartFilePath)
	if err != nil {
		return nil, fmt.Errorf("get chart dependencies: %w", err)
	}

	if err = em.validateDependencies(absChartPath, dependencies); err != nil {
		return nil, fmt.Errorf("validate dependencies: %w", err)
	}

	if len(updates) > 0 {
		if err = em.processDependencyUpdates(absChartFilePath, dependencies, updates); err != nil {
			return nil, fmt.Errorf("process dependency updates: %w", err)
		}
	}

	if err = em.setupRepositories(ctx, dependencies); err != nil {
		return nil, fmt.Errorf("setup repositories: %w", err)
	}

	return em.update(absChartPath)
}

// Build builds the chart dependencies for the given chart path. It reads the
// Chart.yaml file to get the list of dependencies, validates them, and then
// uses the Helm downloader manager to build the dependencies.
//
// Note that if the chart currently does not have a Chart.lock file, the
// build operation will create one by performing an update operation.
func (em *EphemeralDependencyManager) Build(ctx context.Context, chartPath string) error {
	absChartPath, err := securejoin.SecureJoin(em.workDir, chartPath)
	if err != nil {
		return err
	}

	absChartFilePath := filepath.Join(absChartPath, "Chart.yaml")
	dependencies, err := GetChartDependencies(absChartFilePath)
	if err != nil {
		return fmt.Errorf("get chart dependencies: %w", err)
	}

	if len(dependencies) == 0 {
		return nil
	}

	if err = em.validateDependencies(absChartPath, dependencies); err != nil {
		return fmt.Errorf("validate dependencies: %w", err)
	}

	if err = em.setupRepositories(ctx, dependencies); err != nil {
		return fmt.Errorf("setup repositories: %w", err)
	}

	return em.build(absChartPath)
}

// update performs the actual update of the chart dependencies. It reads the
// initial versions from the Chart.lock file, creates a backup of the lock file,
// and then uses the Helm downloader manager to update the dependencies.
//
// After the update, it reads the updated versions from the lock file and
// compares them with the initial versions to determine if any changes were made.
// If no changes were made, it restores the original Chart.lock file from the
// backup to preserve the timestamp and avoid unnecessary changes.
//
// It returns a map of changes where the keys are the names of the dependencies
// and the values are the new versions. If a dependency was removed, the value
// will be an empty string. If a dependency was updated, the value will be a
// string indicating the old and new versions in the format
// "oldVersion -> newVersion".
func (em *EphemeralDependencyManager) update(chartPath string) (_ map[string]string, err error) {
	// Download the repository indexes. This is necessary to ensure that the
	// cache is properly populated, as otherwise the download manager will
	// attempt to download the repository indexes to the default cache path
	// instead of using the cache path set in the environment settings.
	if err = em.fetchRepositoryIndexes(); err != nil {
		return nil, fmt.Errorf("fetch repository indexes: %w", err)
	}

	// Read initial dependency versions from Chart.lock
	lockFile := filepath.Join(chartPath, "Chart.lock")
	initialVersions, err := GetChartDependencies(lockFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		// If the Chart.lock file does not exist, we can proceed without it
		// but if it exists and cannot be read, we return an error
		return nil, fmt.Errorf("read initial dependencies from Chart.lock: %w", err)
	}

	// Create a backup of Chart.lock before making changes
	lockBackup, err := newFileBackup(lockFile)
	if err != nil {
		return nil, fmt.Errorf("create backup of Chart.lock: %w", err)
	}
	defer func() {
		if restoreErr := lockBackup.Remove(); restoreErr != nil {
			err = kerrors.Reduce(kerrors.NewAggregate([]error{err, restoreErr}))
		}
	}()

	regClient, err := NewRegistryClient(em.authorizer.Client)
	if err != nil {
		return nil, fmt.Errorf("create registry client: %w", err)
	}

	// Update the chart dependencies using the downloader manager
	m := &downloader.Manager{
		Out:       io.Discard,
		ChartPath: chartPath,
		Verify:    downloader.VerifyNever,
		// Note: because we fetched the repository indexes earlier, we can skip
		// the update step here. The download manager will use the cached indexes
		// to resolve the dependencies.
		SkipUpdate: true,
		Getters: getter.All(&cli.EnvSettings{
			RepositoryConfig: em.repositoryConfig(),
			RepositoryCache:  em.repositoryCache(),
		}),
		RegistryClient:   regClient,
		RepositoryConfig: em.repositoryConfig(),
		RepositoryCache:  em.repositoryCache(),
	}
	if err = m.Update(); err != nil {
		return nil, fmt.Errorf("update chart dependencies: %w", err)
	}

	// Read the dependencies from Chart.lock after the update
	//
	// NB: We rely on the lock file to determine the version changes because
	// the dependency update process may change the version of a dependency
	// without updating the Chart.yaml. For example, because a new version is
	// available in the repository for a dependency that has a version range
	// specified in the Chart.yaml.
	updatedVersions, err := GetChartDependencies(lockBackup.originalPath)
	if err != nil {
		return nil, fmt.Errorf("read updated dependencies from Chart.lock: %w", err)
	}

	// Compare the initial and updated versions to see if any changes were made
	changes := compareChartVersions(initialVersions, updatedVersions)
	if len(changes) == 0 {
		// No changes made, restore the original Chart.lock. We do this because
		// the timestamp of the lock file will change even if no changes were
		// made.
		if err = lockBackup.Restore(); err != nil {
			return nil, fmt.Errorf("restore original Chart.lock: %w", err)
		}
	}
	return changes, nil
}

// build builds the chart dependencies for the given chart path. It reads the
// Chart.yaml file to get the list of dependencies, validates them, and then
// uses the Helm downloader manager to build the dependencies.
//
// It returns an error if any of the dependencies are invalid or if the
// dependencies cannot be built for any reason, such as if the repository
// indexes cannot be fetched or if the downloader manager fails to build the
// dependencies.
func (em *EphemeralDependencyManager) build(chartPath string) error {
	// Download the repository indexes. This is necessary to ensure that the
	// cache is properly populated, as otherwise the download manager will
	// attempt to download the repository indexes to the default cache path
	// instead of using the cache path set in the environment settings.
	if err := em.fetchRepositoryIndexes(); err != nil {
		return fmt.Errorf("fetch repository indexes: %w", err)
	}

	regClient, err := NewRegistryClient(em.authorizer.Client)
	if err != nil {
		return fmt.Errorf("create registry client: %w", err)
	}

	// Build the chart dependencies using the downloader manager
	m := &downloader.Manager{
		Out:       io.Discard,
		ChartPath: chartPath,
		Verify:    downloader.VerifyNever,
		// Note: because we fetched the repository indexes earlier, we can skip
		// the update step here. The download manager will use the cached indexes
		// to resolve the dependencies.
		SkipUpdate: true,
		Getters: getter.All(&cli.EnvSettings{
			RepositoryConfig: em.repositoryConfig(),
			RepositoryCache:  em.repositoryCache(),
		}),
		RegistryClient:   regClient,
		RepositoryConfig: em.repositoryConfig(),
		RepositoryCache:  em.repositoryCache(),
	}
	if err = m.Build(); err != nil {
		return fmt.Errorf("build chart dependencies: %w", err)
	}
	return nil
}

// validateDependencies checks if the dependencies specified in the chart's
// Chart.yaml file are valid. It ensures that local dependencies (those with
// a "file://" repository URL) are relative paths and do not point outside the
// work directory. It returns an error if any of the dependencies are invalid.
func (em *EphemeralDependencyManager) validateDependencies(chartPath string, dependencies []ChartDependency) error {
	for _, dep := range dependencies {
		if strings.HasPrefix(dep.Repository, "file://") {
			if err := em.validateLocalDependency(chartPath, dep.Repository); err != nil {
				return fmt.Errorf("invalid dependency %q: %w", dep.Repository, err)
			}
		}
	}
	return nil
}

// validateLocalDependency checks if a local dependency (with a "file://" URL)
// is a valid relative path and does not point outside the work directory.
func (em *EphemeralDependencyManager) validateLocalDependency(chartPath, depPath string) error {
	depPath = filepath.FromSlash(strings.TrimPrefix(depPath, "file://"))
	if filepath.IsAbs(depPath) {
		return fmt.Errorf("dependency path %q must be relative", depPath)
	}

	// Resolve the dependency path relative to the chart directory
	depPath = filepath.Join(chartPath, depPath)

	// Check if the resolved path is within the work directory
	resolvedDepPath, err := filepath.EvalSymlinks(depPath)
	if err != nil {
		return fmt.Errorf("resolve dependency path: %w", err)
	}
	if !fs.IsSubPath(em.workDir, resolvedDepPath) {
		return fmt.Errorf("dependency path is outside the work directory")
	}

	// Recursively check for symlinks that go outside the work directory,
	// as Helm follows symlinks when packaging charts
	return fs.ValidateSymlinks(em.workDir, depPath, 100)
}

// processDependencyUpdates processes the updates for chart dependencies
// specified in the updates slice. It checks if the dependencies exist in the
// current dependencies list and if the version requires an update. If so, it
// updates the version in the Chart.yaml file.
func (em *EphemeralDependencyManager) processDependencyUpdates(
	chartFilePath string,
	dependencies, updates []ChartDependency,
) error {
	// For each update, check if the dependency exists in the list of
	// dependencies and if the version requires an update.
	changes := make([]intyaml.Update, 0, len(dependencies))
	for _, update := range updates {
		var found bool
		for i, dep := range dependencies {
			if dep.Name == update.Name && dep.Repository == update.Repository {
				found = true
				// If the version is different, add the updated dependency
				if dep.Version != update.Version {
					changes = append(changes, intyaml.Update{
						Key:   fmt.Sprintf("dependencies.%d.version", i),
						Value: update.Version,
					})
				}
				break
			}
		}
		if !found {
			return fmt.Errorf(
				"no dependency in Chart.yaml matches update with repository %q and name %q",
				update.Repository, update.Name,
			)
		}
	}

	if len(changes) == 0 {
		// No changes made, nothing to persist.
		return nil
	}

	// Write the changes back to the Chart.yaml file
	if err := intyaml.SetValuesInFile(chartFilePath, changes); err != nil {
		return fmt.Errorf("update Chart.yaml with dependency changes: %w", err)
	}
	return nil
}

// setupRepositories sets up the Helm repositories for the given dependencies
// by writing them to the Helm repository configuration file or by logging in
// to the OCI registry if the dependency is an OCI chart.
//
// It returns an error if any of the repositories cannot be set up, such as
// if the credentials cannot be obtained for a repository
func (em *EphemeralDependencyManager) setupRepositories(ctx context.Context, dependencies []ChartDependency) error {
	repoFile := repo.NewFile()
	hosts := make(map[string]struct{}, len(dependencies))
	for _, dep := range dependencies {
		switch {
		case dep.Repository == "", strings.HasPrefix(dep.Repository, "file://"):
			// Skip local dependencies or those without a repository URL
			continue
		case strings.HasPrefix(dep.Repository, "http://"):
			entry := &repo.Entry{
				Name: nameForRepositoryURL(dep.Repository),
				URL:  dep.Repository,
			}
			repoFile.Update(entry)
		case strings.HasPrefix(dep.Repository, "https://"):
			entry := &repo.Entry{
				Name: nameForRepositoryURL(dep.Repository),
				URL:  dep.Repository,
			}

			creds, err := em.credsDB.Get(ctx, em.project, credentials.TypeHelm, dep.Repository)
			if err != nil {
				return fmt.Errorf("obtain credentials for repository %q: %w", dep.Repository, err)
			}
			if creds != nil {
				entry.Username = creds.Username
				entry.Password = creds.Password
			}

			repoFile.Update(entry)
		case strings.HasPrefix(dep.Repository, "oci://"):
			repository := urls.NormalizeChart(dep.Repository)
			host := hostForRepositoryURL(repository)

			if _, exists := hosts[host]; exists {
				// We already logged in to this host, skip it
				continue
			}

			credURL := "oci://" + path.Join(repository, dep.Name)
			creds, err := em.credsDB.Get(ctx, em.project, credentials.TypeHelm, credURL)
			if err != nil {
				return fmt.Errorf("obtain credentials for repository %q: %w", dep.Repository, err)
			}

			if creds != nil {
				if err = em.authorizer.Login(ctx, host, creds.Username, creds.Password); err != nil {
					return fmt.Errorf("authenticate with chart repository %q: %w", dep.Repository, err)
				}

				// Mark this host as logged in to avoid duplicate logins
				hosts[host] = struct{}{}
			}
		}
	}

	if err := repoFile.WriteFile(em.repositoryConfig(), 0o600); err != nil {
		return fmt.Errorf("write repository config file: %w", err)
	}
	return nil
}

// fetchRepositoryIndexes downloads the index files for all repositories
// defined in the repository configuration file, caching them to the ephemeral
// repositoryCache directory.
func (em *EphemeralDependencyManager) fetchRepositoryIndexes() error {
	repoFile, err := repo.LoadFile(em.repositoryConfig())
	if err != nil {
		return fmt.Errorf("load repository config file: %w", err)
	}

	env := &cli.EnvSettings{
		RepositoryConfig: em.repositoryConfig(),
		RepositoryCache:  em.repositoryCache(),
	}

	for _, entry := range repoFile.Repositories {
		cr, err := repo.NewChartRepository(entry, getter.All(env))
		if err != nil {
			return fmt.Errorf("create chart repository for %q: %w", entry.URL, err)
		}

		// NB: Explicitly overwrite the cache path to avoid using the default
		// cache path from the environment variables. Without this, the download
		// manager will not find the repository index files in the cache, and
		// will attempt to download them again (to the default cache path).
		// I.e. without this, the download manager will not use the isolated
		// cache.
		cr.CachePath = env.RepositoryCache

		if _, err = cr.DownloadIndexFile(); err != nil {
			return fmt.Errorf("download repository index for %q: %w", entry.URL, err)
		}
	}

	return nil
}

// repositoryConfig returns the path to the Helm repository configuration file
// used by the EphemeralDependencyManager. This file is used to store the
// repositories that are used by the dependency manager to resolve chart
// dependencies.
func (em *EphemeralDependencyManager) repositoryConfig() string {
	return filepath.Join(em.helmHome, "repositories.yaml")
}

// repositoryCache returns the path to the Helm repository cache directory
// used by the EphemeralDependencyManager. This directory is used to cache the
// repository index files for the repositories that are used by the dependency
// manager to resolve chart dependencies.
func (em *EphemeralDependencyManager) repositoryCache() string {
	return filepath.Join(em.helmHome, "cache")
}

// backupFile is a struct that represents a backup of a file.
type backupFile struct {
	originalPath string
	backupPath   string
}

// newFileBackup creates a backup of the file at the given path. If the file
// does not exist, it returns a backupFile with an empty backupPath. It returns
// an error if the file is a symlink or if the backup cannot be created for
// any other reason.
func newFileBackup(p string) (*backupFile, error) {
	fi, err := os.Lstat(p)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to stat %q: %w", p, err)
		}
		// If the file does not exist, we cannot create a backup. We do however
		// still return a backupFile with an empty backupPath, so that the
		// caller can use the information to determine that no backup was
		// created and that the original file does not exist.
		return &backupFile{
			originalPath: p,
			backupPath:   "",
		}, nil
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		// If the file is a symlink, we cannot create a backup of it. We return
		// an error to indicate that the backup cannot be created.
		return nil, fmt.Errorf("cannot create backup of symlink %q", p)
	}

	backupPath := fmt.Sprintf("%s.%s.bak", p, time.Now().Format("20060102150405"))
	if err = fs.CopyFile(p, backupPath); err != nil {
		return nil, fmt.Errorf("failed to copy %q to %q: %w", p, backupPath, err)
	}

	return &backupFile{
		originalPath: p,
		backupPath:   backupPath,
	}, nil
}

// Restore restores the original file from the backup file. If the backup file
// does not exist, it returns nil. If the original file cannot be restored,
// it returns an error. After restoring, it clears the backupPath to indicate
// that the backup has been restored and is no longer needed.
func (bf *backupFile) Restore() error {
	if bf.backupPath == "" {
		// No backup was created, so nothing to restore.
		return nil
	}

	if err := os.Rename(bf.backupPath, bf.originalPath); err != nil {
		return fmt.Errorf("failed to restore backup file %q to %q: %w", bf.backupPath, bf.originalPath, err)
	}

	// Clear the backup path to indicate that the backup has been restored.
	bf.backupPath = ""
	return nil
}

// Remove deletes the backup file. If the backup file does not exist, it returns
// nil. If the backup file cannot be removed, it returns an error.
func (bf *backupFile) Remove() error {
	if bf.backupPath == "" {
		// No backup was created, so nothing to remove.
		return nil
	}

	if err := os.Remove(bf.backupPath); err != nil {
		return fmt.Errorf("failed to remove backup file %q: %w", bf.backupPath, err)
	}

	return nil
}

// compareChartVersions compares the versions of chart dependencies before and
// after an update. It returns a map where the keys are the names of the
// dependencies and the values indicate the changes in version.
//
// If a dependency was added, the value will be the new version. If a
// dependency was removed, the value will be an empty string. If a dependency
// was updated, the value will be a string indicating the old and new versions
// in the format "oldVersion -> newVersion".
func compareChartVersions(before, after []ChartDependency) map[string]string {
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

// hostForRepositoryURL extracts the host from a repository URL, removing any
// scheme (e.g., "oci://", "http://", "https://") and any path components
// after the host.
func hostForRepositoryURL(repoURL string) string {
	for _, s := range []string{"oci://", "http://", "https://"} {
		if strings.HasPrefix(repoURL, s) {
			repoURL = strings.TrimPrefix(repoURL, s)
			break
		}
	}
	if idx := strings.Index(repoURL, "/"); idx != -1 {
		repoURL = repoURL[:idx]
	}
	return repoURL
}
