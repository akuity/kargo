package governance

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/v76/github"

	"github.com/akuity/kargo/pkg/logging"
)

// prHandler handles pull request-related events for a specific repository
// according to specific configuration.
type prHandler struct {
	cfg          config
	owner        string
	repo         string
	issuesClient IssuesClient
	prsClient    PullRequestsClient
}

// handleOpened is the handler for pull_request.opened events. It performs
// the following, in order:
//
//  1. Auto-assigns the PR to its author.
//  2. Inherits labels from the linked issue (if any) based on configured
//     prefixes.
//  3. Enforces required-label prefixes, flagging any that are still missing
//     after inheritance.
//  4. Applies PR policy — any configured actions for PRs with no linked
//     issue or with a linked issue that carries blocking labels.
//     Maintainers and bots are exempt from this step only.
//
// Labeling happens before policy so that maintainers have full context on
// the PR's subject matter regardless of whether policy ends up drafting or
// closing it. Inheriting a blocking label onto the PR is intentional: it's
// an additional signal to the author that they've done something
// prematurely.
func (h *prHandler) handleOpened(
	ctx context.Context,
	event *github.PullRequestEvent,
) error {
	if event == nil {
		return nil
	}

	pr := event.GetPullRequest()
	if pr == nil {
		return nil
	}
	number := pr.GetNumber()

	logger := logging.LoggerFromContext(ctx).WithValues("pr", number)
	ctx = logging.ContextWithLogger(ctx, logger)

	login := event.GetSender().GetLogin()

	// Steps run independently: each one's failure is logged and collected,
	// but does not prevent subsequent steps from running. The aggregated
	// error is returned at the end so the webhook delivery shows red in
	// GitHub's UI (GitHub does not auto-retry).
	var errs []error

	// 1. Auto-assign the PR to its author.
	if _, _, err := h.issuesClient.AddAssignees(
		ctx,
		h.owner,
		h.repo,
		number,
		[]string{login},
	); err != nil {
		logger.Error(err, "error assigning PR to author")
		errs = append(errs, fmt.Errorf("assign PR to author: %w", err))
	}

	issueNumber := parseLinkedIssue(pr.GetBody())

	// 2. Inherit labels from the linked issue. If this fails, inheritedLabels
	// is nil and the subsequent required-label check operates on only the
	// labels the PR already has.
	inheritedLabels, err := h.inheritLabels(ctx, number, issueNumber)
	if err != nil {
		logger.Error(err, "error inheriting labels")
		errs = append(errs, fmt.Errorf("inherit labels: %w", err))
	}

	// 3. Enforce required-label prefixes, accounting for labels the PR
	// already has plus any we just inherited.
	if h.cfg.PullRequests != nil {
		existingLabels := make(map[string]struct{})
		for _, l := range pr.Labels {
			existingLabels[l.GetName()] = struct{}{}
		}
		for _, l := range inheritedLabels {
			existingLabels[l] = struct{}{}
		}
		if err := enforceRequiredLabels(
			ctx,
			h.issuesClient,
			h.owner,
			h.repo,
			number,
			existingLabels,
			h.cfg.PullRequests.RequiredLabelPrefixes,
		); err != nil {
			logger.Error(err, "error enforcing required labels")
			errs = append(errs, fmt.Errorf("enforce required labels: %w", err))
		}
	}

	// 4. Apply PR policy. Maintainers and bots are exempt.
	author := pr.GetAuthorAssociation()
	switch {
	case h.isExemptFromPRPolicy(author, login):
		logger.Debug("author is exempt from policy, skipping")
	case h.cfg.PullRequests == nil:
		// No policy configured.
	default:
		if err := applyPRPolicy(
			ctx,
			h.cfg,
			h.issuesClient,
			h.prsClient,
			h.owner,
			h.repo,
			number,
			issueNumber,
		); err != nil {
			logger.Error(err, "error applying PR policy")
			errs = append(errs, fmt.Errorf("apply PR policy: %w", err))
		}
	}

	return errors.Join(errs...)
}

