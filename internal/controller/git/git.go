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
	"time"

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

type SigningKeyType string

const (
	SigningKeyTypeGPG SigningKeyType = "gpg"
)

// User represents the user contributing to a git repository.
type User struct {
	// Name is the user's full name.
	Name string
	// Email is the user's email address.
	Email string
	// SigningKeyType indicates the type of signing key.
	SigningKeyType SigningKeyType
	// SigningKeyPath is an optional path referencing a signing key for
	// signing git objects.
	SigningKeyPath string
}

// CommitOptions represents options for committing changes to a git repository.
type CommitOptions struct {
	// AllowEmpty indicates whether an empty commit should be allowed.
	AllowEmpty bool
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
	// GetDiffPathsForCommitID returns a string slice indicating the paths,
	// relative to the root of the repository, of any files that are new or
	// modified in the commit with the given ID.
	GetDiffPathsForCommitID(commitID string) ([]string, error)
	// IsAncestor returns true if parent branch is an ancestor of child
	IsAncestor(parent string, child string) (bool, error)
	// LastCommitID returns the ID (sha) of the most recent commit to the current
	// branch.
	LastCommitID() (string, error)
	// ListTags returns a slice of tags in the repository with metadata such as
	// commit ID, creator date, and subject.
	ListTags() ([]TagMetadata, error)
	// ListCommits returns a slice of commits in the current branch with
	// metadata such as commit ID, commit date, and subject.
	ListCommits(limit, skip uint) ([]CommitMetadata, error)
	// CommitMessage returns the text of the most recent commit message associated
	// with the specified commit ID.
	CommitMessage(id string) (string, error)
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

// ClientOptions represents options for the git client. Commonly, the
// repository credentials are required to authenticate with a remote
// repository.
type ClientOptions struct {
	// User represents the actor that performs operations against the git
	// repository. This has no effect on authentication, see Credentials for
	// specifying authentication configuration.
	User *User
	// Credentials represents the authentication information.
	Credentials *RepoCredentials
}

const (
	// FilterBlobless is a filter that excludes blobs from the clone.
	// When using this filter, the initial Git clone will download all
	// reachable commits and trees, and only download the blobs for commits
	// when you do a Git checkout (including the first checkout during the
	// clone).
	//
	// When using a blobless clone, you can still explore the commits of the
	// repository without downloading additional data. This means that you can
	// perform commands like `git log`, or even `git log -- <path>` with the
	// same performance as a full clone.
	//
	// Commands like `git diff` or `git blame <path>` require the contents of
	// the paths to compute diffs, so these will trigger blob downloads the
	// first time they are run.
	FilterBlobless = "blob:none"
)

// CloneOptions represents options for cloning a git repository.
type CloneOptions struct {
	// Branch is the name of the branch to clone. If not specified, the default
	// branch will be cloned.
	Branch string
	// SingleBranch indicates whether the clone should be a single-branch clone.
	SingleBranch bool
	// Depth is the number of commits to fetch from the remote repository. If
	// zero, all commits will be fetched.
	Depth uint
	// Filter allows for partially cloning the repository by specifying a
	// filter. When a filter is specified, the server will only send a
	// subset of reachable objects according to a given object filter.
	//
	// For more information, see:
	// - https://git-scm.com/docs/git-clone#Documentation/git-clone.txt-code--filtercodeemltfilter-specgtem
	// - https://git-scm.com/docs/git-rev-list#Documentation/git-rev-list.txt---filterltfilter-specgt
	// - https://github.blog/2020-12-21-get-up-to-speed-with-partial-clone-and-shallow-clone/
	// - https://docs.gitlab.com/ee/topics/git/partial_clone.html
	Filter string
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
	clientOpts *ClientOptions,
	cloneOpts *CloneOptions,
) (Repo, error) {
	homeDir, err := os.MkdirTemp("", "repo-")
	if err != nil {
		return nil, fmt.Errorf("error creating home directory for repo %q: %w", repoURL, err)
	}
	r := &repo{
		url:                   repoURL,
		homeDir:               homeDir,
		dir:                   filepath.Join(homeDir, "repo"),
		insecureSkipTLSVerify: cloneOpts.InsecureSkipTLSVerify,
	}
	if err = r.setupClient(clientOpts); err != nil {
		return nil, err
	}
	return r, r.clone(cloneOpts)
}

func (r *repo) AddAll() error {
	if _, err := libExec.Exec(r.buildGitCommand("add", ".")); err != nil {
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
	if _, err := libExec.Exec(r.buildGitCommand("clean", "-fd")); err != nil {
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
	if opts.Depth > 0 {
		args = append(args, "--depth", fmt.Sprint(opts.Depth))
	}
	args = append(args, r.url, r.dir)
	cmd := r.buildGitCommand(args...)
	cmd.Dir = r.homeDir // Override the cmd.Dir that's set by r.buildGitCommand()
	if _, err := libExec.Exec(cmd); err != nil {
		return fmt.Errorf("error cloning repo %q into %q: %w", r.url, r.dir, err)
	}
	if opts.Branch == "" {
		// If branch wasn't specified as part of options, we need to determine it manually
		resBytes, err := libExec.Exec(r.buildGitCommand(
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
	if _, err := libExec.Exec(r.buildGitCommand(
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

	if _, err := libExec.Exec(r.buildGitCommand(cmdTokens...)); err != nil {
		return fmt.Errorf("error committing changes to branch %q: %w", r.currentBranch, err)
	}
	return nil
}

func (r *repo) RefsHaveDiffs(commit1 string, commit2 string) (bool, error) {
	// `git diff --quiet` returns 0 if no diff, 1 if diff, and non-zero/one for any other error
	_, err := libExec.Exec(r.buildGitCommand(
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
	if _, err := libExec.Exec(r.buildGitCommand(
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
	if _, err := libExec.Exec(r.buildGitCommand(
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
	if _, err := libExec.Exec(r.buildGitCommand(
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
	resBytes, err := libExec.Exec(r.buildGitCommand("status", "-s"))
	if err != nil {
		return false, fmt.Errorf("error checking status of branch %q: %w", r.currentBranch, err)
	}
	return len(resBytes) > 0, nil
}

func (r *repo) GetDiffPathsForCommitID(commitID string) ([]string, error) {
	resBytes, err := libExec.Exec(r.buildGitCommand("show", "--pretty=", "--name-only", commitID))
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

func (r *repo) IsAncestor(parent string, child string) (bool, error) {
	_, err := libExec.Exec(r.buildGitCommand("merge-base", "--is-ancestor", parent, child))
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
	shaBytes, err := libExec.Exec(r.buildGitCommand("rev-parse", "HEAD"))
	if err != nil {
		return "", fmt.Errorf("error obtaining ID of last commit: %w", err)
	}
	return strings.TrimSpace(string(shaBytes)), nil
}

func (r *repo) ListTags() ([]TagMetadata, error) {
	if _, err := libExec.Exec(r.buildGitCommand("fetch", "origin", "--tags")); err != nil {
		return nil, fmt.Errorf("error fetching tags from repo %q: %w", r.url, err)
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
	// The `if`/`then`/`else` logic is used to ensure that we get the commit ID
	// and subject of the tag, regardless of whether it's an annotated or
	// lightweight tag.
	//
	// nolint: lll
	const (
		formatAnnotatedTag   = `%(refname:short)|*|%(*objectname)|*|%(*contents:subject)|*|%(*authorname) %(*authoremail)|*|%(*committername) %(*committeremail)|*|%(*creatordate:iso8601)`
		formatLightweightTag = `%(refname:short)|*|%(objectname)|*|%(contents:subject)|*|%(authorname) %(authoremail)|*|%(committername) %(committeremail)|*|%(creatordate:iso8601)`
		tagFormat            = `%(if)%(*objectname)%(then)` + formatAnnotatedTag + `%(else)` + formatLightweightTag + `%(end)`
	)

	tagsBytes, err := libExec.Exec(r.buildGitCommand(
		"for-each-ref",
		"--sort=-creatordate",
		"--format="+tagFormat,
		"refs/tags",
	))
	if err != nil {
		return nil, fmt.Errorf("error listing tags for repo %q: %w", r.url, err)
	}

	var tags []TagMetadata
	scanner := bufio.NewScanner(bytes.NewReader(tagsBytes))
	for scanner.Scan() {
		line := scanner.Bytes()
		parts := bytes.SplitN(scanner.Bytes(), []byte("|*|"), 6)
		if len(parts) != 6 {
			return nil, fmt.Errorf("unexpected number of fields: %q", line)
		}

		creatorDate, err := time.Parse("2006-01-02 15:04:05 -0700", string(parts[5]))
		if err != nil {
			return nil, fmt.Errorf("error parsing creator date %q: %w", parts[5], err)
		}

		tags = append(tags, TagMetadata{
			Tag:         string(parts[0]),
			CommitID:    string(parts[1]),
			Subject:     string(parts[2]),
			Author:      string(parts[3]),
			Committer:   string(parts[4]),
			CreatorDate: creatorDate,
		})
	}

	return tags, nil
}

func (r *repo) ListCommits(limit, skip uint) ([]CommitMetadata, error) {
	args := []string{
		"log",
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
	if limit > 0 {
		args = append(args, fmt.Sprintf("--max-count=%d", limit))
	}
	if skip > 0 {
		args = append(args, fmt.Sprintf("--skip=%d", skip))
	}

	commitsBytes, err := libExec.Exec(r.buildGitCommand(args...))
	if err != nil {
		return nil, fmt.Errorf("error listing commits for repo %q: %w", r.url, err)
	}

	var commits []CommitMetadata
	scanner := bufio.NewScanner(bytes.NewReader(commitsBytes))
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

	return commits, nil
}

func (r *repo) CommitMessage(id string) (string, error) {
	msgBytes, err := libExec.Exec(
		r.buildGitCommand("log", "-n", "1", "--pretty=format:%s", id),
	)
	if err != nil {
		return "", fmt.Errorf("error obtaining commit message for commit %q: %w", id, err)
	}
	return string(msgBytes), nil
}

func (r *repo) Push(force bool) error {
	args := []string{"push", "origin", r.currentBranch}
	if force {
		args = append(args, "--force")
	}
	if _, err := libExec.Exec(r.buildGitCommand(args...)); err != nil {
		return fmt.Errorf("error pushing branch %q: %w", r.currentBranch, err)
	}
	return nil
}

func (r *repo) RemoteBranchExists(branch string) (bool, error) {
	_, err := libExec.Exec(r.buildGitCommand(
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
	if _, err := libExec.Exec(r.buildGitCommand("reset", "--hard")); err != nil {
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

// setupClient configures the git CLI for authentication using either SSH or
// the "store" (username/password-based) credential helper.
func (r *repo) setupClient(opts *ClientOptions) error {
	if opts == nil {
		opts = &ClientOptions{}
	}

	if opts.User != nil {
		if err := r.setupAuthor(*opts.User); err != nil {
			return fmt.Errorf("error configuring the author: %w", err)
		}
	}

	if opts.Credentials != nil {
		if err := r.setupAuth(*opts.Credentials); err != nil {
			return fmt.Errorf("error configuring the credentials: %w", err)
		}
	}

	return nil
}

// setupAuthor configures the git CLI with a default commit author.
// Optionally, the author can have an associated signing key. When using GPG
// signing, the name and email must match the GPG key identity.
func (r *repo) setupAuthor(author User) error {
	if author.Name == "" {
		author.Name = "Kargo Render"
	}

	cmd := r.buildGitCommand("config", "--global", "user.name", author.Name)
	cmd.Dir = r.homeDir // Override the cmd.Dir that's set by r.buildGitCommand()
	if _, err := libExec.Exec(cmd); err != nil {
		return fmt.Errorf("error configuring git user name: %w", err)
	}

	if author.Email == "" {
		author.Name = "kargo-render@akuity.io"
	}

	cmd = r.buildGitCommand("config", "--global", "user.email", author.Email)
	cmd.Dir = r.homeDir // Override the cmd.Dir that's set by r.buildGitCommand()
	if _, err := libExec.Exec(cmd); err != nil {
		return fmt.Errorf("error configuring git user email: %w", err)
	}

	if author.SigningKeyPath != "" && author.SigningKeyType == SigningKeyTypeGPG {
		cmd = r.buildGitCommand("config", "--global", "commit.gpgsign", "true")
		cmd.Dir = r.homeDir // Override the cmd.Dir that's set by r.buildGitCommand()
		if _, err := libExec.Exec(cmd); err != nil {
			return fmt.Errorf("error configuring commit gpg signing: %w", err)
		}

		cmd = r.buildCommand("gpg", "--import", author.SigningKeyPath)
		cmd.Dir = r.homeDir // Override the cmd.Dir that's set by r.buildCommand()
		if _, err := libExec.Exec(cmd); err != nil {
			return fmt.Errorf("error importing gpg key %q: %w", author.SigningKeyPath, err)
		}
	}

	return nil
}

func (r *repo) setupAuth(creds RepoCredentials) error {
	// If an SSH key was provided, use that.
	if creds.SSHPrivateKey != "" {
		sshPath := filepath.Join(r.homeDir, ".ssh")
		if err := os.Mkdir(sshPath, 0700); err != nil {
			return fmt.Errorf("error creating SSH directory %q: %w", sshPath, err)
		}
		sshConfigPath := filepath.Join(sshPath, "config")
		rsaKeyPath := filepath.Join(sshPath, "id_rsa")
		// nolint: lll
		sshConfig := fmt.Sprintf("Host *\n  StrictHostKeyChecking no\n  UserKnownHostsFile=/dev/null\n  IdentityFile %q\n", rsaKeyPath)
		if err :=
			os.WriteFile(sshConfigPath, []byte(sshConfig), 0600); err != nil {
			return fmt.Errorf("error writing SSH config to %q: %w", sshConfigPath, err)
		}

		if err := os.WriteFile(
			rsaKeyPath,
			[]byte(creds.SSHPrivateKey),
			0600,
		); err != nil {
			return fmt.Errorf("error writing SSH key to %q: %w", rsaKeyPath, err)
		}
		return nil // We're done
	}

	// If we get to here, we're authenticating using a password

	// Set up the credential helper
	cmd := r.buildGitCommand("config", "--global", "credential.helper", "store")
	cmd.Dir = r.homeDir // Override the cmd.Dir that's set by r.buildGitCommand()
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
	if creds.Username == "" {
		creds.Username = "git"
	}
	// Augment the URL with user/pass information.
	credentialURL.User = url.UserPassword(creds.Username, creds.Password)
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

func (r *repo) buildCommand(command string, arg ...string) *exec.Cmd {
	cmd := exec.Command(command, arg...)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("HOME=%s", r.homeDir))
	cmd.Dir = r.dir
	return cmd
}

func (r *repo) buildGitCommand(arg ...string) *exec.Cmd {
	cmd := r.buildCommand("git", arg...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("GIT_SSH_COMMAND=ssh -F %s/.ssh/config", r.homeDir))
	if r.insecureSkipTLSVerify {
		cmd.Env = append(cmd.Env, "GIT_SSL_NO_VERIFY=true")
	}
	return cmd
}
