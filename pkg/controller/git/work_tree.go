package git

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	libExec "github.com/akuity/kargo/pkg/exec"
	"github.com/akuity/kargo/pkg/logging"
)

// WorkTree is an interface for interacting with any working tree of a Git
// repository.
type WorkTree interface {
	// AddAll stages pending changes for commit.
	AddAll() error
	// AddAllAndCommit is a convenience function that stages pending changes for
	// commit to the current branch and then commits them using the provided
	// commit message.
	AddAllAndCommit(message string, commitOpts *CommitOptions) error
	// Clean cleans the working tree.
	Clean() error
	// Clear executes `git rm -rf .` to remove all files from the working tree.
	Clear() error
	// Close cleans up file system resources used by this working tree. This
	// should always be called before a WorkTree goes out of scope.
	Close() error
	// Checkout checks out the specified branch.
	Checkout(branch string) error
	// Commit commits staged changes to the current branch.
	Commit(message string, opts *CommitOptions) error
	// CreateChildBranch creates a new branch that is a child of the current
	// branch.
	CreateChildBranch(branch string) error
	// CreateOrphanedBranch creates a new branch that shares no commit history
	// with any other branch.
	CreateOrphanedBranch(branch string) error
	// CreateTag creates a new tag with the specified name.
	CreateTag(name, msg string) error
	// CurrentBranch returns the current branch
	CurrentBranch() (string, error)
	// DeleteBranch deletes the specified branch
	DeleteBranch(branch string) error
	// Dir returns an absolute path to the working tree.
	Dir() string
	// Fetch fetches updates from the remote repository.
	Fetch(opts *FetchOptions) error
	// HasDiffs returns a bool indicating whether the working tree currently
	// contains any differences from what's already at the head of the current
	// branch.
	HasDiffs() (bool, error)
	// HomeDir returns an absolute path to the home directory of the system user
	// who cloned the repo associated with this working tree.
	HomeDir() string
	// GetDiffPathsForCommitID returns a string slice indicating the paths,
	// relative to the root of the repository, of any files that are new or
	// modified in the commit with the given ID.
	GetDiffPathsForCommitID(commitID string) ([]string, error)
	// IsAncestor returns true if parent branch is an ancestor of child
	IsAncestor(parent string, child string) (bool, error)
	// IsRebasing returns a bool indicating whether the working tree is currently
	// in the middle of a rebase operation.
	IsRebasing() (bool, error)
	// LastCommitID returns the ID (sha) of the most recent commit to the current
	// branch.
	LastCommitID() (string, error)
	// ListTags returns a slice of tags in the repository with metadata such as
	// commit ID, creator date, and subject.
	ListTags() ([]TagMetadata, error)
	// ListCommits returns a slice of commits in the current branch with
	// metadata such as commit ID, commit date, and subject.
	ListCommits(opts *ListCommitsOptions) ([]CommitMetadata, error)
	// CommitMessage returns the text of the most recent commit message associated
	// with the specified commit ID.
	CommitMessage(id string) (string, error)
	// GetCommitSignatureInfo returns signature information for the specified
	// commit, including whether it is signed by a trusted key and the
	// identity of the signer. If opts provides a SigningKey, a temporary
	// keyring is created with that key imported at ultimate trust and used
	// for verification.
	GetCommitSignatureInfo(commitID string) (*CommitSignatureInfo, error)
	// IntegrateRemoteChanges integrates remote changes into the local branch
	// before pushing. This fetches the target branch and applies the operator-
	// configured integration policy (rebase, merge, or fail). It is a no-op
	// if the remote branch does not exist or the policy is None.
	IntegrateRemoteChanges(*IntegrationOptions) error
	// Pull fetches and integrates changes from a remote branch
	// into the current local branch.
	Pull(*PullOptions) error
	// Push pushes from the local repository to the remote repository.
	Push(*PushOptions) error
	// RefsHaveDiffs returns whether there is a diff between two commits/branches
	RefsHaveDiffs(commit1 string, commit2 string) (bool, error)
	// RemoteBranchExists returns a bool indicating if the specified branch exists
	// in the remote repository.
	RemoteBranchExists(branch string) (bool, error)
	// ResetHard performs a hard reset on the working tree.
	ResetHard() error
	// URL returns the remote URL of the repository.
	URL() string
	// UpdateSubmodules updates the submodules in the working tree.
	UpdateSubmodules() error
}

