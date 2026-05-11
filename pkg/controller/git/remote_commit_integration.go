package git

import (
	"context"
	"fmt"
	"strings"

	libExec "github.com/akuity/kargo/pkg/exec"
	"github.com/akuity/kargo/pkg/logging"
)

// IntegrationOptions represents options for integrating remote changes into
// the local branch before pushing.
type IntegrationOptions struct {
	// TargetBranch is the remote branch to integrate changes from. If empty, the
	// current branch is used.
	TargetBranch string
	// IntegrationPolicy controls how remote changes are integrated. If empty or
	// set to PushIntegrationPolicyNone, no integration is performed.
	IntegrationPolicy PushIntegrationPolicy
}

func (w *workTree) IntegrateRemoteChanges(opts *IntegrationOptions) error {
	if opts == nil {
		opts = &IntegrationOptions{}
	}
	integrationPolicy := opts.IntegrationPolicy
	if integrationPolicy == "" {
		integrationPolicy = PushIntegrationPolicyNone
	}
	if integrationPolicy == PushIntegrationPolicyNone {
		return nil
	}

	targetBranch := opts.TargetBranch
	if targetBranch == "" {
		var err error
		if targetBranch, err = w.CurrentBranch(); err != nil {
			return err
		}
	}
	exists, err := w.RemoteBranchExists(targetBranch)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	logger := logging.LoggerFromContext(context.TODO()).WithValues(
		"remoteBranch", targetBranch,
		"integrationPolicy", integrationPolicy,
	)

	switch integrationPolicy {
	case PushIntegrationPolicyAlwaysRebase:
		logger.Trace("integrating remote changes via rebase (always)")
		return w.pullRebase(targetBranch)
	case PushIntegrationPolicyAlwaysMerge:
		logger.Trace("integrating remote changes via merge (always)")
		return w.pullMerge(targetBranch)
	case PushIntegrationPolicyRebaseOrMerge, PushIntegrationPolicyRebaseOrFail:
		safe, err := w.canSafelyRebase(targetBranch)
		if err != nil {
			return fmt.Errorf("error checking rebase safety: %w", err)
		}
		if safe {
			logger.Trace("integrating remote changes via rebase")
			return w.pullRebase(targetBranch)
		}
		if integrationPolicy == PushIntegrationPolicyRebaseOrFail {
			logger.Trace(
				"rebase is unsafe and policy prohibits merge fallback",
			)
			return ErrRebaseUnsafe
		}
		logger.Trace("integrating remote changes via merge (rebase unsafe)")
		return w.pullMerge(targetBranch)
	default:
		return fmt.Errorf("unknown push integration policy: %q", integrationPolicy)
	}
}

// pullRebase performs a git pull --rebase against the specified remote branch.
func (w *workTree) pullRebase(targetBranch string) error {
	cmd := w.buildGitCommand("pull", "--rebase", "origin", targetBranch)
	if _, err := libExec.Exec(cmd); err != nil {
		if isRebasing, rbErr := w.IsRebasing(); rbErr == nil && isRebasing {
			return ErrMergeConflict
		}
		return fmt.Errorf("error pulling and rebasing branch: %w", err)
	}
	return nil
}

// pullMerge performs a git pull (merge) against the specified remote branch.
func (w *workTree) pullMerge(targetBranch string) error {
	cmd := w.buildGitCommand("pull", "--no-rebase", "origin", targetBranch)
	if _, err := libExec.Exec(cmd); err != nil {
		// Check for merge conflicts — MERGE_HEAD exists when a merge is
		// in progress and waiting for conflict resolution.
		if _, statErr := libExec.Exec(w.buildGitCommand(
			"rev-parse", "--verify", "MERGE_HEAD",
		)); statErr == nil {
			return ErrMergeConflict
		}
		return fmt.Errorf("error pulling and merging branch: %w", err)
	}
	return nil
}

// canSafelyRebase determines whether it is safe to rebase the local commits
// on top of the specified remote branch without misrepresenting commit
// signature trust. The decision matrix:
//
//   - Signed with trusted key + signing configured: safe (Kargo can re-sign)
//   - Signed with trusted key + signing not configured: unsafe (would strip
//     signature)
//   - Signed with untrusted key: always unsafe (Kargo can't vouch for it)
//   - Unsigned + signing configured: unsafe (would fabricate a signature)
//   - Unsigned + signing not configured: safe (stays unsigned)
func (w *workTree) canSafelyRebase(targetBranch string) (bool, error) {
	commits, err := w.commitsToReplay(targetBranch)
	if err != nil {
		return false, fmt.Errorf(
			"error determining commits to replay: %w", err,
		)
	}
	if len(commits) == 0 {
		return true, nil
	}
	signing, err := w.isSigningConfigured()
	if err != nil {
		return false, fmt.Errorf(
			"error checking signing configuration: %w", err,
		)
	}
	for _, commitID := range commits {
		status, err := w.verifyCommitSignature(commitID)
		if err != nil {
			return false, fmt.Errorf(
				"error verifying signature of commit %s: %w",
				commitID, err,
			)
		}
		if !isRebaseSafeForCommit(signing, status) {
			return false, nil
		}
	}
	return true, nil
}

// commitsToReplay returns the SHAs of commits that would be replayed during
// a rebase of the current branch on top of the specified remote branch.
func (w *workTree) commitsToReplay(
	targetBranch string,
) ([]string, error) {
	remoteRef := fmt.Sprintf("origin/%s", targetBranch)
	// Always fetch the target branch to ensure the remote tracking ref
	// exists and has full, up-to-date history for the git log range below.
	if err := w.Fetch(&FetchOptions{Branch: targetBranch}); err != nil {
		return nil, fmt.Errorf(
			"error fetching remote branch %q: %w", targetBranch, err,
		)
	}
	res, err := libExec.Exec(w.buildGitCommand(
		"log",
		"--format=%H",
		fmt.Sprintf("%s..HEAD", remoteRef),
	))
	if err != nil {
		return nil, fmt.Errorf(
			"error listing commits to replay: %w", err,
		)
	}
	output := strings.TrimSpace(string(res))
	if output == "" {
		return nil, nil
	}
	return strings.Split(output, "\n"), nil
}

// isRebaseSafeForCommit is a pure function implementing the rebase safety
// decision matrix for a single commit. It returns true if rebasing a commit
// with the given signature status is safe given whether signing is configured.
func isRebaseSafeForCommit(signing bool, status signatureStatus) bool {
	switch status {
	case signatureTrusted:
		if !signing {
			// Rebase would strip a valid signature.
			return false
		}
	case signatureUntrusted:
		// Kargo cannot vouch for this commit regardless of
		// signing configuration.
		return false
	case signatureUnsigned:
		if signing {
			// Rebase would fabricate Kargo's signature on a
			// commit it didn't author.
			return false
		}
	}
	return true
}
