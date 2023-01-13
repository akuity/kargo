package git

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	libExec "github.com/akuityio/k8sta/internal/exec"
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
	Commit(message string) error
	// CreateChildBranch creates a new branch that is a child of the current
	// branch.
	CreateChildBranch(branch string) error
	// CreateOrphanedBranch creates a new branch that shares no commit history
	// with any other branch.
	CreateOrphanedBranch(branch string) error
	// HasDiffs returns a bool indicating whether the working directory currently
	// contains any differences from what's already at the head of the current
	// branch.
	HasDiffs() (bool, error)
	// LastCommitID returns the ID (sha) of the most recent commit to the current
	// branch.
	LastCommitID() (string, error)
	// CommitMessage returns the text of the most recent commit message associated
	// with the specified commit ID.
	CommitMessage(id string) (string, error)
	// CommitMessages returns a slice of commit messages starting with id1 and
	// ending with id2. The results exclude id1, but include id2.
	CommitMessages(id1, id2 string) ([]string, error)
	// Push pushes from the current branch to a remote branch by the same name.
	Push() error
	// RemoteBranchExists returns a bool indicating if the specified branch exists
	// in the remote repository.
	RemoteBranchExists(branch string) (bool, error)
	// URL returns the remote URL of the repository.
	URL() string
	// WorkingDir returns an absolute path to the repository's working tree.
	WorkingDir() string
}

// repo is an implementation of the Repo interface for interacting with a git
// repository.
type repo struct {
	url           string
	homeDir       string
	dir           string
	currentBranch string
}

// Clone produces a local clone of the remote git repository at the specified
// URL and returns an implementation of the Repo interface that is stateful and
// NOT suitable for use across multiple goroutines. This function will also
// perform any setup that is required for successfully authenticating to the
// remote repository.
func Clone(
	ctx context.Context,
	url string,
	repoCreds RepoCredentials,
) (Repo, error) {
	homeDir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error creating home directory for repo %q",
			url,
		)
	}
	r := &repo{
		url:     url,
		homeDir: homeDir,
		dir:     filepath.Join(homeDir, "repo"),
	}
	if err = r.setupAuth(ctx, repoCreds); err != nil {
		return nil, err
	}
	return r, r.clone()
}

func (r *repo) AddAll() error {
	_, err := libExec.Exec(r.buildCommand("git", "add", "."))
	return errors.Wrap(err, "error staging changes for commit")
}

func (r *repo) AddAllAndCommit(message string) error {
	if err := r.AddAll(); err != nil {
		return err
	}
	return r.Commit(message)
}

func (r *repo) Clean() error {
	_, err := libExec.Exec(r.buildCommand("git", "clean", "-fd"))
	return errors.Wrapf(err, "error cleaning branch %q", r.currentBranch)
}

func (r *repo) clone() error {
	r.currentBranch = "HEAD"
	cmd := r.buildCommand(
		"git",
		"clone",
		"--no-tags",
		r.url,
		r.dir,
	)
	cmd.Dir = r.homeDir // Override the cmd.Dir that's set by r.buildCommand()
	_, err := libExec.Exec(cmd)
	return errors.Wrapf(
		err,
		"error cloning repo %q into %q",
		r.url,
		r.dir,
	)
}

func (r *repo) Close() error {
	return os.RemoveAll(r.homeDir)
}

func (r *repo) Checkout(branch string) error {
	r.currentBranch = branch
	_, err := libExec.Exec(r.buildCommand(
		"git",
		"checkout",
		branch,
		// The next line makes it crystal clear to git that we're checking out
		// a branch. We need to do this because branch names can often resemble
		// paths within the repo.
		"--",
	))
	return errors.Wrapf(
		err,
		"error checking out branch %q from repo %q",
		branch,
		r.url,
	)
}

func (r *repo) Commit(message string) error {
	_, err := libExec.Exec(r.buildCommand("git", "commit", "-m", message))
	return errors.Wrapf(
		err,
		"error committing changes to branch %q",
		r.currentBranch,
	)
}

func (r *repo) CreateChildBranch(branch string) error {
	r.currentBranch = branch
	_, err := libExec.Exec(r.buildCommand(
		"git",
		"checkout",
		"-b",
		branch,
		// The next line makes it crystal clear to git that we're checking out
		// a branch. We need to do this because branch names can often resemble
		// paths within the repo.
		"--",
	))
	return errors.Wrapf(
		err,
		"error creating new branch %q for repo %q",
		branch,
		r.url,
	)
}

func (r *repo) CreateOrphanedBranch(branch string) error {
	r.currentBranch = branch
	if _, err := libExec.Exec(r.buildCommand(
		"git",
		"switch",
		"--orphan",
		branch,
		"--discard-changes",
	)); err != nil {
		return errors.Wrapf(
			err,
			"error creating orphaned branch %q for repo %q",
			branch,
			r.url,
		)
	}
	return r.Clean()
}

