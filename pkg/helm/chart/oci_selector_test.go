package chart

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
	"oras.land/oras-go/v2/registry/remote"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewOCISelector(t *testing.T) {
	testCases := []struct {
		name       string
		sub        kargoapi.ChartSubscription
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
			name: "error parsing repo url",
			sub: kargoapi.ChartSubscription{
				RepoURL: "https://charts.example.com", // Not OCI
			},
			assertions: func(t *testing.T, _ Selector, err error) {
				require.ErrorContains(t, err, "error parsing repository URL")
			},
		},
		{
			name: "success",
			sub: kargoapi.ChartSubscription{
				RepoURL: "oci://charts.example.com/repo",
				Name:    "my-chart",
			},
			assertions: func(t *testing.T, s Selector, err error) {
				require.NoError(t, err)
				o, ok := s.(*ociSelector)
				require.True(t, ok)
				require.NotNil(t, o.baseSelector)
				require.NotNil(t, o.repo)
			},
		},
		{
			name: "insecureSkipTLSVerify propagated to authorizer",
			sub: kargoapi.ChartSubscription{
				RepoURL:               "oci://charts.example.com/repo",
				InsecureSkipTLSVerify: true,
			},
			assertions: func(t *testing.T, s Selector, err error) {
				require.NoError(t, err)
				o, ok := s.(*ociSelector)
				require.True(t, ok)
				require.True(t, o.insecureSkipTLSVerify)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := newOCISelector(t.Context(), testCase.sub, nil)
			testCase.assertions(t, s, err)
		})
	}
}

func TestOCISelectorAnnotations(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		version     string
		tag         string
		manifest    string
		status      int
		annotations map[string]string
		errContains string
	}{
		{
			name:    "OCI and custom annotations",
			version: "1.2.3",
			tag:     "1.2.3",
			manifest: `{"schemaVersion":2,"annotations":{` +
				`"org.opencontainers.image.revision":"abc123",` +
				`"example.com/release-notes":"https://example.com/releases/1.2.3"}}`,
			annotations: map[string]string{
				"org.opencontainers.image.revision": "abc123",
				"example.com/release-notes":         "https://example.com/releases/1.2.3",
			},
		},
		{
			name:     "no annotations",
			version:  "1.2.3",
			tag:      "1.2.3",
			manifest: `{"schemaVersion":2}`,
		},
		{
			name:        "SemVer build metadata encoded in tag",
			version:     "1.2.3+build.4",
			tag:         "1.2.3_build.4",
			manifest:    `{"schemaVersion":2,"annotations":{"build":"4"}}`,
			annotations: map[string]string{"build": "4"},
		},
		{
			name:        "manifest unavailable",
			version:     "1.2.3",
			tag:         "1.2.3",
			status:      http.StatusNotFound,
			errContains: "error fetching manifest",
		},
		{
			name:        "invalid manifest JSON",
			version:     "1.2.3",
			tag:         "1.2.3",
			manifest:    `not json`,
			errContains: "error decoding manifest",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Any chart layer/config fetch would fail this test.
				if r.URL.Path != "/v2/charts/app/manifests/"+testCase.tag {
					t.Errorf("unexpected registry request: %s %s", r.Method, r.URL.Path)
					http.NotFound(w, r)
					return
				}
				if testCase.status != 0 {
					w.WriteHeader(testCase.status)
					return
				}
				w.Header().Set("Content-Type", ocispec.MediaTypeImageManifest)
				w.Header().Set("Docker-Content-Digest", digest.FromString(testCase.manifest).String())
				w.Header().Set("Content-Length", fmt.Sprint(len(testCase.manifest)))
				_, err := w.Write([]byte(testCase.manifest))
				if err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			repo, err := remote.NewRepository(strings.TrimPrefix(server.URL, "http://") + "/charts/app")
			require.NoError(t, err)
			repo.PlainHTTP = true
			selector := &ociSelector{baseSelector: &baseSelector{repoURL: repo.Reference.String()}, repo: repo}
			annotations, err := selector.Annotations(t.Context(), testCase.version)
			if testCase.errContains != "" {
				require.ErrorContains(t, err, testCase.errContains)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.annotations, annotations)
		})
	}
}

func Test_ociSelector_Select(t *testing.T) {
	// Instead of mocking out an OCI registry, it's more expedient to use Kargo's
	// own chart repo on ghcr.io to test this.
	s, err := newOCISelector(
		t.Context(),
		kargoapi.ChartSubscription{
			RepoURL: "oci://ghcr.io/akuity/kargo-charts/kargo",
		},
		nil,
	)
	require.NoError(t, err)
	versions, err := s.Select(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, versions)
}
