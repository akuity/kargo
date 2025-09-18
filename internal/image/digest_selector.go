package image

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-containerregistry/pkg/v1/remote/transport"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

func init() {
	selectorReg.register(
		kargoapi.ImageSelectionStrategyDigest,
		selectorRegistration{
			predicate: func(sub kargoapi.ImageSubscription) bool {
				return sub.ImageSelectionStrategy == kargoapi.ImageSelectionStrategyDigest
			},
			factory: newDigestSelector,
		},
	)
}

// digestSelector implements the Selector interface for
// kargoapi.ImageSelectionStrategyDigest.
type digestSelector struct {
	*baseSelector
	mutableTag string
}

func newDigestSelector(
	sub kargoapi.ImageSubscription,
	creds *Credentials,
) (Selector, error) {
	base, err := newBaseSelector(sub, creds)
	if err != nil {
		return nil, fmt.Errorf("error building base selector: %w", err)
	}
	s := &digestSelector{
		baseSelector: base,
		mutableTag:   sub.Constraint,
	}
	if s.mutableTag == "" {
		// Fall back on the deprecated SemverConstraint field.
		// TODO: Remove this for v1.9.0.
		s.mutableTag = sub.SemverConstraint // nolint: staticcheck
	}
	return s, nil
}

// MatchesTag implements Selector.
func (d *digestSelector) MatchesTag(tag string) bool {
	return d.mutableTag == tag
}

// Select implements the Selector interface.
func (d *digestSelector) Select(
	ctx context.Context,
) ([]kargoapi.DiscoveredImageReference, error) {
	loggerCtx := append(
		d.getLoggerContext(),
		"selectionStrategy", kargoapi.ImageSelectionStrategyDigest,
		"tag", d.mutableTag,
	)
	logger := logging.LoggerFromContext(ctx).WithValues(loggerCtx...)
	ctx = logging.ContextWithLogger(ctx, logger)

	logger.Trace("selecting image")

	img, err := d.repoClient.getImageByTag(ctx, d.mutableTag, d.platform)
	if err != nil {
		var te *transport.Error
		if errors.As(err, &te) && te.StatusCode == http.StatusNotFound {
			logger.Trace("found no image with tag")
			return nil, nil
		}
		return nil,
			fmt.Errorf("error retrieving image with tag %q: %w", d.mutableTag, err)
	}

	if img == nil {
		logger.Trace("image with tag did not match platform constraints")
		return nil, nil
	}

	logger.Trace("found image with tag")
	return d.imagesToAPIImages([]image{*img}, 0), nil
}
