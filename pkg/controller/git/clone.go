package git

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"

	libExec "github.com/akuity/kargo/pkg/exec"
)

const (
	// headRef is the symbolic ref that points at a remote's default branch.
	headRef = "HEAD"
	// branchRefPrefix is the prefix of every fully qualified branch ref.
	branchRefPrefix = "refs/heads/"
	// remoteBranchRefPrefix is the prefix of every remote-tracking branch ref
	// for the "origin" remote.
	remoteBranchRefPrefix = "refs/remotes/origin/"
	// symrefPrefix prefixes the target of a symbolic ref in the output of
	// git ls-remote --symref.
	symrefPrefix = "ref: "
)

// BaseCloneOptions represents options common to all clones of a Git
// repository.
type BaseCloneOptions struct {
	// BaseDir is an existing directory within which all other directories
	// created and managed by the repository implementation will be created. If
	// not specified, the operating system's temporary directory will be used.
	// Overriding that default is useful under certain circumstances.
	BaseDir string
	// Refs optionally limits the clone to the specified refs. Each must be HEAD
	// (the remote's default branch) or a fully qualified branch ref, e.g.
	// "refs/heads/main". Refs that do not exist in the remote are skipped. If
	// empty, all branches and tags are cloned.
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

// clone clones the remote repository into b.dir. For a regular (non-bare)
// clone, checkoutBranch is the branch to check out, where empty means the
// remote's default branch. For a bare clone, it must be empty.
func (b *baseRepo) clone(
	ctx context.Context,
	opts BaseCloneOptions,
	checkoutBranch string,
	bare bool,
) error {
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
	branch := checkoutBranch
	var fetchRefs []string
	if len(opts.Refs) == 0 {
		if opts.Depth > 0 {
			// --depth implies --single-branch, but all branches were requested
			args = append(args, "--no-single-branch")
		}
	} else {
		var err error
		if branch, fetchRefs, err = b.selectRefs(
			ctx,
			opts,
			checkoutBranch,
			bare,
		); err != nil {
			return err
		}
		// Only the requested refs are cloned, so tags that happen to point into
		// their history are not.
		args = append(args, "--single-branch", "--no-tags")
	}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, b.url, b.dir)
	cmd := b.buildGitCommand(ctx, args...)
	cmd.Dir = b.homeDir // Override the cmd.Dir that's set by b.buildGitCommand()
	if _, err := libExec.Exec(cmd); err != nil {
		return fmt.Errorf("error cloning repo %q into %q: %w", b.url, b.dir, err)
	}
	return b.fetchAdditionalRefs(ctx, opts, bare, fetchRefs)
}

// selectRefs works out how to clone only the refs in opts.Refs. It returns the
// branch to clone with --single-branch, where empty means the remote's default
// branch, and the fully qualified refs to fetch after cloning. A regular clone
// clones checkoutBranch. A bare clone clones the first of opts.Refs that exists
// in the remote or, if none exist, the remote's default branch.
func (b *baseRepo) selectRefs(
	ctx context.Context,
	opts BaseCloneOptions,
	checkoutBranch string,
	bare bool,
) (string, []string, error) {
	if err := validateCloneRefs(opts.Refs); err != nil {
		return "", nil, err
	}
	refs := uniqueRefs(opts.Refs)

	if !bare {
		cloned := headRef
		if checkoutBranch != "" {
			cloned = branchRefPrefix + checkoutBranch
		}
		others := slices.DeleteFunc(refs, func(ref string) bool {
			return ref == cloned
		})
		if len(others) == 0 {
			return checkoutBranch, nil, nil
		}
		existing, defaultBranch, err := b.findRemoteRefs(ctx, others)
		if err != nil {
			return "", nil, err
		}
		if cloned == headRef {
			cloned = defaultBranch
		}
		existing = slices.DeleteFunc(existing, func(ref string) bool {
			return ref == cloned
		})
		return checkoutBranch, existing, nil
	}

	if len(refs) == 1 && refs[0] == headRef {
		return "", nil, nil
	}
	existing, _, err := b.findRemoteRefs(ctx, refs)
	if err != nil {
		return "", nil, err
	}
	if len(existing) == 0 {
		return "", nil, nil
	}
	return strings.TrimPrefix(existing[0], branchRefPrefix), existing[1:], nil
}

// findRemoteRefs returns, in the order requested, those of the specified refs
// that exist in the remote, with HEAD resolved to the fully qualified ref of
// the branch it points to. It also returns that branch's fully qualified ref,
// which is empty if the remote has no default branch.
//
// Equivalent to:
//
//	git ls-remote --symref <url> HEAD <ref>...
func (b *baseRepo) findRemoteRefs(
	ctx context.Context,
	refs []string,
) ([]string, string, error) {
	args := []string{"ls-remote", "--symref", b.url, headRef}
	for _, ref := range refs {
		if ref != headRef {
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
		if ref == headRef {
			if defaultBranch == "" {
				continue
			}
			ref = defaultBranch
		} else if _, ok := remoteRefs[ref]; !ok {
			continue
		}
		if !slices.Contains(existing, ref) {
			existing = append(existing, ref)
		}
	}
	return existing, defaultBranch, nil
}

// fetchAdditionalRefs fetches the specified fully qualified branch refs, and no
// tags, into a freshly cloned repository at the depth of the clone. A bare
// clone stores each branch under its own name. A regular clone stores each as
// a remote-tracking branch.
//
// Equivalent to:
//
//	git fetch --no-tags [--depth=<depth>] origin +<ref>:<local ref>...
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
		localRef := ref
		if !bare {
			localRef = remoteBranchRefPrefix +
				strings.TrimPrefix(ref, branchRefPrefix)
		}
		args = append(args, fmt.Sprintf("+%s:%s", ref, localRef))
	}
	if _, err := libExec.Exec(b.buildGitCommand(ctx, args...)); err != nil {
		return fmt.Errorf(
			"error fetching refs %v from repo %q: %w",
			refs, b.url, err,
		)
	}
	return nil
}

// validateCloneRefs returns an error if any of the specified refs is neither
// HEAD nor a fully qualified branch ref.
func validateCloneRefs(refs []string) error {
	for _, ref := range refs {
		if ref == headRef {
			continue
		}
		if name, ok := strings.CutPrefix(ref, branchRefPrefix); ok && name != "" {
			continue
		}
		return fmt.Errorf(
			"unsupported ref %q: must be %s or a fully qualified branch ref",
			ref, headRef,
		)
	}
	return nil
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
