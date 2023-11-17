package warehouses

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
)

type gitMeta struct {
	Commit  string
	Message string
	Author  string
}

func (r *reconciler) getLatestCommits(
	ctx context.Context,
	namespace string,
	subs []kargoapi.RepoSubscription,
) ([]kargoapi.GitCommit, error) {
	latestCommits := make([]kargoapi.GitCommit, 0, len(subs))
	for _, s := range subs {
		if s.Git == nil {
			continue
		}
		sub := s.Git
		logger := logging.LoggerFromContext(ctx).WithField("repo", sub.RepoURL)
		creds, ok, err :=
			r.credentialsDB.Get(ctx, namespace, credentials.TypeGit, sub.RepoURL)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error obtaining credentials for git repo %q",
				sub.RepoURL,
			)
		}
		var repoCreds *git.RepoCredentials
		if ok {
			repoCreds = &git.RepoCredentials{
				Username:      creds.Username,
				Password:      creds.Password,
				SSHPrivateKey: creds.SSHPrivateKey,
			}
			logger.Debug("obtained credentials for git repo")
		} else {
			logger.Debug("found no credentials for git repo")
		}

		gm, err := r.getLatestCommitMetaFn(ctx, sub.RepoURL, sub.Branch, repoCreds)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error determining latest commit ID of git repo %q",
				sub.RepoURL,
			)
		}
		logger.WithField("commit", gm.Commit).
			Debug("found latest commit from repo")
		latestCommits = append(
			latestCommits,
			kargoapi.GitCommit{
				RepoURL: sub.RepoURL,
				ID:      gm.Commit,
				Branch:  sub.Branch,
				Message: gm.Message,
			},
		)
	}
	return latestCommits, nil
}

func getLatestCommitMeta(
	ctx context.Context,
	repoURL string,
	branch string,
	creds *git.RepoCredentials,
) (*gitMeta, error) {
	logger := logging.LoggerFromContext(ctx).WithField("repo", repoURL)
	if creds == nil {
		creds = &git.RepoCredentials{}
	}
	repo, err := git.Clone(
		repoURL,
		*creds,
		&git.CloneOptions{
			Branch:       branch,
			SingleBranch: true,
			Shallow:      true,
		},
	)
	if err != nil {
		return nil, errors.Wrapf(err, "error cloning git repo %q", repoURL)
	}
	var gm gitMeta
	gm.Commit, err = repo.LastCommitID()
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error determining last commit ID from git repo %q (branch: %q)",
			repoURL,
			branch,
		)
	}
	msg, err := repo.CommitMessage(gm.Commit)
	// Since we currently store commit messages in Stage status, we only capture
	// the first line of the commit message for brevity
	gm.Message = strings.Split(strings.TrimSpace(msg), "\n")[0]
	if err != nil {
		// This is best effort, so just log the error
		logger.Warnf("failed to get message from commit %q: %v", gm.Commit, err)
	}
	// TODO: support git author
	//gm.Author, err = repo.Author(gm.Commit)

	return &gm, nil
}
