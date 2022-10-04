package bookkeeper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/akuityio/k8sta/internal/common/config"
	"github.com/akuityio/k8sta/internal/git"
	"github.com/akuityio/k8sta/internal/helm"
	"github.com/akuityio/k8sta/internal/kustomize"
	"github.com/akuityio/k8sta/internal/ytt"
)

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

func (s *service) Handle(ctx context.Context, req Request) (Response, error) {
	logger := s.logger.WithFields(
		log.Fields{
			"repo":         req.RepoURL,
			"path":         req.Path,
			"targetBranch": req.TargetBranch,
		},
	)

	res := Response{}

	repo, err := git.Clone(ctx, req.RepoURL, req.RepoCreds)
	if err != err {
		return res, errors.Wrap(err, "error cloning remote repository")
	}
	defer repo.Close()

	baseDir := filepath.Join(repo.WorkingDir(), "base")
	envDir := filepath.Join(repo.WorkingDir(), req.Path)

	// Use the caller's preferred config management tool for pre-rendering. Each
	// strategy needs different args, so we use closures here to provide a
	// pre-rendering function that takes no args and, instead, closes over
	// whatever other information it needs.
	var preRenderFn func() ([]byte, error)
	if req.ConfigManagement.Helm != nil {
		preRenderFn = func() ([]byte, error) {
			return helm.Render(
				req.ConfigManagement.Helm.ReleaseName,
				baseDir,
				envDir,
			)
		}
	} else if req.ConfigManagement.Kustomize != nil {
		preRenderFn = func() ([]byte, error) {
			return kustomize.Render(envDir)
		}
	} else if req.ConfigManagement.Ytt != nil {
		preRenderFn = func() ([]byte, error) {
			return ytt.Render(baseDir, envDir)
		}
	} else {
		return res, errors.New(
			"no configuration management strategy was specified by the request",
		)
	}

	// Ensure the existence of the directory into which we will pre-render
	// intermediate state
	bkEnvDir := filepath.Join(repo.WorkingDir(), ".bookkeeper", req.Path)
	if err = kustomize.EnsurePrerenderDir(bkEnvDir); err != nil {
		return res, errors.Wrapf(
			err,
			"error setting up pre-render directory %q",
			bkEnvDir,
		)
	}

	// This is the last commit on the default branch
	lastCommit, err := repo.LastCommitID()
	if err != nil {
		return res, errors.Wrap(
			err,
			"error obtaining ID of the last commit to the default branch",
		)
	}

	// Pre-render
	preRenderedBytes, err := preRenderFn()
	if err != nil {
		return res, errors.Wrapf(
			err,
			"error pre-rendering configuration from %q",
			envDir,
		)
	}
	logger.Debug("pre-rendered configuration")

	// Write/overwrite the pre-rendered config
	allPath := filepath.Join(bkEnvDir, "all.yaml")
	// nolint: gosec
	if err = os.WriteFile(allPath, preRenderedBytes, 0644); err != nil {
		return res, errors.Wrapf(
			err,
			"error writing pre-rendered configuration to %q in the default branch",
			allPath,
		)
	}
	logger.Debug("wrote pre-rendered configuration to the default branch")

	// Commit pre-rendered config to the local default branch
	if err = repo.AddAllAndCommit(
		fmt.Sprintf(
			"bookkeeper: pre-rendering configuration from %s",
			lastCommit,
		),
	); err != nil {
		return res, errors.Wrap(
			err,
			"error committing pre-rendered configuration to the default branch",
		)
	}
	logger.Debug("committed pre-rendered configuration to the default branch")

	// Push the pre-rendered configuration to the default branch
	if err = repo.Push(); err != nil {
		return res, errors.Wrap(
			err,
			"error pushing pre-rendered configuration to the default branch",
		)
	}
	logger.Debug("pushed pre-rendered configuration to the default branch")

	// Now take everything the last mile with kustomize and write the
	// fully-rendered config to a target branch...

	// This is the NEW last commit on the default branch
	if lastCommit, err = repo.LastCommitID(); err != nil {
		return res, errors.Wrap(
			err,
			"error obtaining ID of the last commit to the default branch",
		)
	}

	// Last mile rendering
	renderedBytes, err := kustomize.Render(bkEnvDir)
	if err != nil {
		return res, errors.Wrapf(
			err,
			"error rendering configuration from %q",
			bkEnvDir,
		)
	}

	// Switch to the target branch. This means checking out from a remote branch
	// if it exists or else creating a new orphaned branch.
	if err = s.switchToTargetBranch(repo, req.TargetBranch); err != nil {
		return res, errors.Wrap(err, "error switching to target branch")
	}

	// Remove existing fully-rendered config (or files from the default branch
	// that were left behind from the default branch when the orphaned target
	// branch was created)
	if err = s.deleteAll(repo); err != nil {
		return res, errors.Wrapf(
			err,
			"error deleting existing files from %q",
			repo.WorkingDir(),
		)
	}
	logger.Debug("removed existing fully-rendered configuration")

	// Write the new fully-rendered config to the root of the repo
	allPath = filepath.Join(repo.WorkingDir(), "all.yaml")
	// nolint: gosec
	if err = os.WriteFile(allPath, renderedBytes, 0644); err != nil {
		return res, errors.Wrapf(
			err,
			"error writing fully-rendered configuration to %q",
			allPath,
		)
	}
	logger.Debug("wrote fully-rendered configuration to the target branch")

	// Commit the fully-rendered configuration to the target branch
	if err = repo.AddAllAndCommit(
		fmt.Sprintf(
			"bookkeeper: rendering configuration from %s",
			lastCommit,
		),
	); err != nil {
		return res, errors.Wrapf(
			err,
			"error committing fully-rendered configuration to the target branch",
		)
	}
	logger.Debug("committed fully-rendered configuration to the target branch")

	// Push the fully-rendered configuration to the remote target branch
	if err = repo.Push(); err != nil {
		return res, errors.Wrap(
			err,
			"error pushing fully-rendered configuration to the target branch",
		)
	}
	logger.Debug("pushed fully-rendered configuration to the target branch")

	// Get the ID of the last commit on the target branch
	if res.CommitID, err = repo.LastCommitID(); err != nil {
		return res, err
	}
	logger.Debug("obtained sha of commit to the target branch")
	return res, nil
}

