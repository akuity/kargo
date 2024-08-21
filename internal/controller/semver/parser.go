package semver

import (
	"strings"

	"github.com/Masterminds/semver/v3"
)

func Parse(s string, strict bool) *semver.Version {
	// We do the non-strict parsing first, because it will ensure that
	// sv.Original() returns the original string from which we have not
	// potentially stripped a leading "v".
	sv, err := semver.NewVersion(s)
	if err != nil {
		return nil // tag wasn't a semantic version
	}
	if strict {
		if _, err = semver.StrictNewVersion(strings.TrimPrefix(s, "v")); err != nil {
			return nil // tag wasn't a strict semantic version
		}
	}
	return sv
}
