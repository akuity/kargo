package image

import (
	"cmp"
	"context"
	"fmt"
	"regexp"
	"slices"
	"sort"

	"github.com/Masterminds/semver/v3"

	"github.com/akuity/kargo/internal/logging"
)

// semVerSelector implements the Selector interface for SelectionStrategySemVer.
type semVerSelector struct {
	repoClient     *repositoryClient
	allowRegex     *regexp.Regexp
	ignore         []string
	constraint     *semver.Constraints
	platform       *platformConstraint
	discoveryLimit int
}

// newSemVerSelector returns an implementation of the Selector interface for
// SelectionStrategySemVer.
func newSemVerSelector(
	repoClient *repositoryClient,
	allowRegex *regexp.Regexp,
	ignore []string,
	constraint string,
	platform *platformConstraint,
	discoveryLimit int,
) (Selector, error) {
	var semverConstraint *semver.Constraints
	if constraint != "" {
		var err error
		if semverConstraint, err = semver.NewConstraint(constraint); err != nil {
			return nil, fmt.Errorf(
				"error parsing semver constraint %q: %w",
				constraint,
				err,
			)
		}
	}
	return &semVerSelector{
		repoClient:     repoClient,
		allowRegex:     allowRegex,
		ignore:         ignore,
		constraint:     semverConstraint,
		platform:       platform,
		discoveryLimit: discoveryLimit,
	}, nil
}

// Select implements the Selector interface.
func (s *semVerSelector) Select(ctx context.Context) ([]Image, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"registry", s.repoClient.registry.name,
		"image", s.repoClient.repoURL,
		"selectionStrategy", SelectionStrategySemVer,
		"platformConstrained", s.platform != nil,
		"discoveryLimit", s.discoveryLimit,
	)
	logger.Trace("discovering images")

	ctx = logging.ContextWithLogger(ctx, logger)

	images, err := s.selectImages(ctx)
	if err != nil {
		return nil, err
	}

	limit := s.discoveryLimit
	if limit == 0 || limit > len(images) {
		limit = len(images)
	}
	discoveredImages := make([]Image, 0, limit)

	for _, svImage := range images {
		if len(discoveredImages) >= limit {
			break
		}

		image, err := s.repoClient.getImageByTag(ctx, svImage.Tag, s.platform)
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
		if allowsTag(tag, s.allowRegex) && !ignoresTag(tag, s.ignore) {
			var sv *semver.Version
			if sv, err = semver.NewVersion(tag); err != nil {
				continue // tag wasn't a semantic version
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

// sortImagesBySemVer sorts the provided Images in place, in descending order by semantic version.
func sortImagesBySemVer(images []Image) {
	slices.SortFunc(images, func (a, b Image) int {
		if comp := a.semVer.Compare(b.semVer); comp != 0 {
			return -comp
		}
		// If the semvers tie, break the tie lexically using the original strings
		// used to construct the semvers. This ensures a deterministic comparison
		// of equivalent semvers, e.g., 1.0 and 1.0.0.
		return cmp.Compare(a.semVer.Original(), b.semVer.Original())
	})
}
