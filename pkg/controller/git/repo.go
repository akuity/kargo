package git

import (
	"context"
	"fmt"
	"os"
)

// Repo is an interface for interacting with a Git repository with a single
// working tree.
type Repo interface {
	// Close cleans up file system resources used by this repository. This should
	// always be called before a repository goes out of scope.
	Close(ctx context.Context) error
	// Dir returns an absolute path to the repository.
	Dir() string
	// HomeDir returns an absolute path to the home directory of the system user
	// who has cloned this repo.
	HomeDir() string
	// URL returns the remote URL of the repository as it was provided when the
	// repository was cloned, except that any credentials embedded in it are
	// removed when the repository was cloned with credentials of its own.
	URL() string
	WorkTree
}

// repo is an implementation of the Repo interface for interacting with a Git
// repository.
type repo struct {
	*baseRepo
	*workTree
}

// CloneOptions represents options for cloning a Git repository with a single
// working tree.
type CloneOptions struct {
	BaseCloneOptions
	// Checkout optionally specifies what to check out, in any of the forms that
	// Refs accepts. It is cloned whether or not Refs includes it. Checking out a
	// tag or commit leaves HEAD detached. If not specified, the remote's default
	// branch is checked out.
	Checkout string
}

// Clone produces a local clone of the remote git repository at the specified
// URL and returns an implementation of the Repo interface that is stateful and
// NOT suitable for use across multiple goroutines. This function will also
// perform any setup that is required for successfully authenticating to the
// remote repository.
func Clone(
	ctx context.Context,
	repoURL string,
	clientOpts *ClientOptions,
	cloneOpts *CloneOptions,
) (_ Repo, err error) {
	if err = validateRepoURL(repoURL); err != nil {
		return nil, err
	}
	if clientOpts == nil {
		clientOpts = &ClientOptions{}
	}
	if cloneOpts == nil {
		cloneOpts = &CloneOptions{}
	}
	baseRepo := &baseRepo{
		creds: clientOpts.Credentials,
		url:   repoURL,
	}
	r := &repo{
		baseRepo: baseRepo,
		workTree: &workTree{
			baseRepo: baseRepo,
		},
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(r.homeDir)
		}
	}()
	if err = r.setupDirs(ctx, cloneOpts.BaseDir); err != nil {
		return nil, err
	}
	if err = r.setupClient(ctx, clientOpts); err != nil {
		return nil, err
	}
	if err = r.clone(
		ctx,
		cloneOpts.BaseCloneOptions,
		cloneOpts.Checkout,
		false,
	); err != nil {
		return nil, err
	}
	if err = r.saveDirs(ctx); err != nil {
		return nil, err
	}
	return r, nil
}

type LoadRepoOptions struct {
	Credentials *RepoCredentials
}

func LoadRepo(
	ctx context.Context,
	path string,
	opts *LoadRepoOptions,
) (Repo, error) {
	if opts == nil {
		opts = &LoadRepoOptions{}
	}
	baseRepo := &baseRepo{
		creds: opts.Credentials,
		dir:   path,
	}
	r := &repo{
		baseRepo: baseRepo,
		workTree: &workTree{
			baseRepo: baseRepo,
		},
	}
	if err := r.loadHomeDir(ctx); err != nil {
		return nil, fmt.Errorf("error reading repo home dir from config: %w", err)
	}
	if err := r.loadURL(ctx); err != nil {
		return nil,
			fmt.Errorf(`error reading URL of remote "origin" from config: %w`, err)
	}
	if err := r.setupAuth(); err != nil {
		return nil, fmt.Errorf("error configuring the credentials: %w", err)
	}
	return r, nil
}

func (r *repo) Close(context.Context) error {
	return os.RemoveAll(r.homeDir)
}
