package governance

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
)

const actionKindConvertToDraft = "convertToDraft"

func init() {
	defaultActionRunnerRegistry.MustRegister(actionRunnerRegistration{
		Name:  actionKindConvertToDraft,
		Value: convertToDraftRunner{},
	})
}

// convertToDraftRunner converts a pull request to a draft. Its config is
// a bool — `false` is a no-op. Silently skipped for issues (the YAML
// equivalent for the issues code path can include this action without
// erroring).
type convertToDraftRunner struct{}

func (convertToDraftRunner) run(
	ctx context.Context,
	ac *actionContext,
	cfg []byte,
) error {
	var enabled bool
	if err := yaml.Unmarshal(cfg, &enabled); err != nil {
		return fmt.Errorf("decoding convertToDraft config: %w", err)
	}
	if !enabled || !ac.isPR {
		return nil
	}
	if err := ac.prsClient.ConvertToDraft(
		ctx, ac.owner, ac.repo, ac.number,
	); err != nil {
		return fmt.Errorf("error converting PR to draft: %w", err)
	}
	return nil
}
