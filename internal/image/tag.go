package image

import (
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/opencontainers/go-digest"
)

// Tag is a representation of a container image Tag.
type Tag struct {
	Name      string
	Digest    digest.Digest
	CreatedAt *time.Time
	semVer    *semver.Version
}

// newTag initializes and returns a tag.
func newTag(name string, date *time.Time, digest digest.Digest) Tag {
	t := Tag{
		Name:      name,
		CreatedAt: date,
		Digest:    digest,
	}
	// It's ok if the tag doesn't parse as semver, but if it does, store it
	if sv, err := semver.NewVersion(name); err == nil {
		t.semVer = sv
	}
	return t
}