func (s *service) switchToTargetBranch(
	repo git.Repo,
	targetBranch string,
) error {
	logger := s.logger.WithFields(
		log.Fields{
			"repo":         repo.URL(),
			"targetBranch": targetBranch,
		},
	)
	// Check if the target branch exists on the remote
	if envBranchExists,
		err := repo.RemoteBranchExists(targetBranch); err != nil {
		return errors.Wrap(err, "error checking for existence of target branch")
	} else if envBranchExists {
		logger.Debug("target branch exists on remote")
		if err = repo.Checkout(targetBranch); err != nil {
			return errors.Wrap(err, "error checking out target branch")
		}
		logger.Debug("checked out target branch")
	} else {
		logger.Debug("target branch does not exist on remote")
		if err = repo.CreateOrphanedBranch(targetBranch); err != nil {
			return errors.Wrap(err, "error creating orphaned target branch")
		}
		logger.Debug("created orphaned target branch")
	}
	return nil
}

// deleteAll deletes everything from the working copy of the specified repo
// EXCEPT the .git directory.
func (s *service) deleteAll(repo git.Repo) error {
	files, err := filepath.Glob(filepath.Join(repo.WorkingDir(), "*"))
	if err != nil {
		return errors.Wrapf(err, "error listing files in %q", repo.WorkingDir())
	}
	for _, file := range files {
		if _, fileName := filepath.Split(file); fileName == ".git" {
			continue
		}
		if err = os.RemoveAll(file); err != nil {
			return errors.Wrapf(err, "error deleting %q", file)
		}
	}
	return nil
}
