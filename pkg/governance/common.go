package governance

import (
	"context"
	"fmt"
	"strings"

	"github.com/akuity/kargo/pkg/logging"
)

// isMaintainer reports whether the given login is considered a maintainer
// per the configured MaintainerAssociations.
//
// Fast path: the supplied authorAssoc (from a webhook payload) is matched
// against the configured associations.
//
// Slow path: if MEMBER is configured but the fast path didn't match, fall
// back to querying org membership directly. This catches concealed (private)
// org members — GitHub reports their author_association as CONTRIBUTOR in
// webhook payloads regardless of the App's permissions, but the
// orgs/{org}/members/{user} endpoint honors the App's Organization Members
// permission and returns the true membership state.
//
// The fallback is skipped when orgsClient is nil, login is empty, or MEMBER
// is not in the configured associations.
func isMaintainer(
	ctx context.Context,
	cfg config,
	org string,
	authorAssoc string,
	login string,
	orgsClient OrganizationsClient,
) (bool, error) {
	wantMember := false
	for _, assoc := range cfg.MaintainerAssociations {
		if strings.EqualFold(authorAssoc, assoc) {
			return true, nil
		}
		if strings.EqualFold(assoc, "MEMBER") {
			wantMember = true
		}
	}
	if !wantMember || login == "" || orgsClient == nil {
		return false, nil
	}
	isMember, _, err := orgsClient.IsMember(ctx, org, login)
	if err != nil {
		return false, fmt.Errorf(
			"error checking org membership of %q: %w", login, err,
		)
	}
	return isMember, nil
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
