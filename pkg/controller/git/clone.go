package git

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	libExec "github.com/akuity/kargo/pkg/exec"
)

const (
	// headRef is the symbolic ref that points at a remote's default branch.
	headRef = "HEAD"
	// branchRefPrefix is the prefix of every fully qualified branch ref.
	branchRefPrefix = "refs/heads/"
	// tagRefPrefix is the prefix of every fully qualified tag ref.
	tagRefPrefix = "refs/tags/"
	// remoteBranchRefPrefix is the prefix of every remote-tracking branch ref
	// for the "origin" remote.
	remoteBranchRefPrefix = "refs/remotes/origin/"
	// symrefPrefix prefixes the target of a symbolic ref in the output of
	// git ls-remote --symref.
	symrefPrefix = "ref: "
	// peeledSuffix suffixes the name of an annotated tag in the output of git
	// ls-remote on the line that gives the commit the tag points to.
	peeledSuffix = "^{}"
)

// commitIDRegex matches a full SHA-1 or SHA-256 commit ID.
var commitIDRegex = regexp.MustCompile(`^([0-9a-fA-F]{40}|[0-9a-fA-F]{64})$`)

// BaseCloneOptions represents options common to all clones of a Git
// repository.
type BaseCloneOptions struct {
	// BaseDir is an existing directory within which all other directories
	// created and managed by the repository implementation will be created. If
	// not specified, the operating system's temporary directory will be used.
	// Overriding that default is useful under certain circumstances.
	BaseDir string
	// Refs optionally limits the clone to the specified refs. Each must be HEAD
	// (the remote's default branch), a fully qualified branch or tag ref, e.g.
	// "refs/heads/main" or "refs/tags/v1.0.0", or a full commit ID. Branches and
	// tags that do not exist in the remote are skipped. If empty, all branches
	// and tags are cloned.
	//
	// A commit is cloned only through a branch or tag that reaches it, never by
	// its ID, since some servers serve commits by ID that no branch or tag of
	// the repository reaches, such as commits that exist only in a fork. A
	// branch or tag whose tip is the commit is cloned in its place. Failing
	// that, every branch and tag is fetched, as deep as it takes to reach the
	// commit. Cloning a commit that no branch or tag reaches fails.
	Refs []string
	// Depth optionally limits the clone to the specified number of commits from
	// the tip of each cloned ref. A value of 0 (the default) indicates no limit.
	Depth uint
	// Blobless enables blobless cloning (--filter=blob:none). When set, the
	// initial clone downloads all commits and trees but defers blob downloads
	// until checkout. Combine with sparse checkout to minimize disk usage on
	// large repositories. The server must support partial clones; if it does
	// not, the clone will fail.
	Blobless bool
}

// clonePlan describes how a clone is carried out.
type clonePlan struct {
	// primary is what git clone itself fetches: a fully qualified branch or tag
	// ref. If empty, git clone fetches the remote's default branch, or
	// everything if no refs were requested.
	primary string
	// fetch holds the fully qualified refs fetched after cloning.
	fetch []string
	// commits holds the IDs of the requested commits, each of which must be
	// present once cloning is done.
	commits []string
	// detachAt is the commit, or for a regular clone the tag, that HEAD is
	// detached at after cloning, if any. A regular clone checks it out.
	detachAt string
}

