package helm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"oras.land/oras-go/pkg/registry"
	"oras.land/oras-go/pkg/registry/remote"
	"oras.land/oras-go/pkg/registry/remote/auth"

	libExec "github.com/akuityio/kargo/internal/exec"
)

// GetLatestChartVersion connects to the Helm chart registry specified by
// registryURL and retrieves all available versions of the chart found therein.
// The registry can be either a classic chart registry (using HTTP/S) or an OCI
// registry. If no semverConstraint is provided (empty string is passed), then
// the version that is semantically greatest will be returned. If a
// semverConstraint is specified, then the semantically greatest version
// satisfying that constraint will be returned. If no version satisfies the
// constraint, the empty string is returned. Provided credentials may be nil for
// public registries, but must be non-nil for private registries.
func GetLatestChartVersion(
	ctx context.Context,
	registryURL string,
	chart string,
	semverConstraint string,
	creds *Credentials,
) (string, error) {
	var versions []string
	var err error
	if strings.HasPrefix(registryURL, "http://") ||
		strings.HasPrefix(registryURL, "https://") {
		versions, err =
			getChartVersionsFromClassicRegistry(registryURL, chart, creds)
	} else if strings.HasPrefix(registryURL, "oci://") {
		versions, err =
			getChartVersionsFromOCIRegistry(ctx, registryURL, chart, creds)
	} else {
		return "", errors.Errorf("registry URL %q is invalid", registryURL)
	}
	if err != nil {
		return "", errors.Wrapf(
			err,
			"error retrieving versions of chart %q from registry %q",
			chart,
			registryURL,
		)
	}
	latestVersion, err := getLatestVersion(versions, semverConstraint)
	return latestVersion, errors.Wrapf(
		err,
		"error determining latest version of chart %q from  registry %q",
		chart,
		registryURL,
	)
}

// getChartVersionsFromClassicRegistry connects to the classic (HTTP/S) chart
// registry specified by registryURL and retrieves all available versions of the
// specified chart. The provided registryURL MUST begin with protocol http:// or
// https://. Provided credentials may be nil for public registries, but must be
// non-nil for private registries.
func getChartVersionsFromClassicRegistry(
	registryURL string,
	chart string,
	creds *Credentials,
) ([]string, error) {
	indexURL := fmt.Sprintf("%s/index.yaml", strings.TrimSuffix(registryURL, "/"))
	req, err := http.NewRequest(http.MethodGet, indexURL, nil)
	if err != nil {
		return nil,
			errors.Wrapf(err, "error preparing HTTP/S request to %q", indexURL)
	}
	if creds != nil {
		req.SetBasicAuth(creds.Username, creds.Password)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil,
			errors.Wrapf(err, "error querying registry index at %q", indexURL)
	}
	if res.StatusCode != http.StatusOK {
		return nil,
			errors.Errorf(
				"received unexpected HTTP %d when querying registry index at %q",
				res.StatusCode,
				indexURL,
			)
	}
	defer res.Body.Close()
	resBodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil,
			errors.Wrapf(err, "error reading registry index from %q", indexURL)
	}
	index := struct {
		Entries map[string][]struct {
			Version string `json:"version,omitempty"`
		} `json:"entries,omitempty"`
	}{}
	if err = yaml.Unmarshal(resBodyBytes, &index); err != nil {
		return nil,
			errors.Wrapf(err, "error unmarshaling registry index from %q", indexURL)
	}
	entries, ok := index.Entries[chart]
	if !ok {
		return nil, errors.Errorf(
			"no versions of chart %q found in registry index from %q",
			chart,
			indexURL,
		)
	}
	versions := make([]string, len(entries))
	for i, entry := range entries {
		versions[i] = entry.Version
	}
	return versions, nil
}

// getChartVersionsFromOCIRegistry connects to the OCI registry specified by
// registryURL and retrieves all available versions of the specified chart.
// Provided credentials may be nil for public registries, but must be non-nil
// for private registries.
func getChartVersionsFromOCIRegistry(
	ctx context.Context,
	registryURL string,
	chart string,
	creds *Credentials,
) ([]string, error) {
	rep := &remote.Repository{
		Reference: registry.Reference{
			Registry:   strings.TrimPrefix(registryURL, "oci://"),
			Repository: chart,
		},
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
	versions := []string{}
	return versions, errors.Wrapf(
		rep.Tags(ctx, func(t []string) error {
			versions = append(versions, t...)
			return nil
		}),
		"error retrieving versions of chart %q from registry %q",
		chart,
		registryURL,
	)
}

// getLatestVersion returns the semantically greatest version from the versions
// provided which satisfies the provided constraints. If no constraints are
// specified (the empty string is passed), the absolute semantically greatest
// version will be returned. The empty string will be returned when the provided
// list of versions is nil or empty.
func getLatestVersion(versions []string, constraintStr string) (string, error) {
	semvers := make([]*semver.Version, len(versions))
	for i, version := range versions {
		var err error
		if semvers[i], err = semver.NewVersion(version); err != nil {
			return "", errors.Wrapf(err, "error parsing version %q", version)
		}
	}
	sort.Sort(semver.Collection(semvers))
	if constraintStr == "" {
		return semvers[len(semvers)-1].String(), nil
	}
	constraint, err := semver.NewConstraint(constraintStr)
	if err != nil {
		return "", errors.Wrapf(err, "error parsing constraint %q", constraintStr)
	}
	for i := len(semvers) - 1; i >= 0; i-- {
		if constraint.Check(semvers[i]) {
			return semvers[i].String(), nil
		}
	}
	return "", nil
}

func UpdateChartDependencies(homePath, chartPath string) error {
	cmd := exec.Command("helm", "dependency", "update", chartPath)
	homeEnvVar := fmt.Sprintf("HOME=%s", homePath)
	if cmd.Env == nil {
		cmd.Env = []string{homeEnvVar}
	} else {
		cmd.Env = append(cmd.Env, homeEnvVar)
	}
	_, err := libExec.Exec(cmd)
	return errors.Wrapf(
		err,
		"error running `helm dependency update` for chart at %q",
		chartPath,
	)
}
