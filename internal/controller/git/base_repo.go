package git

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	libExec "github.com/akuity/kargo/internal/exec"
)

// baseRepo implements the common underpinnings of a Git repository with a
// single working tree, a bare repository, or working tree associated with a
// bare repository.
type baseRepo struct {
	creds                 *RepoCredentials
	dir                   string
	homeDir               string
	insecureSkipTLSVerify bool
	url                   string
}

// ClientOptions represents options for a repository-specific Git client.
type ClientOptions struct {
	// User represents the actor that performs operations against the git
	// repository. This has no effect on authentication, see Credentials for
	// specifying authentication configuration.
	User *User
	// Credentials represents the authentication information.
	Credentials *RepoCredentials
}

// setupClient configures the git CLI for authentication using either SSH or
// the "store" (username/password-based) credential helper.
func (b *baseRepo) setupClient(opts *ClientOptions) error {
	if opts == nil {
		opts = &ClientOptions{}
	}

	if err := b.setupAuthor(opts.User); err != nil {
		return fmt.Errorf("error configuring the author: %w", err)
	}

	if err := b.setupAuth(); err != nil {
		return fmt.Errorf("error configuring the credentials: %w", err)
	}

	return nil
}

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

// setupAuthor configures the git CLI with a default commit author.
// Optionally, the author can have an associated signing key. When using GPG
// signing, the name and email must match the GPG key identity.
func (b *baseRepo) setupAuthor(author *User) error {
	if author == nil {
		author = &User{}
	}

	if author.Name == "" {
		author.Name = "Kargo"
	}

	cmd := b.buildGitCommand("config", "--global", "user.name", author.Name)
	cmd.Dir = b.homeDir // Override the cmd.Dir that's set by r.buildGitCommand()
	if _, err := libExec.Exec(cmd); err != nil {
		return fmt.Errorf("error configuring git user name: %w", err)
	}

	if author.Email == "" {
		author.Name = "kargo@akuity.io"
	}

	cmd = b.buildGitCommand("config", "--global", "user.email", author.Email)
	cmd.Dir = b.homeDir // Override the cmd.Dir that's set by r.buildGitCommand()
	if _, err := libExec.Exec(cmd); err != nil {
		return fmt.Errorf("error configuring git user email: %w", err)
	}

	if author.SigningKeyPath != "" && author.SigningKeyType == SigningKeyTypeGPG {
		cmd = b.buildGitCommand("config", "--global", "commit.gpgsign", "true")
		cmd.Dir = b.homeDir // Override the cmd.Dir that's set by r.buildGitCommand()
		if _, err := libExec.Exec(cmd); err != nil {
			return fmt.Errorf("error configuring commit gpg signing: %w", err)
		}

		cmd = b.buildCommand("gpg", "--import", author.SigningKeyPath)
		cmd.Dir = b.homeDir // Override the cmd.Dir that's set by r.buildCommand()
		if _, err := libExec.Exec(cmd); err != nil {
			return fmt.Errorf("error importing gpg key %q: %w", author.SigningKeyPath, err)
		}
	}

	return nil
}

func (b *baseRepo) setupAuth() error {
	if b.creds == nil {
		return nil
	}
	// If an SSH key was provided, use that.
	if b.creds.SSHPrivateKey != "" {
		sshPath := filepath.Join(b.homeDir, ".ssh")
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
			[]byte(b.creds.SSHPrivateKey),
			0600,
		); err != nil {
			return fmt.Errorf("error writing SSH key to %q: %w", rsaKeyPath, err)
		}
		return nil // We're done
	}

	// If no password is specified, we're done'.
	if b.creds.Password == "" {
		return nil
	}

	lowerURL := strings.ToLower(b.url)
	if strings.HasPrefix(lowerURL, "http://") || strings.HasPrefix(lowerURL, "https://") {
		u, err := url.Parse(b.url)
		if err != nil {
			return fmt.Errorf("error parsing URL %q: %w", b.url, err)
		}
		u.User = url.User(b.creds.Username)
		b.url = u.String()
	}

	return nil
}

func (b *baseRepo) buildCommand(command string, arg ...string) *exec.Cmd {
	cmd := exec.Command(command, arg...)
	homeEnvVar := fmt.Sprintf("HOME=%s", b.homeDir)
	if cmd.Env == nil {
		cmd.Env = []string{homeEnvVar}
	} else {
		cmd.Env = append(cmd.Env, homeEnvVar)
	}
	cmd.Dir = b.dir
	return cmd
}

func (b *baseRepo) buildGitCommand(arg ...string) *exec.Cmd {
	cmd := b.buildCommand("git", arg...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("GIT_SSH_COMMAND=ssh -F %s/.ssh/config", b.homeDir))
	if b.creds != nil && b.creds.Password != "" {
		cmd.Env = append(
			cmd.Env,
			"GIT_ASKPASS=/usr/local/bin/credential-helper",
			fmt.Sprintf("GIT_PASSWORD=%s", b.creds.Password),
		)
	}
	if b.insecureSkipTLSVerify {
		cmd.Env = append(cmd.Env, "GIT_SSL_NO_VERIFY=true")
	}
	return cmd
}

func (b *baseRepo) Dir() string {
	return b.dir
}

func (b *baseRepo) HomeDir() string {
	return b.homeDir
}

func (b *baseRepo) URL() string {
	return b.url
}
