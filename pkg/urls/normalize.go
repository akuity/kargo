package urls

type UrlType string

const (
	UrlTypeGit   = "git"
	UrlTypeImage = "image"
	UrlTypeChart = "chart"
)

// Normalize normalizes a URL based on its type.
// If the UrlType is unrecognized, the original URL is returned.
func Normalize(t UrlType, url string) string {
	switch t {
	case UrlTypeGit:
		return NormalizeGit(url)
	case UrlTypeImage:
		return NormalizeImage(url)
	case UrlTypeChart:
		return NormalizeChart(url)
	default:
		return url
	}
}
