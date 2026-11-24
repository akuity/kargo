package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// siteDir is the directory whose contents are published to GitHub Pages.
const siteDir = "_site"

func main() {
	releases, err := fetchBestReleases(context.Background(), releasesBaseURL)
	if err != nil {
		panic(err)
	}
	if err = writeBestReleasesJSON(siteDir, releases); err != nil {
		panic(err)
	}
	if err = writeVersionFiles(siteDir, releases); err != nil {
		panic(err)
	}
}

// writeBestReleasesJSON writes <dir>/best-releases.json, listing every release.
func writeBestReleasesJSON(dir string, releases []Release) error {
	data, err := json.MarshalIndent(
		struct {
			Releases []Release `json:"releases"`
		}{
			Releases: releases,
		},
		"",
		"  ",
	)
	if err != nil {
		return fmt.Errorf("error marshaling releases: %w", err)
	}
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dir, err)
	}
	path := filepath.Join(dir, "best-releases.json")
	// #nosec G306 -- Published content must be world-readable.
	if err = os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("error writing %s: %w", path, err)
	}
	return nil
}

// writeVersionFiles writes <dir>/<major>.<minor>/version.txt for each release.
// Each file contains only that release's version, e.g. v1.11.8, with no
// trailing newline.
func writeVersionFiles(dir string, releases []Release) error {
	for _, r := range releases {
		lineDir := filepath.Join(
			dir,
			fmt.Sprintf("%d.%d", r.Version.Major(), r.Version.Minor()),
		)
		if err := os.MkdirAll(lineDir, 0o755); err != nil {
			return fmt.Errorf("error creating directory %s: %w", lineDir, err)
		}
		path := filepath.Join(lineDir, "version.txt")
		// #nosec G306 -- Published content must be world-readable.
		if err := os.WriteFile(
			path,
			[]byte(r.Version.Original()),
			0o644,
		); err != nil {
			return fmt.Errorf("error writing %s: %w", path, err)
		}
	}
	return nil
}