// clone clones the remote repository into b.dir. For a regular (non-bare)
// clone, checkout is what to check out, in any of the forms that opts.Refs
// accepts, where empty means the remote's default branch. For a bare clone, it
// must be empty.
func (b *baseRepo) clone(
	ctx context.Context,
	opts BaseCloneOptions,
	checkout string,
	bare bool,
) error {
	plan, err := b.planClone(ctx, opts, checkout, bare)
	if err != nil {
		return err
	}
	args := []string{"clone"}
	if bare {
		args = append(args, "--bare")
	}
	if opts.Blobless {
		args = append(args, "--filter", "blob:none")
	}
	if opts.Depth > 0 {
		args = append(args, "--depth", fmt.Sprint(opts.Depth))
	}
	if len(opts.Refs) == 0 {
		if opts.Depth > 0 {
			// --depth implies --single-branch, but all branches were requested
			args = append(args, "--no-single-branch")
		}
	} else {
		// Only the requested refs are cloned, so tags that happen to point into
		// their history are not.
		args = append(args, "--single-branch", "--no-tags")
	}
	if plan.detachAt != "" && !bare {
		args = append(args, "--no-checkout")
	}
	switch {
	case plan.primary == "":
	case isBranchRef(plan.primary):
		args = append(
			args,
			"--branch", strings.TrimPrefix(plan.primary, branchRefPrefix),
		)
	default:
		// Unlike --branch, --revision takes a fully qualified ref, so a tag cannot
		// be mistaken for a branch of the same name.
		args = append(args, "--revision", plan.primary)
	}
	args = append(args, b.url, b.dir)
	cmd := b.buildGitCommand(ctx, args...)
	cmd.Dir = b.homeDir // Override the cmd.Dir that's set by b.buildGitCommand()
	if _, err = libExec.Exec(cmd); err != nil {
		return fmt.Errorf("error cloning repo %q into %q: %w", b.url, b.dir, err)
	}
	if isBranchRef(plan.primary) {
		if err = b.requireHEAD(ctx, plan.primary); err != nil {
			return err
		}
	}
	if err = b.fetchAdditionalRefs(ctx, opts.Depth, bare, plan.fetch); err != nil {
		return err
	}
	if !bare {
		if err = b.trackBranches(ctx, plan.fetch); err != nil {
			return err
		}
	}
	if err = b.requireCommits(ctx, opts.Depth, bare, plan.commits); err != nil {
		return err
	}
	if plan.detachAt == "" {
		return nil
	}
	if bare {
		if _, err = libExec.Exec(b.buildGitCommand(
			ctx,
			"update-ref", "--no-deref", headRef, plan.detachAt,
		)); err != nil {
			return fmt.Errorf("error detaching HEAD at %q: %w", plan.detachAt, err)
		}
		return nil
	}
	if _, err = libExec.Exec(b.buildGitCommand(
		ctx,
		"checkout", "--detach", plan.detachAt,
		// The next argument makes it clear to git that plan.detachAt is a ref
		// and not a path
		"--",
	)); err != nil {
		return fmt.Errorf("error checking out %q: %w", plan.detachAt, err)
	}
	return nil
}

// planClone works out how to clone what opts and checkout ask for. A regular
// clone that is limited to specific refs clones the branch or tag it checks
// out, then fetches the rest. A bare clone that is limited to specific refs
// clones the first of them that exists in the remote, or the remote's default
// branch if none exist, then fetches the rest. A regular clone that checks out
// a commit is planned like a bare clone whose first ref is that commit. A
// commit is cloned through a branch or tag whose tip it is. If there is none,
// requireCommits looks for it after cloning.
func (b *baseRepo) planClone(
	ctx context.Context,
	opts BaseCloneOptions,
	checkout string,
	bare bool,
) (clonePlan, error) {
	if err := validateCloneRefs(opts.Refs); err != nil {
		return clonePlan{}, err
	}
	if checkout == headRef {
		checkout = ""
	}
	if checkout != "" {
		if err := validateCloneRefs([]string{checkout}); err != nil {
			return clonePlan{}, err
		}
	}

	var plan clonePlan
	for _, ref := range append([]string{checkout}, opts.Refs...) {
		if isCommitID(ref) && !slices.Contains(plan.commits, ref) {
			plan.commits = append(plan.commits, ref)
		}
	}
	if len(opts.Refs) == 0 {
		// Everything is cloned. A branch is checked out as part of that. A tag or
		// commit is checked out afterward.
		switch {
		case isBranchRef(checkout):
			plan.primary = checkout
		case checkout != "":
			plan.detachAt = checkout
		}
		return plan, nil
	}

	refs := uniqueRefs(opts.Refs)
	if !bare && !isCommitID(checkout) {
		plan.primary = checkout
		cloned := checkout
		if cloned == "" {
			cloned = headRef
		}
		others := slices.DeleteFunc(refs, func(ref string) bool {
			return ref == cloned
		})
		if len(others) > 0 {
			existing, defaultBranch, err := b.findRemoteRefs(ctx, others)
			if err != nil {
				return clonePlan{}, err
			}
			if cloned == headRef {
				cloned = defaultBranch
			}
			plan.fetch = slices.DeleteFunc(existing, func(ref string) bool {
				return ref == cloned
			})
		}
	} else {
		if !bare {
			refs = uniqueRefs(append([]string{checkout}, refs...))
		}
		if len(refs) > 1 || refs[0] != headRef {
			existing, defaultBranch, err := b.findRemoteRefs(ctx, refs)
			if err != nil {
				return clonePlan{}, err
			}
			plan.detachAt = firstCommit(refs, existing, defaultBranch)
			if len(existing) > 0 {
				plan.primary = existing[0]
				plan.fetch = existing[1:]
			}
		}
	}
	if isTagRef(plan.primary) {
		// git clone --revision does not record the tag, so it is fetched again
		// for that purpose. Its objects are already present.
		plan.fetch = append([]string{plan.primary}, plan.fetch...)
	}
	return plan, nil
}

