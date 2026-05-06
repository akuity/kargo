package governance

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/goccy/go-yaml"
	"github.com/google/go-github/v76/github"
)

const actionKindRemoveLabels = "removeLabels"

func init() {
	defaultActionRunnerRegistry.MustRegister(actionRunnerRegistration{
		Name:  actionKindRemoveLabels,
		Value: removeLabelsRunner{},
	})
}

// removeLabelsRunner removes labels from the issue or pull request. Its
// config is a list of label names. Labels not currently present on the
// resource are a no-op (GitHub returns 404 for those).
type removeLabelsRunner struct{}

func (removeLabelsRunner) run(
	ctx context.Context,
	ac *actionContext,
	cfg []byte,
) error {
	var labels []string
	if err := yaml.Unmarshal(cfg, &labels); err != nil {
		return fmt.Errorf("decoding removeLabels config: %w", err)
	}
	for _, label := range labels {
		_, err := ac.issuesClient.RemoveLabelForIssue(
			ctx, ac.owner, ac.repo, ac.number, label,
		)
		if err == nil {
			continue
		}
		// 404: label wasn't on the resource. End state ("not present")
		// is what we wanted; treat as success.
		var gerr *github.ErrorResponse
		if errors.As(err, &gerr) &&
			gerr.Response != nil &&
			gerr.Response.StatusCode == http.StatusNotFound {
			continue
		}
		return fmt.Errorf("error removing label %q: %w", label, err)
	}
	return nil
}
