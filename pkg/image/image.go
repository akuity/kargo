package image

import (
	"maps"
	"time"

	"github.com/Masterminds/semver/v3"
)

// image is a representation of a container image.
type image struct {
	Tag         string            `json:"tag,omitempty"`
	Digest      string            `json:"digest,omitempty"`
	CreatedAt   *time.Time        `json:"createdAt,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Platforms   []platform        `json:"platforms,omitempty"`
	Semver      *semver.Version   `json:"semver,omitempty"`
}

// newImage initializes and returns an image.
func newImage(tag, digest string, date *time.Time) image {
	t := image{
		Tag:       tag,
		Digest:    digest,
		CreatedAt: date,
	}
	// It's ok if the tag doesn't parse as semver, but if it does, store it
	if sv, err := semver.NewVersion(tag); err == nil {
		t.Semver = sv
	}
	return t
}

func (i *image) getPlatform(pc platformConstraint) *platform {
	for _, p := range i.Platforms {
		if pc.matches(p.OS, p.Arch, p.Variant) {
			return &p
		}
	}
	return nil
}

func (i *image) getAnnotations(pc *platformConstraint) map[string]string {
	annotations := maps.Clone(i.Annotations)
	if pc == nil {
		return annotations
	}
	p := i.getPlatform(*pc)
	if p == nil {
		return annotations
	}
	if annotations == nil {
		annotations = make(map[string]string, len(p.Annotations))
	}
	for k, v := range p.Annotations {
		annotations[k] = v
	}
	return annotations
}
