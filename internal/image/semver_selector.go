package image

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libSemver "github.com/akuity/kargo/internal/controller/semver"
	"github.com/akuity/kargo/internal/logging"
)

func init() {
	selectorReg.register(
		kargoapi.ImageSelectionStrategySemVer,
		selectorRegistration{
			predicate: func(sub kargoapi.ImageSubscription) bool {
				return sub.ImageSelectionStrategy == kargoapi.ImageSelectionStrategySemVer
			},
			factory: newSemverSelector,
		},
	)
}

// semverSelector implements the Selector interface for
// kargoapi.ImageSelectionStrategySemVer.
type semverSelector struct {
	*tagBasedSelector
	constraint    *semver.Constraints
	strictSemvers bool
}

func newSemverSelector(
	sub kargoapi.ImageSubscription,
	creds *Credentials,
) (Selector, error) {
	tagBased, err := newTagBasedSelector(sub, creds)
	if err != nil {
		return nil, fmt.Errorf("error building tag based selector: %w", err)
	}
	s := &semverSelector{
		tagBasedSelector: tagBased,
		strictSemvers:    sub.StrictSemvers,
	}
	if sub.SemverConstraint != "" {
		if s.constraint, err =
			semver.NewConstraint(sub.SemverConstraint); err != nil {
			return nil, fmt.Errorf(
				"error parsing semver constraint %q: %w",
				sub.SemverConstraint, err,
			)
		}
	}
	return s, nil
}

// MatchesTag implements Selector. Note: This differs from tagBasedSelector's
// implementation by imposing additional match criteria beyond those of
// tagBasedSelector's, namely considering whether a tag is parseable as a
// semantic version, and if so, whether it satisfies optional semantic
// versioning constraints.
func (s *semverSelector) MatchesTag(tag string) bool {
	if !s.tagBasedSelector.MatchesTag(tag) {
		return false
	}
	sv := libSemver.Parse(tag, s.strictSemvers)
	if sv == nil {
		// The tag wasn't parseable as a semantic version.
		return false
	}
	// Now it all comes down to whether semantic version constraints were
	// specified and if so, whether the tag satisfies them.
	return s.constraint == nil || s.constraint.Check(sv)
}

// Select implements the Selector interface.
func (s *semverSelector) Select(
	ctx context.Context,
) ([]kargoapi.DiscoveredImageReference, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		s.getLoggerContext(),
		"selectionStrategy", kargoapi.ImageSelectionStrategySemVer,
		"semverConstrained", s.constraint != nil,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	logger.Trace("discovering images")

	tags, err := s.repoClient.getTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing tags: %w", err)
	}
	if len(tags) == 0 {
		logger.Trace("found no tags")
		return nil, nil
	}
	logger.Trace("got all tags")

	// Note: This is calling this type's own implementation of filterTags() and
	// NOT directly calling tagBasedSelector's implementation.
	tags = s.filterTags(tags)
	if len(tags) == 0 {
		logger.Trace("no tags matched criteria")
		return nil, nil
	}
	logger.Trace(
		"tags matched initial criteria",
		"count", len(tags),
	)

	logger.Trace("sorting tags semantically")
	tags = s.sort(tags)

	logger.Trace("sorting tags lexically")
	slices.Sort(tags)
	slices.Reverse(tags)

	images, err := s.getImagesByTags(ctx, tags)
	if err != nil {
		return nil, fmt.Errorf("error getting images by tags")
	}

	if len(images) == 0 {
		logger.Trace("no images matched criteria")
		return nil, nil
	}

	logger.Trace(
		"discovered images",
		"count", len(images),
	)

	return s.imagesToAPIImages(images, s.discoveryLimit), nil
}

// filterTags evaluates all provided tags against the constraints defined by the
// s.matchesTag method, returning only those that satisfied those constraints.
// Note: This implementation uses this type's own matchesTag() implementation
// and not tagBasedSelector's.
func (s *semverSelector) filterTags(tags []string) []string {
	filteredTags := make([]string, 0, len(tags))
	for _, tag := range tags {
		if s.MatchesTag(tag) {
			filteredTags = append(filteredTags, tag)
		}
	}
	return slices.Clip(filteredTags)
}

// sort sorts the provided tags from greatest to least semantic version in
// place. Note: It is assumed that the provided tags have been pre-filtered and
// all are parseable as semantic versions. If any tags are not parseable as
// semantic versions, they will be omitted entirely from the results.
func (s *semverSelector) sort(tags []string) []string {
	semvers := make([]semver.Version, 0, len(tags))
	for _, tag := range tags {
		if sv := libSemver.Parse(tag, s.strictSemvers); sv != nil {
			semvers = append(semvers, *sv)
		}
	}
	slices.SortFunc(semvers, func(lhs, rhs semver.Version) int {
		if comp := rhs.Compare(&lhs); comp != 0 {
			return comp
		}
		// If the semvers tie, break the tie lexically using the original strings
		// used to construct the semvers. This ensures a deterministic comparison of
		// equivalent semvers, e.g., "1.0.0" > "1.0". The semver package's built-in
		// sort does not do this!
		return strings.Compare(rhs.Original(), lhs.Original())
	})
	tags = make([]string, len(semvers))
	for i, sv := range semvers {
		tags[i] = sv.Original()
	}
	return tags
}