// findRemoteRefs returns, in the order requested, those of the specified refs
// that exist in the remote, with HEAD resolved to the fully qualified ref of
// the branch it points to and each commit ID replaced by a branch or tag whose
// tip is that commit. Commits that are the tip of no branch or tag are left
// out. It also returns the fully qualified ref of the remote's default branch,
// which is empty if the remote has none.
//
// Equivalent to:
//
//	git ls-remote --symref <url> HEAD <ref>...
//
// or, if any commit ID is among the refs:
//
//	git ls-remote --symref <url> HEAD 'refs/heads/*' 'refs/tags/*'
func (b *baseRepo) findRemoteRefs(
	ctx context.Context,
	refs []string,
) ([]string, string, error) {
	args := []string{"ls-remote", "--symref", b.url, headRef}
	if slices.ContainsFunc(refs, isCommitID) {
		// Every branch and tag tip is needed to find those that are commits
		args = append(args, branchRefPrefix+"*", tagRefPrefix+"*")
	} else {
		for _, ref := range refs {
			if ref != headRef {
				args = append(args, ref)
			}
		}
	}
	cmd := b.buildGitCommand(ctx, args...)
	// Override the cmd.Dir that's set by b.buildGitCommand(). The repository
	// does not exist yet.
	cmd.Dir = b.homeDir
	out, err := libExec.Exec(cmd)
	if err != nil {
		return nil, "", fmt.Errorf(
			"error listing refs %v in remote repo %q: %w",
			refs, b.url, err,
		)
	}
	remoteRefs, defaultBranch, err := parseLsRemoteSymrefOutput(out)
	if err != nil {
		return nil, "", err
	}
	existing, err := existingRefs(refs, remoteRefs, defaultBranch)
	if err != nil {
		return nil, "", fmt.Errorf(
			"error resolving refs in remote repo %q: %w", b.url, err,
		)
	}
	return existing, defaultBranch, nil
}

// existingRefs returns, in the order requested, those of the specified refs
// that appear among the specified remote refs, with HEAD replaced by the
// specified default branch and each commit ID replaced by a branch or tag whose
// tip is that commit, or left out if there is none. remoteRefs maps the name of
// each remote ref to the ID of the commit it points to. It returns an error if
// HEAD is requested and the remote has refs but no default branch was
// reported, since HEAD then cannot be resolved.
func existingRefs(
	refs []string,
	remoteRefs map[string]string,
	defaultBranch string,
) ([]string, error) {
	existing := make([]string, 0, len(refs))
	for _, ref := range refs {
		switch {
		case ref == headRef:
			if defaultBranch == "" {
				if len(remoteRefs) == 0 {
					// The remote is empty, so it has no default branch
					continue
				}
				return nil, errors.New(
					"cannot determine the default branch: the server did not report " +
						"which branch HEAD points to",
				)
			}
			ref = defaultBranch
		case isCommitID(ref):
			var ok bool
			if ref, ok = refAtCommit(ref, refs, remoteRefs); !ok {
				continue
			}
		default:
			if _, ok := remoteRefs[ref]; !ok {
				continue
			}
		}
		if !slices.Contains(existing, ref) {
			existing = append(existing, ref)
		}
	}
	return existing, nil
}

// refAtCommit returns a fully qualified branch or tag ref among the specified
// remote refs whose tip is the specified commit, preferring one among the
// specified requested refs, and otherwise the first by name. remoteRefs maps
// the name of each remote ref to the ID of the commit it points to. It returns
// false if no branch or tag has the commit at its tip.
func refAtCommit(
	commitID string,
	requested []string,
	remoteRefs map[string]string,
) (string, bool) {
	var candidates []string
	for name, id := range remoteRefs {
		if (isBranchRef(name) || isTagRef(name)) && strings.EqualFold(id, commitID) {
			candidates = append(candidates, name)
		}
	}
	if len(candidates) == 0 {
		return "", false
	}
	slices.Sort(candidates)
	for _, candidate := range candidates {
		if slices.Contains(requested, candidate) {
			return candidate, true
		}
	}
	return candidates[0], true
}

