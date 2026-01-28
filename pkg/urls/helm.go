package urls

import (
	"strings"
)

// NormalizeChart normalizes a chart repository URL for purposes of comparison.
// Crucially, this function removes the oci:// prefix from the URL if there is
// one.
func NormalizeChart(repo string) string {
	repo = rmSpaces(repo)
	if repo == "" {
		return repo
	}
	// Note: We lean a bit on image.NormalizeURL() because it is excellent at
	// normalizing the many different forms of equivalent URLs for Docker Hub
	// repositories.
	return NormalizeImage(strings.TrimPrefix(repo, "oci://"))
}
