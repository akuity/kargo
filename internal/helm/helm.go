package helm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v3"
	"oras.land/oras-go/pkg/registry"
	"oras.land/oras-go/pkg/registry/remote"
	"oras.land/oras-go/pkg/registry/remote/auth"

	libExec "github.com/akuity/kargo/internal/exec"
)

// DiscoverChartVersions connects to the specified Helm chart repository and
// retrieves all available versions of the specified chart, optionally filtering
// by a SemVer constraint. It then returns the versions in descending order.
//
// The repository can be either a classic chart repository (using HTTP/S) or a
// repository within an OCI registry. Classic chart repositories can contain
// differently named charts. When repoURL points to such a repository, the name
// argument must specify the name of the chart within the repository. In the
// case of a repository within an OCI registry, the URL implicitly points to a
// specific chart and the name argument must be empty.
//
// The credentials argument may be nil for public repositories, but must be
// non-nil for private repositories.
//
// It returns an error if the repository cannot be reached or if the versions
// cannot be retrieved, but it does not return an error if no versions of the
// chart are found in the repository.
func DiscoverChartVersions(
	ctx context.Context,
	repoURL string,
	chart string,
	semverConstraint string,
	creds *Credentials,
) ([]string, error) {
	var isOCI bool
	var versions []string
	var err error
	switch {
	case strings.HasPrefix(repoURL, "http://"), strings.HasPrefix(repoURL, "https://"):
		versions, err = getChartVersionsFromClassicRepo(repoURL, chart, creds)
	case strings.HasPrefix(repoURL, "oci://"):
		versions, err = getChartVersionsFromOCIRepo(ctx, repoURL, creds)
		isOCI = true
	default:
		return nil, fmt.Errorf("repository URL %q is invalid", repoURL)
	}
	if err != nil {
		return nil, fmt.Errorf(
			"error retrieving versions of chart %q from repository %q: %w",
			chart,
			repoURL,
			err,
		)
	}

	semvers := versionsToSemVerCollection(versions, isOCI)
	if len(semvers) == 0 {
		return nil, nil
	}

	if semverConstraint != "" {
		if semvers, err = filterSemVers(semvers, semverConstraint); err != nil {
			return nil, fmt.Errorf(
				"error filtering versions of chart %q from repository %q: %w",
				chart,
				repoURL,
				err,
			)
		}
	}

	// Sort versions in descending order
	slices.SortFunc(semvers, func(lhs, rhs *semver.Version) int {
		if comp := rhs.Compare(lhs); comp != 0 {
			return comp
		}
		// If the semvers tie, break the tie lexically using the original strings
		// used to construct the semvers. This ensures a deterministic comparison
		// of equivalent semvers, e.g., "1.0.0" > "1.0"
		return strings.Compare(rhs.Original(), lhs.Original())
	})

	return semVerCollectionToVersions(semvers), nil
}

// getChartVersionsFromClassicRepo connects to the classic (HTTP/S) chart
// repository specified by repoURL and retrieves all available versions of the
// specified chart. The provided repoURL MUST begin with protocol http:// or
// https://. Provided credentials may be nil for public repositories, but must
// be non-nil for private repositories.
func getChartVersionsFromClassicRepo(
	repoURL string,
	chart string,
	creds *Credentials,
) ([]string, error) {
	indexURL := fmt.Sprintf("%s/index.yaml", strings.TrimSuffix(repoURL, "/"))
	req, err := http.NewRequest(http.MethodGet, indexURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error preparing HTTP/S request to %q: %w", indexURL, err)
	}
	if creds != nil {
		req.SetBasicAuth(creds.Username, creds.Password)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error querying repository index at %q: %w", indexURL, err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"received unexpected HTTP %d when querying repository index at %q",
			res.StatusCode,
			indexURL,
		)
	}
	defer res.Body.Close()
	resBodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading repository index from %q: %w", indexURL, err)
	}
	index := struct {
		Entries map[string][]struct {
			Version string `json:"version,omitempty"`
		} `json:"entries,omitempty"`
	}{}
	if err = yaml.Unmarshal(resBodyBytes, &index); err != nil {
		return nil, fmt.Errorf("error unmarshaling repository index from %q: %w", indexURL, err)
	}
	entries, ok := index.Entries[chart]
	if !ok {
		return nil, nil
	}
	versions := make([]string, len(entries))
	for i, entry := range entries {
		versions[i] = entry.Version
	}
	return versions, nil
}

