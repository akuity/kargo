package image

import (
	"context"
	"errors"
	"fmt"

	"github.com/akuity/kargo/internal/logging"
)

// digestSelector implements the Selector interface for SelectionStrategyDigest.
type digestSelector struct {
	repoClient *repositoryClient
	constraint string
	platform   *platformConstraint
}

// newDigestSelector returns an implementation of the Selector interface for
// SelectionStrategyDigest.
func newDigestSelector(
	repoClient *repositoryClient,
	constraint string,
	platform *platformConstraint,
) (Selector, error) {
	if constraint == "" {
		return nil, errors.New("digest selection strategy requires a constraint")
	}
	return &digestSelector{
		repoClient: repoClient,
		constraint: constraint,
		platform:   platform,
	}, nil
}

// Select implements the Selector interface.
func (d *digestSelector) Select(ctx context.Context) (*Image, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"registry", d.repoClient.registry.name,
		"image", d.repoClient.image,
		"selectionStrategy", SelectionStrategyDigest,
		"platformConstrained", d.platform != nil,
	)
	logger.V(2).Info("selecting image")

	ctx = logging.ContextWithLogger(ctx, logger)

	tags, err := d.repoClient.getTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing tags: %w", err)
	}
	if len(tags) == 0 {
		logger.V(2).Info("found no tags")
		return nil, nil
	}
	logger.V(2).Info("got all tags")

	for _, tag := range tags {
		if tag != d.constraint {
			continue
		}
		image, err := d.repoClient.getImageByTag(ctx, tag, d.platform)
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

	logger.V(2).Info("no images matched criteria")
	return nil, nil
}
