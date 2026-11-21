package git

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	libExec "github.com/akuity/kargo/pkg/exec"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	// initialDeepenStep is the number of commits requested on the first
	// deepening fetch when a shallow clone lacks sufficient history to
	// integrate remote changes. Subsequent iterations double this value.
	initialDeepenStep = 16

	// maxDeepenIterations caps the number of exponential deepening fetches
	// attempted before giving up and failing closed.
	maxDeepenIterations = 6
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

func (w *workTree) IntegrateRemoteChanges(
	ctx context.Context,
	opts *IntegrationOptions,
) error {
	if opts == nil {
		opts = &IntegrationOptions{}
	}
	integrationPolicy := opts.IntegrationPolicy
	if integrationPolicy == "" {
		integrationPolicy = PushIntegrationPolicyNone
	}
	switch integrationPolicy {
	case PushIntegrationPolicyNone:
		return nil
	case PushIntegrationPolicyAlwaysRebase,
		PushIntegrationPolicyAlwaysMerge,
		PushIntegrationPolicyRebaseOrMerge,
		PushIntegrationPolicyRebaseOrFail:
	default:
		return fmt.Errorf("unknown push integration policy: %q", integrationPolicy)
	}

	targetBranch := opts.TargetBranch
	if targetBranch == "" {
		var err error
		if targetBranch, err = w.CurrentBranch(ctx); err != nil {
			return err
		}
	}
	exists, err := w.RemoteBranchExists(ctx, targetBranch)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	logger := logging.LoggerFromContext(ctx).WithValues(
		"remoteBranch", targetBranch,
		"integrationPolicy", integrationPolicy,
	)

	complete, err := w.fetchForIntegration(ctx, targetBranch)
	if err != nil {
		return fmt.Errorf(
			"error fetching remote branch %q for integration: %w",
			targetBranch, err,
		)
	}
	if !complete {
		// Without the history that local and remote changes have in common, a
		// rebase could fold commits Kargo cannot see into one, and a merge could
		// fail or use the wrong merge base. This is true under every policy.
		logger.Trace("shallow clone history is insufficient for integration")
		return ErrShallowHistoryIncomplete
	}

	remoteRef := remoteBranchRefPrefix + targetBranch

	switch integrationPolicy {
	case PushIntegrationPolicyAlwaysRebase:
		logger.Trace("integrating remote changes via rebase (always)")
		return w.rebase(ctx, remoteRef)
	case PushIntegrationPolicyAlwaysMerge:
		logger.Trace("integrating remote changes via merge (always)")
		return w.merge(ctx, targetBranch)
	}

	// If we get to here, the policy is RebaseOrMerge or RebaseOrFail
	safe, err := w.canSafelyRebase(ctx, remoteRef)
	if err != nil {
		return fmt.Errorf("error checking rebase safety: %w", err)
	}
	if safe {
		logger.Trace("integrating remote changes via rebase")
		return w.rebase(ctx, remoteRef)
	}
	if integrationPolicy == PushIntegrationPolicyRebaseOrFail {
		logger.Trace("rebase is unsafe and policy prohibits merge fallback")
		return ErrRebaseUnsafe
	}
	logger.Trace("integrating remote changes via merge (rebase unsafe)")
	return w.merge(ctx, targetBranch)
}

// fetchForIntegration fetches targetBranch into refs/remotes/origin/<targetBranch>
//
// For shallow repositories, incrementally deepens history until no shallow
// boundary lies within the symmetric difference HEAD...origin/<targetBranch>.
//
// Returns true if history is complete for integration.
func (w *workTree) fetchForIntegration(
	ctx context.Context,
	targetBranch string,
) (bool, error) {
	shallow, err := w.isShallow(ctx)
	if err != nil {
		return false, err
	}
	if !shallow {
		if err = w.Fetch(ctx, &FetchOptions{Branch: targetBranch}); err != nil {
			return false, err
		}
		return true, nil
	}

	// Fetching without --depth lets Git reuse existing shallow clone.
	fetchOpts := &FetchOptions{Branch: targetBranch}
	hasLocalRef, err := w.hasAnyRef(
		ctx,
		branchRefPrefix+targetBranch,
		remoteBranchRefPrefix+targetBranch,
	)
	if err != nil {
		return false, err
	}
	if !hasLocalRef {
		fetchOpts.Depth = 1
	}
	if err = w.Fetch(ctx, fetchOpts); err != nil {
		return false, err
	}

	return w.ensureCompleteIntegrationHistory(ctx, targetBranch)
}

