package external

import (
	"slices"

	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/image"
)

var containerImageMediaTypes = []string{
	"application/vnd.oci.image.index.v1+json",
	"application/vnd.oci.image.manifest.v1+json",
	"application/vnd.docker.distribution.manifest.list.v2+json",
	"application/vnd.docker.distribution.manifest.v2+json",
}

func isContainerImageMediaType(mediaType string) bool {
	return slices.Contains(containerImageMediaTypes, mediaType)
}

func isHelmChartMediaType(mediaType string) bool {
	return mediaType == "application/vnd.cncf.helm.config.v1+json"
}

// normalizeOCIRepoURL returns a normalized representation the specified OCI
// repository URL based on the specified media type. If the media type is not
// recognized, an empty string is returned.
func normalizeOCIRepoURL(repoURL, mediaType string) string {
	switch {
	case isContainerImageMediaType(mediaType):
		return image.NormalizeURL(repoURL)
	case isHelmChartMediaType(mediaType):
		return helm.NormalizeChartRepositoryURL(repoURL)
	}
	return ""
}
