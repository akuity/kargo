package bookkeeper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/go-github/v47/github"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/akuityio/k8sta/internal/common/config"
	"github.com/akuityio/k8sta/internal/git"
	"github.com/akuityio/k8sta/internal/helm"
	"github.com/akuityio/k8sta/internal/kustomize"
	"github.com/akuityio/k8sta/internal/ytt"
)

// Service is an interface for components that can handle bookkeeping requests.
// Implementations of this interface are transport-agnostic.
type Service interface {
	// RenderConfig handles a bookkeeping request.
	RenderConfig(context.Context, RenderRequest) (Response, error)
}

type service struct {
	logger *log.Logger
}

// NewService returns an implementation of the Service interface for
// handling bookkeeping requests.
func NewService(config config.Config) Service {
	s := &service{
		logger: log.New(),
	}
	s.logger.SetLevel(config.LogLevel)
	return s
}

// nolint: gocyclo
func (s *service) RenderConfig(
	ctx context.Context,
	req RenderRequest,
) (Response, error) {
	logger := s.logger.WithFields(
		log.Fields{
			"repo":         req.RepoURL,
			"targetBranch": req.TargetBranch,
		},
	)

	res := Response{}

	repo, err := git.Clone(ctx, req.RepoURL, req.RepoCreds)
	if err != err {
		return res, errors.Wrap(err, "error cloning remote repository")
	}
	defer repo.Close()
	var sourceCommitID string
	if req.Commit != "" {
		if err = repo.Checkout(req.Commit); err != nil {
			return res, errors.Wrapf(err, "error checking out %q", req.Commit)
		}
		sourceCommitID = req.Commit
	} else if sourceCommitID, err = repo.LastCommitID(); err != nil {
		return res,
			errors.Wrap(err, "error getting last commit ID from the default branch")
	}

	// Pre-render
	preRenderedBytes, err := s.preRender(repo, req)
	if err != nil {
		return res, err
	}

	// Switch to the commit branch. The commit branch might be the target branch,
	// but if req.OpenPR is true, it could be a new child of the target branch.
	commitBranch, err := s.switchToCommitBranch(repo, req)
	if err != nil {
		return res, errors.Wrap(err, "error switching to target branch")
	}
	logger = logger.WithFields(log.Fields{
		"commitBranch": commitBranch,
	})

	// Ensure the .bookkeeper directory exists and is set up correctly
	bkDir := filepath.Join(repo.WorkingDir(), ".bookkeeper")
	if err = kustomize.EnsureBookkeeperDir(bkDir); err != nil {
		return res, errors.Wrapf(
			err,
			"error setting up .bookkeeper directory %q",
			bkDir,
		)
	}

	// Write the pre-rendered config to a temporary location
	preRenderedPath := filepath.Join(bkDir, "ephemeral.yaml")
	// nolint: gosec
	if err = os.WriteFile(preRenderedPath, preRenderedBytes, 0644); err != nil {
		return res, errors.Wrapf(
			err,
			"error writing ephemeral, pre-rendered configuration to %q",
			preRenderedPath,
		)
	}
	logger.Debugf("wrote pre-rendered configuration to %q", preRenderedPath)

	// Deal with new images if any were specified
	var commitMsg string
	if len(req.Images) == 0 {
		commitMsg = fmt.Sprintf(
			"bookkeeper: rendering configuration from %s",
			sourceCommitID,
		)
	} else {
		for _, image := range req.Images {
			if err = kustomize.SetImage(bkDir, image); err != nil {
				return res, errors.Wrapf(
					err,
					"error setting image in pre-render directory %q",
					bkDir,
				)
			}
		}
		if len(req.Images) == 1 {
			commitMsg = fmt.Sprintf(
				"bookkeeper: rendering configuration from %s with new image %s",
				sourceCommitID,
				req.Images[0],
			)
		} else {
			commitMsg = fmt.Sprintf(
				"bookkeeper: rendering configuration from %s with new images",
				sourceCommitID,
			)
			for _, image := range req.Images {
				commitMsg = fmt.Sprintf(
					"%s\n * %s",
					commitMsg,
					image,
				)
			}
		}
	}

	// Now take everything the last mile with kustomize and write the
	// fully-rendered config to the commit branch...

	// Last mile rendering
	fullyRenderedBytes, err := kustomize.Render(bkDir)
	if err != nil {
		return res, errors.Wrapf(
			err,
			"error rendering configuration from %q",
			bkDir,
		)
	}

	// Write the new fully-rendered config to the root of the repo
	allPath := filepath.Join(repo.WorkingDir(), "all.yaml")
	// nolint: gosec
	if err = os.WriteFile(allPath, fullyRenderedBytes, 0644); err != nil {
		return res, errors.Wrapf(
			err,
			"error writing fully-rendered configuration to %q",
			allPath,
		)
	}
	logger.Debug("wrote fully-rendered configuration")

	// Delete the ephemeral, pre-rendered configuration
	if err = os.Remove(preRenderedPath); err != nil {
		return res, errors.Wrapf(
			err,
			"error deleting ephemeral, pre-rendered configuration from %q",
			preRenderedPath,
		)
	}

	// Commit the fully-rendered configuration
	if err = repo.AddAllAndCommit(commitMsg); err != nil {
		return res, errors.Wrapf(
			err,
			"error committing fully-rendered configuration",
		)
	}
	logger.Debug("committed fully-rendered configuration")

	// Push the fully-rendered configuration to the remote commit branch
	if err = repo.Push(); err != nil {
		return res, errors.Wrap(
			err,
			"error pushing fully-rendered configuration",
		)
	}
	logger.Debug("pushed fully-rendered configuration")

	// Open a PR if requested
	//
	// TODO: Support git providers other than GitHub
	//
	// TODO: Move this into its own github package
	if req.OpenPR {
		var owner, repo string
		if owner, repo, err = parseGitHubURL(req.RepoURL); err != nil {
			return res, err
		}
		githubClient := github.NewClient(
			oauth2.NewClient(
				ctx,
				oauth2.StaticTokenSource(
					&oauth2.Token{AccessToken: req.RepoCreds.Password},
				),
			),
		)
		var pr *github.PullRequest
		if pr, _, err = githubClient.PullRequests.Create(
			ctx,
			owner,
			repo,
			&github.NewPullRequest{
				// PR title is just the first line of the commit message
				Title:               github.String(strings.Split(commitMsg, "\n")[0]),
				Base:                github.String(req.TargetBranch),
				Head:                github.String(commitBranch),
				MaintainerCanModify: github.Bool(false),
			},
		); err != nil {
			return res,
				errors.Wrap(err, "error opening pull request to the target branch")
		}
		res.PullRequestURL = *pr.HTMLURL
		return res, nil
	}

	// Get the ID of the last commit on the target branch
	if res.CommitID, err = repo.LastCommitID(); err != nil {
		return res, errors.Wrap(
			err,
			"error getting last commit ID from the target branch",
		)
	}
	logger.Debug("obtained sha of last commit")

	return res, nil
}