// applyPRPolicy runs the configured policy actions when either:
//   - The PR has no linked issue and NoLinkedIssue is configured; or
//   - The PR's linked issue carries one or more labels listed in
//     BlockedIssue.BlockingLabels and BlockedIssue is configured.
//
// Otherwise it's a no-op. Callers should have already verified the PR's
// author is not exempt from policy.
func applyPRPolicy(
	ctx context.Context,
	cfg config,
	issuesClient IssuesClient,
	prsClient PullRequestsClient,
	owner string,
	repo string,
	number int,
	issueNumber int,
) error {
	logger := logging.LoggerFromContext(ctx)

	if issueNumber == 0 {
		if cfg.PullRequests.NoLinkedIssue == nil {
			return nil
		}
		logger.Info("PR has no linked issue, applying policy")
		return executeActions(
			ctx,
			cfg,
			issuesClient,
			prsClient,
			owner,
			repo,
			number,
			true,
			cfg.PullRequests.NoLinkedIssue.Actions,
			nil,
		)
	}

	if cfg.PullRequests.BlockedIssue == nil {
		return nil
	}

	logger = logger.WithValues("linkedIssue", issueNumber)
	ctx = logging.ContextWithLogger(ctx, logger)

	issue, _, err := issuesClient.Get(ctx, owner, repo, issueNumber)
	if err != nil {
		return fmt.Errorf("error fetching linked issue: %w", err)
	}

	issueLabels := make(map[string]bool)
	for _, l := range issue.Labels {
		issueLabels[l.GetName()] = true
	}

	var blockers []string
	for _, blocking := range cfg.PullRequests.BlockedIssue.BlockingLabels {
		if issueLabels[blocking] {
			blockers = append(blockers, blocking)
		}
	}

	if len(blockers) == 0 {
		return nil
	}

	logger.Info("linked issue has blocking labels, applying policy",
		"blockers", blockers,
	)
	return executeActions(
		ctx,
		cfg,
		issuesClient,
		prsClient,
		owner,
		repo,
		number,
		true,
		cfg.PullRequests.BlockedIssue.Actions,
		map[string]string{
			"IssueNumber":    fmt.Sprintf("%d", issueNumber),
			"BlockingLabels": formatBlockers(blockers),
		},
	)
}

// inheritLabels copies labels with configured prefixes from the linked
// issue to the PR. Returns the list of labels that were added.
func (h *prHandler) inheritLabels(
	ctx context.Context,
	prNumber int,
	issueNumber int,
) ([]string, error) {
	if issueNumber == 0 || h.cfg.PullRequests == nil ||
		len(h.cfg.PullRequests.InheritedLabelPrefixes) == 0 {
		return nil, nil
	}

	logger := logging.LoggerFromContext(ctx)

	issue, _, err := h.issuesClient.Get(ctx, h.owner, h.repo, issueNumber)
	if err != nil {
		return nil, fmt.Errorf(
			"error fetching linked issue for label inheritance: %w", err,
		)
	}

	var toAdd []string
	for _, l := range issue.Labels {
		name := l.GetName()
		for _, prefix := range h.cfg.PullRequests.InheritedLabelPrefixes {
			if strings.HasPrefix(name, prefix+"/") {
				toAdd = append(toAdd, name)
				break
			}
		}
	}

	if len(toAdd) == 0 {
		return nil, nil
	}

	logger.Info("inheriting labels from linked issue",
		"labels", toAdd,
		"linkedIssue", issueNumber,
	)
	if _, _, err := h.issuesClient.AddLabelsToIssue(
		ctx,
		h.owner,
		h.repo,
		prNumber,
		toAdd,
	); err != nil {
		return nil, fmt.Errorf("error adding inherited labels: %w", err)
	}
	return toAdd, nil
}

func (h *prHandler) isExemptFromPRPolicy(
	authorAssoc string,
	login string,
) bool {
	if h.cfg.PullRequests != nil {
		if h.cfg.PullRequests.ExemptMaintainers && isMaintainer(h.cfg, authorAssoc) {
			return true
		}
		if h.cfg.PullRequests.ExemptBots && strings.HasSuffix(login, "[bot]") {
			return true
		}
	}
	return false
}

func formatBlockers(blockers []string) string {
	formatted := make([]string, len(blockers))
	for i, b := range blockers {
		formatted[i] = "`" + b + "`"
	}
	return strings.Join(formatted, ", ")
}

// issueRefPattern matches GitHub closing keyword syntax:
//   - Closes #123
//   - Fixes #123
//   - Resolves #123
//   - Close #123, Fix #123, Resolve #123
//   - Closed #123, Fixed #123, Resolved #123
//   - Full URL variants: Closes https://github.com/owner/repo/issues/123
var issueRefPattern = regexp.MustCompile(
	`(?i)(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\s+` +
		`(?:https://github\.com/[^/]+/[^/]+/issues/)?#?(\d+)`,
)

// parseLinkedIssue extracts the first linked issue number from a PR
// body. Returns 0 if no linked issue is found.
func parseLinkedIssue(body string) int {
	match := issueRefPattern.FindStringSubmatch(body)
	if len(match) < 2 {
		return 0
	}
	n, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return n
}