// ensureCompleteIntegrationHistory incrementally deepens a shallow repository
// until no shallow boundary commit appears in HEAD...origin/<targetBranch>.
//
// This ensures every ancestor hidden behind a shallow boundary is common to both
// HEAD and origin/<targetBranch>, so both commitsToReplay and merge bases are
// exact.
//
// Returns false if deepening stops making progress or hits the bound.
func (w *workTree) ensureCompleteIntegrationHistory(
	ctx context.Context,
	targetBranch string,
) (bool, error) {
	remoteRef := remoteBranchRefPrefix + targetBranch

	shallow, err := w.shallowCommits(ctx)
	if err != nil {
		return false, err
	}
	if len(shallow) == 0 {
		return true, nil
	}
	diff, err := w.symmetricDiff(ctx, remoteRef)
	if err != nil {
		return false, err
	}
	if !setsOverlap(diff, shallow) {
		return true, nil
	}

	deepenRefs := []string{branchRefPrefix + targetBranch}
	currentBranch, err := w.CurrentBranch(ctx)
	if err != nil {
		return false, err
	}
	if currentBranch != "" && currentBranch != targetBranch {
		var exists bool
		exists, err = w.RemoteBranchExists(ctx, currentBranch)
		if err != nil {
			return false, err
		}
		if exists {
			deepenRefs = append(deepenRefs, branchRefPrefix+currentBranch)
		}
	}

	step := uint(initialDeepenStep)
	for range maxDeepenIterations {
		if err = w.deepen(ctx, step, deepenRefs...); err != nil {
			return false, err
		}

		var newShallow map[string]struct{}
		newShallow, err = w.shallowCommits(ctx)
		if err != nil {
			return false, err
		}
		if len(newShallow) == 0 {
			return true, nil
		}
		diff, err = w.symmetricDiff(ctx, remoteRef)
		if err != nil {
			return false, err
		}
		if !setsOverlap(diff, newShallow) {
			return true, nil
		}
		if maps.Equal(newShallow, shallow) {
			// Deepening made no progress (e.g. the shallow commit is no longer
			// reachable from the remote refs after a force push).
			return false, nil
		}
		shallow = newShallow
		step *= 2
	}

	return false, nil
}

// rebase rebases the current branch onto the specified ref with --fork-point.
func (w *workTree) rebase(ctx context.Context, ref string) error {
	cmd := w.buildGitCommand(ctx, "rebase", "--fork-point", ref)
	if _, err := libExec.Exec(cmd); err != nil {
		if isRebasing, rbErr := w.IsRebasing(ctx); rbErr == nil && isRebasing {
			return ErrMergeConflict
		}
		return fmt.Errorf("error rebasing branch onto %q: %w", ref, err)
	}
	return nil
}

// merge merges the specified remote branch, which must already be fetched,
// into the current branch. The merge commit gets the message that git pull
// would give it.
func (w *workTree) merge(ctx context.Context, targetBranch string) error {
	ref := remoteBranchRefPrefix + targetBranch
	args := []string{"merge", "--no-edit"}
	msg, err := w.pullMergeMessage(ctx, targetBranch)
	if err != nil {
		return err
	}
	if msg != "" {
		args = append(args, "-m", msg)
	}
	args = append(args, ref)
	if _, err = libExec.Exec(w.buildGitCommand(ctx, args...)); err != nil {
		// Check for merge conflicts — MERGE_HEAD exists when a merge is
		// in progress and waiting for conflict resolution.
		if _, statErr := libExec.Exec(w.buildGitCommand(
			ctx, "rev-parse", "--verify", "MERGE_HEAD",
		)); statErr == nil {
			return ErrMergeConflict
		}
		return fmt.Errorf("error merging %q into branch: %w", ref, err)
	}
	return nil
}

