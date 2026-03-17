package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	libExec "github.com/akuity/kargo/pkg/exec"
	"github.com/akuity/kargo/pkg/logging"
)

// signatureStatus represents the trust level of a commit's GPG signature.
type signatureStatus int

const (
	// signatureUnsigned indicates the commit has no GPG signature.
	signatureUnsigned signatureStatus = iota
	// signatureTrusted indicates the commit is signed by a trusted key.
	signatureTrusted
	// signatureUntrusted indicates the commit is signed but the key is
	// not trusted (or the signature is invalid).
	signatureUntrusted
)

// integrateBeforePush integrates remote changes before pushing.
func (w *workTree) integrateBeforePush(
	targetBranch string,
	committer *User,
	integrationPolicy PushIntegrationPolicy,
) error {
	if integrationPolicy == "" {
		integrationPolicy = PushIntegrationPolicyNone
	}
	if integrationPolicy == PushIntegrationPolicyNone {
		return nil
	}

	var homeDir string
	if committer != nil {
		// This committer is specific to any commits made during rebasing or
		// merging, so we will override repository-level user information by
		// creating a temporary home directory, configuring the user information
		// "globally" within it, and then ensuring git commits use that home
		// directory.
		var err error
		if homeDir, err = os.MkdirTemp(w.homeDir, ""); err != nil {
			return fmt.Errorf(
				"error creating virtual home directory %q for rebase/merge commands: %w",
				homeDir, err,
			)
		}
		defer func() {
			if cleanErr := os.RemoveAll(homeDir); cleanErr != nil {
				logging.LoggerFromContext(context.TODO()).
					Error(cleanErr, "error removing virtual home directory", "path", homeDir)
			}
		}()
		if err = w.setupUser(homeDir, committer); err != nil {
			return fmt.Errorf(
				"error setting up committer information for rebase/merge commands: %w", err,
			)
		}
	}

	logger := logging.LoggerFromContext(context.TODO()).WithValues(
		"remoteBranch", targetBranch,
		"integrationPolicy", integrationPolicy,
	)

	switch integrationPolicy {
	case PushIntegrationPolicyAlwaysRebase:
		logger.Trace("integrating remote changes via rebase (always)")
		return w.pullRebase(targetBranch, homeDir)
	case PushIntegrationPolicyAlwaysMerge:
		logger.Trace("integrating remote changes via merge (always)")
		return w.pullMerge(targetBranch, homeDir)
	case PushIntegrationPolicyRebaseOrMerge, PushIntegrationPolicyRebaseOrFail:
		safe, err := w.canSafelyRebase(targetBranch, homeDir)
		if err != nil {
			return fmt.Errorf("error checking rebase safety: %w", err)
		}
		if safe {
			logger.Trace("integrating remote changes via rebase")
			return w.pullRebase(targetBranch, homeDir)
		}
		if integrationPolicy == PushIntegrationPolicyRebaseOrFail {
			logger.Trace(
				"rebase is unsafe and policy prohibits merge fallback",
			)
			return ErrRebaseUnsafe
		}
		logger.Trace("integrating remote changes via merge (rebase unsafe)")
		return w.pullMerge(targetBranch, homeDir)
	default:
		return fmt.Errorf("unknown push integration policy: %q", integrationPolicy)
	}
}

// pullRebase performs a git pull --rebase against the specified remote branch.
// If homeDir is non-empty, it overrides the home directory used by the git
// command, allowing a custom committer's identity and signing key to be used
// for the replacement commits created by the rebase.
func (w *workTree) pullRebase(
	targetBranch string,
	homeDir string,
) error {
	cmd := w.buildGitCommand(
		"pull", "--rebase", "origin", targetBranch,
	)
	if homeDir != "" {
		// Override the home directory set by w.buildGitCommand().
		w.setCmdHome(cmd, homeDir)
	}
	if _, err := libExec.Exec(cmd); err != nil {
		if isRebasing, rbErr := w.IsRebasing(); rbErr == nil && isRebasing {
			return ErrMergeConflict
		}
		return fmt.Errorf("error pulling and rebasing branch: %w", err)
	}
	return nil
}

