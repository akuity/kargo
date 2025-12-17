package image

import (
	"context"
	"fmt"
	"regexp"
	"slices"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

// tagBasedSelector is a base implementation of Selector that provides common
// functionality for all Selector implementations that select images by
// retrieving, filtering, and sorting image tags. It is not intended to be used
// directly.
type tagBasedSelector struct {
	*baseSelector
	allowTagsRegexes  []*regexp.Regexp
	ignoreTagsRegexes []*regexp.Regexp
	discoveryLimit    int
}

// compileRegexes returns a slice of compiled regular expressions.
func compileRegexes(regexStrs []string) ([]*regexp.Regexp, error) {
	regexes := make([]*regexp.Regexp, len(regexStrs))
	var err error
	for i, regexStr := range regexStrs {
		if regexes[i], err = regexp.Compile(regexStr); err != nil {
			return nil, fmt.Errorf(
				"error compiling regular expression %q: %w",
				regexStr, err,
			)
		}
	}
	return regexes, nil
}

func newTagBasedSelector(
	sub kargoapi.ImageSubscription,
	creds *Credentials,
) (*tagBasedSelector, error) {
	base, err := newBaseSelector(sub, creds, sub.CacheByTag)
	if err != nil {
		return nil, fmt.Errorf("error building base selector: %w", err)
	}
	s := &tagBasedSelector{
		baseSelector:   base,
		discoveryLimit: int(sub.DiscoveryLimit),
	}

	if s.allowTagsRegexes, err = compileRegexes(sub.AllowTagsRegexes); err != nil {
		return nil, fmt.Errorf("error compiling allow tags regex: %w", err)
	}

	// TODO(v1.11.0): Return an error if sub.AllowTags is non-empty.
	// TODO(v1.13.0): Remove this block after the AllowTags field is removed.
	if sub.AllowTags != "" { // nolint: staticcheck
		var allowTagsRegex *regexp.Regexp
		if allowTagsRegex, err = regexp.Compile(sub.AllowTags); err != nil { // nolint: staticcheck
			return nil, fmt.Errorf(
				"error compiling regular expression %q: %w",
				sub.AllowTags, err, // nolint: staticcheck
			)
		}
		s.allowTagsRegexes = append(s.allowTagsRegexes, allowTagsRegex)
	}

	if s.ignoreTagsRegexes, err = compileRegexes(sub.IgnoreTagsRegexes); err != nil {
		return nil, fmt.Errorf("error compiling ignore tags regex: %w", err)
	}

	// TODO(v1.11.0): Return an error if sub.IgnoreTags is non-empty.
	// TODO(v1.13.0): Remove this block after the IgnoreTags field is removed.
	if len(sub.IgnoreTags) > 0 { // nolint: staticcheck
		ignoreTagsRegexStrs := make([]string, len(sub.IgnoreTags))
		for i, ignoreTag := range sub.IgnoreTags { // nolint: staticcheck
			ignoreTagsRegexStrs[i] = fmt.Sprintf("^%s$", regexp.QuoteMeta(ignoreTag))
		}
		ignoreTagsRegexes, err := compileRegexes(ignoreTagsRegexStrs)
		if err != nil {
			return nil, err
		}
		s.ignoreTagsRegexes = append(s.ignoreTagsRegexes, ignoreTagsRegexes...)
	}

	return s, nil
}

// MatchesTag implements Selector.
func (t *tagBasedSelector) MatchesTag(tag string) bool {
	// handle ignoreTagsRegexes
	for _, regex := range t.ignoreTagsRegexes {
		if regex.MatchString(tag) {
			return false
		}
	}

	// if empty allowTagsRegexes, we match all tags
	if len(t.allowTagsRegexes) == 0 {
		return true
	}

	// check if tag matches any allowTagsRegexes
	for _, regex := range t.allowTagsRegexes {
		if regex.MatchString(tag) {
			return true
		}
	}

	return false
}

// getLoggerContext returns key/value pairs that can be used by any selector
// that images by retrieving, filtering, and sorting image tags to enrich
// loggers with valuable context.
func (t *tagBasedSelector) getLoggerContext() []any {
	return append(
		t.baseSelector.getLoggerContext(),
		"tagConstrained", len(t.allowTagsRegexes) > 0 || len(t.ignoreTagsRegexes) > 0,
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

		image, err := t.repoClient.getImageByTag(ctx, tag, t.platformConstraint)
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
