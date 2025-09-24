package external

import (
	"slices"
	"strings"

	"github.com/akuity/kargo/pkg/urls"
)

const (
	ociImageIndexMediaType      = "application/vnd.oci.image.index.v1+json"
	ociImageManifestMediaType   = "application/vnd.oci.image.manifest.v1+json"
	ociImageConfigBlobMediaType = "application/vnd.oci.image.config.v1+json"

	dockerManifestListMediaType    = "application/vnd.docker.distribution.manifest.list.v2+json"
	dockerManifestMediaType        = "application/vnd.docker.distribution.manifest.v2+json"
	dockerImageConfigBlobMediaType = "application/vnd.docker.container.image.v1+json"

	helmChartConfigBlobMediaType = "application/vnd.cncf.helm.config.v1+json"
)

// probableContainerImageMediaTypes is a list of media types that, for our
// purposes, are a dead giveaway that the URL they are associated with
// represents a container image. Note this list deliberately omits
// application/vnd.oci.image.manifest.v1+json because, for our purposes, it
// does not definitively establish whether the URL associated with it represents
// a container image, a Helm chart, or something else entirely.
var probableContainerImageMediaTypes = []string{
	ociImageIndexMediaType,
	dockerManifestListMediaType,
	dockerManifestMediaType,
	ociImageConfigBlobMediaType,
	dockerImageConfigBlobMediaType,
}

// getNormalizedImageRepoURLs returns exactly one or two normalized
// representations of the specified repository URL based on whatever, if
// anything, can be inferred from the specified media type.
//
// In this context, "image repo" could mean a repository in a Docker v2 registry
// OR a repository in an OCI registry. This function assumes that the provided
// repository URL falls into one of those two cases.
//
// Different webhook receivers invoking this function receive and pass on media
// type information of varying degrees of specificity, so no assumptions are
// made about what specific object the specified media type may be associated
// with. The specified media type could reasonably (but non-exhaustively) be
// that of any of the following:
//
//   - A v2 manifest list or an OCI manifest index. These commonly are
//     indicative of a multi-arch container image and imply the specified URL
//     does not reference a chart repository.
//
//   - A v2 or OCI manifest. A v2 manifest is exclusively indicative of a
//     container image and implies the the specified URL references a container
//     image repository. An OCI manifest, however, is more ambiguous and does
//     not imply anything useful about what is referenced by the specified
//     repository URL. That distinction could only be made by examining the
//     mediate type of the config blob.
//
//   - A config blob. If the media type is readily recognizable as Helm-related,
//     it can be inferred that the specified URL references a Helm chart
//     repository. If it's readily recognizable as image-related, it can be
//     inferred that the specified URL references a container image repository.
//
//   - Something entirely unexpected.
//
// If it can be inferred that the repository URL is associated with a container
// image repository, the returned []string will contain only one item -- the
// specified URL normalized as if it represented a container image repository.
//
// If it can be inferred that the repository URL is associated with a Helm chart
// repository, the returned []string will contain only one item -- the specified
// URL normalized as if it represented a Helm chart repository.
//
// In all other cases where it cannot be inferred from the specified media type
// whether the specified URL references a container image repository or a
// Helm chart repository, the returned []string will contain two items -- the
// specified URL normalized both as if it represented a container image
// repository and as if it represented a Helm chart repository. It is
// exclusively the caller's responsibility to deal with this possibility.
func getNormalizedImageRepoURLs(repoURL, mediaType string) []string {
	if strings.HasPrefix(mediaType, "application/vnd.cncf.helm") {
		return []string{urls.NormalizeChart(repoURL)}
	}
	if slices.Contains(probableContainerImageMediaTypes, mediaType) {
		return []string{urls.NormalizeImage(repoURL)}
	}
	// The normalization process for image and chart URLs may change from time to
	// time. At times, the URL normalized as if it were an image URL and the URL
	// normalized as if it were a chart URL may turn out to be identical, in which
	// case, compacting the slice is a cheap, but worthwhile optimization.
	return slices.Compact([]string{
		urls.NormalizeImage(repoURL),
		urls.NormalizeChart(repoURL),
	})
}
