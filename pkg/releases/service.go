package releases

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"sync/atomic"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/jferrl/go-githubauth"
	"github.com/kelseyhightower/envconfig"
	"golang.org/x/oauth2"

	"github.com/akuity/kargo/pkg/logging"
)

// ServiceConfig encapsulates configuration options for the releases service.
type ServiceConfig struct {
	GitHubAppClientID       string `envconfig:"GITHUB_APP_CLIENT_ID"`
	GitHubAppInstallationID int64  `envconfig:"GITHUB_APP_INSTALLATION_ID"`
	GitHubAppPrivateKey     string `envconfig:"GITHUB_APP_PRIVATE_KEY"`
}

// ServiceConfigFromEnv reads configuration from environment variables and
// returns a ServiceConfig.
func ServiceConfigFromEnv() ServiceConfig {
	cfg := ServiceConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// Service provides access to processed Kargo release information.
type Service interface {
	// GetBestReleases returns the latest patch release for each minor version,
	// sorted in descending version order, from a cache that is updated in the
	// background.
	GetBestReleases() []Release
}

type service struct {
	cached atomic.Pointer[[]Release]
	// baseURL is an attribute instead of a constant so it can be overridden in
	// tests.
	baseURL    string
	httpClient *http.Client
}

// NewService returns a new releases Service that fetches from the GitHub API
// and refreshes in the background on the given interval. The context controls
// the lifetime of the background goroutine.
func NewService(ctx context.Context, cfg *ServiceConfig) (Service, error) {
	httpClient, err := newGitHubClient(cfg)
	if err != nil {
		return nil, err
	}
	s := &service{
		baseURL:    "https://api.github.com/repos/akuity/kargo/releases",
		httpClient: httpClient,
	}
	go s.run(ctx)
	return s, nil
}

// newGitHubClient returns an HTTP client that authenticates via a GitHub App
// installation token if credentials are provided, or an unauthenticated client
// otherwise.
func newGitHubClient(cfg *ServiceConfig) (*http.Client, error) {
	if cfg == nil ||
		cfg.GitHubAppClientID == "" ||
		cfg.GitHubAppInstallationID == 0 ||
		cfg.GitHubAppPrivateKey == "" {
		return cleanhttp.DefaultClient(), nil
	}
	appTokenSource, err := githubauth.NewApplicationTokenSource(
		cfg.GitHubAppClientID, []byte(cfg.GitHubAppPrivateKey),
	)
	if err != nil {
		return nil, fmt.Errorf("creating GitHub App token source: %w", err)
	}
	installationTokenSource := githubauth.NewInstallationTokenSource(
		cfg.GitHubAppInstallationID, appTokenSource,
	)
	return oauth2.NewClient(context.Background(), installationTokenSource), nil
}

// GetBestReleases implements Service.
func (s *service) GetBestReleases() []Release {
	if p := s.cached.Load(); p != nil {
		return *p
	}
	return nil
}

// run is the background goroutine that periodically refreshes the releases
// cache.
func (s *service) run(ctx context.Context) {
	s.refresh(ctx) // Seed the cache immediately on startup
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			releases, err := s.fetchReleases(ctx)
			if err != nil {
				logging.LoggerFromContext(ctx).Error(
					err, "failed to refresh releases from GitHub",
				)
				return
			}
			s.cached.Store(&releases)
		}
	}
}

// refresh fetches the latest releases from GitHub and updates the cache. Any
// error is logged and the cache remains unchanged.
func (s *service) refresh(ctx context.Context) {
	releases, err := s.fetchReleases(ctx)
	if err != nil {
		logging.LoggerFromContext(ctx).Error(
			err, "failed to refresh releases from GitHub",
		)
		return
	}
	s.cached.Store(&releases)
}

// fetchReleases retrieves all releases from the GitHub API, handling
// pagination, and returns the latest patch release for each minor version,
// sorted in descending version order.
func (s *service) fetchReleases(ctx context.Context) ([]Release, error) {
	const perPage = 100

	var allReleases []Release
	page := 1

	releasesURL, err := url.Parse(s.baseURL)
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

		resp, err := s.httpClient.Do(req)
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

	return s.pickBestReleases(allReleases), nil
}

// pickBestReleases takes a slice of releases and returns the latest patch
// release for each minor version, sorted in descending version order.
func (s *service) pickBestReleases(raw []Release) []Release {
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