// pullMergeMessage returns the message that git pull would give a merge of the
// specified remote branch, using the description of that branch that git
// recorded in FETCH_HEAD when it was last fetched. It returns an empty string
// if FETCH_HEAD records no fetch of that branch.
//
// Equivalent to:
//
//	grep "branch '<targetBranch>' of " $(git rev-parse --git-path FETCH_HEAD) |
//	  git fmt-merge-msg
func (w *workTree) pullMergeMessage(
	ctx context.Context,
	targetBranch string,
) (string, error) {
	res, err := libExec.Exec(w.buildGitCommand(
		ctx, "rev-parse", "--git-path", "FETCH_HEAD",
	))
	if err != nil {
		return "", fmt.Errorf("error locating FETCH_HEAD: %w", err)
	}
	fetchHeadPath := strings.TrimSpace(string(res))
	if !filepath.IsAbs(fetchHeadPath) {
		fetchHeadPath = filepath.Join(w.dir, fetchHeadPath)
	}
	// #nosec G304 -- Path is resolved by Git from the repository's own gitdir.
	data, err := os.ReadFile(filepath.Clean(fetchHeadPath))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("error reading FETCH_HEAD: %w", err)
	}
	description := fmt.Sprintf("branch '%s' of ", targetBranch)
	for line := range strings.Lines(string(data)) {
		// Each line is "<commit ID>\t[not-for-merge]\t<description>"
		fields := strings.SplitN(strings.TrimSuffix(line, "\n"), "\t", 3)
		if len(fields) != 3 || !strings.HasPrefix(fields[2], description) {
			continue
		}
		cmd := w.buildGitCommand(ctx, "fmt-merge-msg")
		cmd.Stdin = strings.NewReader(fields[0] + "\t\t" + fields[2] + "\n")
		if res, err = libExec.Exec(cmd); err != nil {
			return "", fmt.Errorf("error formatting merge message: %w", err)
		}
		return strings.TrimSpace(string(res)), nil
	}
	return "", nil
}