// firstCommit returns the first of the specified requested refs if it is a
// commit ID once refs missing from the remote are skipped, or an empty string
// otherwise. existing holds those of the requested refs that exist in the
// remote, as findRemoteRefs returns them, and defaultBranch the remote's
// default branch, if it has one. A commit ID counts as existing, since cloning
// fails if it cannot be found.
func firstCommit(
	refs []string,
	existing []string,
	defaultBranch string,
) string {
	for _, ref := range refs {
		switch {
		case isCommitID(ref):
			return ref
		case ref == headRef:
			if defaultBranch != "" {
				return ""
			}
		case slices.Contains(existing, ref):
			return ""
		}
	}
	return ""
}

// requireHEAD returns an error unless HEAD of a freshly cloned repository is the
// specified fully qualified branch ref. git clone --branch falls back to a tag
// of the same name when the branch does not exist, which leaves HEAD detached.
//
// Equivalent to:
//
//	git symbolic-ref -q HEAD
func (b *baseRepo) requireHEAD(ctx context.Context, branchRef string) error {
	res, err := libExec.Exec(b.buildGitCommand(
		ctx, "symbolic-ref", "-q", headRef,
	))
	if err != nil {
		var exitErr *libExec.ExitError
		if !errors.As(err, &exitErr) || exitErr.ExitCode != 1 {
			return fmt.Errorf("error reading HEAD of repo %q: %w", b.url, err)
		}
		// HEAD is detached
	}
	if strings.TrimSpace(string(res)) != branchRef {
		return fmt.Errorf(
			"branch %q does not exist in remote repo %q",
			strings.TrimPrefix(branchRef, branchRefPrefix), b.url,
		)
	}
	return nil
}

// fetchAdditionalRefs fetches the specified fully qualified refs, and no other
// tags, into a freshly cloned repository, limited to the specified depth unless
// it is 0. "refs/heads/*" and "refs/tags/*" stand for all branches and all
// tags. A bare clone stores each branch under its own name. A regular clone
// stores each as a remote-tracking branch. Both store each tag under its own
// name.
//
// Equivalent to:
//
//	git fetch --no-tags [--depth=<depth>] origin <refspec>...
func (b *baseRepo) fetchAdditionalRefs(
	ctx context.Context,
	depth uint,
	bare bool,
	refs []string,
) error {
	if len(refs) == 0 {
		return nil
	}
	args := []string{"fetch", "--no-tags"}
	if depth > 0 {
		args = append(args, "--depth", fmt.Sprint(depth))
	}
	args = append(args, "origin")
	args = append(args, refspecs(refs, bare)...)
	if _, err := libExec.Exec(b.buildGitCommand(ctx, args...)); err != nil {
		return fmt.Errorf("error fetching refs %v from repo %q: %w", refs, b.url, err)
	}
	return nil
}

// requireCommits returns an error unless every one of the specified commits is
// present in a freshly cloned repository, or can be made present without
// fetching any commit by its ID. Every branch and tag is fetched, limited to
// the specified depth unless it is 0, and deepened as far as it takes.
func (b *baseRepo) requireCommits(
	ctx context.Context,
	depth uint,
	bare bool,
	commitIDs []string,
) error {
	missing, err := b.missingCommits(ctx, commitIDs)
	if err != nil || len(missing) == 0 {
		return err
	}
	branches := branchRefPrefix + "*"
	refs := []string{branches, tagRefPrefix + "*"}
	if err = b.fetchAdditionalRefs(ctx, depth, bare, refs); err != nil {
		return err
	}
	if !bare {
		if err = b.trackBranches(ctx, []string{branches}); err != nil {
			return err
		}
	}
	if missing, err = b.deepenUntilPresent(
		ctx,
		missing,
		refspecs(refs, bare),
	); err != nil {
		return err
	}
	if len(missing) > 0 {
		return fmt.Errorf(
			"no branch or tag in remote repo %q reaches commit %s",
			b.url, strings.Join(missing, ", "),
		)
	}
	return nil
}

