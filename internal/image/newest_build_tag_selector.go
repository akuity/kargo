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

// newestBuildTagSelector implements the TagSelector interface for the
// TagSelectionStrategyNewestBuild strategy.
type newestBuildTagSelector struct {
	repoClient *repositoryClient
	allowRegex *regexp.Regexp
	ignore     []string
	platform   *platformConstraint
}

// newNewestBuildTagSelector returns an implementation of the TagSelector
// for the TagSelectionStrategyNewestBuild strategy.
func newNewestBuildTagSelector(
	repoClient *repositoryClient,
	allowRegex *regexp.Regexp,
	ignore []string,
	platform *platformConstraint,
) TagSelector {
	return &newestBuildTagSelector{
		repoClient: repoClient,
		allowRegex: allowRegex,
		ignore:     ignore,
		platform:   platform,
	}
}

// SelectTag implements the TagSelector interface.
func (n *newestBuildTagSelector) SelectTag(ctx context.Context) (*Tag, error) {
	logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
		"registry":            n.repoClient.registry.name,
		"image":               n.repoClient.image,
		"selectionStrategy":   TagSelectionStrategyNewestBuild,
		"platformConstrained": n.platform != nil,
	})
	logger.Trace("selecting tag")

	ctx = logging.ContextWithLogger(ctx, logger)

	tagNames, err := n.repoClient.getTagNames(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error listing tags")
	}
	if len(tagNames) == 0 {
		logger.Trace("found no tag names")
		return nil, nil
	}
	logger.Trace("got all tag names")

	if n.allowRegex != nil || len(n.ignore) > 0 {
		matchedTagNames := make([]string, 0, len(tagNames))
		for _, tagName := range tagNames {
			if allows(tagName, n.allowRegex) && !ignores(tagName, n.ignore) {
				matchedTagNames = append(matchedTagNames, tagName)
			}
		}
		if len(matchedTagNames) == 0 {
			logger.Trace("no tag names matched criteria")
			return nil, nil
		}
		tagNames = matchedTagNames
	}
	logger.Tracef("%d tag names matched criteria", len(tagNames))

	logger.Trace("retrieving all tags that matched criteria")
	tags, err := n.getTagsByNames(ctx, tagNames)
	if err != nil {
		return nil,
			errors.Wrapf(err, "error retrieving tags for all matched tag names")
	}
	if len(tags) == 0 {
		// This shouldn't happen
		return nil, nil
	}

	logger.Trace("sorting tags by date")
	sortTagsByDate(tags)

	if n.platform == nil {
		tag := tags[0]
		logger.WithFields(log.Fields{
			"name":   tag.Name,
			"digest": tag.Digest.String(),
		}).Trace("found tag")
		return &tag, nil
	}

	tagName := tags[0].Name
	digest := tags[0].Digest
	tag, err := n.repoClient.getTagByDigest(ctx, digest, n.platform)
	if err != nil {
		return nil, errors.Wrapf(err, "error retrieving tag %q", digest.String())
	}
	if tag == nil {
		logger.Tracef(
			"tag %q was found, but did not match platform constraint",
			tagName,
		)
		return nil, nil
	}
	tag.Name = tagName

	logger.WithFields(log.Fields{
		"name":   tag.Name,
		"digest": tag.Digest.String(),
	}).Trace("found tag")
	return tag, nil
}

// getTagsByNames returns Tag structs for the provided tag names. Since the
// number of tag names can often be large, this is done concurrently, with a
// package-level semaphore being used to limit the total number of running
// goroutines. The underlying repository client also uses built-in
// registry-level rate-limiting to avoid overwhelming any registry.
func (n *newestBuildTagSelector) getTagsByNames(
	ctx context.Context,
	tagNames []string,
) ([]Tag, error) {
	// We'll cancel this context at the first error we encounter so that other
	// goroutines can stop early.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup

	// This channel is for collecting results
	tagCh := make(chan Tag, len(tagNames))
	// This buffered channel has room for one error
	errCh := make(chan error, 1)

	for _, tagName := range tagNames {
		if err := metaSem.Acquire(ctx, 1); err != nil {
			return nil, errors.Wrapf(
				err,
				"error acquiring semaphore for retrieval of tag %q",
				tagName,
			)
		}
		wg.Add(1)
		go func(tagName string) {
			defer wg.Done()
			defer metaSem.Release(1)
			tag, err := n.repoClient.getTagByName(ctx, tagName, nil)
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
			if tag == nil {
				// This shouldn't happen
				return
			}
			// tagCh is buffered and sized appropriately, so this will never block.
			tagCh <- *tag
		}(tagName)
	}
	wg.Wait()
	// Check for and handle errors
	select {
	case err := <-errCh:
		return nil, err
	default:
	}
	close(tagCh)
	if len(tagCh) == 0 {
		return nil, nil
	}
	// Unpack the channel into a slice
	tags := make([]Tag, len(tagCh))
	for i := range tags {
		// This will never block because we know that the channel is closed,
		// we know exactly how many items are in it, and we don't loop past that
		// number.
		tags[i] = <-tagCh
	}
	return tags, nil
}

// sortTagsByDate sorts the provided tags in place, in chronologically
// descending order, breaking ties lexically by tag name.
func sortTagsByDate(tags []Tag) {
	sort.Slice(tags, func(i, j int) bool {
		if tags[i].CreatedAt.Equal(*tags[j].CreatedAt) {
			// If there's a tie on the date, break the tie lexically by name
			return tags[i].Name > tags[j].Name
		}
		return tags[i].CreatedAt.After(*tags[j].CreatedAt)
	})
}
