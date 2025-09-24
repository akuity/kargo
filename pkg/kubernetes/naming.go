package kubernetes

import "github.com/akuity/kargo/pkg/strings"

// WARNING: Do not change this. See comment on strings.HashShorten().
const shortHashLen = 8

// ShortenResourceName deterministically shortens the provided name to the
// maximum allowed length for the name of a Kubernetes resource by retaining as
// many of the leading characters as possible and replacing as many trailing
// characters as necessary with a short hash of the entire input. The preserved
// characters of the input name and the short hash will be separated by a dash.
// If the length of the input name is already less than or equal to the maximum
// allowed length, then the original name is returned as is.
func ShortenResourceName(name string) string {
	shortName, _ := strings.HashShorten(name, 253, "-", shortHashLen)
	return shortName
}

// ShortenLabelValue deterministically shortens the provided string value to the
// maximum allowed length for the value of a Kubernetes label by retaining
// as many of the leading characters as possible and replacing as many trailing
// characters as necessary with a short hash of the entire input. The preserved
// characters of the input value and the short hash will be separated by a dash.
// If the length of the input name is already less than or equal to the maximum
// allowed length, then the original value is returned as is.
func ShortenLabelValue(value string) string {
	shortName, _ := strings.HashShorten(value, 63, "-", shortHashLen)
	return shortName
}
