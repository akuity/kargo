package git

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	libExec "github.com/akuity/kargo/internal/exec"
)

// RepoCredentials represents the credentials for connecting to a private git
// repository.
type RepoCredentials struct {
	// SSHPrivateKey is a private key that can be used for both reading from and
	// writing to some remote repository.
	SSHPrivateKey string `json:"sshPrivateKey,omitempty"`
	// Username identifies a principal, which combined with the value of the
	// Password field, can be used for both reading from and writing to some
	// remote repository.
	Username string `json:"username,omitempty"`
	// Password, when combined with the principal identified by the Username
	// field, can be used for both reading from and writing to some remote
	// repository.
	Password string `json:"password,omitempty"`
}

// CommitOptions represents options for committing changes to a git repository.
type CommitOptions struct {
	// AllowEmpty indicates whether an empty commit should be allowed.
	AllowEmpty bool
}

// Repo is an interface for interacting with a git repository.
type Repo interface {
	// AddAll stages pending changes for commit.
	AddAll() error
	// AddAllAndCommit is a convenience function that stages pending changes for
	// commit to the current branch and then commits them using the provided
	// commit message.
	AddAllAndCommit(message string) error
	// Clean cleans the working directory.
	Clean() error
	// Close cleans up file system resources used by this repository. This should
	// always be called before a repository goes out of scope.
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
	// CurrentBranch returns the current branch
	CurrentBranch() string
	// DeleteBranch deletes the specified branch
	DeleteBranch(branch string) error
	// HasDiffs returns a bool indicating whether the working directory currently
	// contains any differences from what's already at the head of the current
	// branch.
	HasDiffs() (bool, error)
	// GetDiffPaths returns a string slice indicating the paths, relative to the
	// root of the repository, of any new or modified files.
	GetDiffPaths() ([]string, error)
	// GetDiffPathsForCommitID returns a string slice indicating the paths, relative to the
	// root of the repository, of any files that are new or modified since the given commit ID.
	GetDiffPathsSinceCommitID(commitId string) ([]string, error)
	// IsAncestor returns true if parent branch is an ancestor of child
	IsAncestor(parent string, child string) (bool, error)
	// LastCommitID returns the ID (sha) of the most recent commit to the current
	// branch.
	LastCommitID() (string, error)
	// ListTags returns a slice of tags in the repository.
	ListTags() ([]string, error)
	// CommitMessage returns the text of the most recent commit message associated
	// with the specified commit ID.
	CommitMessage(id string) (string, error)
	// CommitMessages returns a slice of commit messages starting with id1 and
	// ending with id2. The results exclude id1, but include id2.
	CommitMessages(id1, id2 string) ([]string, error)
	// Push pushes from the current branch to a remote branch by the same name.
	Push(force bool) error
	// RefsHaveDiffs returns whether there is a diff between two commits/branches
	RefsHaveDiffs(commit1 string, commit2 string) (bool, error)
	// RemoteBranchExists returns a bool indicating if the specified branch exists
	// in the remote repository.
	RemoteBranchExists(branch string) (bool, error)
	// ResetHard performs a hard reset.
	ResetHard() error
	// URL returns the remote URL of the repository.
	URL() string
	// WorkingDir returns an absolute path to the repository's working tree.
	WorkingDir() string
	// HomeDir returns an absolute path to the home directory of the system user
	// who has cloned this repo.
	HomeDir() string
}

// repo is an implementation of the Repo interface for interacting with a git
// repository.
type repo struct {
	url                   string
	homeDir               string
	dir                   string
	currentBranch         string
	insecureSkipTLSVerify bool
}

// CloneOptions represents options for cloning a git repository.
type CloneOptions struct {
	// Branch is the name of the branch to clone. If not specified, the default
	// branch will be cloned.
	Branch string
	// SingleBranch indicates whether the clone should be a single-branch clone.
	SingleBranch bool
	// Shallow indicates whether the clone should be with a depth of 1. This is
	// useful for speeding up the cloning process when all we care about is the
	// latest commit from a single branch.
	Shallow bool
	// InsecureSkipTLSVerify specifies whether certificate verification errors
	// should be ignored when cloning the repository. The setting will be
	// remembered for subsequent interactions with the remote repository.
	InsecureSkipTLSVerify bool
}

