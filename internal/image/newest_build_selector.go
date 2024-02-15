package image

import (
	"context"
	"regexp"
	"sort"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/akuity/kargo/internal/logging"
)

// newestBuildSelector implements the Selector interface for
// SelectionStrategyNewestBuild.
type newestBuildSelector struct {
	repoClient *repositoryClient
	allowRegex *regexp.Regexp
	ignore     []string
	platform   *platformConstraint
}

// newNewestBuildSelector returns an implementation of the Selector interface
// for SelectionStrategyNewestBuild.
func newNewestBuildSelector(
	repoClient *repositoryClient,
	allowRegex *regexp.Regexp,
	ignore []string,
	platform *platformConstraint,
) Selector {
	return &newestBuildSelector{
		repoClient: repoClient,
		allowRegex: allowRegex,
		ignore:     ignore,
		platform:   platform,
	}
}

// Select implements the Selector interface.
func (n *newestBuildSelector) Select(ctx context.Context) (*Image, error) {
	logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
		"registry":            n.repoClient.registry.name,
		"image":               n.repoClient.image,
		"selectionStrategy":   SelectionStrategyNewestBuild,
		"platformConstrained": n.platform != nil,
	})
	logger.Trace("selecting image")

	ctx = logging.ContextWithLogger(ctx, logger)

	tags, err := n.repoClient.getTags(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error listing tags")
	}
	if len(tags) == 0 {
		logger.Trace("found no tags")
		return nil, nil
	}
	logger.Trace("got all tags")

	if n.allowRegex != nil || len(n.ignore) > 0 {
		matchedTags := make([]string, 0, len(tags))
		for _, tag := range tags {
			if allowsTag(tag, n.allowRegex) && !ignoresTag(tag, n.ignore) {
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

	logger.Trace("retrieving images for all tags that matched criteria")
	images, err := n.getImagesByTags(ctx, tags)
	if err != nil {
		return nil,
			errors.Wrapf(err, "error retrieving images for all matched tags")
	}
	if len(images) == 0 {
		// This shouldn't happen
		return nil, nil
	}

	logger.Trace("sorting images by date")
	sortImagesByDate(images)

	if n.platform == nil {
		image := images[0]
		logger.WithFields(log.Fields{
			"tag":    image.Tag,
			"digest": image.Digest.String(),
		}).Trace("found image")
		return &image, nil
	}

	tag := images[0].Tag
	digest := images[0].Digest
	image, err := n.repoClient.getImageByDigest(ctx, digest, n.platform)
	if err != nil {
		return nil, errors.Wrapf(err, "error retrieving image with digest %q", digest.String())
	}
	if image == nil {
		logger.Tracef(
			"image with digest %q was found, but did not match platform constraint",
			digest.String(),
		)
		return nil, nil
	}
	image.Tag = tag

	logger.WithFields(log.Fields{
		"tag":    image.Tag,
		"digest": image.Digest.String(),
	}).Trace("found image")
	return image, nil
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
			return nil, errors.Wrapf(
				err,
				"error acquiring semaphore for retrieval of image with tag %q",
				tag,
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