// getChartVersionsFromOCIRepo connects to the OCI repository specified by
// repoURL and retrieves all available versions of the specified chart. Provided
// credentials may be nil for public repositories, but must be non-nil for
// private repositories.
func getChartVersionsFromOCIRepo(
	ctx context.Context,
	repoURL string,
	creds *Credentials,
) ([]string, error) {
	ref, err := registry.ParseReference(strings.TrimPrefix(repoURL, "oci://"))
	if err != nil {
		return nil, fmt.Errorf("error parsing repository URL %q: %w", repoURL, err)
	}
	rep := &remote.Repository{
		Reference: ref,
		Client: &auth.Client{
			Credential: func(context.Context, string) (auth.Credential, error) {
				if creds != nil {
					return auth.Credential{
						Username: creds.Username,
						Password: creds.Password,
					}, nil
				}
				return auth.Credential{}, nil
			},
		},
	}

	versions := make([]string, 0, rep.TagListPageSize)
	if err := rep.Tags(ctx, func(t []string) error {
		versions = append(versions, t...)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("error retrieving versions of chart from repository %q: %w", repoURL, err)
	}
	return versions, nil
}

// versionsToSemVerCollection converts a slice of versions to a semver.Collection.
// Any versions that cannot be parsed as SemVer are ignored.
func versionsToSemVerCollection(versions []string, isOCI bool) semver.Collection {
	newSemver := semver.NewVersion
	if isOCI {
		// OCI artifact tags produced by Helm are STRICT SemVer, meaning that
		// they must contain a patch version and do not start with a "v".
		// I.e., "1.0.0" is valid, but "v1.0" is not. This is enforced by Helm
		// itself when publishing charts.
		newSemver = semver.StrictNewVersion
	}

	semvers := make(semver.Collection, 0, len(versions))
	for _, version := range versions {
		// OCI artifact tags are not allowed to contain the "+" character,
		// which is used by SemVer to separate the version from the build
		// metadata. To work around this, Helm uses "_" instead of "+".
		if isOCI {
			version = strings.ReplaceAll(version, "_", "+")
		}
		semverVersion, err := newSemver(version)
		if err == nil {
			semvers = append(semvers, semverVersion)
		}
	}
	return semvers
}

// semVerCollectionToVersions converts a semver.Collection to a slice of
// version strings.
func semVerCollectionToVersions(semvers semver.Collection) []string {
	versions := make([]string, len(semvers))
	for i, semverVersion := range semvers {
		original := semverVersion.Original()
		versions[i] = original
	}
	return versions
}

// filterSemVers filters the provided SemVers by the provided semver
// constraint.
func filterSemVers(semvers semver.Collection, semverConstraint string) (semver.Collection, error) {
	constraint, err := semver.NewConstraint(semverConstraint)
	if err != nil {
		return nil, fmt.Errorf("error parsing constraint %q: %w", semverConstraint, err)
	}

	var filtered = make(semver.Collection, 0, len(semvers))
	for _, version := range semvers {
		if constraint.Check(version) {
			filtered = append(filtered, version)
		}
	}
	return filtered, nil
}

// Login runs `helm registry login` or `helm repo add` for the provided
// repository. The provided homePath is used to set the HOME environment
// variable, as well as the XDG_* environment variables. This ensures that Helm
// uses the provided homePath as its configuration directory, and allows for
// isolation.
func Login(homePath, repository string, credentials Credentials) error {
	var args []string
	switch {
	case strings.HasPrefix(repository, "oci://"):
		// When logging into an OCI registry, both username and password are
		// required. If the password is missing, return an error as otherwise
		// it would prompt the user for it.
		if credentials.Username == "" || credentials.Password == "" {
			return fmt.Errorf("missing username and/or password for OCI registry login")
		}

		// NB: Registry login works _without_ the oci:// prefix.
		args = append(args, "registry", "login", NormalizeChartRepositoryURL(repository))
	case strings.HasPrefix(repository, "https://"):
		// When logging into an HTTPS repository, a password is required if a
		// username is provided. If the password is missing, return an error as
		// otherwise it would prompt the user for it.
		if credentials.Username != "" && credentials.Password == "" {
			return fmt.Errorf("missing password for HTTPS repository login")
		}

		// NB: The repository "alias" does not accept slashes, but does accept
		// any other type of character.
		args = append(args, "repo", "add", strings.ReplaceAll(repository, "/", ""), repository)
	default:
		return fmt.Errorf("unsupported repository URL %q", repository)
	}

	// Flags for username and password are the same for both `helm registry login`
	// and `helm repo add`.
	if credentials.Username != "" {
		args = append(args, "--username", credentials.Username)
	}
	if credentials.Password != "" {
		args = append(args, "--password-stdin")
	}

	cmd := exec.Command("helm", args...)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, helmEnv(homePath)...)

	// If a password is provided, write it to the command's stdin.
	if credentials.Password != "" {
		in, err := cmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("stdin pipe for password: %w", err)
		}
		go func() {
			defer in.Close()
			_, _ = io.WriteString(in, credentials.Password)
		}()
	}

	if _, err := libExec.Exec(cmd); err != nil {
		return err
	}
	return nil
}

// UpdateChartDependencies runs `helm dependency update` for the chart at the
// provided chartPath. The homePath is used to set the HOME environment variable,
// as well as the XDG_* environment variables. This ensures that Helm uses the
// provided homePath as its configuration directory, and allows for isolation.
func UpdateChartDependencies(homePath, chartPath string) error {
	cmd := exec.Command("helm", "dependency", "update", chartPath)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, helmEnv(homePath)...)
	if _, err := libExec.Exec(cmd); err != nil {
		return err
	}
	return nil
}

// NormalizeChartRepositoryURL normalizes a chart repository URL for purposes
// of comparison. Crucially, this function removes the oci:// prefix from the
// URL if there is one.
func NormalizeChartRepositoryURL(repo string) string {
	return strings.TrimPrefix(
		strings.ToLower(
			strings.TrimSpace(repo),
		),
		"oci://",
	)
}

// helmEnv returns a slice of environment variables that should be set when
// running Helm commands. The provided homePath is used to set the HOME
// environment variable, as well as the XDG_* environment variables.
//
// This ensures that Helm uses the provided homePath as its configuration
// directory.
func helmEnv(homePath string) []string {
	return []string{
		fmt.Sprintf("HOME=%s", homePath),
		fmt.Sprintf("XDG_CACHE_HOME=%s/cache", homePath),
		fmt.Sprintf("XDG_CONFIG_HOME=%s/config", homePath),
		fmt.Sprintf("XDG_DATA_HOME=%s/data", homePath),
	}
}