// ListCommitsOptions contains options for listing commits.
type ListCommitsOptions struct {
	// Limit is the maximum number of commits to return. 0 means no limit.
	Limit uint
	// Skip is the number of commits to skip before returning results.
	Skip uint
	// Since limits commits to those at or after this time.
	Since *time.Time
}

// workTree is an implementation of the WorkTree interface for interacting with
// any working tree of a Git repository.
type workTree struct {
	*baseRepo
	bareRepo *bareRepo
}

type LoadWorkTreeOptions struct {
	Credentials *RepoCredentials
}

func LoadWorkTree(path string, opts *LoadWorkTreeOptions) (WorkTree, error) {
	if opts == nil {
		opts = &LoadWorkTreeOptions{}
	}
	w := &workTree{
		baseRepo: &baseRepo{
			creds: opts.Credentials,
			dir:   path,
		},
	}
	res, err := libExec.Exec(w.buildGitCommand(
		"config",
		repoDirConfigKey,
	))
	if err != nil {
		return nil, fmt.Errorf("error reading repo dir from config: %w", err)
	}
	repoPath := strings.TrimSpace(string(res))
	if err = w.loadHomeDir(); err != nil {
		return nil, fmt.Errorf("error reading repo home dir from config: %w", err)
	}
	if err = w.loadURLs(); err != nil {
		return nil,
			fmt.Errorf(`error reading URL of remote "origin" from config: %w`, err)
	}
	if err = w.setupAuth(w.homeDir); err != nil {
		return nil, fmt.Errorf("error configuring the credentials: %w", err)
	}
	br, err := LoadBareRepo(repoPath, &LoadBareRepoOptions{
		Credentials: opts.Credentials,
	})
	if err != nil {
		return nil, err
	}
	w.bareRepo = br.(*bareRepo) // nolint: forcetypeassert
	return w, nil
}

func (w *workTree) AddAll() error {
	if _, err := libExec.Exec(w.buildGitCommand("add", ".")); err != nil {
		return fmt.Errorf("error staging changes for commit: %w", err)
	}
	return nil
}

func (w *workTree) AddAllAndCommit(message string, commitOpts *CommitOptions) error {
	if err := w.AddAll(); err != nil {
		return err
	}
	return w.Commit(message, commitOpts)
}

func (w *workTree) Clean() error {
	if _, err := libExec.Exec(w.buildGitCommand("clean", "-fd")); err != nil {
		return fmt.Errorf("error cleaning worktree: %w", err)
	}
	return nil
}

func (w *workTree) Clear() error {
	if _, err := libExec.Exec(
		w.buildGitCommand("rm", "-rf", "--ignore-unmatch", "."),
	); err != nil {
		return fmt.Errorf("error clearing worktree: %w", err)
	}
	return nil
}

func (w *workTree) Close() error {
	if w.bareRepo != nil {
		return w.bareRepo.RemoveWorkTree(w.dir)
	}
	if err := os.RemoveAll(w.dir); err != nil {
		return fmt.Errorf("error removing working tree at %q: %w", w.dir, err)
	}
	return nil
}

func (w *workTree) Checkout(branch string) error {
	if _, err := libExec.Exec(w.buildGitCommand(
		"checkout",
		branch,
		// The next line makes it crystal clear to git that we're checking out
		// a branch. We need to do this because branch names can often resemble
		// paths within the repo.
		"--",
	)); err != nil {
		return fmt.Errorf(
			"error checking out branch %q from repo %q: %w",
			branch, w.originalURL, err,
		)
	}
	return nil
}

// CommitOptions represents options for committing changes to a git repository.
type CommitOptions struct {
	// AllowEmpty indicates whether an empty commit should be allowed.
	AllowEmpty bool
	// Author is the author of the commit. If nil, the default author already
	// configured in the git repository will be used.
	Author *User
}

