package helm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetChartVersionsFromClassicRepo(t *testing.T) {
	// This is a mock registry. Depending on the request path, it returns a 404,
	// invalid YAML, or valid YAML.
	testServer := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				defer r.Body.Close()
				switch r.URL.Path {
				case "/bad-repo/index.yaml":
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte("this isn't yaml"))
					require.NoError(t, err)
				case "/fake-repo/index.yaml":
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`entries:
  fake-chart:
    - version: 1.0.0
    - version: 1.1.0
    - version: 1.2.0
`))
					require.NoError(t, err)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			},
		),
	)
	defer testServer.Close()
	testCases := []struct {
		name       string
		repoURL    string
		chart      string
		assertions func(t *testing.T, versions []string, err error)
	}{
		{
			name:    "request for repo index returns non-200 status",
			repoURL: fmt.Sprintf("%s/non-existent-repo", testServer.URL),
			chart:   "fake-chart",
			assertions: func(t *testing.T, _ []string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "received unexpected HTTP 404")
			},
		},
		{
			name:    "index isn't valid YAML",
			repoURL: fmt.Sprintf("%s/bad-repo", testServer.URL),
			chart:   "fake-chart",
			assertions: func(t *testing.T, _ []string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error unmarshaling repository index")
			},
		},
		{
			name:    "no versions found",
			repoURL: fmt.Sprintf("%s/fake-repo", testServer.URL),
			chart:   "non-existent-chart",
			assertions: func(t *testing.T, _ []string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "no versions of chart")
				require.Contains(t, err.Error(), "found in repository index")
			},
		},
		{
			name:    "success",
			repoURL: fmt.Sprintf("%s/fake-repo", testServer.URL),
			chart:   "fake-chart",
			assertions: func(t *testing.T, versions []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"1.0.0", "1.1.0", "1.2.0"}, versions)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			versions, err := getChartVersionsFromClassicRepo(
				testCase.repoURL,
				testCase.chart,
				nil,
			)
			testCase.assertions(t, versions, err)
		})
	}
}

func TestGetChartVersionsFromOCIRepo(t *testing.T) {
	// Instead of mocking out an OCI registry, it's more expedient to use Kargo's
	// own chart repo on ghcr.io to test this.
	versions, err := getChartVersionsFromOCIRepo(
		context.Background(),
		"oci://ghcr.io/akuity/kargo-charts/kargo",
		nil,
	)
	require.NoError(t, err)
	require.NotEmpty(t, versions)
}

func TestGetLatestVersion(t *testing.T) {
	testCases := []struct {
		name       string
		unsorted   []string
		constraint string
		assertions func(t *testing.T, latest string, err error)
	}{
		{
			name:     "error parsing versions",
			unsorted: []string{"not-semantic"},
			assertions: func(t *testing.T, latest string, err error) {
				require.NoError(t, err)
				require.Empty(t, latest)
			},
		},
		{
			name:     "success with invalid version ignored",
			unsorted: []string{"not-semantic", "1.0.0"},
			assertions: func(t *testing.T, latest string, err error) {
				require.NoError(t, err)
				require.Equal(t, "1.0.0", latest)
			},
		},
		{
			name:       "error parsing constraint",
			unsorted:   []string{"1.0.0"},
			constraint: "invalid",
			assertions: func(t *testing.T, _ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error parsing constraint")
			},
		},
		{
			name:       "success with constraint",
			unsorted:   []string{"2.0.0", "1.0.0", "1.1.0"},
			constraint: "^1.0.0",
			assertions: func(t *testing.T, latest string, err error) {
				require.NoError(t, err)
				require.Equal(t, "1.1.0", latest)
			},
		},
		{
			name:     "success with no constraint",
			unsorted: []string{"2.0.0", "1.0.0", "1.1.0"},
			assertions: func(t *testing.T, latest string, err error) {
				require.NoError(t, err)
				require.Equal(t, "2.0.0", latest)
			},
		},
		{
			name:       "success with no constraint",
			unsorted:   []string{"2.0.0", "1.0.0", "1.1.0"},
			constraint: "^3.0.0",
			assertions: func(t *testing.T, latest string, err error) {
				require.NoError(t, err)
				require.Equal(t, "", latest)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			latest, err := getLatestVersion(testCase.unsorted, testCase.constraint)
			testCase.assertions(t, latest, err)
		})
	}
}
