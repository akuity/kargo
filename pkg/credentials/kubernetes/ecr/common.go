package ecr

import (
	"crypto/sha256"
	"fmt"
	"regexp"
)

// ecrURLRegex is a regex that matches ECR URLs.
var ecrURLRegex = regexp.MustCompile(`^(?:oci://)?[0-9]{12}\.dkr\.ecr\.(.+)\.amazonaws\.com/`)

// tokenCacheKey returns a cache key in the form of a hash for the given parts.
// Using a hash ensures that any sensitive data is not stored in a decodable
// form.
func tokenCacheKey(parts ...string) string {
	const separator = ":"
	h := sha256.New()
	for i := range parts {
		if i > 0 {
			_, _ = h.Write([]byte(separator))
		}
		_, _ = h.Write([]byte(parts[i]))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
