package governance

import (
	"context"
	"fmt"
	"strings"

	"github.com/akuity/kargo/pkg/logging"
)

func isMaintainer(cfg config, authorAssoc string) bool {
	for _, assoc := range cfg.MaintainerAssociations {
		if strings.EqualFold(authorAssoc, assoc) {
			return true
		}
	}
	return false
}

func enforceRequiredLabels(
	ctx context.Context,
	issuesClient IssuesClient,
	owner string,
	repo string,
	number int,
	existingLabels map[string]struct{},
	prefixes []string,
) error {
	logger := logging.LoggerFromContext(ctx)
	for _, prefix := range prefixes {
		if !needsLabel(prefix, existingLabels) {
			continue
		}
		label := "needs/" + prefix
		logger.Info("adding missing label", "label", label)
		if _, _, err := issuesClient.AddLabelsToIssue(
			ctx,
			owner,
			repo,
			number,
			[]string{label},
		); err != nil {
			return fmt.Errorf("error adding label %q: %w", label, err)
		}
	}
	return nil
}

// needsLabel returns true if no label with the given prefix is present
// in the existing labels.
func needsLabel(
	prefix string,
	existingLabels map[string]struct{},
) bool {
	prefixSlash := prefix + "/"
	for label := range existingLabels {
		if strings.HasPrefix(label, prefixSlash) {
			return false
		}
	}
	return true
}
