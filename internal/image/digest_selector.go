package image

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	log "github.com/sirupsen/logrus"

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
func (d *digestSelector) Select(ctx context.Context) ([]Image, error) {
	tag := d.constraint
	logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
		"registry":            d.repoClient.registry.name,
		"image":               d.repoClient.repoURL,
		"selectionStrategy":   SelectionStrategyDigest,
		"tag":                 tag,
		"platformConstrained": d.platform != nil,
	})
	logger.Trace("selecting image")

	ctx = logging.ContextWithLogger(ctx, logger)

	image, err := d.repoClient.getImageByTag(ctx, tag, d.platform)
	if err != nil {
		var te *transport.Error
		if errors.As(err, &te) && te.StatusCode == http.StatusNotFound {
			logger.Trace("found no image with tag")
			return nil, nil
		}
		return nil, fmt.Errorf("error retrieving image with tag %q: %w", tag, err)
	}

	if image == nil {
		logger.Trace("image with tag did not match platform constraints")
		return nil, nil
	}

	logger.Trace("found image with tag")
	return []Image{*image}, nil
}
