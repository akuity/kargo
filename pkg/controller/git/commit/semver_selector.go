package commit

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	libSemver "github.com/akuity/kargo/pkg/controller/semver"
	"github.com/akuity/kargo/pkg/logging"
)

func init() {
	defaultSelectorRegistry.MustRegister(
		selectorRegistration{
			Predicate: func(_ context.Context, sub kargoapi.GitSubscription) (bool, error) {
				return sub.CommitSelectionStrategy == kargoapi.CommitSelectionStrategySemVer, nil
			},
			Value: newSemverSelector,
		},
	)
}

// semverSelector implements the Selector interface for
// kargoapi.CommitSelectionStrategySemVer.
type semverSelector struct {
	*tagBasedSelector
	constraint    *semver.Constraints
	strictSemvers bool
}

func newSemverSelector(
	sub kargoapi.GitSubscription,
	creds *git.RepoCredentials,
) (Selector, error) {
	tagBased, err := newTagBasedSelector(sub, creds)
	if err != nil {
		return nil, fmt.Errorf("error building tag based selector: %w", err)
	}
	s := &semverSelector{tagBasedSelector: tagBased}
	if sub.StrictSemvers != nil {
		s.strictSemvers = *sub.StrictSemvers
	}
	if sub.SemverConstraint != "" {
		if s.constraint, err = semver.NewConstraint(sub.SemverConstraint); err != nil {
			return nil, fmt.Errorf(
				"error parsing semver constraint %q: %w",
				sub.SemverConstraint, err,
			)
		}
	}
	return s, nil
}

// MatchesRef implements Selector. Note: This differs from tagBasedSelector's
// implementation by invoking an implementation-specific matchesTag() method
// that imposes additional match criteria beyond those of tagBasedSelector's.
func (s *semverSelector) MatchesRef(ref string) bool {
	if !strings.HasPrefix(ref, tagPrefix) {
		return false
	}
	return s.matchesTag(ref)
}

// matchesTag returns a boolean indicating whether the given tag satisfies the
// selector's constraints. Any leading "refs/tags/" is stripped away prior
// to evaluation. Note: This implementation uses tagBasedSelector's matchesTag()
// method, but then imposes additional match criteria -- namely considering
// whether a tag is parseable as a semantic version, and if so, whether it
// satisfies optional semantic versioning constraints.
func (s *semverSelector) matchesTag(tag string) bool {
	tag = strings.TrimPrefix(tag, tagPrefix)
	if !s.tagBasedSelector.matchesTag(tag) {
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
func (s *semverSelector) Select(ctx context.Context) (
	[]kargoapi.DiscoveredCommit,
	error,
) {
	loggerCtx := append(
		s.getLoggerContext(),
		"selectionStrategy", kargoapi.CommitSelectionStrategySemVer,
	)
	logger := logging.LoggerFromContext(ctx).WithValues(loggerCtx...)
	ctx = logging.ContextWithLogger(ctx, logger)

	repo, err := s.clone(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = repo.Close()
	}()

	tags, err := repo.ListTags()
	if err != nil {
		return nil, err
	}

	// Note: This is calling this type's own implementation of filterTags() and
	// NOT directly calling tagBasedSelector's implementation.
	tags = s.filterTags(tags)

	if tags, err = s.filterTagsByExpression(tags); err != nil {
		return nil, fmt.Errorf("error filtering tags by expression: %w", err)
	}

	s.sort(tags)

	if tags, err = s.filterTagsByDiffPathsFn(repo, tags); err != nil {
		return nil, fmt.Errorf("error filtering tags by paths: %w", err)
	}

	return s.tagsToAPICommits(ctx, tags), nil
}

// filterTags evaluates all provided tags against the constraints defined by the
// s.matchesTag method, returning only those that satisfied those constraints.
// Note: This implementation uses this type's own matchesTag() implementation
// and not tagBasedSelector's.
func (s *semverSelector) filterTags(tags []git.TagMetadata) []git.TagMetadata {
	filteredTags := make([]git.TagMetadata, 0, len(tags))
	for _, tag := range tags {
		if s.matchesTag(tag.Tag) {
			filteredTags = append(filteredTags, tag)
		}
	}
	return slices.Clip(filteredTags)
}

// sort sorts the provided tags from greatest to least semantic version in
// place. Note: It is assumed that the provided tags have been pre-filtered and
// all are parseable as semantic versions.
func (s *semverSelector) sort(tags []git.TagMetadata) {
	type semverTag struct {
		tag    git.TagMetadata
		semver *semver.Version
	}
	semverTags := make([]semverTag, len(tags))
	for i, tag := range tags {
		sv := libSemver.Parse(tag.Tag, s.strictSemvers)
		semverTags[i] = semverTag{
			tag:    tag,
			semver: sv,
		}
	}
	slices.SortFunc(semverTags, func(i, j semverTag) int {
		if comp := j.semver.Compare(i.semver); comp != 0 {
			return comp
		}
		// If the semvers tie, break the tie lexically using the original strings
		// used to construct the semvers. This ensures a deterministic comparison
		// of equivalent semvers, e.g., 1.0 and 1.0.0.
		return strings.Compare(j.semver.Original(), i.semver.Original())
	})
	for i, semverTag := range semverTags {
		tags[i] = semverTag.tag
	}
}
