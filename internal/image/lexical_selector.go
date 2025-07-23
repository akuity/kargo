package image

import (
	"context"
	"fmt"
	"slices"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

func init() {
	selectorReg.register(
		kargoapi.ImageSelectionStrategyLexical,
		selectorRegistration{
			predicate: func(sub kargoapi.ImageSubscription) bool {
				return sub.ImageSelectionStrategy == kargoapi.ImageSelectionStrategyLexical
			},
			factory: newLexicalSelector,
		},
	)
}

// lexicalSelector implements the Selector interface for
// kargoapi.ImageSelectionStrategyLexical.
type lexicalSelector struct {
	*tagBasedSelector
}

func newLexicalSelector(
	sub kargoapi.ImageSubscription,
	creds *Credentials,
) (Selector, error) {
	tagBased, err := newTagBasedSelector(sub, creds)
	if err != nil {
		return nil, fmt.Errorf("error building tag based selector: %w", err)
	}
	return &lexicalSelector{tagBasedSelector: tagBased}, nil
}

// Select implements the Selector interface.
func (l *lexicalSelector) Select(
	ctx context.Context,
) ([]kargoapi.DiscoveredImageReference, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		l.getLoggerContext(),
		"selectionStrategy", kargoapi.ImageSelectionStrategyLexical,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	logger.Trace("discovering images")

	tags, err := l.repoClient.getTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing tags: %w", err)
	}
	if len(tags) == 0 {
		logger.Trace("found no tags")
		return nil, nil
	}
	logger.Trace("got all tags")

	tags = l.filterTags(tags)
	if len(tags) == 0 {
		logger.Trace("no tags matched criteria")
		return nil, nil
	}
	logger.Trace(
		"tags matched initial criteria",
		"count", len(tags),
	)

	logger.Trace("sorting tags lexically")
	slices.Sort(tags)
	slices.Reverse(tags)

	images, err := l.getImagesByTags(ctx, tags)
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

	return l.imagesToAPIImages(images, l.discoveryLimit), nil
}
