package image

import (
	"context"
	"fmt"
	"regexp"
	"slices"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

// tagBasedSelector is a base implementation of Selector that provides common
// functionality for all Selector implementations that select images by
// retrieving, filtering, and sorting image tags. It is not intended to be used
// directly.
type tagBasedSelector struct {
	*baseSelector
	allows         *regexp.Regexp
	ignores        []string
	discoveryLimit int
}

func newTagBasedSelector(
	sub kargoapi.ImageSubscription,
	creds *Credentials,
) (*tagBasedSelector, error) {
	base, err := newBaseSelector(sub, creds)
	if err != nil {
		return nil, fmt.Errorf("error building base selector: %w", err)
	}
	s := &tagBasedSelector{
		baseSelector:   base,
		ignores:        sub.IgnoreTags,
		discoveryLimit: int(sub.DiscoveryLimit),
	}
	if sub.AllowTags != "" {
		if s.allows, err = regexp.Compile(sub.AllowTags); err != nil {
			return nil, fmt.Errorf(
				"error compiling regular expression %q: %w",
				sub.AllowTags, err,
			)
		}
	}
	return s, nil
}

// MatchesTag implements Selector.
func (t *tagBasedSelector) MatchesTag(tag string) bool {
	return (t.allows == nil || t.allows.MatchString(tag)) &&
		!slices.Contains(t.ignores, tag)
}

// getLoggerContext returns key/value pairs that can be used by any selector
// that images by retrieving, filtering, and sorting image tags to enrich
// loggers with valuable context.
func (t *tagBasedSelector) getLoggerContext() []any {
	return append(
		t.baseSelector.getLoggerContext(),
		"tagConstrained", t.allows != nil || len(t.ignores) > 0,
		"discoveryLimit", t.discoveryLimit,
	)
}

// filterTags evaluates all provided tags against the constraints defined by the
// t.MatchesTag method, returning only those that satisfied those constraints.
func (t *tagBasedSelector) filterTags(tags []string) []string {
	filteredTags := make([]string, 0, len(tags))
	for _, tag := range tags {
		if t.MatchesTag(tag) {
			filteredTags = append(filteredTags, tag)
		}
	}
	return slices.Clip(filteredTags)
}

// getImagesByTags retrieves image metadata for the provided tags SEQUENTIALLY.
// It discards any that does not match the selector's criteria. This repeats
// until the list of provided tags has been exhausted or it has found an amount
// of image metadata equal to the selector's discovery limit.
func (t *tagBasedSelector) getImagesByTags(
	ctx context.Context,
	tags []string,
) ([]image, error) {
	logger := logging.LoggerFromContext(ctx)

	limit := t.discoveryLimit
	if limit == 0 || limit > len(tags) {
		limit = len(tags)
	}
	images := make([]image, 0, limit)
	for _, tag := range tags {
		if len(images) >= limit {
			break
		}

		image, err := t.repoClient.getImageByTag(ctx, tag, t.platform)
		if err != nil {
			return nil, fmt.Errorf("error retrieving image with tag %q: %w", tag, err)
		}
		if image == nil {
			logger.Trace(
				"image was found, but did not match platform constraint",
				"tag", tag,
			)
			continue
		}

		logger.Trace(
			"discovered image",
			"tag", image.Tag,
			"digest", image.Digest,
		)
		images = append(images, *image)
	}

	return slices.Clip(images), nil
}