// pullMerge performs a git pull (merge) against the specified remote branch.
// If homeDir is non-empty, it overrides the home directory used by the git
// command, allowing a custom committer's identity and signing key to be used
// for the merge commit.
func (w *workTree) pullMerge(
	targetBranch string,
	homeDir string,
) error {
	cmd := w.buildGitCommand(
		"pull", "--no-rebase", "origin", targetBranch,
	)
	if homeDir != "" {
		// Override the home directory set by w.buildGitCommand().
		w.setCmdHome(cmd, homeDir)
	}
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
// signature trust. If homeDir is non-empty, it overrides the home directory
// used by git commands so that a custom committer's GPG trust database and
// signing configuration are consulted. The decision matrix:
//
//   - Signed with trusted key + signing configured: safe (Kargo can re-sign)
//   - Signed with trusted key + signing not configured: unsafe (would strip
//     signature)
//   - Signed with untrusted key: always unsafe (Kargo can't vouch for it)
//   - Unsigned + signing configured: unsafe (would fabricate a signature)
//   - Unsigned + signing not configured: safe (stays unsigned)
func (w *workTree) canSafelyRebase(
	targetBranch string,
	homeDir string,
) (bool, error) {
	commits, err := w.commitsToReplay(targetBranch)
	if err != nil {
		return false, fmt.Errorf(
			"error determining commits to replay: %w", err,
		)
	}
	if len(commits) == 0 {
		return true, nil
	}
	signing, err := w.isSigningConfigured(homeDir)
	if err != nil {
		return false, fmt.Errorf(
			"error checking signing configuration: %w", err,
		)
	}
	for _, commitID := range commits {
		status, err := w.verifyCommitSignature(commitID, homeDir)
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

// isSigningConfigured returns true if GPG commit signing is enabled in the
// git configuration for this repository. If homeDir is non-empty, it overrides
// the home directory used by the git command so that a custom committer's
// configuration is consulted.
func (w *workTree) isSigningConfigured(homeDir string) (bool, error) {
	cmd := w.buildGitCommand("config", "--get", "commit.gpgSign")
	if homeDir != "" {
		// Override the home directory set by w.buildGitCommand().
		w.setCmdHome(cmd, homeDir)
	}
	res, err := libExec.Exec(cmd)
	if err != nil {
		var exitErr *libExec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode == 1 {
			// Exit code 1 means the key was not found — signing is
			// not configured.
			return false, nil
		}
		return false, fmt.Errorf(
			"error reading commit.gpgSign config: %w", err,
		)
	}
	return strings.TrimSpace(string(res)) == "true", nil
}

// verifyCommitSignature checks the GPG signature status of the specified
// commit. It uses git's %G? format which returns:
//
//   - G: good (valid) signature from a trusted key
//   - U: good signature from an untrusted key
//   - B: bad signature
//   - X: good signature that has expired
//   - Y: good signature made by an expired key
//   - R: good signature made by a revoked key
//   - E: signature cannot be checked (missing key)
//   - N: no signature
//
// If homeDir is non-empty, it overrides the home directory used by the git
// command so that a custom committer's GPG trust database is consulted.
func (w *workTree) verifyCommitSignature(
	commitID string,
	homeDir string,
) (signatureStatus, error) {
	cmd := w.buildGitCommand(
		"log", "-1", "--format=%G?", commitID, "--",
	)
	if homeDir != "" {
		// Override the home directory set by w.buildGitCommand().
		w.setCmdHome(cmd, homeDir)
	}
	res, err := libExec.Exec(cmd)
	if err != nil {
		return signatureUnsigned, fmt.Errorf(
			"error checking signature of commit %s: %w",
			commitID, err,
		)
	}
	switch strings.TrimSpace(string(res)) {
	case "G":
		return signatureTrusted, nil
	case "N", "":
		return signatureUnsigned, nil
	default:
		// U, B, X, Y, R, E — all treated as untrusted.
		return signatureUntrusted, nil
	}
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
