package governance

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/google/go-github/v76/github"

	"github.com/akuity/kargo/pkg/logging"
)

// pathExemptionPerPage is the per-page size requested when fetching changed
// files for the path-based exemption check. PRs whose file count meets or
// exceeds this threshold are treated as not path-exempt without bothering
// to paginate — they're not drive-bys by definition.
const pathExemptionPerPage = 100

// prHandler handles pull request-related events for a specific repository
// according to specific configuration.
type prHandler struct {
	cfg          config
	owner        string
	repo         string
	issuesClient IssuesClient
	prsClient    PullRequestsClient
}

type handlePROpenedOpts struct {
	applyPolicyOnly bool
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
	opts *handlePROpenedOpts,
) error {
	if opts == nil {
		opts = &handlePROpenedOpts{}
	}

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

	issueNumber := parseLinkedIssue(pr.GetBody())

	// Steps run independently: each one's failure is logged and collected,
	// but does not prevent subsequent steps from running. The aggregated
	// error is returned at the end so the webhook delivery shows red in
	// GitHub's UI (GitHub does not auto-retry).
	var errs []error

	if !opts.applyPolicyOnly {

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

	}

	// 4. Apply PR policy unless the PR is exempt. On exemption-check error,
	// log + accumulate but treat as not exempt (safer default — apply
	// policy).
	exempt, err := h.isExemptFromPRPolicy(ctx, pr, login)
	if err != nil {
		logger.Error(err, "error checking PR policy exemption")
		errs = append(errs, fmt.Errorf("check PR policy exemption: %w", err))
	}
	switch {
	case exempt:
		logger.Debug("PR is exempt from policy, skipping")
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

// applyPRPolicy evaluates PR policy and runs the actions configured for the
// matching outcome:
//   - The PR has no linked issue and OnNoLinkedIssue is configured; or
//   - The PR's linked issue carries one or more labels listed in
//     OnBlockedIssue.BlockingLabels and OnBlockedIssue is configured; or
//   - The PR passes (linked issue is not blocked, or the relevant check is
//     not configured) and OnPass is configured.
//
// Callers should have already verified the PR's author is not exempt from
// policy.
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

	// Blocking outcome: PR has no linked issue.
	if issueNumber == 0 {
		if cfg.PullRequests.OnNoLinkedIssue != nil {
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
				cfg.PullRequests.OnNoLinkedIssue.Actions,
				nil,
			)
		}
		// OnNoLinkedIssue not configured: treat as a pass. Fall through.
	} else if cfg.PullRequests.OnBlockedIssue != nil {
		// Has a linked issue and OnBlockedIssue is configured: check for
		// blocking labels.
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
		for _, blocking := range cfg.PullRequests.OnBlockedIssue.BlockingLabels {
			if issueLabels[blocking] {
				blockers = append(blockers, blocking)
			}
		}

		// Blocking outcome: linked issue has blocking labels.
		if len(blockers) > 0 {
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
				cfg.PullRequests.OnBlockedIssue.Actions,
				map[string]string{
					"IssueNumber":    fmt.Sprintf("%d", issueNumber),
					"BlockingLabels": formatBlockers(blockers),
				},
			)
		}
		// No blocking labels: fall through to OnPass.
	}

	// Passing outcome: no blocking action fired. Run OnPass if configured.
	if cfg.PullRequests.OnPass == nil {
		return nil
	}
	logger.Info("PR passes policy, applying OnPass actions")
	return executeActions(
		ctx,
		cfg,
		issuesClient,
		prsClient,
		owner,
		repo,
		number,
		true,
		cfg.PullRequests.OnPass.Actions,
		nil,
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

// isExemptFromPRPolicy reports whether the PR matches any of the configured
// exemption criteria (maintainer, bot, size, path). Criteria are OR'd.
// Cheaper checks run first; the path check makes a network call (ListFiles)
// and is only reached if the cheaper checks didn't already exempt the PR.
//
// On error from the path check, returns (false, err) — the caller is
// expected to treat that as "not exempt" and apply policy.
func (h *prHandler) isExemptFromPRPolicy(
	ctx context.Context,
	pr *github.PullRequest,
	login string,
) (bool, error) {
	if h.cfg.PullRequests == nil || h.cfg.PullRequests.Exemptions == nil {
		return false, nil
	}
	ex := h.cfg.PullRequests.Exemptions

	if ex.Maintainers && isMaintainer(h.cfg, pr.GetAuthorAssociation()) {
		return true, nil
	}
	if ex.Bots && strings.HasSuffix(login, "[bot]") {
		return true, nil
	}
	if ex.MaxChangedLines > 0 &&
		pr.GetAdditions()+pr.GetDeletions() <= ex.MaxChangedLines {
		return true, nil
	}
	if len(ex.PathPatterns) > 0 {
		exempt, err := h.allFilesMatchPathPatterns(
			ctx, pr.GetNumber(), ex.PathPatterns,
		)
		if err != nil {
			return false, err
		}
		if exempt {
			return true, nil
		}
	}
	return false, nil
}

// allFilesMatchPathPatterns returns true if every file changed by the PR
// matches at least one of the supplied gitignore-style patterns. Returns
// false (without error) when the PR's file count meets or exceeds the
// per-page limit — at that scale the PR is not a drive-by and we don't
// bother paginating.
func (h *prHandler) allFilesMatchPathPatterns(
	ctx context.Context,
	prNumber int,
	patterns []string,
) (bool, error) {
	files, _, err := h.prsClient.ListFiles(
		ctx, h.owner, h.repo, prNumber,
		&github.ListOptions{PerPage: pathExemptionPerPage},
	)
	if err != nil {
		return false, fmt.Errorf("error listing PR files: %w", err)
	}
	if len(files) == 0 || len(files) >= pathExemptionPerPage {
		return false, nil
	}

	parsed := make([]gitignore.Pattern, 0, len(patterns))
	for _, p := range patterns {
		parsed = append(parsed, gitignore.ParsePattern(p, nil))
	}
	matcher := gitignore.NewMatcher(parsed)

	for _, f := range files {
		if !matcher.Match(strings.Split(f.GetFilename(), "/"), false) {
			return false, nil
		}
	}
	return true, nil
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