// Clone produces a local clone of the remote git repository at the specified
// URL and returns an implementation of the Repo interface that is stateful and
// NOT suitable for use across multiple goroutines. This function will also
// perform any setup that is required for successfully authenticating to the
// remote repository.
func Clone(
	repoURL string,
	repoCreds RepoCredentials,
	opts *CloneOptions,
) (Repo, error) {
	homeDir, err := os.MkdirTemp("", "repo-")
	if err != nil {
		return nil, fmt.Errorf("error creating home directory for repo %q: %w", repoURL, err)
	}
	r := &repo{
		url:                   repoURL,
		homeDir:               homeDir,
		dir:                   filepath.Join(homeDir, "repo"),
		insecureSkipTLSVerify: opts.InsecureSkipTLSVerify,
	}
	if err = r.setupAuth(repoCreds); err != nil {
		return nil, err
	}
	return r, r.clone(opts)
}

func (r *repo) AddAll() error {
	if _, err := libExec.Exec(r.buildCommand("add", ".")); err != nil {
		return fmt.Errorf("error staging changes for commit: %w", err)
	}
	return nil
}

func (r *repo) AddAllAndCommit(message string) error {
	if err := r.AddAll(); err != nil {
		return err
	}
	return r.Commit(message, nil)
}

func (r *repo) Clean() error {
	if _, err := libExec.Exec(r.buildCommand("clean", "-fd")); err != nil {
		return fmt.Errorf("error cleaning branch %q: %w", r.currentBranch, err)
	}
	return nil
}

func (r *repo) clone(opts *CloneOptions) error {
	if opts == nil {
		opts = &CloneOptions{}
	}
	args := []string{"clone", "--no-tags"}
	if opts.Branch != "" {
		args = append(args, "--branch", opts.Branch)
		r.currentBranch = opts.Branch
	}
	if opts.SingleBranch {
		args = append(args, "--single-branch")
	}
	if opts.Shallow {
		args = append(args, "--depth=1")
	}
	args = append(args, r.url, r.dir)
	cmd := r.buildCommand(args...)
	cmd.Dir = r.homeDir // Override the cmd.Dir that's set by r.buildCommand()
	if _, err := libExec.Exec(cmd); err != nil {
		return fmt.Errorf("error cloning repo %q into %q: %w", r.url, r.dir, err)
	}
	if opts.Branch == "" {
		// If branch wasn't specified as part of options, we need to determine it manually
		resBytes, err := libExec.Exec(r.buildCommand(
			"branch",
			"--show-current",
		))
		if err != nil {
			return fmt.Errorf("error determining branch after cloning: %w", err)
		}
		r.currentBranch = strings.TrimSpace(string(resBytes))
	}
	return nil
}

func (r *repo) Close() error {
	return os.RemoveAll(r.homeDir)
}

func (r *repo) Checkout(branch string) error {
	r.currentBranch = branch
	if _, err := libExec.Exec(r.buildCommand(
		"checkout",
		branch,
		// The next line makes it crystal clear to git that we're checking out
		// a branch. We need to do this because branch names can often resemble
		// paths within the repo.
		"--",
	)); err != nil {
		return fmt.Errorf("error checking out branch %q from repo %q: %w", branch, r.url, err)
	}
	return nil
}

func (r *repo) Commit(message string, opts *CommitOptions) error {
	if opts == nil {
		opts = &CommitOptions{}
	}
	cmdTokens := []string{"commit", "-m", message}
	if opts.AllowEmpty {
		cmdTokens = append(cmdTokens, "--allow-empty")
	}

	if _, err := libExec.Exec(r.buildCommand(cmdTokens...)); err != nil {
		return fmt.Errorf("error committing changes to branch %q: %w", r.currentBranch, err)
	}
	return nil
}

