package image

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/akuity/kargo/internal/logging"
)

// lexicalSelector implements the Selector interface for
// SelectionStrategyLexical.
type lexicalSelector struct {
	repoClient *repositoryClient
	opts       SelectorOptions
}

// newLexicalSelector returns an implementation of the Selector interface for
// SelectionStrategyLexical.
func newLexicalSelector(repoClient *repositoryClient, opts SelectorOptions) Selector {
	return &lexicalSelector{
		repoClient: repoClient,
		opts:       opts,
	}
}

// Select implements the Selector interface.
func (l *lexicalSelector) Select(ctx context.Context) ([]Image, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"registry", l.repoClient.registry.name,
		"image", l.repoClient.repoURL,
		"selectionStrategy", SelectionStrategyLexical,
		"platformConstrained", l.opts.platform != nil,
		"discoveryLimit", l.opts.DiscoveryLimit,
	)
	logger.Trace("discovering images")

	ctx = logging.ContextWithLogger(ctx, logger)

	tags, err := l.selectTags(ctx)
	if err != nil || len(tags) == 0 {
		return nil, err
	}

	limit := l.opts.DiscoveryLimit
	if limit == 0 || limit > len(tags) {
		limit = len(tags)
	}
	images := make([]Image, 0, limit)

	for _, tag := range tags {
		if len(images) >= limit {
			break
		}

		image, err := l.repoClient.getImageByTag(ctx, tag, l.opts.platform)
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

	if len(images) == 0 {
		logger.Trace("no images matched criteria")
		return nil, nil
	}

	logger.Trace(
		"discovered images",
		"count", len(images),
	)
	return images, nil
}

// selectTags retrieves all tags from the repository and filters them based on
// the allowRegex and ignore fields of the lexicalSelector. If no tags match
// the criteria, nil is returned.
func (l *lexicalSelector) selectTags(ctx context.Context) ([]string, error) {
	logger := logging.LoggerFromContext(ctx)

	tags, err := l.repoClient.getTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing tags: %w", err)
	}
	if len(tags) == 0 {
		logger.Trace("found no tags")
		return nil, nil
	}
	logger.Trace("got all tags")

	if l.opts.allowRegex != nil || len(l.opts.Ignore) > 0 {
		matchedTags := make([]string, 0, len(tags))
		for _, tag := range tags {
			if allowsTag(tag, l.opts.allowRegex) && !ignoresTag(tag, l.opts.Ignore) {
				matchedTags = append(matchedTags, tag)
			}
		}
		if len(matchedTags) == 0 {
			logger.Trace("no tags matched criteria")
			return nil, nil
		}
		tags = matchedTags
	}
	logger.Trace(
		"tags matched criteria",
		"count", len(tags),
	)

	logger.Trace("sorting tags lexically")
	sortTagsLexically(tags)
	return tags, nil
}

// sortTagsLexically sorts the provided tags in place, in lexically descending
// order.
func sortTagsLexically(tags []string) {
	slices.SortFunc(tags, func(lhs, rhs string) int {
		return strings.Compare(rhs, lhs)
	})
}