func (w *workTree) Commit(message string, opts *CommitOptions) error {
	if opts == nil {
		opts = &CommitOptions{}
	}

	var homeDir string
	if opts.Author != nil {
		// This author + committer is specific to this commit, so we will override
		// repository-level user information by creating a temporary home directory,
		// configuring the user information "globally" within it, and then ensuring
		// the git commit command uses that home directory.
		var err error
		if homeDir, err = os.MkdirTemp(w.homeDir, ""); err != nil {
			return fmt.Errorf(
				"error creating virtual home directory %q for commit command: %w",
				homeDir, err,
			)
		}
		defer func() {
			if cleanErr := os.RemoveAll(homeDir); cleanErr != nil {
				logging.LoggerFromContext(context.TODO()).
					Error(cleanErr, "error removing virtual home directory", "path", homeDir)
			}
		}()
		if err = w.setupUser(homeDir, opts.Author); err != nil {
			return fmt.Errorf(
				"error setting up author + committer information for commit command: %w", err,
			)
		}
	}

	cmdTokens := []string{"commit", "-m", message}
	if opts.AllowEmpty {
		cmdTokens = append(cmdTokens, "--allow-empty")
	}

	cmd := w.buildGitCommand(cmdTokens...)
	if homeDir != "" {
		// Override the home directory set by b.buildGitCommand().
		w.setCmdHome(cmd, homeDir)
	}
	if _, err := libExec.Exec(cmd); err != nil {
		return fmt.Errorf("error committing changes: %w", err)
	}
	return nil
}

func (w *workTree) CommitMessage(id string) (string, error) {
	msgBytes, err := libExec.Exec(
		w.buildGitCommand(
			"log",
			"-n",
			"1",
			"--pretty=format:%B",
			id,
			// `--` clarifies that id is a branch name and not a file name
			"--",
		),
	)
	if err != nil {
		return "", fmt.Errorf("error obtaining commit message for commit %q: %w", id, err)
	}
	return string(msgBytes), nil
}

func (w *workTree) CreateChildBranch(branch string) error {
	if _, err := libExec.Exec(w.buildGitCommand(
		"checkout",
		"-b",
		branch,
		// The next line makes it crystal clear to git that we're checking out
		// a branch. We need to do this because branch names can often resemble
		// paths within the repo.
		"--",
	)); err != nil {
		return fmt.Errorf(
			"error creating new branch %q for repo %q: %w",
			branch, w.originalURL, err,
		)
	}
	return nil
}

func (w *workTree) CreateOrphanedBranch(branch string) error {
	if _, err := libExec.Exec(w.buildGitCommand(
		"switch",
		"--orphan",
		branch,
		"--discard-changes",
	)); err != nil {
		return fmt.Errorf(
			"error creating orphaned branch %q for repo %q: %w",
			branch, w.originalURL, err,
		)
	}
	return w.Clean()
}

func (w *workTree) CreateTag(tag, msg string) error {
	cmd := w.buildGitCommand("tag", "-a", tag, "-m", msg)
	if _, err := libExec.Exec(cmd); err != nil {
		return fmt.Errorf("error creating annotated tag %q: %w", tag, err)
	}
	return nil
}

func (w *workTree) CurrentBranch() (string, error) {
	res, err := libExec.Exec(w.buildGitCommand("branch", "--show-current"))
	if err != nil {
		return "", fmt.Errorf(
			"error checking current branch for repo %q: %w",
			w.originalURL, err,
		)
	}
	return strings.TrimSpace(string(res)), nil
}

func (w *workTree) DeleteBranch(branch string) error {
	if _, err := libExec.Exec(w.buildGitCommand(
		"branch",
		"--delete",
		"--force",
		branch,
	)); err != nil {
		return fmt.Errorf("error deleting branch %q for repo %q: %w", branch, w.accessURL, err)
	}
	return nil
}

