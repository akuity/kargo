package governance

import (
	"context"
	"fmt"

	"github.com/google/go-github/v76/github"

	"github.com/akuity/kargo/pkg/logging"
)

// issueHandler handles issue-related events for a specific repository
// according to specific configuration. It embeds repoContext so the shared
// per-repository dependencies and their methods are accessible directly on
// the handler.
type issueHandler struct {
	repoContext
}

// handleOpened is the handler for the "issues.opened" event.
func (h *issueHandler) handleOpened(
	ctx context.Context,
	event *github.IssuesEvent,
) error {
	if event == nil || h.cfg.Issues == nil ||
		len(h.cfg.Issues.RequiredLabelPrefixes) == 0 {
		return nil
	}

	issue := event.GetIssue()
	if issue == nil {
		return nil
	}
	number := issue.GetNumber()

	logger := logging.LoggerFromContext(ctx).WithValues("issue", number)
	ctx = logging.ContextWithLogger(ctx, logger)

	existingLabels := make(map[string]struct{})
	for _, l := range issue.Labels {
		existingLabels[l.GetName()] = struct{}{}
	}

	if err := h.enforceRequiredLabels(
		ctx,
		number,
		existingLabels,
		h.cfg.Issues.RequiredLabelPrefixes,
	); err != nil {
		return fmt.Errorf("error enforcing required labels: %w", err)
	}

	return nil
}
