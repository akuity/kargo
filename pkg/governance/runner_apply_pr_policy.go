package governance

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
)

const actionKindApplyPRPolicy = "applyPRPolicy"

func init() {
	defaultActionRunnerRegistry.MustRegister(actionRunnerRegistration{
		Name:  actionKindApplyPRPolicy,
		Value: applyPRPolicyRunner{},
	})
}

// applyPRPolicyRunner re-evaluates PR policy against the target. Its
// config is a bool — `false` is a no-op. Silently skipped for issues.
//
// Like the webhook-driven PR policy path, this consults exemptions:
// blocking outcomes are gated by exemption while OnPass runs regardless.
type applyPRPolicyRunner struct{}

func (applyPRPolicyRunner) run(
	ctx context.Context,
	ac *actionContext,
	cfg []byte,
) error {
	var enabled bool
	if err := yaml.Unmarshal(cfg, &enabled); err != nil {
		return fmt.Errorf("decoding applyPRPolicy config: %w", err)
	}
	if !enabled || !ac.isPR {
		return nil
	}
	pr, _, err := ac.prsClient.Get(ctx, ac.owner, ac.repo, ac.number)
	if err != nil {
		return fmt.Errorf("error fetching PR for policy check: %w", err)
	}
	if err := applyPRPolicy(
		ctx,
		ac.cfg,
		ac.issuesClient,
		ac.prsClient,
		ac.owner,
		ac.repo,
		pr,
		pr.GetUser().GetLogin(),
	); err != nil {
		return fmt.Errorf("error applying PR policy: %w", err)
	}
	return nil
}
