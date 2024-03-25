package image

import (
	"context"
	"fmt"
	"regexp"
	"sort"

	"github.com/akuity/kargo/internal/logging"
)

// lexicalSelector implements the Selector interface for
// SelectionStrategyLexical.
type lexicalSelector struct {
	repoClient *repositoryClient
	allowRegex *regexp.Regexp
	ignore     []string
	platform   *platformConstraint
}

// newLexicalSelector returns an implementation of the Selector interface for
// SelectionStrategyLexical.
func newLexicalSelector(
	repoClient *repositoryClient,
	allowRegex *regexp.Regexp,
	ignore []string,
	platform *platformConstraint,
) Selector {
	return &lexicalSelector{
		repoClient: repoClient,
		allowRegex: allowRegex,
		ignore:     ignore,
		platform:   platform,
	}
}

// Select implements the Selector interface.
func (l *lexicalSelector) Select(ctx context.Context) (*Image, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"registry", l.repoClient.registry.name,
		"image", l.repoClient.image,
		"selectionStrategy", SelectionStrategyLexical,
		"platformConstrained", l.platform != nil,
	)
	logger.V(2).Info("selecting image")

	ctx = logging.ContextWithLogger(ctx, logger)

	tags, err := l.repoClient.getTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing tags: %w", err)
	}
	if len(tags) == 0 {
		logger.V(2).Info("found no tags")
		return nil, nil
	}
	logger.V(2).Info("got all tags")

	if l.allowRegex != nil || len(l.ignore) > 0 {
		matchedTags := make([]string, 0, len(tags))
		for _, tag := range tags {
			if allowsTag(tag, l.allowRegex) && !ignoresTag(tag, l.ignore) {
				matchedTags = append(matchedTags, tag)
			}
		}
		if len(matchedTags) == 0 {
			logger.V(2).Info("no tags matched criteria")
			return nil, nil
		}
		tags = matchedTags
	}
	logger.V(2).Info("tags matched criteria", "numberOfTags", len(tags))

	logger.V(2).Info("sorting tags lexically")
	sortTagsLexically(tags)

	tag := tags[0]
	image, err := l.repoClient.getImageByTag(ctx, tag, l.platform)
	if err != nil {
		return nil, fmt.Errorf("error retrieving image with tag %q: %w", tag, err)
	}
	if image == nil {
		logger.V(2).Info(
			"image was found, but did not match platform constraint",
			"tag", tag,
		)
		return nil, nil
	}

	logger.WithValues(
		"tag", image.Tag,
		"digest", image.Digest.String(),
	).V(2).Info("found image")
	return image, nil
}

// sortTagsLexically sorts the provided tags in place, in lexically descending
// order.
func sortTagsLexically(tags []string) {
	sort.Slice(tags, func(i, j int) bool {
		return tags[i] > tags[j]
	})
}
