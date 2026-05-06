package governance

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/google/go-github/v76/github"
)

const actionKindClose = "close"

// stateReasonNotPlanned is the value used in IssueRequest.StateReason
// when closing an issue. PRs don't take a state_reason.
const stateReasonNotPlanned = "not_planned"

func init() {
	defaultActionRunnerRegistry.MustRegister(actionRunnerRegistration{
		Name:  actionKindClose,
		Value: closeRunner{},
	})
}

// closeRunner closes the issue or pull request. Its config is a bool —
// `false` is a no-op (preserves the previous semantics where the close
// action was opt-in).
type closeRunner struct{}

func (closeRunner) run(
	ctx context.Context,
	ac *actionContext,
	cfg []byte,
) error {
	var enabled bool
	if err := yaml.Unmarshal(cfg, &enabled); err != nil {
		return fmt.Errorf("decoding close config: %w", err)
	}
	if !enabled {
		return nil
	}
	if ac.isPR {
		if _, _, err := ac.prsClient.Edit(
			ctx, ac.owner, ac.repo, ac.number,
			&github.PullRequest{State: github.Ptr(prStateClosed)},
		); err != nil {
			return fmt.Errorf("error closing PR: %w", err)
		}
		return nil
	}
	if _, _, err := ac.issuesClient.Edit(
		ctx, ac.owner, ac.repo, ac.number,
		&github.IssueRequest{
			State:       github.Ptr(issueStateClosed),
			StateReason: github.Ptr(stateReasonNotPlanned),
		},
	); err != nil {
		return fmt.Errorf("error closing issue: %w", err)
	}
	return nil
}
