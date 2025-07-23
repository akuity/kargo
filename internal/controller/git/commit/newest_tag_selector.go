package commit

import (
	"context"
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/logging"
)

func init() {
	selectorReg.register(
		kargoapi.CommitSelectionStrategyNewestTag,
		selectorRegistration{
			predicate: func(sub kargoapi.GitSubscription) bool {
				return sub.CommitSelectionStrategy == kargoapi.CommitSelectionStrategyNewestTag
			},
			factory: newNewestTagSelector,
		},
	)
}

// newestTagSelector implements the Selector interface for
// kargoapi.CommitSelectionStrategyNewestTag.
type newestTagSelector struct {
	*tagBasedSelector
}

func newNewestTagSelector(
	sub kargoapi.GitSubscription,
	creds *git.RepoCredentials,
) (Selector, error) {
	tagBased, err := newTagBasedSelector(sub, creds)
	if err != nil {
		return nil, fmt.Errorf("error building tag based selector: %w", err)
	}
	return &newestTagSelector{tagBasedSelector: tagBased}, nil
}

// Select implements the Selector interface.
func (n *newestTagSelector) Select(ctx context.Context) (
	[]kargoapi.DiscoveredCommit,
	error,
) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		n.getLoggerContext(),
		"selectionStrategy", kargoapi.CommitSelectionStrategyNewestTag,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	repo, err := n.clone(ctx)
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

	tags = n.filterTags(tags)

	if tags, err = n.filterTagsByExpression(tags); err != nil {
		return nil, fmt.Errorf("error filtering tags by expression: %w", err)
	}

	// Note: Tags are already sorted in descending order by creation date when
	// retrieved. No further sorting is required.

	if tags, err = n.filterTagsByDiffPathsFn(repo, tags); err != nil {
		return nil, fmt.Errorf("error filtering tags by paths: %w", err)
	}

	return n.tagsToAPICommits(ctx, tags), nil
}