func (r *repo) RefsHaveDiffs(commit1 string, commit2 string) (bool, error) {
	// `git diff --quiet` returns 0 if no diff, 1 if diff, and non-zero/one for any other error
	_, err := libExec.Exec(r.buildCommand(
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

func (r *repo) CreateChildBranch(branch string) error {
	r.currentBranch = branch
	if _, err := libExec.Exec(r.buildCommand(
		"checkout",
		"-b",
		branch,
		// The next line makes it crystal clear to git that we're checking out
		// a branch. We need to do this because branch names can often resemble
		// paths within the repo.
		"--",
	)); err != nil {
		return fmt.Errorf("error creating new branch %q for repo %q: %w", branch, r.url, err)
	}
	return nil
}

func (r *repo) CreateOrphanedBranch(branch string) error {
	r.currentBranch = branch
	if _, err := libExec.Exec(r.buildCommand(
		"switch",
		"--orphan",
		branch,
		"--discard-changes",
	)); err != nil {
		return fmt.Errorf("error creating orphaned branch %q for repo %q: %w", branch, r.url, err)
	}
	return r.Clean()
}

func (r *repo) CurrentBranch() string {
	return r.currentBranch
}

func (r *repo) DeleteBranch(branch string) error {
	if _, err := libExec.Exec(r.buildCommand(
		"branch",
		"--delete",
		"--force",
		branch,
	)); err != nil {
		return fmt.Errorf("error deleting branch %q for repo %q: %w", branch, r.url, err)
	}
	return nil
}

func (r *repo) HasDiffs() (bool, error) {
	resBytes, err := libExec.Exec(r.buildCommand("status", "-s"))
	if err != nil {
		return false, fmt.Errorf("error checking status of branch %q: %w", r.currentBranch, err)
	}
	return len(resBytes) > 0, nil
}

func (r *repo) GetDiffPaths() ([]string, error) {
	resBytes, err := libExec.Exec(r.buildCommand("status", "-s"))
	if err != nil {
		return nil, fmt.Errorf("error checking status of branch %q: %w", r.currentBranch, err)
	}
	var paths []string
	scanner := bufio.NewScanner(bytes.NewReader(resBytes))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		paths = append(
			paths,
			strings.SplitN(strings.TrimSpace(scanner.Text()), " ", 2)[1],
		)
	}
	return paths, nil
}

func (r *repo) GetDiffPathsSinceCommitID(commitId string) ([]string, error) {
	resBytes, err := libExec.Exec(r.buildCommand("diff", "--name-only", commitId+"..HEAD"))
	if err != nil {
		return nil,
			fmt.Errorf("error getting diffs since commit %q %w", commitId, err)
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

func (r *repo) IsAncestor(parent string, child string) (bool, error) {
	_, err := libExec.Exec(r.buildCommand("merge-base", "--is-ancestor", parent, child))
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

func (r *repo) LastCommitID() (string, error) {
	shaBytes, err := libExec.Exec(r.buildCommand("rev-parse", "HEAD"))
	if err != nil {
		return "", fmt.Errorf("error obtaining ID of last commit: %w", err)
	}
	return strings.TrimSpace(string(shaBytes)), nil
}

func (r *repo) ListTags() ([]string, error) {
	if _, err :=
		libExec.Exec(r.buildCommand("fetch", "origin", "--tags")); err != nil {
		return nil, fmt.Errorf("error fetching tags from repo %q: %w", r.url, err)
	}
	tagsBytes, err := libExec.Exec(r.buildCommand("tag", "--list", "--sort", "-creatordate"))
	if err != nil {
		return nil, fmt.Errorf("error listing tags for repo %q: %w", r.url, err)
	}
	var tags []string
	scanner := bufio.NewScanner(bytes.NewReader(tagsBytes))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		tags = append(tags, strings.TrimSpace(scanner.Text()))
	}
	return tags, nil
}

func (r *repo) CommitMessage(id string) (string, error) {
	msgBytes, err := libExec.Exec(
		r.buildCommand("log", "-n", "1", "--pretty=format:%s", id),
	)
	if err != nil {
		return "", fmt.Errorf("error obtaining commit message for commit %q: %w", id, err)
	}
	return string(msgBytes), nil
}

func (r *repo) CommitMessages(id1, id2 string) ([]string, error) {
	allMsgBytes, err := libExec.Exec(r.buildCommand(
		"log",
		"--pretty=oneline",
		"--decorate-refs=",
		"--decorate-refs-exclude=",
		fmt.Sprintf("%s..%s", id1, id2),
	))
	if err != nil {
		return nil, fmt.Errorf("error obtaining commit messages between commits %q and %q: %w", id1, id2, err)
	}
	msgsBytes := bytes.Split(allMsgBytes, []byte("\n"))
	var msgs []string
	for _, msgBytes := range msgsBytes {
		msgStr := string(msgBytes)
		// There's usually a trailing newline in the result. We could just discard
		// the last line, but this feels more resilient against the admittedly
		// remote possibility that that could change one day.
		if strings.TrimSpace(msgStr) != "" {
			msgs = append(msgs, string(msgBytes))
		}
	}
	return msgs, nil
}

func (r *repo) Push(force bool) error {
	args := []string{"push", "origin", r.currentBranch}
	if force {
		args = append(args, "--force")
	}
	if _, err := libExec.Exec(r.buildCommand(args...)); err != nil {
		return fmt.Errorf("error pushing branch %q: %w", r.currentBranch, err)
	}
	return nil
}

func (r *repo) RemoteBranchExists(branch string) (bool, error) {
	_, err := libExec.Exec(r.buildCommand(
		"ls-remote",
		"--heads",
		"--exit-code", // Return 2 if not found
		r.url,
		branch,
	))
	var exitErr *libExec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode == 2 {
		// Branch does not exist
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf(
			"error checking for existence of branch %q in remote repo %q: %w",
			branch,
			r.url,
			err,
		)
	}
	return true, nil
}

func (r *repo) ResetHard() error {
	if _, err := libExec.Exec(r.buildCommand("reset", "--hard")); err != nil {
		return fmt.Errorf("error resetting branch working tree: %w", err)
	}
	return nil
}

func (r *repo) URL() string {
	return r.url
}

func (r *repo) HomeDir() string {
	return r.homeDir
}

func (r *repo) WorkingDir() string {
	return r.dir
}

// SetupAuth configures the git CLI for authentication using either SSH or the
// "store" (username/password-based) credential helper.
func (r *repo) setupAuth(repoCreds RepoCredentials) error {
	// Configure the git client
	cmd := r.buildCommand("config", "--global", "user.name", "Kargo Render")
	cmd.Dir = r.homeDir // Override the cmd.Dir that's set by r.buildCommand()
	if _, err := libExec.Exec(cmd); err != nil {
		return fmt.Errorf("error configuring git username: %w", err)
	}
	cmd =
		r.buildCommand("config", "--global", "user.email", "kargo-render@akuity.io")
	cmd.Dir = r.homeDir // Override the cmd.Dir that's set by r.buildCommand()
	if _, err := libExec.Exec(cmd); err != nil {
		return fmt.Errorf("error configuring git user email address: %w", err)
	}

	// If an SSH key was provided, use that.
	if repoCreds.SSHPrivateKey != "" {
		sshConfigPath := filepath.Join(r.homeDir, ".ssh", "config")
		// nolint: lll
		const sshConfig = "Host *\n  StrictHostKeyChecking no\n  UserKnownHostsFile=/dev/null"
		if err :=
			os.WriteFile(sshConfigPath, []byte(sshConfig), 0600); err != nil {
			return fmt.Errorf("error writing SSH config to %q: %w", sshConfigPath, err)
		}

		rsaKeyPath := filepath.Join(r.homeDir, ".ssh", "id_rsa")
		if err := os.WriteFile(
			rsaKeyPath,
			[]byte(repoCreds.SSHPrivateKey),
			0600,
		); err != nil {
			return fmt.Errorf("error writing SSH key to %q: %w", rsaKeyPath, err)
		}
		return nil // We're done
	}

	// If we get to here, we're authenticating using a password

	// Set up the credential helper
	cmd = r.buildCommand("config", "--global", "credential.helper", "store")
	cmd.Dir = r.homeDir // Override the cmd.Dir that's set by r.buildCommand()
	if _, err := libExec.Exec(cmd); err != nil {
		return fmt.Errorf("error configuring git credential helper: %w", err)
	}

	credentialURL, err := url.Parse(r.url)
	if err != nil {
		return fmt.Errorf("error parsing URL %q: %w", r.url, err)
	}
	// Remove path and query string components from the URL
	credentialURL.Path = ""
	credentialURL.RawQuery = ""
	// If the username is the empty string, we assume we're working with a git
	// provider like GitHub that only requires the username to be non-empty. We
	// arbitrarily set it to "git".
	if repoCreds.Username == "" {
		repoCreds.Username = "git"
	}
	// Augment the URL with user/pass information.
	credentialURL.User = url.UserPassword(repoCreds.Username, repoCreds.Password)
	// Write the augmented URL to the location used by the "stored" credential
	// helper.
	credentialsPath := filepath.Join(r.homeDir, ".git-credentials")
	if err := os.WriteFile(
		credentialsPath,
		[]byte(credentialURL.String()),
		0600,
	); err != nil {
		return fmt.Errorf("error writing credentials to %q: %w", credentialsPath, err)
	}
	return nil
}

func (r *repo) buildCommand(arg ...string) *exec.Cmd {
	cmd := exec.Command("git", arg...)
	homeEnvVar := fmt.Sprintf("HOME=%s", r.homeDir)
	if cmd.Env == nil {
		cmd.Env = []string{homeEnvVar}
	} else {
		cmd.Env = append(cmd.Env, homeEnvVar)
	}
	if r.insecureSkipTLSVerify {
		cmd.Env = append(cmd.Env, "GIT_SSL_NO_VERIFY=true")
	}
	cmd.Dir = r.dir
	return cmd
}
