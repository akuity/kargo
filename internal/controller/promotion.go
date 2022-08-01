package controller

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// TODO: Break this up into smaller, more testable functions
func (t *ticketReconciler) promoteImage(
	ctx context.Context,
	imageRepo string,
	imageTag string,
	gitopsRepoURL string,
	envBranch string,
) (string, error) {
	defer tearDownGitAuth() // nolint: errcheck
	if err := t.setupGitAuth(ctx, gitopsRepoURL); err != nil {
		return "", errors.Wrapf(
			err,
			"error setting up authentication for repo %s",
			gitopsRepoURL,
		)
	}

	// Create a temporary workspace
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", errors.Wrapf(
			err,
			"error creating temporary workspace for cloning repo %s",
			gitopsRepoURL,
		)
	}
	defer os.RemoveAll(tempDir)
	t.logger.WithFields(logrus.Fields{
		"path": tempDir,
	}).Debug("created temporary workspace")

	repoDir := filepath.Join(tempDir, "repo")
	// We assume the environment-specific overlay path within the source branch ==
	// the name of the environment-specific branch that the final rendered YAML
	// will live in.
	envDir := filepath.Join(repoDir, envBranch)

	// Clone the repo
	cmd := exec.Command("git", "clone", "--no-tags", gitopsRepoURL, repoDir)
	if err = t.execCommand(cmd); err != nil {
		return "", errors.Wrapf(
			err,
			"error cloning repo %s into %s",
			gitopsRepoURL,
			repoDir,
		)
	}
	t.logger.WithFields(logrus.Fields{
		"path": repoDir,
		"repo": gitopsRepoURL,
	}).Debug("clone git repository")

	// Set the image
	cmd = exec.Command( // nolint: gosec
		"kustomize",
		"edit",
		"set",
		"image",
		fmt.Sprintf("%s=%s:%s", imageRepo, imageRepo, imageTag),
	)
	cmd.Dir = envDir // We need to be in the overlay directory to do this
	if err = t.execCommand(cmd); err != nil {
		return "", errors.Wrap(err, "error setting image")
	}
	t.logger.WithFields(logrus.Fields{
		"repo":      gitopsRepoURL,
		"envBranch": envBranch,
		"imageRepo": imageRepo,
		"imageTag":  imageTag,
	}).Debug("ran kustomize edit set image")

	// Render environment-specific YAML
	// TODO: We may need to buffer this or use a file instead
	cmd = exec.Command("kustomize", "build")
	cmd.Dir = envDir // We need to be in the overlay directory to do this
	yamlBytes, err := cmd.Output()
	if err != nil {
		return "",
			errors.Wrapf(err, "error rendering YAML for branch %s", envBranch)
	}
	t.logger.WithFields(logrus.Fields{
		"repo":      gitopsRepoURL,
		"envBranch": envBranch,
		"imageRepo": imageRepo,
		"imageTag":  imageTag,
	}).Debug("rendered environment-specific YAML")

	// Commit the changes to the source branch
	cmd = exec.Command( // nolint: gosec
		"git",
		"commit",
		"-am",
		fmt.Sprintf(
			"k8sta: updating %s to use image %s:%s",
			envBranch,
			imageRepo,
			imageTag,
		),
	)
	cmd.Dir = repoDir // We need to be in the root of the repo for this
	if err = t.execCommand(cmd); err != nil {
		return "", errors.Wrap(err, "error committing changes to source branch")
	}
	t.logger.WithFields(logrus.Fields{
		"repo":      gitopsRepoURL,
		"envBranch": envBranch,
		"imageRepo": imageRepo,
		"imageTag":  imageTag,
	}).Debug("committed changes to the source branch")

	// Push the changes to the source branch
	cmd = exec.Command("git", "push", "origin", "HEAD")
	cmd.Dir = repoDir // We need to be anywhere in the root of the repo for this
	if err = t.execCommand(cmd); err != nil {
		return "", errors.Wrap(err, "error pushing changes to source branch")
	}
	t.logger.WithFields(logrus.Fields{
		"repo":      gitopsRepoURL,
		"envBranch": envBranch,
		"imageRepo": imageRepo,
		"imageTag":  imageTag,
	}).Debug("pushed changes to the source branch")

	// Switch to the env-specific branch
	// TODO: Should we do something about the possibility that the branch doesn't
	// already exist, e.g. `git checkout --orphan <envBranch> --`
	cmd = exec.Command(
		"git",
		"checkout",
		envBranch,
		// The next line makes it crystal clear to git that we're checking out
		// a branch. We need to do this since we operate under an assumption that
		// the path to the overlay within the repo == the branch name.
		"--",
	)
	cmd.Dir = repoDir // We need to be anywhere in the root of the repo for this
	if err = t.execCommand(cmd); err != nil {
		return "", errors.Wrapf(
			err,
			"error checking out environment-specific branch %s from repo %s",
			envBranch,
			gitopsRepoURL,
		)
	}
	t.logger.WithFields(logrus.Fields{
		"repo":      gitopsRepoURL,
		"envBranch": envBranch,
		"imageRepo": imageRepo,
		"imageTag":  imageTag,
	}).Debug("checked out environment-specific branch")

	// Remove existing rendered YAML
	files, err := filepath.Glob(filepath.Join(repoDir, "*"))
	if err != nil {
		return "", errors.Wrapf(
			err,
			"error listing files in environment-specific branch %s",
			envBranch,
		)
	}
	for _, file := range files {
		if _, fileName := filepath.Split(file); fileName == ".git" {
			continue
		}
		if err = os.RemoveAll(file); err != nil {
			return "", errors.Wrapf(
				err,
				"error deleting file %s environment-specific branch %s",
				file,
				envBranch,
			)
		}
	}
	t.logger.WithFields(logrus.Fields{
		"repo":      gitopsRepoURL,
		"envBranch": envBranch,
		"imageRepo": imageRepo,
		"imageTag":  imageTag,
	}).Debug("removed existing rendered YAML")

	// Write the new rendered YAML
	if err = os.WriteFile( // nolint: gosec
		filepath.Join(repoDir, "all.yaml"),
		yamlBytes,
		0644,
	); err != nil {
		return "", errors.Wrapf(
			err,
			"error writing rendered YAML to environment-specific branch %s",
			envBranch,
		)
	}
	t.logger.WithFields(logrus.Fields{
		"repo":      gitopsRepoURL,
		"envBranch": envBranch,
		"imageRepo": imageRepo,
		"imageTag":  imageTag,
	}).Debug("wrote new rendered YAML")

	// Commit the changes to the environment-specific branch
	cmd = exec.Command( // nolint: gosec
		"git",
		"commit",
		"-am",
		fmt.Sprintf(
			"k8sta: use image %s:%s",
			imageRepo,
			imageTag,
		),
	)
	cmd.Dir = repoDir // We need to be in the root of the repo for this
	if err = t.execCommand(cmd); err != nil {
		return "", errors.Wrapf(
			err,
			"error committing changes to environment-specific branch %s",
			envBranch,
		)
	}
	t.logger.WithFields(logrus.Fields{
		"repo":      gitopsRepoURL,
		"envBranch": envBranch,
		"imageRepo": imageRepo,
		"imageTag":  imageTag,
	}).Debug("committed changes to environment-specific branch")

	// Push the changes to the environment-specific branch
	cmd = exec.Command("git", "push", "origin", envBranch)
	cmd.Dir = repoDir // We need to be anywhere in the root of the repo for this
	if err = t.execCommand(cmd); err != nil {
		return "", errors.Wrapf(
			err,
			"error pushing changes to environment-specific branch %s",
			envBranch,
		)
	}
	t.logger.WithFields(logrus.Fields{
		"repo":      gitopsRepoURL,
		"envBranch": envBranch,
		"imageRepo": imageRepo,
		"imageTag":  imageTag,
	}).Debug("pushed changes to environment-specific branch")

	// Get the ID of the last commit
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoDir // We need to be anywhere in the root of the repo for this
	shaBytes, err := cmd.Output()
	if err != nil {
		return "", errors.Wrapf(
			err,
			"error obtaining last commit ID for branch %s",
			envBranch,
		)
	}
	sha := strings.TrimSpace(string(shaBytes))
	t.logger.WithFields(logrus.Fields{
		"repo":      gitopsRepoURL,
		"envBranch": envBranch,
		"imageRepo": imageRepo,
		"imageTag":  imageTag,
		"sha":       sha,
	}).Debug("obtained sha of commit to environment-specific branch")

	return sha, nil
}

