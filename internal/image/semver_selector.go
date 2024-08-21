package image

import (
	"context"
	"fmt"
	"sort"

	"github.com/Masterminds/semver/v3"

	libSemver "github.com/akuity/kargo/internal/controller/semver"
	"github.com/akuity/kargo/internal/logging"
)

// semVerSelector implements the Selector interface for SelectionStrategySemVer.
type semVerSelector struct {
	repoClient *repositoryClient
	opts       SelectorOptions
	constraint *semver.Constraints
}

// newSemVerSelector returns an implementation of the Selector interface for
// SelectionStrategySemVer.
func newSemVerSelector(repoClient *repositoryClient, opts SelectorOptions) (Selector, error) {
	var semverConstraint *semver.Constraints
	if opts.Constraint != "" {
		var err error
		if semverConstraint, err = semver.NewConstraint(opts.Constraint); err != nil {
			return nil, fmt.Errorf(
				"error parsing semver constraint %q: %w",
				opts.Constraint,
				err,
			)
		}
	}
	return &semVerSelector{
		repoClient: repoClient,
		opts:       opts,
		constraint: semverConstraint,
	}, nil
}

// Select implements the Selector interface.
func (s *semVerSelector) Select(ctx context.Context) ([]Image, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"registry", s.repoClient.registry.name,
		"image", s.repoClient.repoURL,
		"selectionStrategy", SelectionStrategySemVer,
		"platformConstrained", s.opts.platform != nil,
		"discoveryLimit", s.opts.DiscoveryLimit,
	)
	logger.Trace("discovering images")

	ctx = logging.ContextWithLogger(ctx, logger)

	images, err := s.selectImages(ctx)
	if err != nil {
		return nil, err
	}

	limit := s.opts.DiscoveryLimit
	if limit == 0 || limit > len(images) {
		limit = len(images)
	}
	discoveredImages := make([]Image, 0, limit)

	for _, svImage := range images {
		if len(discoveredImages) >= limit {
			break
		}

		image, err := s.repoClient.getImageByTag(ctx, svImage.Tag, s.opts.platform)
		if err != nil {
			return nil, fmt.Errorf("error retrieving image with tag %q: %w", svImage.Tag, err)
		}
		if image == nil {
			logger.Trace(
				"image was found, but did not match platform constraint",
				"tag", svImage.Tag,
			)
			continue
		}

		logger.Trace(
			"discovered image",
			"tag", image.Tag,
			"digest", image.Digest,
		)
		discoveredImages = append(discoveredImages, *image)
	}

	if len(discoveredImages) == 0 {
		logger.Trace("no images matched criteria")
		return nil, nil
	}

	logger.Trace(
		"discovered images",
		"count", len(discoveredImages),
	)
	return discoveredImages, nil
}

func (s *semVerSelector) selectImages(ctx context.Context) ([]Image, error) {
	logger := logging.LoggerFromContext(ctx)

	tags, err := s.repoClient.getTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing tags: %w", err)
	}
	if len(tags) == 0 {
		logger.Trace("found no tags")
		return nil, nil
	}
	logger.Trace("got all tags")

	images := make([]Image, 0, len(tags))
	for _, tag := range tags {
		if allowsTag(tag, s.opts.allowRegex) && !ignoresTag(tag, s.opts.Ignore) {
			sv := libSemver.Parse(tag, s.opts.StrictSemvers)
			if sv == nil {
				continue
			}
			if s.constraint != nil && !s.constraint.Check(sv) {
				continue
			}
			images = append(
				images,
				Image{
					Tag:    tag,
					semVer: sv,
				},
			)
		}
	}
	if len(images) == 0 {
		logger.Trace("no tags matched criteria")
		return nil, nil
	}
	logger.Trace(
		"tags matched criteria",
		"count", len(images),
	)

	logger.Trace("sorting images by semantic version")
	sortImagesBySemVer(images)
	return images, nil
}

// sortImagesBySemVer sorts the provided Images in place, in descending order by
// semantic version.
func sortImagesBySemVer(images []Image) {
	sort.Slice(images, func(i, j int) bool {
		if comp := images[i].semVer.Compare(images[j].semVer); comp != 0 {
			return comp > 0
		}
		// If the semvers tie, break the tie lexically using the original strings
		// used to construct the semvers. This ensures a deterministic comparison
		// of equivalent semvers, e.g., 1.0 and 1.0.0.
		return images[i].semVer.Original() > images[j].semVer.Original()
	})
}