func (s *service) switchToCommitBranch(
	repo git.Repo,
	req RenderRequest,
) (string, error) {
	logger := s.logger.WithFields(
		log.Fields{
			"repo":         repo.URL(),
			"targetBranch": req.TargetBranch,
		},
	)

	var commitBranch = req.TargetBranch
	if req.OpenPR {
		commitBranch = fmt.Sprintf("bookkeeper/%s", uuid.NewV4().String())
	}

	// Check if the target branch exists on the remote
	if envBranchExists,
		err := repo.RemoteBranchExists(req.TargetBranch); err != nil {
		return commitBranch,
			errors.Wrap(err, "error checking for existence of target branch")
	} else if envBranchExists {
		logger.Debug("target branch exists on remote")
		if err = repo.Checkout(req.TargetBranch); err != nil {
			return commitBranch,
				errors.Wrap(err, "error checking out target branch")
		}
		logger.Debug("checked out target branch")
		// If we're supposed to be opening a PR instead of committing directly to
		// the target branch, we should create and check out a new child of the
		// target branch.
		if req.OpenPR {
			if err = repo.CreateChildBranch(commitBranch); err != nil {
				return commitBranch,
					errors.Wrap(err, "error creating child of target branch")
			}
			logger.Debug("created child of target branch")
		}
	} else {
		logger.Debug("target branch does not exist on remote")
		// If we're supposed to be opening a PR instead of committing directly to
		// the target branch, then we really cannot accommodate that scenario here
		// because you cannot open a PR against a branch that doesn't exist.
		if req.OpenPR {
			return commitBranch,
				errors.New("can not open a PR against a non-existing branch")
		}
		if err = repo.CreateOrphanedBranch(req.TargetBranch); err != nil {
			return commitBranch,
				errors.Wrap(err, "error creating orphaned target branch")
		}
		logger.Debug("created orphaned target branch")
	}
	if err := repo.Reset(); err != nil {
		return commitBranch, errors.Wrap(err, "error resetting repo")
	}
	return commitBranch, errors.Wrap(repo.Clean(), "error cleaning repo")
}

func (s *service) preRender(repo git.Repo, req RenderRequest) ([]byte, error) {
	baseDir := filepath.Join(repo.WorkingDir(), "base")
	envDir := filepath.Join(repo.WorkingDir(), req.TargetBranch)

	// Use the caller's preferred config management tool for pre-rendering.
	var preRenderedBytes []byte
	var err error
	if req.ConfigManagement.Helm != nil {
		preRenderedBytes, err = helm.Render(
			req.ConfigManagement.Helm.ReleaseName,
			baseDir,
			envDir,
		)
	} else if req.ConfigManagement.Kustomize != nil {
		preRenderedBytes, err = kustomize.Render(envDir)
	} else if req.ConfigManagement.Ytt != nil {
		preRenderedBytes, err = ytt.Render(baseDir, envDir)
	} else {
		return nil, errors.New(
			"no configuration management strategy was specified by the request",
		)
	}

	return preRenderedBytes, err
}

func parseGitHubURL(url string) (string, string, error) {
	regex := regexp.MustCompile(`^https\://github\.com/([\w-]+)/([\w-]+).*`)
	parts := regex.FindStringSubmatch(url)
	if len(parts) != 3 {
		return "", "", errors.Errorf("error parsing github repository URL %q", url)
	}
	return parts[1], parts[2], nil
}