// setupGitAuth, if necessary, configures the git CLI for authentication using
// either SSH or the "store" (username/password-based) credential helper.
func (t *ticketReconciler) setupGitAuth(
	ctx context.Context,
	repoURL string,
) error {
	const repoTypeGit = "git"
	var sshKey, username, password string
	repo, err := t.argoDB.GetRepository(ctx, repoURL)
	if err != nil {
		return errors.Wrapf(
			err,
			"error getting Repository (Secret) for %s",
			repoURL,
		)
	}
	if repo.Type == repoTypeGit || repo.Type != "" {
		sshKey = repo.SSHPrivateKey
		username = repo.Username
		password = repo.Password
	} else {
		var repoCreds *argocd.RepoCreds
		if repoCreds, err =
			t.argoDB.GetRepositoryCredentials(ctx, repoURL); err != nil {
			return errors.Wrapf(
				err,
				"error getting Repository Credentials (Secret) for %s",
				repoURL,
			)
		}
		if repoCreds.Type == repoTypeGit || repoCreds.Type != "" {
			sshKey = repoCreds.SSHPrivateKey
			username = repoCreds.Username
			password = repoCreds.Password
		}
	}

	homeDir, err := homedir.Dir()
	if err != nil {
		return errors.Wrap(err, "error finding user's home directory")
	}

	// If an SSH key was provided, use that.
	if sshKey != "" {
		rsaKeyPath := filepath.Join(homeDir, ".ssh", "id_rsa")
		if err := ioutil.WriteFile(rsaKeyPath, []byte(sshKey), 0600); err != nil {
			return errors.Wrapf(err, "error writing SSH key to %q", rsaKeyPath)
		}
		return nil // We're done
	}

	// If a password was provided, use that.
	if password != "" {
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
		return nil // We're done
	}

	return errors.Errorf(
		"no authentication method was provided for repo %s",
		repoURL,
	)
}

