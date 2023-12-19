package helm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetChartVersionsFromClassicRegistry(t *testing.T) {
	// This is a mock registry. Depending on the request path, it returns a 404,
	// invalid YAML, or valid YAML.
	testServer := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				defer r.Body.Close()
				switch r.URL.Path {
				case "/bad-registry/index.yaml":
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte("this isn't yaml"))
					require.NoError(t, err)
				case "/fake-registry/index.yaml":
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
		name        string
		registryURL string
		chart       string
		assertions  func(versions []string, err error)
	}{
		{
			name:        "request for registry index returns non-200 status",
			registryURL: fmt.Sprintf("%s/non-existent-registry", testServer.URL),
			chart:       "fake-chart",
			assertions: func(versions []string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "received unexpected HTTP 404")
			},
		},
		{
			name:        "index isn't valid YAML",
			registryURL: fmt.Sprintf("%s/bad-registry", testServer.URL),
			chart:       "fake-chart",
			assertions: func(versions []string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error unmarshaling registry index")
			},
		},
		{
			name:        "no versions found",
			registryURL: fmt.Sprintf("%s/fake-registry", testServer.URL),
			chart:       "non-existent-chart",
			assertions: func(versions []string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "no versions of chart")
				require.Contains(t, err.Error(), "found in registry index")
			},
		},
		{
			name:        "success",
			registryURL: fmt.Sprintf("%s/fake-registry", testServer.URL),
			chart:       "fake-chart",
			assertions: func(versions []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"1.0.0", "1.1.0", "1.2.0"}, versions)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				getChartVersionsFromClassicRegistry(
					testCase.registryURL,
					testCase.chart,
					nil,
				),
			)
		})
	}
}

func TestGetChartVersionsFromOCIRegistry(t *testing.T) {
	// Instead of mocking out an OCI registry, it's more expedient to use Kargo's
	// own chart repo on ghcr.io to test this.
	testCases := []struct {
		registryURL string
		chart       string
	}{
		// These test cases validate that Kargo is forgiving in terms of what the
		// user provides for registryURL and for chart.
		{
			registryURL: "oci://ghcr.io",
			chart:       "akuity/kargo-charts/kargo",
		},
		{
			registryURL: "oci://ghcr.io/akuity",
			chart:       "kargo-charts/kargo",
		},
		{
			registryURL: "oci://ghcr.io/akuity/kargo-charts",
			chart:       "kargo",
		},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			versions, err := getChartVersionsFromOCIRegistry(
				context.Background(),
				testCase.registryURL,
				testCase.chart,
				nil,
			)
			require.NoError(t, err)
			require.NotEmpty(t, versions)
		})
	}
}

func TestGetLatestVersion(t *testing.T) {
	testCases := []struct {
		name       string
		unsorted   []string
		constraint string
		assertions func(latest string, err error)
	}{
		{
			name:     "error parsing versions",
			unsorted: []string{"not-semantic"},
			assertions: func(_ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error parsing version")
			},
		},
		{
			name:       "error parsing constraint",
			unsorted:   []string{"1.0.0"},
			constraint: "invalid",
			assertions: func(_ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error parsing constraint")
			},
		},
		{
			name:       "success with constraint",
			unsorted:   []string{"2.0.0", "1.0.0", "1.1.0"},
			constraint: "^1.0.0",
			assertions: func(latest string, err error) {
				require.NoError(t, err)
				require.Equal(t, "1.1.0", latest)
			},
		},
		{
			name:     "success with no constraint",
			unsorted: []string{"2.0.0", "1.0.0", "1.1.0"},
			assertions: func(latest string, err error) {
				require.NoError(t, err)
				require.Equal(t, "2.0.0", latest)
			},
		},
		{
			name:       "success with no constraint",
			unsorted:   []string{"2.0.0", "1.0.0", "1.1.0"},
			constraint: "^3.0.0",
			assertions: func(latest string, err error) {
				require.NoError(t, err)
				require.Equal(t, "", latest)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				getLatestVersion(testCase.unsorted, testCase.constraint),
			)
		})
	}
}
