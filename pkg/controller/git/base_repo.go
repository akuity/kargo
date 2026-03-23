package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	libExec "github.com/akuity/kargo/pkg/exec"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	defaultUsername = "Kargo"
	defaultEmail    = "no-reply@kargo.io"

	repoDirConfigKey         = "kargo.repoDir"
	repoHomeDirConfigKey     = "kargo.repoHomeDir"
	repoOriginalURLConfigKey = "kargo.repoOriginalURL"
)

// baseRepo implements the common underpinnings of a Git repository with a
// single working tree, a bare repository, or working tree associated with a
// bare repository.
type baseRepo struct {
	creds   *RepoCredentials
	dir     string
	homeDir string
	// Store the URL two ways:
	// 1. Exactly as it was originally provided
	// 2. Modified in ways required for internal use by this type's methods
	originalURL string
	accessURL   string
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

func (b *baseRepo) setupDirs(baseDir string) error {
	var err error
	if b.homeDir, err = os.MkdirTemp(baseDir, "repo-"); err != nil {
		return fmt.Errorf(
			"error creating home directory for repo %q: %w",
			b.originalURL, err,
		)
	}
	if b.homeDir, err = filepath.EvalSymlinks(b.homeDir); err != nil {
		return fmt.Errorf("error resolving symlinks in path %s: %w", b.homeDir, err)
	}
	b.dir = filepath.Join(b.homeDir, "repo")
	cmd := b.buildGitCommand("config", "--global", "init.defaultBranch", "main")
	// Override the cmd.Dir that's set by b.buildGitCommand(). It's normally the
	// repository's path, but if this method was called as part of the cloning
	// process, that path may not exist yet.
	cmd.Dir = b.homeDir
	if _, err := libExec.Exec(cmd); err != nil {
		return fmt.Errorf("error configuring init.defaultBranch: %w", err)
	}
	return nil
}

// setupClient sets up "global" git configuration with author and authentication
// details in the specified virtual home directory.
func (b *baseRepo) setupClient(opts *ClientOptions) error {
	if opts == nil {
		opts = &ClientOptions{}
	}

	if _, err := b.setupAuthor(b.homeDir, opts.User); err != nil {
		return fmt.Errorf("error configuring the author: %w", err)
	}

	if err := b.setupAuth(b.homeDir); err != nil {
		return fmt.Errorf("error configuring the credentials: %w", err)
	}

	if opts.InsecureSkipTLSVerify {
		cmd := b.buildGitCommand("config", "--global", "http.sslVerify", "false")
		// Override the cmd.Dir that's set by b.buildGitCommand(). It's normally the
		// repository's path, but if this method was called as part of the cloning
		// process, that path may not exist yet.
		cmd.Dir = b.homeDir
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
	// gpgFingerprint is the fingerprint of the imported GPG signing key.
	// It is set by setupAuthor after a successful key import.
	gpgFingerprint string
}

// setupAuthor configures the git CLI with a default commit author.
// Optionally, the author can have an associated signing key. When using GPG
// signing, the name and email must match the GPG key identity. The directory
// specified by homeDir is used as a virtual home directory for all commands
// executed by this method. Returns the fully resolved User (with
// gpgFingerprint set when a signing key was imported).
func (b *baseRepo) setupAuthor(
	homeDir string,
	author *User,
) (*User, error) {
	if author == nil {
		author = &User{}
	}

	if author.Name == "" {
		author.Name = defaultUsername
	}

	cmd := b.buildGitCommand("config", "--global", "user.name", author.Name)
	// Override cmd.Dir set by buildGitCommand(). The repo path may not exist
	// yet if called during clone. homeDir is safe since we're only writing
	// "global" git config for a synthetic user.
	cmd.Dir = homeDir
	// Override HOME set by buildGitCommand(). The caller may provide a
	// different homeDir to set up ephemeral config for a per-commit author
	// identity.
	b.setCmdHome(cmd, homeDir)
	if _, err := libExec.Exec(cmd); err != nil {
		return nil, fmt.Errorf("error configuring git user name: %w", err)
	}

	if author.Email == "" {
		author.Email = defaultEmail
	}

	cmd = b.buildGitCommand("config", "--global", "user.email", author.Email)
	// See justification for both of these overrides above.
	cmd.Dir = homeDir
	b.setCmdHome(cmd, homeDir)
	if _, err := libExec.Exec(cmd); err != nil {
		return nil, fmt.Errorf("error configuring git user email: %w", err)
	}

	// For now, since only GPG signing is supported, we will assume GPG if the
	// signing key type is not specified.
	if author.SigningKeyType == SigningKeyTypeGPG || author.SigningKeyType == "" {
		if author.SigningKey != "" || author.SigningKeyPath != "" {
			cmd = b.buildGitCommand(
				"config", "--global", "commit.gpgSign", "true",
			)
			// See justification for both of these overrides above.
			cmd.Dir = homeDir
			b.setCmdHome(cmd, homeDir)
			if _, err := libExec.Exec(cmd); err != nil {
				return nil, fmt.Errorf(
					"error configuring commit gpg signing: %w", err,
				)
			}

			// Enable signing for tags as well.
			cmd = b.buildGitCommand(
				"config", "--global", "tag.gpgSign", "true",
			)
			cmd.Dir = homeDir
			b.setCmdHome(cmd, homeDir)
			if _, err := libExec.Exec(cmd); err != nil {
				return nil, fmt.Errorf(
					"error configuring tag gpg signing: %w", err,
				)
			}

			fingerprint, err := b.importGPGSigningKey(homeDir, author)
			if err != nil {
				return nil, err
			}
			author.gpgFingerprint = fingerprint
			return author, nil
		}
	}

	return author, nil
}

// importGPGSigningKey imports a GPG signing key into the keyring rooted at
// homeDir. If author.SigningKey (raw content) is set, it is written to a
// temporary file for import. Returns the key fingerprint.
func (b *baseRepo) importGPGSigningKey(
	homeDir string,
	author *User,
) (string, error) {
	keyPath := author.SigningKeyPath
	if author.SigningKey != "" {
		keyPath = filepath.Join(homeDir, "signing-key.asc")
		if err := os.WriteFile(
			keyPath,
			[]byte(author.SigningKey),
			0600,
		); err != nil {
			return "", fmt.Errorf(
				"error writing signing key to %q: %w", keyPath, err,
			)
		}
		defer func() {
			if err := os.Remove(keyPath); err != nil {
				logging.LoggerFromContext(context.TODO()).Error(
					err,
					"error removing file",
					"file", keyPath,
				)
			}
		}()
	}

	if keyPath == "" {
		return "", nil
	}

	cmd := b.buildCommand(
		"gpg", "--import",
		"--import-options", "import-show",
		"--with-colons",
		keyPath,
	)
	cmd.Dir = homeDir
	b.setCmdHome(cmd, homeDir)
	res, err := libExec.Exec(cmd)
	if err != nil {
		return "", fmt.Errorf(
			"error importing gpg key %q: %w", keyPath, err,
		)
	}
	return ExtractFingerprint(res), nil
}

// importGPGPublicKey exports a public key identified by fingerprint from
// srcHome's GPG keyring and imports it into dstHome's keyring.
func (b *baseRepo) importGPGPublicKey(
	srcHome, dstHome, fingerprint string,
) error {
	exportCmd := b.buildCommand(
		"gpg", "--export", "--armor", fingerprint,
	)
	exportCmd.Dir = srcHome
	b.setCmdHome(exportCmd, srcHome)
	pubKey, err := libExec.Exec(exportCmd)
	if err != nil {
		return fmt.Errorf(
			"error exporting public key %s: %w", fingerprint, err,
		)
	}
	if len(pubKey) == 0 {
		return fmt.Errorf(
			"no public key data exported for fingerprint %s", fingerprint,
		)
	}

	importCmd := b.buildCommand("gpg", "--import")
	importCmd.Dir = dstHome
	b.setCmdHome(importCmd, dstHome)
	importCmd.Stdin = bytes.NewReader(pubKey)
	if _, err = libExec.Exec(importCmd); err != nil {
		return fmt.Errorf(
			"error importing public key %s into keyring: %w",
			fingerprint, err,
		)
	}

	return nil
}

// fprRegex matches the first fingerprint line in GPG's --with-colons output.
// The format is: fpr:::::::::FINGERPRINT:
var fprRegex = regexp.MustCompile(`(?m)^fpr:{9}([A-F0-9]+):`)

// ExtractFingerprint parses the output of `gpg --with-colons` and returns the
// first key fingerprint found.
func ExtractFingerprint(output []byte) string {
	matches := fprRegex.FindSubmatch(output)
	if matches == nil {
		return ""
	}
	return string(matches[1])
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
		if err := os.WriteFile(sshConfigPath, []byte(sshConfig), 0600); err != nil {
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

	// Add username@ to the URL that will be used internally...
	lowerURL := strings.ToLower(b.accessURL)
	if strings.HasPrefix(lowerURL, "http://") || strings.HasPrefix(lowerURL, "https://") {
		u, err := url.Parse(b.accessURL)
		if err != nil {
			return fmt.Errorf("error parsing URL %q: %w", b.accessURL, err)
		}
		u.User = url.User(b.creds.Username)
		b.accessURL = u.String()
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
		repoDirConfigKey,
		b.dir,
	)); err != nil {
		return fmt.Errorf("error saving repo dir as config: %w", err)
	}
	if _, err := libExec.Exec(b.buildGitCommand(
		"config",
		repoHomeDirConfigKey,
		b.homeDir,
	)); err != nil {
		return fmt.Errorf("error saving repo home dir as config: %w", err)
	}
	return nil
}

// saveOriginalURL saves the original URL of the repository to the repository's
// configuration. This is useful for reliably determining this information when
// an existing repository or working tree is loaded from the file system.
func (b *baseRepo) saveOriginalURL() error {
	if _, err := libExec.Exec(b.buildGitCommand(
		"config",
		repoOriginalURLConfigKey,
		b.originalURL,
	)); err != nil {
		return fmt.Errorf("error saving original URL as config: %w", err)
	}
	return nil
}

// loadHomeDir restores the repository's home directory from the repository's
// configuration. This is useful for reliably determining this information when
// an existing repository or working tree is loaded from the file system.
func (b *baseRepo) loadHomeDir() error {
	res, err := libExec.Exec(b.buildGitCommand(
		"config",
		repoHomeDirConfigKey,
	))
	if err != nil {
		return fmt.Errorf("error reading repo home dir from config: %w", err)
	}
	b.homeDir = strings.TrimSpace(string(res))
	return nil
}

// loadURLs restores the repository's original and access URLs from the
// repository's configuration. This is useful for reliably determining this
// information when an existing repository or working tree is loaded from the
// file system.
func (b *baseRepo) loadURLs() error {
	res, err := libExec.Exec(b.buildGitCommand("config", repoOriginalURLConfigKey))
	if err != nil {
		return fmt.Errorf(`error getting original URL of remote "origin": %w`, err)
	}
	b.originalURL = strings.TrimSpace(string(res))
	if res, err = libExec.Exec(b.buildGitCommand(
		"config",
		"remote.origin.url",
	)); err != nil {
		return fmt.Errorf(`error getting URL of remote "origin": %w`, err)
	}
	b.accessURL = strings.TrimSpace(string(res))
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
		b.accessURL,
		"refs/heads/"+branch,
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
			b.originalURL,
			err,
		)
	}
	return true, nil
}

func (b *baseRepo) URL() string {
	return b.originalURL
}

func (b *baseRepo) setCmdHome(cmd *exec.Cmd, homeDir string) {
	if cmd.Env == nil {
		cmd.Env = []string{fmt.Sprintf("HOME=%s", homeDir)}
	} else {
		cmd.Env = append(cmd.Env, fmt.Sprintf("HOME=%s", homeDir))
	}
}
