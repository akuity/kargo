package helm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v3"
	"oras.land/oras-go/pkg/registry"
	"oras.land/oras-go/pkg/registry/remote"
	"oras.land/oras-go/pkg/registry/remote/auth"

	libExec "github.com/akuity/kargo/internal/exec"
)

// SelectChartVersion connects to the specified Helm chart repository and
// determines the latest version of the specified chart, optionally filtering
// by a SemVer constraint. It then returns the version.
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
// If no versions of the chart are found in the repository, an error is
// returned.
func SelectChartVersion(
	ctx context.Context,
	repoURL string,
	chart string,
	semverConstraint string,
	creds *Credentials,
) (string, error) {
	versions, err := DiscoverChartVersions(ctx, repoURL, chart, semverConstraint, creds)
	if err != nil {
		return "", err
	}
	if len(versions) == 0 {
		err := fmt.Errorf("no versions of chart %q found in repository %q", chart, repoURL)
		if semverConstraint != "" {
			err = fmt.Errorf("%s that satisfy constraint %q", err, semverConstraint)
		}
		return "", err
	}
	return versions[0], nil
}

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
	var versions []string
	var err error
	switch {
	case strings.HasPrefix(repoURL, "http://"), strings.HasPrefix(repoURL, "https://"):
		versions, err = getChartVersionsFromClassicRepo(repoURL, chart, creds)
	case strings.HasPrefix(repoURL, "oci://"):
		versions, err = getChartVersionsFromOCIRepo(ctx, repoURL, creds)
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

	semvers := versionsToSemVerCollection(versions)
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

	// NB: semver.Collection sorts in ascending order by default. We want to
	// return the versions in descending order.
	sort.Sort(sort.Reverse(semvers))

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
func versionsToSemVerCollection(versions []string) semver.Collection {
	semvers := make(semver.Collection, 0, len(versions))
	for _, version := range versions {
		semverVersion, err := semver.NewVersion(version)
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
		versions[i] = semverVersion.Original()
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

func UpdateChartDependencies(homePath, chartPath string) error {
	cmd := exec.Command("helm", "dependency", "update", chartPath)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("HOME=%s", homePath))
	if _, err := libExec.Exec(cmd); err != nil {
		return fmt.Errorf("error running `helm dependency update` for chart at %q: %w", chartPath, err)
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
