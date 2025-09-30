package chart

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/helm"
)

func TestNewHTTPSelector(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.ChartSubscription
		creds      *helm.Credentials
		assertions func(*testing.T, Selector, error)
	}{
		{
			name: "error building base selector",
			sub: kargoapi.ChartSubscription{
				SemverConstraint: "invalid", // This will force an error
			},
			assertions: func(t *testing.T, _ Selector, err error) {
				require.ErrorContains(t, err, "error building base selector")
			},
		},
		{
			name: "success",
			sub: kargoapi.ChartSubscription{
				RepoURL: "https://charts.example.com",
				Name:    "my-chart",
			},
			creds: &helm.Credentials{
				Username: "foo",
				Password: "bar",
			},
			assertions: func(t *testing.T, s Selector, err error) {
				require.NoError(t, err)
				h, ok := s.(*httpSelector)
				require.True(t, ok)
				require.NotNil(t, h.baseSelector)
				require.True(
					t,
					strings.HasPrefix(h.indexURL, "https://charts.example.com"),
				)
				require.Equal(t, "my-chart", h.chartName)
				require.Equal(
					t,
					&helm.Credentials{
						Username: "foo",
						Password: "bar",
					},
					h.creds,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newHTTPSelector(testCase.sub, testCase.creds)
			testCase.assertions(t, s, err)
		})
	}
}

func Test_httpSelector_Select(t *testing.T) {
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
				require.ErrorContains(t, err, "received unexpected HTTP 404")
			},
		},
		{
			name:    "index isn't valid YAML",
			repoURL: fmt.Sprintf("%s/bad-repo", testServer.URL),
			chart:   "fake-chart",
			assertions: func(t *testing.T, _ []string, err error) {
				require.ErrorContains(t, err, "error unmarshaling repository index")
			},
		},
		{
			name:    "no versions found",
			repoURL: fmt.Sprintf("%s/fake-repo", testServer.URL),
			chart:   "non-existent-chart",
			assertions: func(t *testing.T, _ []string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:    "success",
			repoURL: fmt.Sprintf("%s/fake-repo", testServer.URL),
			chart:   "fake-chart",
			assertions: func(t *testing.T, versions []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"1.2.0", "1.1.0", "1.0.0"}, versions)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s := &httpSelector{
				baseSelector: &baseSelector{repoURL: testCase.repoURL},
				indexURL: fmt.Sprintf(
					"%s/index.yaml",
					strings.TrimSuffix(testCase.repoURL, "/"),
				),
				chartName: testCase.chart,
			}
			versions, err := s.Select(context.Background())
			testCase.assertions(t, versions, err)
		})
	}
}
