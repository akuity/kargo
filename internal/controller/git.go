package controller

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/db"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// setupGitAuth configures the git CLI for authentication using either SSH or
// the "store" (username/password-based) credential helper.
func setupGitAuth(
	ctx context.Context,
	repoURL string,
	homeDir string,
	argoDB db.ArgoDB,
	logger *log.Entry,
) error {
	// Configure the git client
	cmd := exec.Command("git", "config", "--global", "user.name", "k8sta")
	if _, err := execGitCommand(cmd, homeDir, logger); err != nil {
		return errors.Wrapf(err, "error configuring git username")
	}
	cmd = exec.Command(
		"git",
		"config",
		"--global",
		"user.email",
		"k8sta@akuity.io",
	)
	if _, err := execGitCommand(cmd, homeDir, logger); err != nil {
		return errors.Wrapf(err, "error configuring git user email address")
	}

	const repoTypeGit = "git"
	var sshKey, username, password string
	// NB: This next call returns an empty Repository if no such Repository is
	// found, so instead of continuing to look for credentials if no Repository is
	// found, what we'll do is continue looking for credentials if the Repository
	// we get back doesn't have anything we can use, i.e. no SSH private key or
	// password.
	repo, err := argoDB.GetRepository(ctx, repoURL)
	if err != nil {
		return errors.Wrapf(
			err,
			"error getting Repository (Secret) for repo %q",
			repoURL,
		)
	}
	if repo.Type == repoTypeGit || repo.Type == "" {
		sshKey = repo.SSHPrivateKey
		username = repo.Username
		password = repo.Password
	}
	if sshKey == "" && password == "" {
		// We didn't find any creds yet, so keep looking
		var repoCreds *argocd.RepoCreds
		repoCreds, err = argoDB.GetRepositoryCredentials(ctx, repoURL)
		if err != nil {
			return errors.Wrapf(
				err,
				"error getting Repository Credentials (Secret) for repo %q",
				repoURL,
			)
		}
		if repoCreds.Type == repoTypeGit || repoCreds.Type == "" {
			sshKey = repo.SSHPrivateKey
			username = repo.Username
			password = repo.Password
		}
	}

	// We didn't find any creds, so we're done. We can't promote without creds.
	if sshKey == "" && password == "" {
		return errors.Errorf("could not find any credentials for repo %q", repoURL)
	}

	// If an SSH key was provided, use that.
	if sshKey != "" {
		sshConfigPath := filepath.Join(homeDir, ".ssh", "config")
		// nolint: lll
		const sshConfig = "Host *\n  StrictHostKeyChecking no\n  UserKnownHostsFile=/dev/null"
		if err =
			ioutil.WriteFile(sshConfigPath, []byte(sshConfig), 0600); err != nil {
			return errors.Wrapf(err, "error writing SSH config to %q", sshConfigPath)
		}

		rsaKeyPath := filepath.Join(homeDir, ".ssh", "id_rsa")
		if err = ioutil.WriteFile(rsaKeyPath, []byte(sshKey), 0600); err != nil {
			return errors.Wrapf(err, "error writing SSH key to %q", rsaKeyPath)
		}
		return nil // We're done
	}

	// If we get to here, we're authenticating using a password

	// Set up the credential helper
	cmd = exec.Command("git", "config", "--global", "credential.helper", "store")
	if _, err = execGitCommand(cmd, homeDir, logger); err != nil {
		return errors.Wrapf(err, "error configuring git credential helper")
	}

	credentialURL, err := url.Parse(repoURL)
	if err != nil {
		return errors.Wrapf(err, "error parsing URL %q", repoURL)
	}
	// Remove path and query string components from the URL
	credentialURL.Path = ""
	credentialURL.RawQuery = ""
	// If the username is the empty string, we assume we're working with a git
	// provider like GitHub that only requires the username to be non-empty. We
	// arbitrarily set it to "git".
	if username == "" {
		username = "git"
	}
	// Augment the URL with user/pass information.
	credentialURL.User = url.UserPassword(username, password)
	// Write the augmented URL to the location used by the "stored" credential
	// helper.
	credentialsPath := filepath.Join(homeDir, ".git-credentials")
	if err := ioutil.WriteFile(
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

func cloneRepo(repoURL, homeDir string, logger *log.Entry) (string, error) {
	repoDir := filepath.Join(homeDir, "repo")
	cmd := exec.Command( // nolint: gosec
		"git",
		"clone",
		"--no-tags",
		repoURL,
		repoDir,
	)
	if _, err := execGitCommand(cmd, homeDir, logger); err != nil {
		return "", errors.Wrapf(
			err,
			"error cloning repo %q into %q",
			repoURL,
			repoDir,
		)
	}
	logger.WithFields(log.Fields{
		"repo": repoURL,
		"path": repoDir,
	}).Debug("cloned git repository")
	return repoDir, nil
}

func getLastCommitID(repoDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoDir // We need to be anywhere in the root of the repo for this
	shaBytes, err := cmd.Output()
	return strings.TrimSpace(string(shaBytes)),
		errors.Wrap(err, "error obtaining ID of last commit")
}

func execGitCommand(
	cmd *exec.Cmd,
	homeDir string,
	logger *log.Entry,
) ([]byte, error) {
	homeEnvVar := fmt.Sprintf("HOME=%s", homeDir)
	if cmd.Env == nil {
		cmd.Env = []string{homeEnvVar}
	} else {
		cmd.Env = append(cmd.Env, homeEnvVar)
	}
	output, err := cmd.CombinedOutput()
	logger.Debug(string(output))
	return output, err
}
