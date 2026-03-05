package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"

	"github.com/Masterminds/semver/v3"
	"github.com/hashicorp/go-cleanhttp"
)

const releasesBaseURL = "https://api.github.com/repos/akuity/kargo/releases"

// fetchBestReleases retrieves all releases from the GitHub API, handling
// pagination, and returns the latest patch release for each minor version,
// sorted in descending version order.
func fetchBestReleases(ctx context.Context, baseURL string) ([]Release, error) {
	const perPage = 100

	httpClient := cleanhttp.DefaultClient()
	var allReleases []Release
	page := 1

	releasesURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing releases URL: %w", err)
	}
	query := releasesURL.Query()
	query.Add("per_page", fmt.Sprintf("%d", perPage))
	releasesURL.RawQuery = query.Encode()

	for {
		query.Set("page", fmt.Sprintf("%d", page))
		releasesURL.RawQuery = query.Encode()
		req, err := http.NewRequestWithContext(
			ctx, http.MethodGet, releasesURL.String(), nil,
		)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Accept", "application/vnd.github+json")

		// #nosec G704 -- The URL is controlled by us, so this is not a security
		// risk.
		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching releases page %d: %w", page, err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf(
				"unexpected status %d from GitHub releases API",
				resp.StatusCode,
			)
		}

		var pageReleases githubReleases
		if err := json.Unmarshal(body, &pageReleases); err != nil {
			return nil, fmt.Errorf("unmarshaling releases: %w", err)
		}

		allReleases = append(allReleases, pageReleases...)

		if len(pageReleases) < perPage {
			break
		}
		page++
	}

	return pickBestReleases(allReleases), nil
}

// pickBestReleases takes a slice of releases and returns the latest patch
// release for each minor version, sorted in descending version order.
func pickBestReleases(raw []Release) []Release {
	best := make(map[string]*Release)

	v1 := semver.MustParse("v1.0.0")
	for i, r := range raw {
		if r.Version.LessThan(v1) {
			continue
		}
		key := fmt.Sprintf("%d.%d", r.Version.Major(), r.Version.Minor())
		if existing, ok := best[key]; !ok || r.Version.GreaterThan(existing.Version) {
			best[key] = &raw[i]
		}
	}

	results := make([]Release, 0, len(best))
	for _, r := range best {
		results = append(results, *r)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Version.GreaterThan(results[j].Version)
	})

	if len(results) > 0 {
		results[0].Latest = true
	}

	return results
}
