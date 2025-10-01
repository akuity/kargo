package gar

import (
	"crypto/sha256"
	"fmt"
	"regexp"
)

const accessTokenUsername = "oauth2accesstoken"

var (
	// TODO(krancour): Repo URLs are not currently normalized prior to credential
	// providers being invoked. When that is fixed, the optional leading `oci://`
	// can be removed from these regular expressions.
	gcrURLRegex = regexp.MustCompile(`^(?:oci://)?(?:.+\.)?gcr\.io/`) // Legacy
	garURLRegex = regexp.MustCompile(`^(?:oci://)?(.+-docker\.pkg\.dev/)`)
)

func tokenCacheKey(key string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
}
