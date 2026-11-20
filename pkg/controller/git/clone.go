package git

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
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
	// unadvertisedObjectMsg appears in git's output when a server refuses to
	// serve a commit requested by ID.
	unadvertisedObjectMsg = "allow request for unadvertised object"
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
	// tags that do not exist in the remote are skipped. Commits are fetched by
	// ID, which the server must allow. If empty, all branches and tags are
	// cloned.
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
	// ref or a commit ID. If empty, git clone fetches the remote's default
	// branch, or everything if no refs were requested.
	primary string
	// fetch holds the refs and commit IDs fetched after cloning.
	fetch []string
	// detachAt is what a regular clone checks out after cloning, if anything.
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
	if plan.detachAt != "" {
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
		return fmt.Errorf(
			"error cloning repo %q into %q: %w",
			b.url, b.dir, explainCommitFetchError(err),
		)
	}
	if err = b.fetchAdditionalRefs(ctx, opts, bare, plan.fetch); err != nil {
		return err
	}
	if !bare {
		if err = b.trackBranches(ctx, plan.fetch); err != nil {
			return err
		}
	}
	if plan.detachAt == "" {
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
// clone that is limited to specific refs clones what it checks out, then
// fetches the rest. A bare clone that is limited to specific refs clones the
// first of them that exists in the remote, or the remote's default branch if
// none exist, then fetches the rest.
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
	if len(opts.Refs) == 0 {
		// Everything is cloned. A branch is checked out as part of that. A tag or
		// commit is checked out afterward.
		switch {
		case isBranchRef(checkout):
			plan.primary = checkout
		case checkout != "":
			plan.detachAt = checkout
			if isCommitID(checkout) {
				// A commit need not be reachable from any branch or tag
				plan.fetch = []string{checkout}
			}
		}
		return plan, nil
	}

	refs := uniqueRefs(opts.Refs)
	if !bare {
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
	} else if len(refs) > 1 || refs[0] != headRef {
		existing, _, err := b.findRemoteRefs(ctx, refs)
		if err != nil {
			return clonePlan{}, err
		}
		if len(existing) > 0 {
			plan.primary = existing[0]
			plan.fetch = existing[1:]
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
// the branch it points to. Commit IDs are returned as they are, since a remote
// cannot be asked whether it has a commit. It also returns the fully qualified
// ref of the remote's default branch, which is empty if the remote has none or
// if no remote was consulted.
//
// Equivalent to:
//
//	git ls-remote --symref <url> HEAD <ref>...
func (b *baseRepo) findRemoteRefs(
	ctx context.Context,
	refs []string,
) ([]string, string, error) {
	if !slices.ContainsFunc(refs, func(ref string) bool {
		return !isCommitID(ref)
	}) {
		return refs, "", nil
	}
	args := []string{"ls-remote", "--symref", b.url, headRef}
	for _, ref := range refs {
		if ref != headRef && !isCommitID(ref) {
			args = append(args, ref)
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
	existing := make([]string, 0, len(refs))
	for _, ref := range refs {
		switch {
		case ref == headRef:
			if defaultBranch == "" {
				continue
			}
			ref = defaultBranch
		case isCommitID(ref):
		default:
			if _, ok := remoteRefs[ref]; !ok {
				continue
			}
		}
		if !slices.Contains(existing, ref) {
			existing = append(existing, ref)
		}
	}
	return existing, defaultBranch, nil
}

// fetchAdditionalRefs fetches the specified fully qualified refs and commit
// IDs, and no other tags, into a freshly cloned repository at the depth of the
// clone. A bare clone stores each branch under its own name. A regular clone
// stores each as a remote-tracking branch. Both store each tag under its own
// name.
//
// Equivalent to:
//
//	git fetch --no-tags [--depth=<depth>] origin <refspec>...
func (b *baseRepo) fetchAdditionalRefs(
	ctx context.Context,
	opts BaseCloneOptions,
	bare bool,
	refs []string,
) error {
	if len(refs) == 0 {
		return nil
	}
	args := []string{"fetch", "--no-tags"}
	if opts.Depth > 0 {
		args = append(args, "--depth", fmt.Sprint(opts.Depth))
	}
	args = append(args, "origin")
	for _, ref := range refs {
		switch {
		case isCommitID(ref):
			args = append(args, ref)
		case isBranchRef(ref) && !bare:
			args = append(args, fmt.Sprintf(
				"+%s:%s%s",
				ref, remoteBranchRefPrefix, strings.TrimPrefix(ref, branchRefPrefix),
			))
		default:
			args = append(args, fmt.Sprintf("+%s:%s", ref, ref))
		}
	}
	if _, err := libExec.Exec(b.buildGitCommand(ctx, args...)); err != nil {
		return fmt.Errorf(
			"error fetching refs %v from repo %q: %w",
			refs, b.url, explainCommitFetchError(err),
		)
	}
	return nil
}

// trackBranches adds any fully qualified branch refs among the specified refs
// to the "origin" remote's configured fetch refspecs. Git treats a
// remote-tracking branch as one only if a configured refspec covers it, so
// without this, a branch fetched after cloning could not be checked out by
// name, and later fetches would not update it.
//
// Equivalent to:
//
//	git remote set-branches --add origin <branch>...
func (b *baseRepo) trackBranches(ctx context.Context, refs []string) error {
	var branches []string
	for _, ref := range refs {
		if branch, ok := strings.CutPrefix(ref, branchRefPrefix); ok {
			branches = append(branches, branch)
		}
	}
	if len(branches) == 0 {
		return nil
	}
	args := append([]string{"remote", "set-branches", "--add", "origin"}, branches...)
	if _, err := libExec.Exec(b.buildGitCommand(ctx, args...)); err != nil {
		return fmt.Errorf(
			"error tracking branches %v of repo %q: %w",
			branches, b.url, err,
		)
	}
	return nil
}

// explainCommitFetchError adds an explanation to err if it reports that the
// server refused to serve a commit requested by ID.
func explainCommitFetchError(err error) error {
	var exitErr *libExec.ExitError
	if errors.As(err, &exitErr) &&
		bytes.Contains(exitErr.Output, []byte(unadvertisedObjectMsg)) {
		return fmt.Errorf("server does not support fetching commits by ID: %w", err)
	}
	return err
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
// returns the set of ref names listed and the target of the symbolic ref HEAD,
// which is empty if HEAD was not listed as a symbolic ref.
func parseLsRemoteSymrefOutput(
	output []byte,
) (map[string]struct{}, string, error) {
	refs := map[string]struct{}{}
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
		refs[name] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("error scanning ls-remote output: %w", err)
	}
	return refs, headTarget, nil
}
