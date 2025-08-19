package helm

import (
	"strings"

	"github.com/akuity/kargo/internal/image"
)

// NormalizeChartRepositoryURL normalizes a chart repository URL for purposes
// of comparison. Crucially, this function removes the oci:// prefix from the
// URL if there is one.
func NormalizeChartRepositoryURL(repo string) string {
	// Note: We lean a bit on image.NormalizeURL() because it is excellent at
	// normalizing the many different forms of equivalent URLs for Docker Hub
	// repositories.
	return image.NormalizeURL(
		strings.TrimPrefix(
			strings.ToLower(
				strings.TrimSpace(repo),
			),
			"oci://",
		),
	)
}
