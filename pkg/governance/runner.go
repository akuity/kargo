package governance

import (
	"context"
	"fmt"

	"github.com/akuity/kargo/pkg/component"
)

// actionContext bundles resources every action runner might need. It's
// constructed by callers of executeActions and threaded through to each
// runner.
type actionContext struct {
	cfg          config
	issuesClient IssuesClient
	prsClient    PullRequestsClient
	owner        string
	repo         string
	number       int
	isPR         bool
	templateData map[string]string
}

// actionRunner is the behavior side of an action — looked up by kind
// name and invoked with the action's raw YAML config bytes.
type actionRunner interface {
	run(ctx context.Context, ac *actionContext, cfg []byte) error
}

type actionRunnerRegistration = component.NameBasedRegistration[
	actionRunner, struct{},
]

// defaultActionRunnerRegistry holds all built-in runners. They
// self-register from their respective files' init().
var defaultActionRunnerRegistry = component.MustNewNameBasedRegistry[
	actionRunner, struct{},
](&component.NameBasedRegistryOptions{})

// executeActions dispatches each action through the runner registry by
// the action's kind. Operations are fail-fast: a failed action
// short-circuits subsequent actions in the same list. Operator-authored
// action sequences have intentional ordering and a partial run is rarely
// desirable.
func executeActions(
	ctx context.Context,
	ac *actionContext,
	actions []action,
) error {
	for _, a := range actions {
		reg, err := defaultActionRunnerRegistry.Get(a.kind)
		if err != nil {
			return fmt.Errorf("unknown action %q: %w", a.kind, err)
		}
		if err := reg.Value.run(ctx, ac, a.config); err != nil {
			return fmt.Errorf("error running %s action: %w", a.kind, err)
		}
	}
	return nil
}
