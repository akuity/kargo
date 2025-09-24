package image

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/urls"
)

// baseSelector is a base implementation of Selector that provides common
// functionality for all Selector implementations. It is not intended to be used
// directly.
type baseSelector struct {
	platform   *platformConstraint
	repoClient *repositoryClient
}

func newBaseSelector(
	sub kargoapi.ImageSubscription,
	creds *Credentials,
) (*baseSelector, error) {
	var err error
	s := &baseSelector{}
	if sub.Platform != "" {
		if s.platform, err = parsePlatformConstraint(sub.Platform); err != nil {
			return nil, fmt.Errorf(
				"error parsing platform constraint %q: %w",
				sub.Platform, err,
			)
		}
	}
	repoURL := urls.NormalizeImage(sub.RepoURL)
	if s.repoClient, err = newRepositoryClient(
		repoURL,
		sub.InsecureSkipTLSVerify,
		creds,
	); err != nil {
		return nil, fmt.Errorf(
			"error creating repository client for image %q: %w",
			repoURL,
			err,
		)
	}
	return s, nil
}

// getLoggerContext returns key/value pairs that can be used by any selector to
// enrich loggers with valuable context.
func (b *baseSelector) getLoggerContext() []any {
	return []any{
		"registry", b.repoClient.registry.name,
		"image", b.repoClient.repoURL,
		"platformConstrained", b.platform != nil,
	}
}

// imagesToAPIImages converts a slice of internal image to a slice of
// kargoapi.DiscoveredImageReference, which can be directly used by a caller
// performing artifact discovery. If the number of tags provided exceeds the
// selector's discovery limit, the slice returned will be truncated so as not to
// exceed that limit.
func (b *baseSelector) imagesToAPIImages(
	images []image,
	limit int,
) []kargoapi.DiscoveredImageReference {
	if limit <= 0 || limit > len(images) {
		limit = len(images)
	}
	apiImages := make([]kargoapi.DiscoveredImageReference, limit)
	for i, img := range images[:limit] {
		apiImages[i] = kargoapi.DiscoveredImageReference{
			Tag:         img.Tag,
			Digest:      img.Digest,
			Annotations: img.Annotations,
		}
		if img.CreatedAt != nil {
			apiImages[i].CreatedAt = &metav1.Time{Time: *img.CreatedAt}
		}
	}
	return apiImages
}
