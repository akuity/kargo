package image

import (
	"context"
	"regexp"
	"sort"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/akuity/kargo/internal/logging"
)

// semVerTagSelector implements the TagSelector interface for the
// TagSelectionStrategySemVer strategy.
type semVerTagSelector struct {
	repoClient *repositoryClient
	allowRegex *regexp.Regexp
	ignore     []string
	constraint *semver.Constraints
	platform   *platformConstraint
}

// newSemVerTagSelector returns an implementation of the TagSelector
// for the TagSelectionStrategySemVer strategy.
func newSemVerTagSelector(
	repoClient *repositoryClient,
	allowRegex *regexp.Regexp,
	ignore []string,
	constraint string,
	platform *platformConstraint,
) (TagSelector, error) {
	var semverConstraint *semver.Constraints
	if constraint != "" {
		var err error
		if semverConstraint, err = semver.NewConstraint(constraint); err != nil {
			return nil, errors.Wrapf(
				err,
				"error parsing semver constraint %q",
				constraint,
			)
		}
	}
	return &semVerTagSelector{
		repoClient: repoClient,
		allowRegex: allowRegex,
		ignore:     ignore,
		constraint: semverConstraint,
		platform:   platform,
	}, nil
}

// SelectTag implements the TagSelector interface.
func (s *semVerTagSelector) SelectTag(ctx context.Context) (*Tag, error) {
	logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
		"registry":            s.repoClient.registry.name,
		"image":               s.repoClient.image,
		"selectionStrategy":   TagSelectionStrategySemVer,
		"platformConstrained": s.platform != nil,
	})
	logger.Trace("selecting tag")

	ctx = logging.ContextWithLogger(ctx, logger)

	tagNames, err := s.repoClient.getTagNames(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error listing tags")
	}
	if len(tagNames) == 0 {
		logger.Trace("found no tag names")
		return nil, nil
	}
	logger.Trace("got all tag names")

	tags := make([]Tag, 0, len(tagNames))
	for _, tagName := range tagNames {
		if allows(tagName, s.allowRegex) && !ignores(tagName, s.ignore) {
			var sv *semver.Version
			if sv, err = semver.NewVersion(tagName); err != nil {
				continue // tagName wasn't a semantic version
			}
			if s.constraint != nil && !s.constraint.Check(sv) {
				continue
			}
			tags = append(
				tags,
				Tag{
					Name:   tagName,
					semVer: sv,
				},
			)
		}
	}
	if len(tags) == 0 {
		logger.Trace("no tag names matched criteria")
		return nil, nil
	}
	logger.Tracef("%d tag names matched criteria", len(tags))

	logger.Trace("sorting tags by semantic version")
	sortTagsBySemVer(tags)

	tagName := tags[0].Name
	tag, err := s.repoClient.getTagByName(ctx, tagName, s.platform)
	if err != nil {
		return nil, errors.Wrapf(err, "error retrieving tag %q", tagName)
	}
	if tag == nil {
		logger.Tracef(
			"tag %q was found, but did not match platform constraint",
			tagName,
		)
		return nil, nil
	}

	logger.WithFields(log.Fields{
		"name":   tag.Name,
		"digest": tag.Digest.String(),
	}).Trace("found tag")
	return tag, nil
}

// sortTagsBySemVer sorts the provided tags in place, in descending order by
// semantic version.
func sortTagsBySemVer(tags []Tag) {
	sort.Slice(tags, func(i, j int) bool {
		if comp := tags[i].semVer.Compare(tags[j].semVer); comp != 0 {
			return comp > 0
		}
		// If the semvers tie, break the tie lexically using the original strings
		// used to construct the semvers. This ensures a deterministic comparison
		// of equivalent semvers, e.g., 1.0 and 1.0.0.
		return tags[i].semVer.Original() > tags[j].semVer.Original()
	})
}
