package external

import (
	"slices"

	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/image"
)

const (
	ociImageIndexMediaType    = "application/vnd.oci.image.index.v1+json"
	ociImageManifestMediaType = "application/vnd.oci.image.manifest.v1+json"

	dockerManifestListMediaType = "application/vnd.docker.distribution.manifest.list.v2+json"
	dockerManifestMediaType     = "application/vnd.docker.distribution.manifest.v2+json"

	helmChartMediaType = "application/vnd.cncf.helm.config.v1+json"
)

var containerImageMediaTypes = []string{
	ociImageIndexMediaType,
	ociImageManifestMediaType,
	dockerManifestListMediaType,
	dockerManifestMediaType,
}

func isContainerImageMediaType(mediaType string) bool {
	return slices.Contains(containerImageMediaTypes, mediaType)
}

func isHelmChartMediaType(mediaType string) bool {
	return mediaType == helmChartMediaType
}

// normalizeOCIRepoURL returns a normalized representation of the specified OCI
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