func tearDownGitAuth() error {
	homeDir, err := homedir.Dir()
	if err != nil {
		return errors.Wrap(err, "error finding user's home directory")
	}
	rsaKeyPath := filepath.Join(homeDir, ".ssh", "id_rsa")
	if err = os.RemoveAll(rsaKeyPath); err != nil {
		return errors.Wrapf(err, "error deleting %s", rsaKeyPath)
	}
	credentialsPath := filepath.Join(homeDir, ".git-credentials")
	if err = os.RemoveAll(credentialsPath); err != nil {
		return errors.Wrapf(err, "error deleting %s", credentialsPath)
	}
	return nil
}

// TODO: This isn't adding much value. Let's can it.
func (t *ticketReconciler) execCommand(cmd *exec.Cmd) error {
	// stdoutReader, err := cmd.StdoutPipe()
	// if err != nil {
	// 	return errors.Wrap(err, "error obtaining stdout pipe")
	// }
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		// scanner := bufio.NewScanner(stdoutReader)
		// for scanner.Scan() {
		// 	t.logger.Debug(scanner.Text())
		// }
	}()
	// stderrReader, err := cmd.StderrPipe()
	// if err != nil {
	// 	return errors.Wrap(err, "error obtaining stderr pipe")
	// }
	wg.Add(1)
	go func() {
		defer wg.Done()
		// scanner := bufio.NewScanner(stderrReader)
		// for scanner.Scan() {
		// 	t.logger.Debug(scanner.Text())
		// }
	}()
	if err := cmd.Start(); err != nil {
		return errors.Wrapf(err, "error starting command %q", cmd.String())
	}
	if err := cmd.Wait(); err != nil {
		return errors.Wrapf(err, "error waiting for command %q", cmd.String())
	}
	wg.Wait() // Make sure we got all the output before returning
	return nil
}
