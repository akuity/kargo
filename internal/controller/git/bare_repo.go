package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	libExec "github.com/akuity/kargo/internal/exec"
)

// BareRepo is an interface for interacting with a bare Git repository.
type BareRepo interface {
	// AddWorkTree adds a working tree to the repository. The working tree will be
	// created at the specified path and will be checked out to the specified ref.
	AddWorkTree(path, ref string) (WorkTree, error)
	// Close cleans up file system resources used by this repository. This should
	// always be called before a repository goes out of scope.
	Close() error
	// Dir returns an absolute path to the repository.
	Dir() string
	// HomeDir returns an absolute path to the home directory of the system user
	// who has cloned this repo.
	HomeDir() string
	// RemoveWorkTree removes a working tree from the repository. The working tree
	// will be removed from the file system.
	RemoveWorkTree(path string) error
	// URL returns the remote URL of the repository.
	URL() string
	// WorkTrees returns a list of working trees associated with the repository.
	WorkTrees() ([]WorkTree, error)
}

// bareRepo is an implementation of the BareRepo interface for interacting with
// a bare Git repository.
type bareRepo struct {
	*baseRepo
}

// BareCloneOptions represents options for cloning a Git repository without a
// working tree.
type BareCloneOptions struct {
	// BaseDir is an existing directory within which all other directories created
	// and managed by the BareRepo implementation will be created. If not
	// specified, the operating system's temporary directory will be used.
	// Overriding that default is useful under certain circumstances.
	BaseDir string
	// InsecureSkipTLSVerify specifies whether certificate verification errors
	// should be ignored when cloning the repository. The setting will be
	// remembered for subsequent interactions with the remote repository.
	InsecureSkipTLSVerify bool
}

// CloneBare produces a local, bare clone of the remote Git repository at the
// specified URL and returns an implementation of the BareRepo interface that is
// stateful and NOT suitable for use across multiple goroutines. This function
// will also perform any setup that is required for successfully authenticating
// to the remote repository.
func CloneBare(
	repoURL string,
	clientOpts *ClientOptions,
	cloneOpts *BareCloneOptions,
) (BareRepo, error) {
	if clientOpts == nil {
		clientOpts = &ClientOptions{}
	}
	if cloneOpts == nil {
		cloneOpts = &BareCloneOptions{}
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
	b := &bareRepo{
		baseRepo: &baseRepo{
			creds:                 clientOpts.Credentials,
			dir:                   filepath.Join(homeDir, "repo"),
			homeDir:               homeDir,
			insecureSkipTLSVerify: cloneOpts.InsecureSkipTLSVerify,
			url:                   repoURL,
		},
	}
	if err = b.setupClient(clientOpts); err != nil {
		return nil, err
	}
	return b, b.clone()
}

func (b *bareRepo) clone() error {
	cmd := b.buildGitCommand("clone", "--bare", b.url, b.dir)
	cmd.Dir = b.homeDir // Override the cmd.Dir that's set by r.buildGitCommand()
	if _, err := libExec.Exec(cmd); err != nil {
		return fmt.Errorf("error cloning repo %q into %q: %w", b.url, b.dir, err)
	}
	return nil
}

type LoadBareRepoOptions struct {
	Credentials           *RepoCredentials
	InsecureSkipTLSVerify bool
}

func LoadBareRepo(path string, opts *LoadBareRepoOptions) (BareRepo, error) {
	if opts == nil {
		opts = &LoadBareRepoOptions{}
	}
	b := &bareRepo{
		baseRepo: &baseRepo{
			creds:                 opts.Credentials,
			dir:                   path,
			homeDir:               filepath.Dir(path),
			insecureSkipTLSVerify: opts.InsecureSkipTLSVerify,
		},
	}
	res, err := libExec.Exec(b.buildGitCommand("config", "--get", "remote.origin.url"))
	if err != nil {
		return nil, fmt.Errorf(`error getting URL of remote "origin": %w`, err)
	}
	b.url = strings.TrimSpace(string(res))
	if err = b.setupAuth(); err != nil {
		return nil, fmt.Errorf("error configuring the credentials: %w", err)
	}
	return b, nil
}

func (b *bareRepo) AddWorkTree(path, ref string) (WorkTree, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("error resolving absolute path for %s: %w", path, err)
	}
	workTreePaths, err := b.workTrees()
	if err != nil {
		return nil, err
	}
	if slices.Contains(workTreePaths, path) {
		return nil, fmt.Errorf("working tree already exists at %q", path)
	}
	if _, err = libExec.Exec(
		b.buildGitCommand("worktree", "add", path, ref),
	); err != nil {
		return nil, fmt.Errorf("error adding working tree at %q: %w", path, err)
	}
	if path, err = filepath.EvalSymlinks(path); err != nil {
		return nil, fmt.Errorf("error resolving symlinks in path %s: %w", path, err)
	}
	return &workTree{
		baseRepo: &baseRepo{
			creds:                 b.creds,
			dir:                   path,
			homeDir:               b.homeDir,
			insecureSkipTLSVerify: b.insecureSkipTLSVerify,
			url:                   b.url,
		},
		bareRepo: b,
	}, nil
}

func (b *bareRepo) Close() error {
	workTreePaths, err := b.workTrees()
	if err != nil {
		return err
	}
	for _, workTreePath := range workTreePaths {
		if err := b.RemoveWorkTree(workTreePath); err != nil {
			return err
		}
	}
	return os.RemoveAll(b.homeDir)
}

func (b *bareRepo) RemoveWorkTree(path string) error {
	workTreePaths, err := b.workTrees()
	if err != nil {
		return err
	}
	if !slices.Contains(workTreePaths, path) {
		return fmt.Errorf("no working tree exists at %q", path)
	}
	if _, err := libExec.Exec(
		b.buildGitCommand("worktree", "remove", path),
	); err != nil {
		return fmt.Errorf("error removing working tree at %q: %w", path, err)
	}
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("error removing working tree at %q: %w", path, err)
	}
	return nil
}

func (b *bareRepo) WorkTrees() ([]WorkTree, error) {
	workTreePaths, err := b.workTrees()
	if err != nil {
		return nil, err
	}
	workTrees := make([]WorkTree, len(workTreePaths))
	for i, workTreePath := range workTreePaths {
		workTrees[i] = &workTree{
			baseRepo: &baseRepo{
				creds:                 b.creds,
				dir:                   workTreePath,
				homeDir:               b.homeDir,
				insecureSkipTLSVerify: b.insecureSkipTLSVerify,
				url:                   b.url,
			},
			bareRepo: b,
		}
	}
	return workTrees, err
}

func (b *bareRepo) workTrees() ([]string, error) {
	res, err := libExec.Exec(b.buildGitCommand("worktree", "list"))
	if err != nil {
		return nil, fmt.Errorf("error listing working trees: %w", err)
	}
	workTrees := []string{}
	scanner := bufio.NewScanner(bytes.NewReader(res))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasSuffix(line, "(bare)") {
			fields := strings.Fields(line)
			if len(fields) != 3 {
				return nil, fmt.Errorf("unexpected number of fields: %q", line)
			}
			workTrees = append(workTrees, fields[0])
		}
	}
	return workTrees, err
}