// canSafelyRebase determines whether it is safe to rebase the local commits
// on top of the specified already-fetched remote ref without misrepresenting
// commit signature trust. The decision matrix:
//
//   - Signed with trusted key + signing configured: safe (Kargo can re-sign)
//   - Signed with trusted key + signing not configured: unsafe (would strip
//     signature)
//   - Signed with untrusted key: always unsafe (Kargo can't vouch for it)
//   - Unsigned + signing configured: unsafe (would fabricate a signature)
//   - Unsigned + signing not configured: safe (stays unsigned)
func (w *workTree) canSafelyRebase(
	ctx context.Context,
	remoteRef string,
) (bool, error) {
	commits, err := w.commitsToReplay(ctx, remoteRef)
	if err != nil {
		return false, fmt.Errorf(
			"error determining commits to replay: %w", err,
		)
	}
	if len(commits) == 0 {
		return true, nil
	}
	shallow, err := w.shallowCommits(ctx)
	if err != nil {
		return false, fmt.Errorf(
			"error reading shallow boundaries: %w", err,
		)
	}

	if len(shallow) > 0 {
		var diff map[string]struct{}
		diff, err = w.symmetricDiff(ctx, remoteRef)
		if err != nil {
			return false, fmt.Errorf(
				"error checking shallow history completeness: %w", err,
			)
		}

		// never approve a rebase if a shallow boundary lies within the symmetric
		// difference between HEAD and remoteRef, since commitsToReplay could omit
		// hidden ancestors or misclassify shared ones.
		if setsOverlap(diff, shallow) {
			return false, nil
		}
	}

	signing, err := w.isSigningConfigured(ctx)
	if err != nil {
		return false, fmt.Errorf(
			"error checking signing configuration: %w", err,
		)
	}
	for _, commitID := range commits {
		status, err := w.verifyCommitSignature(ctx, commitID)
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

// forkPoint returns the fork-point commit between remoteRef's reflog and HEAD,
// or an empty string if no fork point can be determined.
//
// Equivalent to:
//
//	git merge-base --fork-point <remoteRef> HEAD
func (w *workTree) forkPoint(
	ctx context.Context,
	remoteRef string,
) (string, error) {
	res, err := libExec.Exec(w.buildGitCommand(
		ctx, "merge-base", "--fork-point", remoteRef, "HEAD",
	))
	if err != nil {
		var exitErr *libExec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode == 1 {
			return "", nil
		}
		return "", fmt.Errorf(
			"error finding fork point for %q: %w", remoteRef, err,
		)
	}
	return strings.TrimSpace(string(res)), nil
}

// commitsToReplay returns the SHAs of commits that would be replayed during
// a rebase of the current branch on top of the specified already-fetched
// remote ref, with fork-point detection.
func (w *workTree) commitsToReplay(
	ctx context.Context,
	remoteRef string,
) ([]string, error) {
	fp, err := w.forkPoint(ctx, remoteRef)
	if err != nil {
		return nil, err
	}
	args := []string{"log", "--format=%H", "HEAD", "^" + remoteRef}
	if fp != "" {
		args = append(args, "^"+fp)
	}
	res, err := libExec.Exec(w.buildGitCommand(ctx, args...))
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

// isShallow returns true if the repository is a shallow clone.
//
// Equivalent to:
//
//	git rev-parse --is-shallow-repository
func (w *workTree) isShallow(ctx context.Context) (bool, error) {
	res, err := libExec.Exec(w.buildGitCommand(
		ctx, "rev-parse", "--is-shallow-repository",
	))
	if err != nil {
		return false, fmt.Errorf(
			"error checking if repository is shallow: %w", err,
		)
	}
	return strings.TrimSpace(string(res)) == "true", nil
}

// shallowCommits returns the set of commit SHAs recorded in the repository's
// shallow file, or nil if the repository is not shallow.
//
// Equivalent to:
//
//	git rev-parse --git-path shallow
func (w *workTree) shallowCommits(
	ctx context.Context,
) (map[string]struct{}, error) {
	res, err := libExec.Exec(w.buildGitCommand(
		ctx, "rev-parse", "--git-path", "shallow",
	))
	if err != nil {
		return nil, fmt.Errorf("error locating shallow file: %w", err)
	}
	shallowPath := strings.TrimSpace(string(res))
	if !filepath.IsAbs(shallowPath) {
		shallowPath = filepath.Join(w.dir, shallowPath)
	}
	// #nosec G304 -- Path is resolved by Git from the repository's own gitdir.
	data, err := os.ReadFile(filepath.Clean(shallowPath))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"error reading shallow file %q: %w", shallowPath, err,
		)
	}
	commits := make(map[string]struct{})
	for line := range strings.SplitSeq(string(data), "\n") {
		sha := strings.TrimSpace(line)
		if sha != "" {
			commits[sha] = struct{}{}
		}
	}
	return commits, nil
}

// symmetricDiff returns the set of commit SHAs in the symmetric difference
// between HEAD and remoteRef.
//
// Equivalent to:
//
//	git rev-list HEAD...<remoteRef>
func (w *workTree) symmetricDiff(
	ctx context.Context,
	remoteRef string,
) (map[string]struct{}, error) {
	res, err := libExec.Exec(w.buildGitCommand(
		ctx, "rev-list", fmt.Sprintf("HEAD...%s", remoteRef),
	))
	if err != nil {
		return nil, fmt.Errorf(
			"error listing symmetric difference HEAD...%s: %w", remoteRef, err,
		)
	}
	commits := make(map[string]struct{})
	for line := range strings.SplitSeq(string(res), "\n") {
		if sha := strings.TrimSpace(line); sha != "" {
			commits[sha] = struct{}{}
		}
	}
	return commits, nil
}

// setsOverlap returns true if a and b have at least one key in common.
func setsOverlap[K comparable](a, b map[K]struct{}) bool {
	if len(a) > len(b) {
		a, b = b, a
	}
	for k := range a {
		if _, ok := b[k]; ok {
			return true
		}
	}
	return false
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
