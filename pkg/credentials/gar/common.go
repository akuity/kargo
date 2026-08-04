package gar

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"time"
)

const (
	accessTokenUsername    = "oauth2accesstoken"
	tokenCacheExpiryMargin = 5 * time.Minute

	// tokenAcquisitionTimeout bounds a single token acquisition. Because an
	// acquisition executes under a context detached from any caller's, nothing
	// but this bounds how long one may run overall. It is generous because it
	// serves only as a fail-safe: exceeding it fails every caller waiting on that
	// acquisition, and no fresh acquisition for the same key can begin until it
	// returns. It cannot bound an individual request that ignores its context,
	// which is what tokenRequestTimeout is for.
	tokenAcquisitionTimeout = 30 * time.Second

	// tokenRequestTimeout bounds an individual HTTP request made while acquiring
	// a token. It exists because oauth2 resolves its HTTP client from a context
	// but then issues the request without one, leaving the acquisition's own
	// deadline with nothing to act on. A request that is never answered would
	// otherwise keep the acquisition running, and with it the singleflight key it
	// occupies, failing every later caller for that key.
	tokenRequestTimeout = 30 * time.Second
)

var (
	gcrURLRegex = regexp.MustCompile(`^(?:.+\.)?gcr\.io/`) // Legacy
	garURLRegex = regexp.MustCompile(`^.+-docker\.pkg\.dev/`)
)

func tokenCacheKey(key string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
}
