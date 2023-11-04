package image

import (
	"context"
	"regexp"
	"sort"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/akuity/kargo/internal/logging"
)

// lexicalTagSelector implements the TagSelector interface for the
// TagSelectionStrategyLexical strategy.
type lexicalTagSelector struct {
	repoClient *repositoryClient
	allowRegex *regexp.Regexp
	ignore     []string
	platform   *platformConstraint
}

// newLexicalTagSelector returns an implementation of the TagSelector
// for the TagSelectionStrategyLexical strategy.
func newLexicalTagSelector(
	repoClient *repositoryClient,
	allowRegex *regexp.Regexp,
	ignore []string,
	platform *platformConstraint,
) TagSelector {
	return &lexicalTagSelector{
		repoClient: repoClient,
		allowRegex: allowRegex,
		ignore:     ignore,
		platform:   platform,
	}
}

// SelectTag implements the TagSelector interface.
func (l *lexicalTagSelector) SelectTag(ctx context.Context) (*Tag, error) {
	logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
		"registry":            l.repoClient.registry.name,
		"image":               l.repoClient.image,
		"selectionStrategy":   TagSelectionStrategyLexical,
		"platformConstrained": l.platform != nil,
	})
	logger.Trace("selecting tag")

	ctx = logging.ContextWithLogger(ctx, logger)

	tagNames, err := l.repoClient.getTagNames(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error listing tags")
	}
	if len(tagNames) == 0 {
		logger.Trace("found no tag names")
		return nil, nil
	}
	logger.Trace("got all tag names")

	if l.allowRegex != nil || len(l.ignore) > 0 {
		matchedTagNames := make([]string, 0, len(tagNames))
		for _, tagName := range tagNames {
			if allows(tagName, l.allowRegex) && !ignores(tagName, l.ignore) {
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

	logger.Trace("sorting tags lexically")
	sortTagNamesLexically(tagNames)

	tagName := tagNames[0]
	tag, err := l.repoClient.getTagByName(ctx, tagName, l.platform)
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

// sortTagNamesLexically sorts the provided tag names in place, in lexically
// descending order.
func sortTagNamesLexically(tagNames []string) {
	sort.Slice(tagNames, func(i, j int) bool {
		return tagNames[i] > tagNames[j]
	})
}
