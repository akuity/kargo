package external

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/urls"
)

func TestGetNormalizedImageRepoURLs(t *testing.T) {
	testCases := []struct {
		name       string
		repoURL    string
		mediaType  string
		assertions func(t *testing.T, repoURLs []string)
	}{
		{
			name:      "v2 manifest list",
			repoURL:   "example/repo",
			mediaType: dockerManifestListMediaType,
			assertions: func(t *testing.T, repoURLs []string) {
				require.Equal(t, []string{urls.NormalizeImage("example/repo")}, repoURLs)
			},
		},
		{
			name:      "OCI manifest index",
			repoURL:   "ghcr.io/example/repo",
			mediaType: ociImageIndexMediaType,
			assertions: func(t *testing.T, repoURLs []string) {
				require.Equal(
					t,
					[]string{urls.NormalizeImage("ghcr.io/example/repo")},
					repoURLs,
				)
			},
		},
		{
			name:      "v2 manifest",
			repoURL:   "example/repo",
			mediaType: dockerManifestMediaType,
			assertions: func(t *testing.T, repoURLs []string) {
				require.Equal(t, []string{urls.NormalizeImage("example/repo")}, repoURLs)
			},
		},
		{
			name:      "OCI manifest",
			repoURL:   "ghcr.io/example/repo",
			mediaType: ociImageManifestMediaType,
			assertions: func(t *testing.T, repoURLs []string) {
				// Current image URL and chart URL normalization logic yield the same
				// result for this input, so we expect that getNormalizedImageRepoURLs()
				// will have compacted the results.
				require.Len(t, repoURLs, 1)
				require.Equal(t, urls.NormalizeImage("ghcr.io/example/repo"), repoURLs[0])
				require.Equal(
					t,
					urls.NormalizeChart("ghcr.io/example/repo"),
					repoURLs[0],
				)
			},
		},
		{
			name:      "v2 image config blob",
			repoURL:   "example/repo",
			mediaType: dockerImageConfigBlobMediaType,
			assertions: func(t *testing.T, repoURLs []string) {
				require.Equal(t, []string{urls.NormalizeImage("example/repo")}, repoURLs)
			},
		},
		{
			name:      "OCI image config blob",
			repoURL:   "ghcr.io/example/repo",
			mediaType: ociImageConfigBlobMediaType,
			assertions: func(t *testing.T, repoURLs []string) {
				require.Equal(
					t,
					[]string{urls.NormalizeImage("ghcr.io/example/repo")},
					repoURLs,
				)
			},
		},
		{
			name:      "Helm chart config blob",
			repoURL:   "ghcr.io/example/repo",
			mediaType: helmChartConfigBlobMediaType,
			assertions: func(t *testing.T, repoURLs []string) {
				require.Equal(
					t,
					[]string{urls.NormalizeChart("ghcr.io/example/repo")},
					repoURLs,
				)
			},
		},
		{
			name:      "something completely unexpected",
			repoURL:   "ghcr.io/example/repo",
			mediaType: "nonsense",
			assertions: func(t *testing.T, repoURLs []string) {
				// Current image URL and chart URL normalization logic yield the same
				// result for this input, so we expect that getNormalizedImageRepoURLs()
				// will have compacted the results.
				require.Len(t, repoURLs, 1)
				require.Equal(
					t,
					urls.NormalizeImage("ghcr.io/example/repo"),
					repoURLs[0],
				)
				require.Equal(
					t,
					urls.NormalizeChart("ghcr.io/example/repo"),
					repoURLs[0],
				)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.assertions(t, getNormalizedImageRepoURLs(tc.repoURL, tc.mediaType))
		})
	}
}