func (r *repo) HasDiffs() (bool, error) {
	resBytes, err := libExec.Exec(r.buildCommand("git", "status", "-s"))
	return len(resBytes) > 0,
		errors.Wrapf(err, "error checking status of branch %q", r.currentBranch)
}

func (r *repo) LastCommitID() (string, error) {
	shaBytes, err := libExec.Exec(r.buildCommand("git", "rev-parse", "HEAD"))
	return strings.TrimSpace(string(shaBytes)),
		errors.Wrap(err, "error obtaining ID of last commit")
}

func (r *repo) CommitMessage(id string) (string, error) {
	msgBytes, err := libExec.Exec(
		r.buildCommand("git", "log", "-n", "1", "--pretty=format:%s", id),
	)
	return string(msgBytes),
		errors.Wrapf(err, "error obtaining commit message for commit %q", id)
}

func (r *repo) CommitMessages(id1, id2 string) ([]string, error) {
	allMsgBytes, err := libExec.Exec(r.buildCommand(
		"git",
		"log",
		"--pretty=oneline",
		"--decorate-refs=",
		"--decorate-refs-exclude=",
		fmt.Sprintf("%s..%s", id1, id2),
	))
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error obtaining commit messages between commits %q and %q",
			id1,
			id2,
		)
	}
	msgsBytes := bytes.Split(allMsgBytes, []byte("\n"))
	msgs := []string{}
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

func (r *repo) Push() error {
	_, err :=
		libExec.Exec(r.buildCommand("git", "push", "origin", r.currentBranch))
	return errors.Wrapf(err, "error pushing branch %q", r.currentBranch)
}

func (r *repo) RemoteBranchExists(branch string) (bool, error) {
	_, err := libExec.Exec(r.buildCommand(
		"git",
		"ls-remote",
		"--heads",
		"--exit-code", // Return 2 if not found
		r.url,
		branch,
	))
	if exitErr, ok := err.(*libExec.ExitError); ok && exitErr.ExitCode == 2 {
		// Branch does not exist
		return false, nil
	}
	return err == nil, errors.Wrapf(
		err,
		"error checking for existence of branch %q in remote repo %q",
		branch,
		r.url,
	)
}

func (r *repo) URL() string {
	return r.url
}

func (r *repo) WorkingDir() string {
	return r.dir
}

// SetupAuth configures the git CLI for authentication using either SSH or the
// "store" (username/password-based) credential helper.
func (r *repo) setupAuth(ctx context.Context, repoCreds RepoCredentials) error {
	// Configure the git client
	cmd := r.buildCommand(
		"git",
		"config",
		"--global",
		"user.name",
		"K8sTA",
	)
	cmd.Dir = r.homeDir // Override the cmd.Dir that's set by r.buildCommand()
	if _, err := libExec.Exec(cmd); err != nil {
		return errors.Wrapf(err, "error configuring git username")
	}
	cmd = r.buildCommand(
		"git",
		"config",
		"--global",
		"user.email",
		"k8sta@akuity.io",
	)
	cmd.Dir = r.homeDir // Override the cmd.Dir that's set by r.buildCommand()
	if _, err := libExec.Exec(cmd); err != nil {
		return errors.Wrapf(err, "error configuring git user email address")
	}

	// If an SSH key was provided, use that.
	if repoCreds.SSHPrivateKey != "" {
		sshConfigPath := filepath.Join(r.homeDir, ".ssh", "config")
		// nolint: lll
		const sshConfig = "Host *\n  StrictHostKeyChecking no\n  UserKnownHostsFile=/dev/null"
		if err :=
			os.WriteFile(sshConfigPath, []byte(sshConfig), 0600); err != nil {
			return errors.Wrapf(err, "error writing SSH config to %q", sshConfigPath)
		}

		rsaKeyPath := filepath.Join(r.homeDir, ".ssh", "id_rsa")
		if err := os.WriteFile(
			rsaKeyPath,
			[]byte(repoCreds.SSHPrivateKey),
			0600,
		); err != nil {
			return errors.Wrapf(err, "error writing SSH key to %q", rsaKeyPath)
		}
		return nil // We're done
	}

	// If we get to here, we're authenticating using a password

	// Set up the credential helper
	cmd =
		r.buildCommand("git", "config", "--global", "credential.helper", "store")
	cmd.Dir = r.homeDir // Override the cmd.Dir that's set by r.buildCommand()
	if _, err := libExec.Exec(cmd); err != nil {
		return errors.Wrapf(err, "error configuring git credential helper")
	}

	credentialURL, err := url.Parse(r.url)
	if err != nil {
		return errors.Wrapf(err, "error parsing URL %q", r.url)
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
		return errors.Wrapf(
			err,
			"error writing credentials to %q",
			credentialsPath,
		)
	}
	return nil
}

func (r *repo) buildCommand(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	homeEnvVar := fmt.Sprintf("HOME=%s", r.homeDir)
	if cmd.Env == nil {
		cmd.Env = []string{homeEnvVar}
	} else {
		cmd.Env = append(cmd.Env, homeEnvVar)
	}
	cmd.Dir = r.dir
	return cmd
}
