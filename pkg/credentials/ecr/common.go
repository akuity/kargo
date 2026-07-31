package ecr

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"time"
)

const (
	tokenCacheExpiryMargin = 5 * time.Minute

	// tokenAcquisitionTimeout bounds a single token acquisition. Because an
	// acquisition executes under a context detached from any caller's, this is
	// the only thing bounding its duration. It is generous because it serves only
	// as a fail-safe: exceeding it fails every caller waiting on that
	// acquisition, and no fresh acquisition for the same key can begin until it
	// returns.
	tokenAcquisitionTimeout = 30 * time.Second
)

// ecrURLRegex is a regex that matches ECR URLs.
var ecrURLRegex = regexp.MustCompile(`^[0-9]{12}\.dkr\.ecr\.(.+)\.amazonaws\.com/`)

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
