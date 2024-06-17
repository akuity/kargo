package gar

import "regexp"

const accessTokenUsername = "oauth2accesstoken"

var (
	gcrURLRegex = regexp.MustCompile(`^(?:.+\.)?gcr\.io/`) // Legacy
	garURLRegex = regexp.MustCompile(`^.+-docker\.pkg\.dev/`)
)
