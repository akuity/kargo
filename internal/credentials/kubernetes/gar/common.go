package gar

import (
	"crypto/sha256"
	"fmt"
	"regexp"
)

const accessTokenUsername = "oauth2accesstoken"

var (
	gcrURLRegex = regexp.MustCompile(`^(?:.+\.)?gcr\.io/`) // Legacy
	garURLRegex = regexp.MustCompile(`^.+-docker\.pkg\.dev/`)
)

func tokenCacheKey(key string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
}
