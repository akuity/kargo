package image

import (
	"time"

	"github.com/Masterminds/semver/v3"
)

// Image is a representation of a container image.
type Image struct {
	Tag         string
	Digest      string
	Annotations map[string]string
	CreatedAt   *time.Time

	semVer *semver.Version
}

// newImage initializes and returns an Image.
func newImage(tag, digest string, date *time.Time) Image {
	t := Image{
		Tag:       tag,
		Digest:    digest,
		CreatedAt: date,
	}
	// It's ok if the tag doesn't parse as semver, but if it does, store it
	if sv, err := semver.NewVersion(tag); err == nil {
		t.semVer = sv
	}
	return t
}
