package governance

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
)

const actionKindAddLabels = "addLabels"

func init() {
	defaultActionRunnerRegistry.MustRegister(actionRunnerRegistration{
		Name:  actionKindAddLabels,
		Value: addLabelsRunner{},
	})
}

// addLabelsRunner adds labels to the issue or pull request. Its config
// is a list of label names.
type addLabelsRunner struct{}

func (addLabelsRunner) run(
	ctx context.Context,
	ac *actionContext,
	cfg []byte,
) error {
	var labels []string
	if err := yaml.Unmarshal(cfg, &labels); err != nil {
		return fmt.Errorf("decoding addLabels config: %w", err)
	}
	if len(labels) == 0 {
		return nil
	}
	if _, _, err := ac.issuesClient.AddLabelsToIssue(
		ctx, ac.owner, ac.repo, ac.number, labels,
	); err != nil {
		return fmt.Errorf("error adding labels: %w", err)
	}
	return nil
}
