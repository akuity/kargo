package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	libExec "github.com/akuity/kargo/internal/exec"
)

// Repo is an interface for interacting with a Git repository with a single
// working tree.
type Repo interface {
	// Close cleans up file system resources used by this repository. This should
	// always be called before a repository goes out of scope.
	Close() error
	// Dir returns an absolute path to the repository.
	Dir() string
	// HomeDir returns an absolute path to the home directory of the system user
	// who has cloned this repo.
	HomeDir() string
	// URL returns the remote URL of the repository.
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
	// BaseDir is an existing directory within which all other directories created
	// and managed by the Repo implementation will be created. If not specified,
	// the operating system's temporary directory will be used. Overriding that
	// default is useful under certain circumstances.
	BaseDir string
	// Branch is the name of the branch to clone. If not specified, the default
	// branch will be cloned. This option is ignored if Bare is true.
	Branch string
	// Depth is the number of commits to fetch from the remote repository. If
	// zero, all commits will be fetched. This option is ignored if Bare is true.
	Depth uint
	// Filter allows for partially cloning the repository by specifying a
	// filter. When a filter is specified, the server will only send a
	// subset of reachable objects according to a given object filter.
	//
	// For more information, see:
	// - https://git-scm.com/docs/git-clone#Documentation/git-clone.txt-code--filtercodeemltfilter-specgtem
	// - https://git-scm.com/docs/git-rev-list#Documentation/git-rev-list.txt---filterltfilter-specgt
	// - https://github.blog/2020-12-21-get-up-to-speed-with-partial-clone-and-shallow-clone/
	// - https://docs.gitlab.com/ee/topics/git/partial_clone.html
	Filter string
	// InsecureSkipTLSVerify specifies whether certificate verification errors
	// should be ignored when cloning the repository. The setting will be
	// remembered for subsequent interactions with the remote repository.
	InsecureSkipTLSVerify bool
	// SingleBranch indicates whether the clone should be a single-branch clone.
	// This option is ignored if Bare is true.
	SingleBranch bool
}

// Clone produces a local clone of the remote git repository at the specified
// URL and returns an implementation of the Repo interface that is stateful and
// NOT suitable for use across multiple goroutines. This function will also
// perform any setup that is required for successfully authenticating to the
// remote repository.
func Clone(
	repoURL string,
	clientOpts *ClientOptions,
	cloneOpts *CloneOptions,
) (Repo, error) {
	if clientOpts == nil {
		clientOpts = &ClientOptions{}
	}
	if cloneOpts == nil {
		cloneOpts = &CloneOptions{}
	}
	homeDir, err := os.MkdirTemp(cloneOpts.BaseDir, "repo-")
	if err != nil {
		return nil,
			fmt.Errorf("error creating home directory for repo %q: %w", repoURL, err)
	}
	if homeDir, err = filepath.EvalSymlinks(homeDir); err != nil {
		return nil,
			fmt.Errorf("error resolving symlinks in path %s: %w", homeDir, err)
	}
	baseRepo := &baseRepo{
		creds:                 clientOpts.Credentials,
		dir:                   filepath.Join(homeDir, "repo"),
		homeDir:               homeDir,
		insecureSkipTLSVerify: cloneOpts.InsecureSkipTLSVerify,
		url:                   repoURL,
	}
	r := &repo{
		baseRepo: baseRepo,
		workTree: &workTree{
			baseRepo: baseRepo,
		},
	}
	if err = r.setupClient(clientOpts); err != nil {
		return nil, err
	}
	return r, r.clone(cloneOpts)
}

func (r *repo) clone(opts *CloneOptions) error {
	if opts == nil {
		opts = &CloneOptions{}
	}
	args := []string{"clone", "--no-tags"}
	if opts.Branch != "" {
		args = append(args, "--branch", opts.Branch)
	}
	if opts.SingleBranch {
		args = append(args, "--single-branch")
	}
	if opts.Depth > 0 {
		args = append(args, "--depth", fmt.Sprint(opts.Depth))
	}
	args = append(args, r.url, r.dir)
	cmd := r.buildGitCommand(args...)
	cmd.Dir = r.homeDir // Override the cmd.Dir that's set by r.buildGitCommand()
	if _, err := libExec.Exec(cmd); err != nil {
		return fmt.Errorf("error cloning repo %q into %q: %w", r.url, r.dir, err)
	}
	return nil
}

type LoadRepoOptions struct {
	Credentials           *RepoCredentials
	InsecureSkipTLSVerify bool
}

func LoadRepo(path string, opts *LoadRepoOptions) (Repo, error) {
	if opts == nil {
		opts = &LoadRepoOptions{}
	}
	baseRepo := &baseRepo{
		creds:                 opts.Credentials,
		dir:                   path,
		homeDir:               filepath.Dir(path),
		insecureSkipTLSVerify: opts.InsecureSkipTLSVerify,
	}
	r := &repo{
		baseRepo: baseRepo,
		workTree: &workTree{
			baseRepo: baseRepo,
		},
	}
	res, err := libExec.Exec(r.buildGitCommand("config", "--get", "remote.origin.url"))
	if err != nil {
		return nil, fmt.Errorf(`error getting URL of remote "origin": %w`, err)
	}
	r.url = strings.TrimSpace(string(res))
	if err = r.setupAuth(); err != nil {
		return nil, fmt.Errorf("error configuring the credentials: %w", err)
	}
	return r, nil
}

func (r *repo) Close() error {
	return os.RemoveAll(r.homeDir)
}
