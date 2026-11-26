package governance

import (
	"context"
	"errors"
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

// handleOpened is the handler for the "issues.opened" event. It enforces
// required-label prefixes, then runs any configured OnOpened actions. A
// failure in the first step does not prevent the second from running; errors
// from both are returned together.
func (h *issueHandler) handleOpened(
	ctx context.Context,
	event *github.IssuesEvent,
) error {
	if event == nil || h.cfg.Issues == nil {
		return nil
	}

	issue := event.GetIssue()
	if issue == nil {
		return nil
	}
	number := issue.GetNumber()

	logger := logging.LoggerFromContext(ctx).WithValues("issue", number)
	ctx = logging.ContextWithLogger(ctx, logger)

	var errs []error

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
		logger.Error(err, "error enforcing required labels")
		errs = append(
			errs,
			fmt.Errorf("error enforcing required labels: %w", err),
		)
	}

	if h.cfg.Issues.OnOpened != nil {
		if err := executeActions(
			ctx,
			&actionContext{
				repoContext: h.repoContext,
				number:      number,
			},
			h.cfg.Issues.OnOpened.Actions,
		); err != nil {
			logger.Error(err, "error running onOpened actions")
			errs = append(
				errs,
				fmt.Errorf("error running onOpened actions: %w", err),
			)
		}
	}

	return errors.Join(errs...)
}
