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
	allowTagsRegex  []*regexp.Regexp
	ignoreTagsRegex []*regexp.Regexp
	discoveryLimit  int
}

// compileRegexes returns a slice of compiled regular expressions.
func compileRegexes(regexStrs []string) ([]*regexp.Regexp, error) {
	regexes := make([]*regexp.Regexp, len(regexps))
	for i, regexStr := range regexStrs {
		var err error
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
	base, err := newBaseSelector(sub, creds)
	if err != nil {
		return nil, fmt.Errorf("error building base selector: %w", err)
	}
	s := &tagBasedSelector{
		baseSelector:   base,
		discoveryLimit: int(sub.DiscoveryLimit),
	}

	var err error
	if s.allowTagsRegex, err = compileRegexes(sub.AllowTagsRegex); err != nil {
		return nil, fmt.Errorf("error compiling allow tags regex: %w", err)
	}

	// TODO(v1.11.0): Return an error if sub.AllowTags is non-empty.
	// TODO(v1.13.0): Remove this block after the AllowTags field is removed.
	if sub.AllowTags != "" {
		allowTagsRegex, err := regexp.Compile(sub.AllowTags)
		if err != nil {
			return nil, fmt.Errorf(
				"error compiling regular expression %q: %w",
				sub.AllowTags, err,
			)
		}
		s.allowTagsRegex = append(s.allowTagsRegex, allowTagsRegex)
	}

	if s.ignoreTagsRegex, err = compileRegexes(sub.IgnoreTagsRegex); err != nil {
		return nil, fmt.Errorf("error compiling ignore tags regex: %w", err)
	}

	// TODO(v1.11.0): Return an error if sub.IgnoreTags is non-empty.
	// TODO(v1.13.0): Remove this block after the IgnoreTags field is removed.
	if len(sub.IgnoreTags) {
		ignoreTagsRegexStrs := make([]string, len(sub.IgnoreTags))
		for i, ignoreTag := range sub.IgnoreTags {
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
	// handle ignoreTagsRegex
	for _, regex := range t.ignoreTagsRegex {
		if regex.MatchString(tag) {
			return false
		}
	}

	// if empty allowTagsRegex, we match all tags
	if len(t.allowTagsRegex) == 0 {
		return true
	}

	// check if tag matches any allowTagsRegex
	for _, regex := range t.allowTagsRegex {
		if regex.MatchString(regex, tag) {
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
		"tagConstrained", len(t.allowTagsRegex) > 0 || len(t.ignoreTags) > 0,
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