// FetchOptions represents options for fetching from a remote git repository.
type FetchOptions struct {
	// Branch optionally specifies a single branch to fetch. If empty, all
	// branches are fetched.
	Branch string
	// Depth optionally limits fetching to the specified number of commits. A
	// value of 0 (the default) indicates no depth limit.
	Depth uint
}

func (w *workTree) Fetch(opts *FetchOptions) error {
	if opts == nil {
		opts = &FetchOptions{}
	}
	args := []string{"fetch", "origin"}
	if opts.Branch != "" {
		args = append(args,
			fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s",
				opts.Branch, opts.Branch,
			),
		)
	} else {
		args = append(args, "+refs/heads/*:refs/remotes/origin/*")
	}
	if opts.Depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", opts.Depth))
	}
	if _, err := libExec.Exec(w.buildGitCommand(args...)); err != nil {
		return fmt.Errorf(
			"error fetching from repo %q: %w", w.originalURL, err,
		)
	}
	return nil
}

func (w *workTree) GetDiffPathsForCommitID(commitID string) ([]string, error) {
	resBytes, err := libExec.Exec(w.buildGitCommand("show", "--pretty=", "--name-only", "--first-parent", commitID))
	if err != nil {
		return nil, fmt.Errorf("error getting diff paths for commit %q: %w", commitID, err)
	}
	var paths []string
	scanner := bufio.NewScanner(bytes.NewReader(resBytes))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		paths = append(
			paths,
			scanner.Text(),
		)
	}
	return paths, nil
}

func (w *workTree) HasDiffs() (bool, error) {
	resBytes, err := libExec.Exec(w.buildGitCommand("status", "-s"))
	if err != nil {
		return false, fmt.Errorf("error checking status of branch: %w", err)
	}
	return len(resBytes) > 0, nil
}

func (w *workTree) IsAncestor(parent string, child string) (bool, error) {
	_, err := libExec.Exec(w.buildGitCommand("merge-base", "--is-ancestor", parent, child))
	if err == nil {
		return true, nil
	}
	var execErr *libExec.ExitError
	if errors.As(err, &execErr) {
		if execErr.ExitCode == 1 {
			return false, nil
		}
	}
	return false, fmt.Errorf("error testing ancestry of branches %q, %q: %w", parent, child, err)
}

