package commit

import (
	"context"
	"fmt"
	"slices"
	"strings"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/logging"
)

func init() {
	defaultSelectorRegistry.MustRegister(
		selectorRegistration{
			Predicate: func(_ context.Context, sub kargoapi.GitSubscription) (bool, error) {
				return sub.CommitSelectionStrategy == kargoapi.CommitSelectionStrategyLexical, nil
			},
			Value: newLexicalSelector,
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
	loggerCtx := append(
		l.getLoggerContext(),
		"selectionStrategy", kargoapi.CommitSelectionStrategyLexical,
	)
	logger := logging.LoggerFromContext(ctx).WithValues(loggerCtx...)
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