// deepenUntilPresent deepens a shallow repository along the specified refspecs,
// doubling the number of commits it asks for each time, until none of the
// specified commits are missing, the repository is no longer shallow, or
// deepening brings in nothing new. It returns the commits that are still
// missing.
func (b *baseRepo) deepenUntilPresent(
	ctx context.Context,
	commitIDs []string,
	refspecs []string,
) ([]string, error) {
	missing := commitIDs
	for step := uint(initialDeepenStep); ; step *= 2 {
		var err error
		if missing, err = b.missingCommits(ctx, missing); err != nil {
			return nil, err
		}
		if len(missing) == 0 {
			return nil, nil
		}
		shallow, err := b.shallowCommits(ctx)
		if err != nil {
			return nil, err
		}
		if len(shallow) == 0 {
			return missing, nil
		}
		if err = b.deepen(ctx, step, refspecs...); err != nil {
			return nil, err
		}
		deepened, err := b.shallowCommits(ctx)
		if err != nil {
			return nil, err
		}
		if maps.Equal(deepened, shallow) {
			// What remains hidden is not reachable from the refspecs
			return missing, nil
		}
	}
}

// validateCloneRefs returns an error if any of the specified refs is not HEAD,
// a fully qualified branch or tag ref, or a full commit ID.
func validateCloneRefs(refs []string) error {
	for _, ref := range refs {
		if !isValidCloneRef(ref) {
			return fmt.Errorf(
				"unsupported ref %q: must be %s, a fully qualified branch or tag "+
					"ref, or a full commit ID",
				ref, headRef,
			)
		}
	}
	return nil
}

func isValidCloneRef(ref string) bool {
	if ref == headRef || isCommitID(ref) {
		return true
	}
	for _, prefix := range []string{branchRefPrefix, tagRefPrefix} {
		if name, ok := strings.CutPrefix(ref, prefix); ok {
			return name != ""
		}
	}
	return false
}

// refspecs returns the refspecs that fetch the specified fully qualified refs
// into a bare or regular repository. "refs/heads/*" and "refs/tags/*" stand
// for all branches and all tags. A bare repository stores each branch under its
// own name. A regular repository stores each as a remote-tracking branch. Both
// store each tag under its own name.
func refspecs(refs []string, bare bool) []string {
	specs := make([]string, len(refs))
	for i, ref := range refs {
		if isBranchRef(ref) && !bare {
			specs[i] = fmt.Sprintf(
				"+%s:%s%s",
				ref, remoteBranchRefPrefix, strings.TrimPrefix(ref, branchRefPrefix),
			)
		} else {
			specs[i] = fmt.Sprintf("+%s:%s", ref, ref)
		}
	}
	return specs
}

func isBranchRef(ref string) bool {
	return strings.HasPrefix(ref, branchRefPrefix)
}

func isTagRef(ref string) bool {
	return strings.HasPrefix(ref, tagRefPrefix)
}

func isCommitID(ref string) bool {
	return commitIDRegex.MatchString(ref)
}

// uniqueRefs returns a copy of refs with duplicates removed, preserving the
// order in which each ref first appears.
func uniqueRefs(refs []string) []string {
	unique := make([]string, 0, len(refs))
	for _, ref := range refs {
		if !slices.Contains(unique, ref) {
			unique = append(unique, ref)
		}
	}
	return unique
}

// parseLsRemoteSymrefOutput parses the output of git ls-remote --symref. It
// returns a map of the name of each ref listed to the ID of the commit it
// points to, which for an annotated tag is the commit the tag points to, not
// the tag itself. It also returns the target of the symbolic ref HEAD, which is
// empty if HEAD was not listed as a symbolic ref.
func parseLsRemoteSymrefOutput(
	output []byte,
) (map[string]string, string, error) {
	refs := map[string]string{}
	peeled := map[string]string{}
	var headTarget string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		value, name, ok := strings.Cut(scanner.Text(), "\t")
		if !ok {
			continue
		}
		if target, isSymref := strings.CutPrefix(value, symrefPrefix); isSymref {
			if name == headRef {
				headTarget = target
			}
			continue
		}
		if tag, isPeeled := strings.CutSuffix(name, peeledSuffix); isPeeled {
			peeled[tag] = value
			continue
		}
		refs[name] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("error scanning ls-remote output: %w", err)
	}
	maps.Copy(refs, peeled)
	return refs, headTarget, nil
}