func (w *workTree) IsRebasing() (bool, error) {
	res, err := libExec.Exec(w.buildGitCommand("rev-parse", "--git-path", "rebase-merge"))
	if err != nil {
		return false, fmt.Errorf("error determining rebase status: %w", err)
	}
	rebaseMerge := filepath.Join(w.dir, strings.TrimSpace(string(res)))
	if _, err = os.Stat(rebaseMerge); !os.IsNotExist(err) {
		if err != nil {
			return false, err
		}
		return true, nil
	}
	if res, err = libExec.Exec(w.buildGitCommand("rev-parse", "--git-path", "rebase-apply")); err != nil {
		return false, fmt.Errorf("error determining rebase status: %w", err)
	}
	rebaseApply := filepath.Join(w.dir, strings.TrimSpace(string(res)))
	if _, err = os.Stat(rebaseApply); !os.IsNotExist(err) {
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (w *workTree) LastCommitID() (string, error) {
	shaBytes, err := libExec.Exec(w.buildGitCommand("rev-parse", "HEAD"))
	if err != nil {
		return "", fmt.Errorf("error obtaining ID of last commit: %w", err)
	}
	return strings.TrimSpace(string(shaBytes)), nil
}

type CommitMetadata struct {
	// CommitID is the ID (sha) of the commit.
	ID string
	// CommitDate is the date of the commit.
	CommitDate time.Time
	// Author is the author of the commit, in the format "Name <email>".
	Author string
	// Committer is the person who committed the commit, in the format
	// "Name <email>".
	Committer string
	// Subject is the subject (first line) of the commit message.
	Subject string
}

func (w *workTree) ListCommits(opts *ListCommitsOptions) ([]CommitMetadata, error) {
	if opts == nil {
		opts = &ListCommitsOptions{}
	}
	args := []string{
		"log",
		"--first-parent",
		// This format is designed to output the following fields, separated by
		// tabs (%x09):
		//
		// - commit ID
		// - commit date
		// - author name and email
		// - committer name and email
		// - subject
		"--pretty=format:%H%x09%ci%x09%an <%ae>%x09%cn <%ce>%x09%s",
	}
	if opts.Limit > 0 {
		args = append(args, fmt.Sprintf("--max-count=%d", opts.Limit))
	}
	if opts.Skip > 0 {
		args = append(args, fmt.Sprintf("--skip=%d", opts.Skip))
	}
	if opts.Since != nil {
		args = append(args, fmt.Sprintf("--since=%s", opts.Since.Format(time.RFC3339)))
	}

	b, err := libExec.Exec(w.buildGitCommand(args...))
	if err != nil {
		return nil, fmt.Errorf(
			"error listing commits for repo %q: %w",
			w.originalURL, err,
		)
	}

	var commits []CommitMetadata
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		line := scanner.Bytes()
		parts := bytes.SplitN(scanner.Bytes(), []byte("\t"), 5)
		if len(parts) != 5 {
			return nil, fmt.Errorf("unexpected number of fields: %q", line)
		}

		commitDate, err := time.Parse("2006-01-02 15:04:05 -0700", string(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("error parsing commit date %q: %w", parts[1], err)
		}

		commits = append(commits, CommitMetadata{
			ID:         string(parts[0]),
			CommitDate: commitDate,
			Author:     string(parts[2]),
			Committer:  string(parts[3]),
			Subject:    string(parts[4]),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning commits output: %w", err)
	}
	return commits, nil
}

// TagMetadata represents metadata associated with a Git tag.
type TagMetadata struct {
	// Tag is the name of the tag.
	Tag string
	// CommitID is the ID (sha) of the commit associated with the tag.
	CommitID string
	// CreatorDate is the creation date of an annotated tag, or the commit date
	// of a lightweight tag.
	CreatorDate time.Time
	// Author is the author of the commit message associated with the tag, in
	// the format "Name <email>".
	Author string
	// Committer is the person who committed the commit associated with the tag,
	// in the format "Name <email>".
	Committer string
	// Subject is the subject (first line) of the commit message associated
	// with the tag.
	Subject string
	// Tagger is the person who created the tag, in the format "Name <email>".
	// This field is only populated for annotated tags.
	Tagger string
	// Annotation is the annotation of the tag, if it is an annotated tag.
	Annotation string
}

func parseTagMetadataLine(line []byte) (TagMetadata, error) {
	parts := bytes.Split(line, []byte("|*|"))
	if l := len(parts); l < 6 || l == 7 {
		return TagMetadata{}, fmt.Errorf("unexpected number of fields: %q", line)
	}

	creatorDate, err := time.Parse("2006-01-02 15:04:05 -0700", string(parts[5]))
	if err != nil {
		return TagMetadata{}, fmt.Errorf("error parsing creator date %q: %w", parts[5], err)
	}

	metadata := TagMetadata{
		Tag:         string(parts[0]),
		CommitID:    string(parts[1]),
		Subject:     string(parts[2]),
		Author:      string(parts[3]),
		Committer:   string(parts[4]),
		CreatorDate: creatorDate,
	}

	if len(parts) >= 8 {
		metadata.Tagger = string(parts[6])
		metadata.Annotation = string(bytes.Join(parts[7:], []byte("|*|")))
	}

	return metadata, nil
}

func (w *workTree) ListTags() ([]TagMetadata, error) {
	if _, err := libExec.Exec(w.buildGitCommand("fetch", "origin", "--tags")); err != nil {
		return nil, fmt.Errorf(
			"error fetching tags from repo %q: %w",
			w.originalURL, err,
		)
	}

	// These formats are quite complex, so we break them down into smaller
	// pieces for readability.
	//
	// They are designed to output the following fields, separated by `|*|`:
	// - tag name
	// - commit ID
	// - subject
	// - author name and email
	// - committer name and email
	// - creator date
	//
	// For annotated tags, we also output the following fields:
	// - tagger name and email
	// - tag annotation (the first line of the tag message)
	//
	// The `if`/`then`/`else` logic is used to ensure that we get the commit ID
	// and subject of the tag, regardless of whether it's an annotated or
	// lightweight tag.
	//
	// nolint: lll
	const (
		formatAnnotatedTag   = `%(refname:short)|*|%(*objectname)|*|%(*contents:subject)|*|%(*authorname) %(*authoremail)|*|%(*committername) %(*committeremail)|*|%(creatordate:iso8601)|*|%(taggername) %(taggeremail)|*|%(contents:subject)`
		formatLightweightTag = `%(refname:short)|*|%(objectname)|*|%(contents:subject)|*|%(authorname) %(authoremail)|*|%(committername) %(committeremail)|*|%(creatordate:iso8601)`
		tagFormat            = `%(if)%(*objectname)%(then)` + formatAnnotatedTag + `%(else)` + formatLightweightTag + `%(end)`
	)

	tagsBytes, err := libExec.Exec(w.buildGitCommand(
		"for-each-ref",
		"--sort=-creatordate",
		"--format="+tagFormat,
		"refs/tags",
	))
	if err != nil {
		return nil, fmt.Errorf(
			"error listing tags for repo %q: %w",
			w.originalURL, err,
		)
	}

	var tags []TagMetadata
	scanner := bufio.NewScanner(bytes.NewReader(tagsBytes))
	for scanner.Scan() {
		tag, err := parseTagMetadataLine(scanner.Bytes())
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

// PullOptions represents options for pulling changes from a remote branch.
type PullOptions struct {
	// Branch is the remote branch to pull from. If not specified, the current
	// branch will be pulled.
	Branch string
	// Force indicates whether the local branch should be reset to match
	// the remote exactly, discarding any local state.
	Force bool
}

func (w *workTree) Pull(opts *PullOptions) error {
	if opts == nil {
		opts = &PullOptions{}
	}
	branch := opts.Branch
	if branch == "" {
		var err error
		if branch, err = w.CurrentBranch(); err != nil {
			return err
		}
	}
	if err := w.Fetch(&FetchOptions{Branch: branch}); err != nil {
		return fmt.Errorf("error fetching branch %q: %w", branch, err)
	}
	if opts.Force {
		cmd := w.buildGitCommand(
			"reset", "--hard", fmt.Sprintf("origin/%s", branch),
		)
		if _, err := libExec.Exec(cmd); err != nil {
			return fmt.Errorf(
				"error resetting to origin/%s: %w", branch, err,
			)
		}
		return nil
	}
	cmd := w.buildGitCommand("merge", fmt.Sprintf("origin/%s", branch))
	if _, err := libExec.Exec(cmd); err != nil {
		return fmt.Errorf(
			"error merging origin/%s: %w", branch, err,
		)
	}
	return nil
}

// PushOptions represents options for pushing changes to a remote git
// repository.
type PushOptions struct {
	// Force indicates whether the push should be forced.
	Force bool
	// TargetBranch specifies the branch to push to. If empty, the current branch
	// TargetBranch specifies a remote branch to push to. If empty, the remote
	// branch's name will be assumed to be the same as the branch checked out
	// locally. Whether this field is empty or non-empty, if Tag is non-empty,
	// it takes precedence and the tag will be pushed -- the branch will not.
	TargetBranch string
	// IntegrationPolicy controls how remote changes are integrated before
	// pushing. If empty or set to PushIntegrationPolicyNone, no integration
	// is performed.
	IntegrationPolicy PushIntegrationPolicy
	// Tag specifies a tag to push to the remote repository. If this field and
	// TargetBranch are both non-empty, this field takes precedence and the tag
	// will be pushed -- the branch will not.
	Tag string
}

// https://regex101.com/r/f7kTjs/1
//
// nolint: lll
var nonFastForwardRegex = regexp.MustCompile(`(?m)^\s*!\s+\[(?:remote )?rejected].+\((?:non-fast-forward|fetch first|cannot lock ref.*|incorrect old value provided)\)\s*$`)

func (w *workTree) Push(opts *PushOptions) error {
	if opts == nil {
		opts = &PushOptions{}
	}
	if opts.IntegrationPolicy == "" {
		opts.IntegrationPolicy = PushIntegrationPolicyNone
	}

	args := []string{"push", "origin"}
	if opts.Tag != "" {
		args = append(args, "tag", opts.Tag)
		if _, err := libExec.Exec(w.buildGitCommand(args...)); err != nil {
			return fmt.Errorf("error pushing tag: %w", err)
		}
		return nil
	}

	targetBranch := opts.TargetBranch
	if targetBranch == "" {
		var err error
		if targetBranch, err = w.CurrentBranch(); err != nil {
			return err
		}
	}

	args = append(args, fmt.Sprintf("HEAD:%s", targetBranch))
	if err := w.IntegrateRemoteChanges(&IntegrationOptions{
		TargetBranch:      targetBranch,
		IntegrationPolicy: opts.IntegrationPolicy,
	}); err != nil {
		return err
	}

	if opts.Force {
		args = append(args, "--force")
	}

	if res, err := libExec.Exec(w.buildGitCommand(args...)); err != nil {
		if nonFastForwardRegex.MatchString(string(res)) {
			return fmt.Errorf("error pushing branch: %w", ErrNonFastForward)
		}
		return fmt.Errorf("error pushing branch: %w", err)
	}
	return nil
}

func (w *workTree) RefsHaveDiffs(commit1 string, commit2 string) (bool, error) {
	// `git diff --quiet` returns 0 if no diff, 1 if diff, and non-zero/one for any other error
	_, err := libExec.Exec(w.buildGitCommand(
		"diff", "--quiet", fmt.Sprintf("%s..%s", commit1, commit2), "--"))
	if err == nil {
		return false, nil
	}
	var execErr *libExec.ExitError
	if errors.As(err, &execErr) {
		if execErr.ExitCode == 1 {
			return true, nil
		}
	}
	return false, fmt.Errorf("error diffing commits %s..%s: %w", commit1, commit2, err)
}

func (w *workTree) ResetHard() error {
	if _, err := libExec.Exec(w.buildGitCommand("reset", "--hard")); err != nil {
		return fmt.Errorf("error resetting branch working tree: %w", err)
	}
	return nil
}

func (w *workTree) UpdateSubmodules() error {
	if _, err := libExec.Exec(w.buildGitCommand("submodule", "update", "--init", "--recursive")); err != nil {
		return fmt.Errorf("error updating submodules: %w", err)
	}
	return nil
}

// validateSparsePatterns validates sparse checkout patterns for cone mode.
func validateSparsePatterns(patterns []string) error {
	for _, p := range patterns {
		if filepath.IsAbs(p) {
			return fmt.Errorf("sparse checkout pattern must be relative: %s", p)
		}
		if strings.HasPrefix(p, "..") {
			return fmt.Errorf("sparse checkout pattern cannot escape repository: %s", p)
		}
		// Cone mode only supports directory patterns, not globs
		if strings.ContainsAny(p, "*?[") {
			return fmt.Errorf(
				"sparse checkout pattern cannot contain glob characters (*, ?, [): %s; "+
					"use directory paths like 'src/app' instead",
				p,
			)
		}
	}
	return nil
}

// configureSparseCheckout configures sparse checkout in cone mode for the
// working tree with the specified directory patterns.
func (w *workTree) configureSparseCheckout(patterns []string) error {
	if err := validateSparsePatterns(patterns); err != nil {
		return err
	}

	// Initialize sparse checkout in cone mode (fast, directory-based)
	if _, err := libExec.Exec(
		w.buildGitCommand("sparse-checkout", "init", "--cone"),
	); err != nil {
		return fmt.Errorf("error initializing sparse checkout: %w", err)
	}

	// Set the sparse checkout patterns
	setArgs := append([]string{"sparse-checkout", "set"}, patterns...)
	if _, err := libExec.Exec(w.buildGitCommand(setArgs...)); err != nil {
		return fmt.Errorf("error setting sparse checkout patterns %v: %w", patterns, err)
	}
	return nil
}
