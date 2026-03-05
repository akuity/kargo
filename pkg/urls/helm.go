package urls

import (
	"net/url"
	"strings"
)

// NormalizeChart normalizes a chart repository URL for purposes of comparison.
// Crucially, this function removes the oci:// prefix from the URL if there is
// one.
func NormalizeChart(repo string) string {
	ogRepo := repo
	repo = SanitizeURL(repo)
	// just to check validity
	if _, err := url.Parse(repo); err != nil {
		return ogRepo
	}
	// Note: We lean a bit on image.NormalizeURL() because it is excellent at
	// normalizing the many different forms of equivalent URLs for Docker Hub
	// repositories.
	return NormalizeImage(
		strings.TrimPrefix(
			strings.ToLower(repo),
			"oci://",
		),
	)
}
