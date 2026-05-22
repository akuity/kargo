package governance

import (
	"context"
	"fmt"
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

// repoContext bundles the per-repository dependencies shared across the
// webhook, slash-command, and action-execution paths: the loaded
// governance config, the repo coordinates (owner/repo), and the GitHub
// API clients. Handler types embed it; actionContext composes it for
// action runners.
//
// Methods on repoContext are the verbs the repo can perform against
// inputs (users, PRs, issue numbers, etc.). Inputs are passed as method
// parameters; they are not part of the repo's identity.
type repoContext struct {
	cfg          config
	owner        string
	repo         string
	issuesClient IssuesClient
	prsClient    PullRequestsClient
	orgsClient   OrganizationsClient
}

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
func (rc *repoContext) isMaintainer(
	ctx context.Context,
	authorAssoc string,
	login string,
) (bool, error) {
	wantMember := false
	for _, assoc := range rc.cfg.MaintainerAssociations {
		if strings.EqualFold(authorAssoc, assoc) {
			return true, nil
		}
		if strings.EqualFold(assoc, "MEMBER") {
			wantMember = true
		}
	}
	if !wantMember || login == "" || rc.orgsClient == nil {
		return false, nil
	}
	isMember, _, err := rc.orgsClient.IsMember(ctx, rc.owner, login)
	if err != nil {
		return false, fmt.Errorf(
			"error checking org membership of %q: %w", login, err,
		)
	}
	return isMember, nil
}

// enforceRequiredLabels adds a `needs/<prefix>` label for any required
// prefix not already present in existingLabels. Used by both the issue and
// PR opened-event paths.
func (rc *repoContext) enforceRequiredLabels(
	ctx context.Context,
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
		if _, _, err := rc.issuesClient.AddLabelsToIssue(
			ctx,
			rc.owner,
			rc.repo,
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

// applyPRPolicy evaluates PR policy and runs the actions configured for the
// matching outcome:
//   - Blocking outcomes (OnNoLinkedIssue, OnBlockedIssue) fire only when
//     the PR is NOT exempt from policy.
//   - OnPass fires whenever neither blocking outcome fired — including for
//     exempt PRs. This keeps cleanup-style OnPass actions (e.g. removing
//     stale policy/* labels) working for PRs that became exempt after
//     they'd already been drafted.
//
// PullRequests must be non-nil on the receiver's config. The exemption
// check makes network calls (ListFiles for path patterns, IsMember for
// concealed-member detection) when those checks are reached; errors there
// propagate to the caller.
//
// senderLogin is the event sender's login — for opened events that's the
// PR author, for reopened / ready_for_review events that may be a different
// user. A maintainer sender is treated as exempt even when the PR author
// isn't, giving maintainers a way to clear stuck PRs by re-triggering
// policy evaluation.
func (rc *repoContext) applyPRPolicy(
	ctx context.Context,
	pr *github.PullRequest,
	senderLogin string,
) error {
	if rc.cfg.PullRequests == nil {
		return nil
	}
	logger := logging.LoggerFromContext(ctx)
	number := pr.GetNumber()
	issueNumber := parseLinkedIssue(pr.GetBody())

	exempt, err := rc.isPRExempt(ctx, pr, senderLogin)
	if err != nil {
		return fmt.Errorf("error checking PR policy exemption: %w", err)
	}

	// Blocking outcomes — gated by exemption.
	if !exempt {
		if issueNumber == 0 {
			if rc.cfg.PullRequests.OnNoLinkedIssue != nil {
				logger.Info("PR has no linked issue, applying policy")
				return executeActions(
					ctx,
					&actionContext{
						repoContext: *rc,
						number:      number,
						isPR:        true,
					},
					rc.cfg.PullRequests.OnNoLinkedIssue.Actions,
				)
			}
			// OnNoLinkedIssue not configured: fall through to OnPass.
		} else if rc.cfg.PullRequests.OnBlockedIssue != nil {
			// Has a linked issue and OnBlockedIssue is configured: check
			// for blocking labels.
			logger = logger.WithValues("linkedIssue", issueNumber)
			ctx = logging.ContextWithLogger(ctx, logger)

			issue, _, err := rc.issuesClient.Get(ctx, rc.owner, rc.repo, issueNumber)
			if err != nil {
				return fmt.Errorf("error fetching linked issue: %w", err)
			}

			issueLabels := make(map[string]bool)
			for _, l := range issue.Labels {
				issueLabels[l.GetName()] = true
			}

			var blockers []string
			for _, blocking := range rc.cfg.PullRequests.OnBlockedIssue.BlockingLabels {
				if issueLabels[blocking] {
					blockers = append(blockers, blocking)
				}
			}

			if len(blockers) > 0 {
				logger.Info("linked issue has blocking labels, applying policy",
					"blockers", blockers,
				)
				return executeActions(
					ctx,
					&actionContext{
						repoContext: *rc,
						number:      number,
						isPR:        true,
						templateData: map[string]string{
							"IssueNumber":    fmt.Sprintf("%d", issueNumber),
							"BlockingLabels": formatBlockers(blockers),
						},
					},
					rc.cfg.PullRequests.OnBlockedIssue.Actions,
				)
			}
			// No blocking labels: fall through to OnPass.
		}
	}

	// Passing outcome (or exempt): run OnPass if configured.
	if rc.cfg.PullRequests.OnPass == nil {
		return nil
	}
	logger.Info("PR passes policy, applying OnPass actions", "exempt", exempt)
	return executeActions(
		ctx,
		&actionContext{
			repoContext: *rc,
			number:      number,
			isPR:        true,
		},
		rc.cfg.PullRequests.OnPass.Actions,
	)
}

// isPRExempt reports whether the PR matches any of the configured
// exemption criteria (maintainer, bot, size, path). Criteria are OR'd.
// Cheaper checks run first; the path and concealed-member checks make
// network calls and are only reached when the cheaper checks didn't
// already exempt the PR.
//
// For the maintainer criterion, both the PR author and the event sender are
// considered. If either is a maintainer, the PR is exempt. The sender
// distinction matters on reopened / ready_for_review events: a maintainer
// can re-trigger policy on someone else's stuck PR by marking it ready, and
// that signal is treated as an implicit endorsement.
//
// On error from any of the network checks, returns (false, err) — the
// caller is expected to treat that as "not exempt" and apply policy.
func (rc *repoContext) isPRExempt(
	ctx context.Context,
	pr *github.PullRequest,
	senderLogin string,
) (bool, error) {
	if rc.cfg.PullRequests == nil || rc.cfg.PullRequests.Exemptions == nil {
		return false, nil
	}
	ex := rc.cfg.PullRequests.Exemptions

	if ex.Maintainers {
		// Author check: payload association (cheap), with org-membership
		// fallback to catch concealed members.
		authorLogin := pr.GetUser().GetLogin()
		authorExempt, err := rc.isMaintainer(
			ctx, pr.GetAuthorAssociation(), authorLogin,
		)
		if err != nil {
			return false, err
		}
		if authorExempt {
			return true, nil
		}
		// Sender check: only meaningful when the sender differs from the
		// author (i.e. reopened / ready_for_review by someone else). No
		// payload association is available for the sender, so this relies
		// on the org-membership fallback.
		if senderLogin != "" && senderLogin != authorLogin {
			senderExempt, err := rc.isMaintainer(ctx, "", senderLogin)
			if err != nil {
				return false, err
			}
			if senderExempt {
				return true, nil
			}
		}
	}
	if ex.Bots && strings.HasSuffix(senderLogin, "[bot]") {
		return true, nil
	}
	// MaxChangedLines is uint; pr additions/deletions are non-negative ints
	// from go-github. The cast is safe (config-bound value).
	if ex.MaxChangedLines > 0 &&
		pr.GetAdditions()+pr.GetDeletions() <= int(ex.MaxChangedLines) { //nolint:gosec
		return true, nil
	}
	if len(ex.PathPatterns) > 0 {
		exempt, err := rc.allFilesMatchPathPatterns(
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
func (rc *repoContext) allFilesMatchPathPatterns(
	ctx context.Context,
	prNumber int,
	patterns []string,
) (bool, error) {
	files, _, err := rc.prsClient.ListFiles(
		ctx, rc.owner, rc.repo, prNumber,
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
