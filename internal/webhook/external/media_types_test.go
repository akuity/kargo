package external

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/image"
)

func TestGetNormalizedImageRepoURLs(t *testing.T) {
	testCases := []struct {
		name      string
		repoURL   string
		mediaType string
		expected  []string
	}{
		{
			name:      "v2 manifest list",
			repoURL:   "example/repo",
			mediaType: dockerManifestListMediaType,
			expected:  []string{image.NormalizeURL("example/repo")},
		},
		{
			name:      "OCI manifest index",
			repoURL:   "ghcr.io/example/repo",
			mediaType: ociImageIndexMediaType,
			expected:  []string{image.NormalizeURL("ghcr.io/example/repo")},
		},
		{
			name:      "v2 manifest",
			repoURL:   "example/repo",
			mediaType: dockerManifestMediaType,
			expected:  []string{image.NormalizeURL("example/repo")},
		},
		{
			name:      "OCI manifest",
			repoURL:   "ghcr.io/example/repo",
			mediaType: ociImageManifestMediaType,
			expected: []string{
				image.NormalizeURL("ghcr.io/example/repo"),
				helm.NormalizeChartRepositoryURL("ghcr.io/example/repo"),
			},
		},
		{
			name:      "v2 image config blob",
			repoURL:   "example/repo",
			mediaType: dockerImageConfigBlobMediaType,
			expected:  []string{image.NormalizeURL("example/repo")},
		},
		{
			name:      "OCI image config blob",
			repoURL:   "ghcr.io/example/repo",
			mediaType: ociImageConfigBlobMediaType,
			expected:  []string{image.NormalizeURL("ghcr.io/example/repo")},
		},
		{
			name:      "Helm chart config blob",
			repoURL:   "ghcr.io/example/repo",
			mediaType: helmChartConfigBlobMediaType,
			expected:  []string{helm.NormalizeChartRepositoryURL("ghcr.io/example/repo")},
		},
		{
			name:      "something completely unexpected",
			repoURL:   "ghcr.io/example/repo",
			mediaType: "nonsense",
			expected: []string{
				image.NormalizeURL("ghcr.io/example/repo"),
				helm.NormalizeChartRepositoryURL("ghcr.io/example/repo"),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			urls := getNormalizedImageRepoURLs(tc.repoURL, tc.mediaType)
			require.Equal(t, tc.expected, urls)
		})
	}
}
