package image

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/akuity/kargo/internal/logging"
)

// digestTagSelector implements the TagSelector interface for the
// TagSelectionStrategyDigest strategy.
type digestTagSelector struct {
	repoClient *repositoryClient
	constraint string
	platform   *platformConstraint
}

// newDigestTagSelector returns an implementation of the TagSelector
// for the TagSelectionStrategyDigest strategy.
func newDigestTagSelector(
	repoClient *repositoryClient,
	constraint string,
	platform *platformConstraint,
) (TagSelector, error) {
	if constraint == "" {
		return nil, errors.New("digest selection strategy requires a constraint")
	}
	return &digestTagSelector{
		repoClient: repoClient,
		constraint: constraint,
		platform:   platform,
	}, nil
}

// SelectTag implements the TagSelector interface.
func (d *digestTagSelector) SelectTag(ctx context.Context) (*Tag, error) {
	logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
		"registry":            d.repoClient.registry.name,
		"image":               d.repoClient.image,
		"selectionStrategy":   TagSelectionStrategyDigest,
		"platformConstrained": d.platform != nil,
	})
	logger.Trace("selecting tag")

	ctx = logging.ContextWithLogger(ctx, logger)

	tagNames, err := d.repoClient.getTagNames(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error listing tags")
	}
	if len(tagNames) == 0 {
		logger.Trace("found no tag names")
		return nil, nil
	}
	logger.Trace("got all tag names")

	for _, tagName := range tagNames {
		if tagName != d.constraint {
			continue
		}
		tag, err := d.repoClient.getTagByName(ctx, tagName, d.platform)
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

	logger.Trace("no tag names matched criteria")
	return nil, nil
}
