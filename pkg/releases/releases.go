package releases

import (
	"encoding/json"
	"regexp"

	"github.com/Masterminds/semver/v3"

	"github.com/akuity/kargo/pkg/x/version"
)

// Note(krancour): Everything in this file could be collectively summarized as:
// GitHub API release information in, filtered and transformed Kargo release
// information out. This means the serialization and deserialization of this
// information is asymmetrical. This was a small, but deliberate tradeoff I made
// to keep the implementation of the Service interface simpler and focused
// entirely on retrieval, selecting the latest patch version of each minor
// version, and sorting the results.

// githubReleases is a slice of Release. GitHub API responses can be unmarshaled
// directly into this type, which will handle all necessary filtering and
// transformations to produce a slice of valid Release objects.
type githubReleases []Release

// UnmarshalJSON parses the GitHub Releases API response, filtering out draft
// and prerelease entries, validating tag names as semver versions, and
// unmarshaling valid entries into Release objects.
func (g *githubReleases) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	filtered := make([]Release, 0, len(raw))
	for _, entry := range raw {
		var meta struct {
			TagName    string `json:"tag_name"`
			Draft      bool   `json:"draft"`
			Prerelease bool   `json:"prerelease"`
		}
		if err := json.Unmarshal(entry, &meta); err != nil {
			return err
		}
		if meta.Draft || meta.Prerelease {
			continue
		}
		relVer, err := semver.NewVersion(meta.TagName)
		if err != nil {
			continue
		}
		var r Release
		if err := json.Unmarshal(entry, &r); err != nil {
			return err
		}
		if relVer.Major() == version.GetVersion().Semver.Major() &&
			relVer.Minor() == version.GetVersion().Semver.Minor() {
			r.Current = true
		}
		filtered = append(filtered, r)
	}
	*g = filtered
	return nil
}

// Release represents a release of Kargo. GitHub release information can be
// unmarshaled directly into this struct. Custom unmarshaling will handle all
// necessary filtering and transformations.
type Release struct {
	// Version is the semver version of the release, parsed from the GitHub tag
	// name.
	Version *semver.Version `json:"version"`
	// Latest indicates whether this release is the latest release.
	Latest bool `json:"latest,omitempty"`
	// Current indicates whether this release matches the minor version of the
	// API server.
	Current bool `json:"current,omitempty"`
	// CLIBinaries maps OS and architecture combinations to their corresponding
	// download URLs for the kargo CLI binary.
	CLIBinaries CLIBinaries `json:"cliBinaries"`
} // @name Release

// MarshalJSON formats the Release for JSON output, notably converting the
// semver Version to a string.
func (r Release) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Version     string      `json:"version"`
		Latest      bool        `json:"latest,omitempty"`
		Current     bool        `json:"current,omitempty"`
		CLIBinaries CLIBinaries `json:"cliBinaries"`
	}{
		Version:     r.Version.Original(),
		Latest:      r.Latest,
		Current:     r.Current,
		CLIBinaries: r.CLIBinaries,
	})
}

// cliAssetPattern matches asset names of the form kargo-{os}-{arch}[.exe],
// capturing os and arch as groups.
var cliAssetPattern = regexp.MustCompile(`^kargo-([a-z]+)-([a-z0-9]+)(?:\.exe)?$`)

// UnmarshalJSON parses a GitHub Releases API entry, extracting the tag name as
// a semver.Version and building CLIBinaries from assets matching the
// kargo-{os}-{arch}[.exe] naming convention.
func (r *Release) UnmarshalJSON(data []byte) error {
	var raw struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	v, err := semver.NewVersion(raw.TagName)
	if err != nil {
		return err
	}
	r.Version = v
	r.CLIBinaries = CLIBinaries{}
	for _, a := range raw.Assets {
		if m := cliAssetPattern.FindStringSubmatch(a.Name); m != nil {
			os, arch := m[1], m[2]
			if r.CLIBinaries[os] == nil {
				r.CLIBinaries[os] = make(map[string]string)
			}
			r.CLIBinaries[os][arch] = a.BrowserDownloadURL
		}
	}
	return nil
}

// CLIBinaries maps OS to architectures and their download URLs.
// e.g. {"linux": {"amd64": "https://...", "arm64": "https://..."}}
type CLIBinaries map[string]map[string]string
