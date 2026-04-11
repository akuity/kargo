package governance

import (
	"context"
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

// handleOpened is the handler for pull_request.opened events. It performs the
// following actions:
//   - Auto-assigns the PR to its author.
//   - Checks PR policy: if the PR has no linked issue or if the linked issue
//     has blocking labels, take configured actions (e.g. add labels, comment,
//     close).
//   - Inherits labels from the linked issue based on configured prefixes.
//   - Flags missing required labels based on configured prefixes.
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

	// Auto-assign the PR to its author.
	login := event.GetSender().GetLogin()
	if _, _, err := h.issuesClient.AddAssignees(
		ctx,
		h.owner,
		h.repo,
		number,
		[]string{login},
	); err != nil {
		return fmt.Errorf("error assigning PR to author: %w", err)
	}

	// Policy check: exempt maintainers and bots.
	author := pr.GetAuthorAssociation()
	if h.isExemptFromPRPolicy(author, login) {
		logger.Debug("author is exempt from policy check, skipping")
	} else if h.cfg.PullRequests != nil {
		closed, err := h.checkPRPolicy(ctx, number, pr)
		if err != nil {
			return fmt.Errorf("error checking PR policy: %w", err)
		}
		if closed {
			return nil
		}
	}

	// Label inheritance: copy labels from linked issue to PR.
	issueNumber := h.parseLinkedIssue(pr.GetBody())
	inheritedLabels, err := h.inheritLabels(ctx, number, issueNumber)
	if err != nil {
		return fmt.Errorf("error inheriting labels: %w", err)
	}

	// Label governance: flag missing required labels, accounting for
	// both the PR's own labels and any we just inherited.
	existingLabels := make(map[string]struct{})
	for _, l := range pr.Labels {
		existingLabels[l.GetName()] = struct{}{}
	}
	for _, l := range inheritedLabels {
		existingLabels[l] = struct{}{}
	}
	if h.cfg.PullRequests != nil {
		if err := enforceRequiredLabels(
			ctx,
			h.issuesClient,
			h.owner,
			h.repo,
			number,
			existingLabels,
			h.cfg.PullRequests.RequiredLabelPrefixes,
		); err != nil {
			return fmt.Errorf("error enforcing required labels: %w", err)
		}
	}

	return nil
}

// checkPRPolicy checks whether a PR has a linked issue and whether
// that issue has blocking labels. Returns true if the PR was closed.
func (h *prHandler) checkPRPolicy(
	ctx context.Context,
	number int,
	pr *github.PullRequest,
) (bool, error) {
	if pr == nil {
		return false, nil
	}

	logger := logging.LoggerFromContext(ctx)
	issueNumber := h.parseLinkedIssue(pr.GetBody())

	if issueNumber == 0 && h.cfg.PullRequests.NoLinkedIssue != nil {
		logger.Info("PR has no linked issue, closing")
		if err := executeActions(
			ctx,
			h.issuesClient,
			h.prsClient,
			h.owner,
			h.repo,
			number,
			true,
			h.cfg.PullRequests.NoLinkedIssue.Actions,
			nil,
		); err != nil {
			return false, err
		}
		return true, nil
	}

	if issueNumber == 0 {
		return false, nil
	}

	logger = logger.WithValues("linkedIssue", issueNumber)
	ctx = logging.ContextWithLogger(ctx, logger)

	if h.cfg.PullRequests.BlockedIssue == nil {
		return false, nil
	}

	issue, _, err := h.issuesClient.Get(ctx, h.owner, h.repo, issueNumber)
	if err != nil {
		return false, fmt.Errorf("error fetching linked issue: %w", err)
	}

	issueLabels := make(map[string]bool)
	for _, l := range issue.Labels {
		issueLabels[l.GetName()] = true
	}

	var blockers []string
	for _, blocking := range h.cfg.PullRequests.BlockedIssue.BlockingLabels {
		if issueLabels[blocking] {
			blockers = append(blockers, blocking)
		}
	}

	if len(blockers) > 0 {
		logger.Info("linked issue has blocking labels, closing PR",
			"blockers", blockers,
		)
		if err := executeActions(
			ctx,
			h.issuesClient,
			h.prsClient,
			h.owner,
			h.repo,
			number,
			true,
			h.cfg.PullRequests.BlockedIssue.Actions,
			map[string]string{
				"IssueNumber":    fmt.Sprintf("%d", issueNumber),
				"BlockingLabels": h.formatBlockers(blockers),
			},
		); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
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

func (h *prHandler) formatBlockers(blockers []string) string {
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
func (h *prHandler) parseLinkedIssue(body string) int {
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
