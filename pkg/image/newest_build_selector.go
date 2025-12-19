package image

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

func init() {
	defaultSelectorRegistry.MustRegister(
		selectorRegistration{
			Predicate: func(_ context.Context, sub kargoapi.ImageSubscription) (bool, error) {
				return sub.ImageSelectionStrategy == kargoapi.ImageSelectionStrategyNewestBuild, nil
			},
			Value: newNewestBuildSelector,
		},
	)
}

// newestBuildSelector implements the Selector interface for
// kargoapi.ImageSelectionStrategyNewestBuild.
type newestBuildSelector struct {
	*tagBasedSelector
}

func newNewestBuildSelector(
	sub kargoapi.ImageSubscription,
	creds *Credentials,
) (Selector, error) {
	tagBased, err := newTagBasedSelector(sub, creds)
	if err != nil {
		return nil, fmt.Errorf("error building tag based selector: %w", err)
	}
	return &newestBuildSelector{tagBasedSelector: tagBased}, nil
}

// Select implements the Selector interface.
func (n *newestBuildSelector) Select(
	ctx context.Context,
) ([]kargoapi.DiscoveredImageReference, error) {
	loggerCtx := append(
		n.getLoggerContext(),
		"selectionStrategy", kargoapi.ImageSelectionStrategyNewestBuild,
	)
	logger := logging.LoggerFromContext(ctx).WithValues(loggerCtx...)
	ctx = logging.ContextWithLogger(ctx, logger)

	logger.Trace("discovering images")

	tags, err := n.repoClient.getTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing tags: %w", err)
	}
	if len(tags) == 0 {
		logger.Trace("found no tags")
		return nil, nil
	}
	logger.Trace("got all tags")

	tags = n.filterTags(tags)
	if len(tags) == 0 {
		logger.Trace("no tags matched criteria")
		return nil, nil
	}
	logger.Trace(
		"tags matched initial criteria",
		"count", len(tags),
	)

	logger.Trace("retrieving images for all tags that matched initial criteria")
	images, err := n.getImagesByTags(ctx, tags)
	if err != nil {
		return nil, fmt.Errorf("error retrieving images for all matched tags: %w", err)
	}
	if len(images) == 0 {
		// This shouldn't happen
		return nil, nil
	}

	logger.Trace("sorting images by date")
	n.sort(images)

	limit := n.discoveryLimit
	if limit == 0 || limit > len(images) {
		limit = len(images)
	}

	for _, image := range images[:limit] {
		logger.Trace(
			"discovered image",
			"tag", image.Tag,
			"digest", image.Digest,
		)
	}

	logger.Trace(
		"discovered images",
		"count", limit,
	)

	return n.imagesToAPIImages(images, n.discoveryLimit), nil
}

// getImagesByTags returns Image structs for the provided tags. Since the number
// of tags can often be large, this is done CONCURRENTLY, with a package-level
// semaphore being used to limit the total number of running goroutines. The
// underlying repository client also uses built-in registry-level rate-limiting
// to avoid overwhelming any registry.
func (n *newestBuildSelector) getImagesByTags(
	ctx context.Context,
	tags []string,
) ([]image, error) {
	// We'll cancel this context at the first error we encounter so that other
	// goroutines can stop early.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup

	// This channel is for collecting results
	imageCh := make(chan image, len(tags))
	// This buffered channel has room for one error
	errCh := make(chan error, 1)

	for _, tag := range tags {
		if err := metaSem.Acquire(ctx, 1); err != nil {
			return nil, fmt.Errorf(
				"error acquiring semaphore for retrieval of image with tag %q: %w",
				tag,
				err,
			)
		}
		wg.Add(1)
		go func(tag string) {
			defer wg.Done()
			defer metaSem.Release(1)
			image, err := n.repoClient.getImageByTag(ctx, tag, n.platformConstraint)
			if err != nil {
				// Report the error right away or not at all. errCh is a buffered
				// channel with room for one error, so if we can't send the error
				// right away, we know that another goroutine has already sent one.
				select {
				case errCh <- err:
					cancel() // Stop all other goroutines
				default:
				}
				return
			}
			if image == nil {
				// This shouldn't happen
				return
			}
			// imageCh is buffered and sized appropriately, so this will never block.
			imageCh <- *image
		}(tag)
	}
	wg.Wait()
	// Check for and handle errors
	select {
	case err := <-errCh:
		return nil, err
	default:
	}
	close(imageCh)
	if len(imageCh) == 0 {
		return nil, nil
	}
	// Unpack the channel into a slice
	images := make([]image, len(imageCh))
	for i := range images {
		// This will never block because we know that the channel is closed,
		// we know exactly how many items are in it, and we don't loop past that
		// number.
		images[i] = <-imageCh
	}
	return images, nil
}

// sort sorts the provided images in place, in chronologically descending order,
// breaking ties lexically by tag.
func (n *newestBuildSelector) sort(images []image) {
	slices.SortFunc(images, func(lhs, rhs image) int {
		if comp := rhs.CreatedAt.Compare(*lhs.CreatedAt); comp != 0 {
			return comp
		}
		return strings.Compare(rhs.Tag, lhs.Tag)
	})
}
