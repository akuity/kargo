package commit

import (
	"context"
	"fmt"
	"slices"
	"strings"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/logging"
)

func init() {
	selectorReg.register(
		kargoapi.CommitSelectionStrategyLexical,
		selectorRegistration{
			predicate: func(sub kargoapi.GitSubscription) bool {
				return sub.CommitSelectionStrategy == kargoapi.CommitSelectionStrategyLexical
			},
			factory: newLexicalSelector,
		},
	)
}

// lexicalSelector implements the Selector interface for
// kargoapi.CommitSelectionStrategyLexical.
type lexicalSelector struct {
	*tagBasedSelector
}

func newLexicalSelector(
	sub kargoapi.GitSubscription,
	creds *git.RepoCredentials,
) (Selector, error) {
	tagBased, err := newTagBasedSelector(sub, creds)
	if err != nil {
		return nil, fmt.Errorf("error building tag based selector: %w", err)
	}
	return &lexicalSelector{tagBasedSelector: tagBased}, nil
}

// Select implements the Selector interface.
func (l *lexicalSelector) Select(ctx context.Context) (
	[]kargoapi.DiscoveredCommit,
	error,
) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		l.getLoggerContext(),
		"selectionStrategy", kargoapi.CommitSelectionStrategyLexical,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	repo, err := l.clone(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = repo.Close()
	}()

	tags, err := repo.ListTags()
	if err != nil {
		return nil, err
	}

	tags = l.filterTags(tags)

	if tags, err = l.filterTagsByExpression(tags); err != nil {
		return nil, fmt.Errorf("error filtering tags by expression: %w", err)
	}

	// Sort in reverse lexicographic order.
	slices.SortFunc(tags, func(i, j git.TagMetadata) int {
		return strings.Compare(j.Tag, i.Tag)
	})

	if tags, err = l.filterTagsByDiffPathsFn(repo, tags); err != nil {
		return nil, fmt.Errorf("error filtering tags by paths: %w", err)
	}

	return l.tagsToAPICommits(ctx, tags), nil
}
