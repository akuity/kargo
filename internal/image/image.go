package image

import (
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/opencontainers/go-digest"
)

// Image is a representation of a container image.
type Image struct {
	Tag       string
	Digest    digest.Digest
	CreatedAt *time.Time
	semVer    *semver.Version
}

// newImage initializes and returns an Image.
func newImage(tag string, date *time.Time, digest digest.Digest) Image {
	t := Image{
		Tag:       tag,
		CreatedAt: date,
		Digest:    digest,
	}
	// It's ok if the tag doesn't parse as semver, but if it does, store it
	if sv, err := semver.NewVersion(tag); err == nil {
		t.semVer = sv
	}
	return t
}
