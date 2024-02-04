package image

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// platformConstraint represents an operating system, system architecture, and
// (optionally) variant thereof that can be used to filter images by platform.
type platformConstraint struct {
	os      string
	arch    string
	variant string
}

// String implements fmt.Stringer.
func (p *platformConstraint) String() string {
	if p.variant == "" {
		return fmt.Sprintf("%s/%s", p.os, p.arch)
	}
	return fmt.Sprintf("%s/%s/%s", p.os, p.arch, p.variant)
}

// ValidatePlatformConstraint returns a boolean indicating whether the provided
// platform constraint string is valid.
func ValidatePlatformConstraint(platformStr string) bool {
	_, err := parsePlatformConstraint(platformStr)
	return err == nil
}

// parsePlatformConstraint parses a the provided platform constraint string
// and returns a platformConstraint struct.
func parsePlatformConstraint(platformStr string) (platformConstraint, error) {
	tokens := strings.SplitN(platformStr, "/", 3)
	if len(tokens) < 2 {
		return platformConstraint{},
			errors.Errorf("error parsing platform constraint %q", platformStr)
	}
	platform := platformConstraint{
		os:   tokens[0],
		arch: tokens[1],
	}
	if len(tokens) == 3 {
		platform.variant = tokens[2]
	}
	return platform, nil
}

// matches returns a boolean indicating whether the provided operating system,
// system architecture, and variant satisfy the platform constraint.
func (p *platformConstraint) matches(os, arch, variant string) bool {
	return p.os == os &&
		p.arch == arch &&
		p.variant == variant
}
