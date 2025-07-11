package git

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	libExec "github.com/akuity/kargo/internal/exec"
	"github.com/akuity/kargo/internal/logging"
)

const (
	defaultUsername = "Kargo"
	defaultEmail    = "no-reply@kargo.io"
)

// baseRepo implements the common underpinnings of a Git repository with a
// single working tree, a bare repository, or working tree associated with a
// bare repository.
type baseRepo struct {
	creds   *RepoCredentials
	dir     string
	homeDir string
	url     string
}

// ClientOptions represents options for a repository-specific Git client.
type ClientOptions struct {
	// User represents the actor that performs operations against the git
	// repository. This has no effect on authentication, see Credentials for
	// specifying authentication configuration.
	User *User
	// Credentials represents the authentication information.
	Credentials *RepoCredentials
	// InsecureSkipTLSVerify indicates whether to ignore certificate verification
	// errors when interacting with the remote repository.
	InsecureSkipTLSVerify bool
}

// setupClient sets up "global" git configuration with author and authentication
// details in the specified virtual home directory.
func (b *baseRepo) setupClient(homeDir string, opts *ClientOptions) error {
	if opts == nil {
		opts = &ClientOptions{}
	}

	if err := b.setupAuthor(homeDir, opts.User); err != nil {
		return fmt.Errorf("error configuring the author: %w", err)
	}

	if err := b.setupAuth(homeDir); err != nil {
		return fmt.Errorf("error configuring the credentials: %w", err)
	}

	if opts.InsecureSkipTLSVerify {
		cmd := b.buildGitCommand("config", "--global", "http.sslVerify", "false")
		// Override the home directory set by b.buildGitCommand().
		b.setCmdHome(cmd, homeDir)
		// Override the cmd.Dir that's set by b.buildGitCommand(). It's normally the
		// repository's path, but if this method was called as part of the cloning
		// process, that path may not exist yet.
		cmd.Dir = homeDir
		if _, err := libExec.Exec(cmd); err != nil {
			return fmt.Errorf("error configuring http.sslVerify: %w", err)
		}
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
	// SigningKey is an optional string containing the raw signing key content.
	// If provided, it takes precedence over SigningKeyPath.
	SigningKey string
	// SigningKeyPath is an optional path referencing a signing key for
	// signing git objects. Ignored if SigningKey is provided.
	SigningKeyPath string
}

// setupAuthor configures the git CLI with a default commit author.
// Optionally, the author can have an associated signing key. When using GPG
// signing, the name and email must match the GPG key identity. The directory
// specified by homeDir is used as a virtual home directory for all commands
// executed by this method.
func (b *baseRepo) setupAuthor(homeDir string, author *User) error {
	if author == nil {
		author = &User{}
	}

	if author.Name == "" {
		author.Name = defaultUsername
	}

	cmd := b.buildGitCommand("config", "--global", "user.name", author.Name)
	// Override the home directory set by b.buildGitCommand().
	b.setCmdHome(cmd, homeDir)
	// Override the cmd.Dir that's set by b.buildGitCommand(). It's normally the
	// repository's path, but if this method was called as part of the cloning
	// process, that path may not exist yet.
	cmd.Dir = homeDir
	if _, err := libExec.Exec(cmd); err != nil {
		return fmt.Errorf("error configuring git user name: %w", err)
	}

	if author.Email == "" {
		author.Email = defaultEmail
	}

	cmd = b.buildGitCommand("config", "--global", "user.email", author.Email)
	// Override the home directory set by b.buildGitCommand().
	b.setCmdHome(cmd, homeDir)
	// Override the cmd.Dir that's set by b.buildGitCommand(). It's normally the
	// repository's path, but if this method was called as part of the cloning
	// process, that path may not exist yet.
	cmd.Dir = homeDir
	if _, err := libExec.Exec(cmd); err != nil {
		return fmt.Errorf("error configuring git user email: %w", err)
	}

	// For now, since only GPG signing is supported, we will assume GPG if the
	// signing key type is not specified.
	if author.SigningKeyType == SigningKeyTypeGPG || author.SigningKeyType == "" {

		if author.SigningKey != "" {
			author.SigningKeyPath = filepath.Join(homeDir, "signing-key.asc")
			if err := os.WriteFile(
				author.SigningKeyPath,
				[]byte(author.SigningKey),
				0600,
			); err != nil {
				return fmt.Errorf("error writing signing key to %q: %w", author.SigningKeyPath, err)
			}
			defer func() {
				if err := os.Remove(author.SigningKeyPath); err != nil {
					logging.LoggerFromContext(context.TODO()).Error(
						err,
						"error removing file",
						"file", author.SigningKeyPath,
					)
				}
			}()
		}

		if author.SigningKeyPath != "" {
			cmd = b.buildGitCommand("config", "--global", "commit.gpgsign", "true")
			// Override the home directory set by b.buildGitCommand().
			b.setCmdHome(cmd, homeDir)
			// Override the cmd.Dir that's set by b.buildGitCommand(). It's normally the
			// repository's path, but if this method was called as part of the cloning
			// process, that path may not exist yet.
			cmd.Dir = homeDir
			if _, err := libExec.Exec(cmd); err != nil {
				return fmt.Errorf("error configuring commit gpg signing: %w", err)
			}

			cmd = b.buildCommand("gpg", "--import", author.SigningKeyPath)
			// Override the home directory set by b.buildCommand().
			b.setCmdHome(cmd, homeDir)
			// Override the cmd.Dir that's set by b.buildCommand(). It's normally the
			// repository's path, but if this method was called as part of the cloning
			// process, that path may not exist yet.
			cmd.Dir = homeDir
			if _, err := libExec.Exec(cmd); err != nil {
				return fmt.Errorf("error importing gpg key %q: %w", author.SigningKeyPath, err)
			}
		}

	}

	return nil
}

// setupAuth configures the git CLI with authentication information. The
// directory specified by homeDir is used as a virtual home directory for
// storing ssh keys if applicable.
func (b *baseRepo) setupAuth(homeDir string) error {
	if b.creds == nil {
		return nil
	}
	// If an SSH key was provided, use that.
	if b.creds.SSHPrivateKey != "" {
		sshPath := filepath.Join(homeDir, ".ssh")
		if err := os.MkdirAll(sshPath, 0700); err != nil {
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

// saveDirs saves information about the repository's directories to the
// repository's configuration. This is useful for reliably determining this
// information later if an existing repository or working tree is loaded from
// the file system.
func (b *baseRepo) saveDirs() error {
	if _, err := libExec.Exec(b.buildGitCommand(
		"config",
		"kargo.repoDir",
		b.dir,
	)); err != nil {
		return fmt.Errorf("error saving repo dir as config: %w", err)
	}
	if _, err := libExec.Exec(b.buildGitCommand(
		"config",
		"kargo.repoHomeDir",
		b.homeDir,
	)); err != nil {
		return fmt.Errorf("error saving repo home dir as config: %w", err)
	}
	return nil
}

// loadHomeDir restores the repository's home directory from the repository's
// configuration. This is useful for reliably determining this information when
// an existing repository or working tree is loaded from the file system.
func (b *baseRepo) loadHomeDir() error {
	res, err := libExec.Exec(b.buildGitCommand(
		"config",
		"kargo.repoHomeDir",
	))
	if err != nil {
		return fmt.Errorf("error reading repo home dir from config: %w", err)
	}
	b.homeDir = strings.TrimSpace(string(res))
	return nil
}

func (b *baseRepo) loadURL() error {
	res, err := libExec.Exec(b.buildGitCommand("config", "remote.origin.url"))
	if err != nil {
		return fmt.Errorf(`error getting URL of remote "origin": %w`, err)
	}
	b.url = strings.TrimSpace(string(res))
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
	if httpProxy := os.Getenv("http_proxy"); httpProxy != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("http_proxy=%s", httpProxy))
	}
	if httpsProxy := os.Getenv("https_proxy"); httpsProxy != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("https_proxy=%s", httpsProxy))
	}
	if noProxy := os.Getenv("no_proxy"); noProxy != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("no_proxy=%s", noProxy))
	}
	return cmd
}

func (b *baseRepo) Dir() string {
	return b.dir
}

func (b *baseRepo) HomeDir() string {
	return b.homeDir
}

func (b *baseRepo) RemoteBranchExists(branch string) (bool, error) {
	_, err := libExec.Exec(b.buildGitCommand(
		"ls-remote",
		"--heads",
		"--exit-code", // Return 2 if not found
		b.url,
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
			b.url,
			err,
		)
	}
	return true, nil
}

func (b *baseRepo) URL() string {
	return b.url
}

func (b *baseRepo) setCmdHome(cmd *exec.Cmd, homeDir string) {
	if cmd.Env == nil {
		cmd.Env = []string{fmt.Sprintf("HOME=%s", homeDir)}
	} else {
		cmd.Env = append(cmd.Env, fmt.Sprintf("HOME=%s", homeDir))
	}
}
