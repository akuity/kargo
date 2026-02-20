package releases

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_GetBestReleases(t *testing.T) {
	t.Run("returns nil when no cached data", func(t *testing.T) {
		s := &service{}
		result := s.GetBestReleases()
		assert.Nil(t, result)
	})

	t.Run("returns cached data", func(t *testing.T) {
		s := &service{}
		data := []Release{{Version: v("1.0.0")}}
		s.cached.Store(&data)

		result := s.GetBestReleases()
		require.Len(t, result, 1)
		assert.True(t, v("1.0.0").Equal(result[0].Version))
	})
}

func TestService_fetchReleases(t *testing.T) {
	type rawAsset struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	}
	type rawRelease struct {
		TagName string     `json:"tag_name"`
		Assets  []rawAsset `json:"assets"`
	}

	t.Run("paginates and returns best releases", func(t *testing.T) {
		const perPage = 100
		asset := rawAsset{Name: "kargo-linux-amd64", BrowserDownloadURL: "https://example.com/dl"}

		page1 := make([]rawRelease, perPage)
		for i := range page1 {
			page1[i] = rawRelease{
				TagName: fmt.Sprintf("v1.0.%d", i),
				Assets:  []rawAsset{asset},
			}
		}
		page2 := []rawRelease{
			{
				TagName: "v1.1.0",
				Assets:  []rawAsset{asset},
			},
		}

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			page := r.URL.Query().Get("page")
			var data any
			switch page {
			case "1", "":
				data = page1
			case "2":
				data = page2
			default:
				data = []rawRelease{}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(data) //nolint: errcheck
		}))
		defer srv.Close()

		s := &service{baseURL: srv.URL, httpClient: srv.Client()}
		results, err := s.fetchReleases(context.Background())
		require.NoError(t, err)

		// page1 has v1.0.0 through v1.0.99 (latest patch = v1.0.99)
		// page2 has v1.1.0
		require.Len(t, results, 2)
		assert.True(t, v("1.1.0").Equal(results[0].Version))
		assert.True(t, v("1.0.99").Equal(results[1].Version))
	})

	t.Run("HTTP error returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer srv.Close()

		s := &service{baseURL: srv.URL, httpClient: srv.Client()}
		results, err := s.fetchReleases(context.Background())
		require.Error(t, err)
		assert.Nil(t, results)
		assert.Contains(t, err.Error(), "403")
	})
}

func TestService_pickBestReleases(t *testing.T) {
	t.Run("nil input returns empty", func(t *testing.T) {
		results := (&service{}).pickBestReleases(nil)
		assert.Empty(t, results)
	})

	t.Run("picks latest patch per minor, sorted descending", func(t *testing.T) {
		raw := []Release{
			{
				Version:     v("1.0.0"),
				CLIBinaries: CLIBinaries{"linux": {"amd64": "https://example.com/v1.0.0/kargo-linux-amd64"}},
			},
			{
				Version:     v("1.0.1"),
				CLIBinaries: CLIBinaries{"linux": {"amd64": "https://example.com/v1.0.1/kargo-linux-amd64"}},
			},
			{
				Version:     v("1.0.2"),
				CLIBinaries: CLIBinaries{"linux": {"amd64": "https://example.com/v1.0.2/kargo-linux-amd64"}},
			},
			{
				Version:     v("1.1.0"),
				CLIBinaries: CLIBinaries{"linux": {"amd64": "https://example.com/v1.1.0/kargo-linux-amd64"}},
			},
			{
				Version:     v("1.1.1"),
				CLIBinaries: CLIBinaries{"linux": {"amd64": "https://example.com/v1.1.1/kargo-linux-amd64"}},
			},
			{
				Version:     v("2.0.0"),
				CLIBinaries: CLIBinaries{"linux": {"amd64": "https://example.com/v2.0.0/kargo-linux-amd64"}},
			},
		}

		results := (&service{}).pickBestReleases(raw)

		require.Len(t, results, 3)
		assert.True(t, v("2.0.0").Equal(results[0].Version))
		assert.True(t, v("1.1.1").Equal(results[1].Version))
		assert.True(t, v("1.0.2").Equal(results[2].Version))
	})
}
