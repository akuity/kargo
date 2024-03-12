package image

import (
	"context"
	"fmt"
	"regexp"
	"sort"

	log "github.com/sirupsen/logrus"

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
	logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
		"registry":            l.repoClient.registry.name,
		"image":               l.repoClient.image,
		"selectionStrategy":   SelectionStrategyLexical,
		"platformConstrained": l.platform != nil,
	})
	logger.Trace("selecting image")

	ctx = logging.ContextWithLogger(ctx, logger)

	tags, err := l.repoClient.getTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing tags: %w", err)
	}
	if len(tags) == 0 {
		logger.Trace("found no tags")
		return nil, nil
	}
	logger.Trace("got all tags")

	if l.allowRegex != nil || len(l.ignore) > 0 {
		matchedTags := make([]string, 0, len(tags))
		for _, tag := range tags {
			if allowsTag(tag, l.allowRegex) && !ignoresTag(tag, l.ignore) {
				matchedTags = append(matchedTags, tag)
			}
		}
		if len(matchedTags) == 0 {
			logger.Trace("no tags matched criteria")
			return nil, nil
		}
		tags = matchedTags
	}
	logger.Tracef("%d tags matched criteria", len(tags))

	logger.Trace("sorting tags lexically")
	sortTagsLexically(tags)

	tag := tags[0]
	image, err := l.repoClient.getImageByTag(ctx, tag, l.platform)
	if err != nil {
		return nil, fmt.Errorf("error retrieving image with tag %q: %w", tag, err)
	}
	if image == nil {
		logger.Tracef(
			"image with tag %q was found, but did not match platform constraint",
			tag,
		)
		return nil, nil
	}

	logger.WithFields(log.Fields{
		"tag":    image.Tag,
		"digest": image.Digest.String(),
	}).Trace("found image")
	return image, nil
}

// sortTagsLexically sorts the provided tags in place, in lexically descending
// order.
func sortTagsLexically(tags []string) {
	sort.Slice(tags, func(i, j int) bool {
		return tags[i] > tags[j]
	})
}
