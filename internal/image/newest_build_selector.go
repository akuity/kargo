package image

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/akuity/kargo/internal/logging"
)

// newestBuildSelector implements the Selector interface for
// SelectionStrategyNewestBuild.
type newestBuildSelector struct {
	repoClient *repositoryClient
	opts       SelectorOptions
}

// newNewestBuildSelector returns an implementation of the Selector interface
// for SelectionStrategyNewestBuild.
func newNewestBuildSelector(repoClient *repositoryClient, opts SelectorOptions) Selector {
	return &newestBuildSelector{
		repoClient: repoClient,
		opts:       opts,
	}
}

// Select implements the Selector interface.
func (n *newestBuildSelector) Select(ctx context.Context) ([]Image, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"registry", n.repoClient.registry.name,
		"image", n.repoClient.repoURL,
		"selectionStrategy", SelectionStrategyNewestBuild,
		"platformConstrained", n.opts.platform != nil,
		"discoveryLimit", n.opts.DiscoveryLimit,
	)
	logger.Trace("discovering images")

	ctx = logging.ContextWithLogger(ctx, logger)

	images, err := n.selectImages(ctx)
	if err != nil || len(images) == 0 {
		return nil, err
	}

	limit := n.opts.DiscoveryLimit
	if limit == 0 || limit > len(images) {
		limit = len(images)
	}

	if n.opts.platform == nil {
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
		return images[:limit], nil
	}

	// TODO(hidde): this could be more efficient, as we are fetching the image
	// _again_ to check if it matches the platform constraint (although we do
	// cache it indefinitely). We should consider refactoring this to avoid
	// fetching the image twice.
	discoveredImages := make([]Image, 0, limit)
	for _, image := range images {
		if len(discoveredImages) >= limit {
			break
		}

		discoveredImage, err := n.repoClient.getImageByDigest(ctx, image.Digest, n.opts.platform)
		if err != nil {
			return nil, fmt.Errorf("error retrieving image with digest %q: %w", image.Digest, err)
		}

		if discoveredImage == nil {
			logger.Trace(
				"image was found, but did not match platform constraint",
				"digest", image.Digest,
			)
			continue
		}

		discoveredImage.Tag = image.Tag
		discoveredImages = append(discoveredImages, *discoveredImage)

		logger.Trace(
			"discovered image",
			"tag", discoveredImage.Tag,
			"digest", discoveredImage.Digest,
			"createdAt", discoveredImage.CreatedAt.Format(time.RFC3339),
		)
	}

	if len(discoveredImages) == 0 {
		logger.Trace("no images matched platform constraint")
		return nil, nil
	}

	logger.Trace(
		"discovered images",
		"count", len(discoveredImages),
	)
	return discoveredImages, nil
}

func (n *newestBuildSelector) selectImages(ctx context.Context) ([]Image, error) {
	logger := logging.LoggerFromContext(ctx)

	tags, err := n.repoClient.getTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing tags: %w", err)
	}
	if len(tags) == 0 {
		logger.Trace("found no tags")
		return nil, nil
	}
	logger.Trace("got all tags")

	if n.opts.allowRegex != nil || len(n.opts.Ignore) > 0 {
		matchedTags := make([]string, 0, len(tags))
		for _, tag := range tags {
			if allowsTag(tag, n.opts.allowRegex) && !ignoresTag(tag, n.opts.Ignore) {
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

	logger.Trace("retrieving images for all tags that matched criteria")
	images, err := n.getImagesByTags(ctx, tags)
	if err != nil {
		return nil, fmt.Errorf("error retrieving images for all matched tags: %w", err)
	}
	if len(images) == 0 {
		// This shouldn't happen
		return nil, nil
	}

	logger.Trace("sorting images by date")
	sortImagesByDate(images)
	return images, nil
}

// getImagesByTags returns Image structs for the provided tags. Since the number
// of tags can often be large, this is done concurrently, with a package-level
// semaphore being used to limit the total number of running goroutines. The
// underlying repository client also uses built-in registry-level rate-limiting
// to avoid overwhelming any registry.
func (n *newestBuildSelector) getImagesByTags(
	ctx context.Context,
	tags []string,
) ([]Image, error) {
	// We'll cancel this context at the first error we encounter so that other
	// goroutines can stop early.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup

	// This channel is for collecting results
	imageCh := make(chan Image, len(tags))
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
			image, err := n.repoClient.getImageByTag(ctx, tag, nil)
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
	images := make([]Image, len(imageCh))
	for i := range images {
		// This will never block because we know that the channel is closed,
		// we know exactly how many items are in it, and we don't loop past that
		// number.
		images[i] = <-imageCh
	}
	return images, nil
}

// sortImagesByDate sorts the provided images in place, in chronologically
// descending order, breaking ties lexically by tag.
func sortImagesByDate(images []Image) {
	sort.Slice(images, func(i, j int) bool {
		if images[i].CreatedAt.Equal(*images[j].CreatedAt) {
			// If there's a tie on the date, break the tie lexically by name
			return images[i].Tag > images[j].Tag
		}
		return images[i].CreatedAt.After(*images[j].CreatedAt)
	})
}
