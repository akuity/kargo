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
	// AddWorkTree adds a working tree to the repository.
	AddWorkTree(path string, opts *AddWorkTreeOptions) (WorkTree, error)
	// Close cleans up file system resources used by this repository. This should
	// always be called before a repository goes out of scope.
	Close() error
	// Dir returns an absolute path to the repository.
	Dir() string
	// HomeDir returns an absolute path to the home directory of the system user
	// who has cloned this repo.
	HomeDir() string
	// RemoteBranchExists returns a bool indicating if the specified branch exists
	// in the remote repository.
	RemoteBranchExists(branch string) (bool, error)
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

// workTreeInfo represents information about a working tree.
type workTreeInfo struct {
	// Path is the absolute path to the working tree.
	Path string
	// HEAD is the commit ID of the HEAD of the working tree.
	HEAD string
	// Branch is the name of the branch that the working tree is on,
	// or an empty string if the working tree is in a detached HEAD state.
	Branch string
	// Bare is true if the working tree is a bare repository.
	Bare bool
	// Detached is true if the working tree is in a detached HEAD state.
	Detached bool
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
			creds:   clientOpts.Credentials,
			dir:     filepath.Join(homeDir, "repo"),
			homeDir: homeDir,
			url:     repoURL,
		},
	}
	if err = b.setupClient(clientOpts); err != nil {
		return nil, err
	}
	if err = b.clone(); err != nil {
		return nil, err
	}
	if err = b.saveDirs(); err != nil {
		return nil, err
	}
	return b, nil
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
	Credentials *RepoCredentials
}

func LoadBareRepo(path string, opts *LoadBareRepoOptions) (BareRepo, error) {
	if opts == nil {
		opts = &LoadBareRepoOptions{}
	}
	b := &bareRepo{
		baseRepo: &baseRepo{
			creds: opts.Credentials,
			dir:   path,
		},
	}
	if err := b.loadHomeDir(); err != nil {
		return nil, fmt.Errorf("error reading repo home dir from config: %w", err)
	}
	if err := b.loadURL(); err != nil {
		return nil,
			fmt.Errorf(`error reading URL of remote "origin" from config: %w`, err)
	}
	if err := b.setupAuth(); err != nil {
		return nil, fmt.Errorf("error configuring the credentials: %w", err)
	}
	return b, nil
}

// AddWorkTreeOptions represents options for adding a working tree to a bare
// repository.
type AddWorkTreeOptions struct {
	// Orphan specifies whether the working tree should be created from a new,
	// orphaned branch. If true, the Ref field will be ignored.
	Orphan bool
	// Ref specifies the branch or commit to check out in the working tree. Will
	// be ignored if Orphan is true.
	Ref string
}

func (b *bareRepo) AddWorkTree(path string, opts *AddWorkTreeOptions) (WorkTree, error) {
	if opts == nil {
		opts = &AddWorkTreeOptions{}
	}
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
	args := []string{"worktree", "add", path}
	if opts.Orphan {
		args = append(args, "--orphan")
	} else {
		args = append(args, opts.Ref)
	}
	if _, err = libExec.Exec(b.buildGitCommand(args...)); err != nil {
		return nil, fmt.Errorf("error adding working tree at %q: %w", path, err)
	}
	if path, err = filepath.EvalSymlinks(path); err != nil {
		return nil, fmt.Errorf("error resolving symlinks in path %s: %w", path, err)
	}
	return &workTree{
		baseRepo: &baseRepo{
			creds:   b.creds,
			dir:     path,
			homeDir: b.homeDir,
			url:     b.url,
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
				creds:   b.creds,
				dir:     workTreePath,
				homeDir: b.homeDir,
				url:     b.url,
			},
			bareRepo: b,
		}
	}
	return workTrees, err
}

func (b *bareRepo) workTrees() ([]string, error) {
	res, err := libExec.Exec(b.buildGitCommand("worktree", "list", "--porcelain"))
	if err != nil {
		return nil, fmt.Errorf("error listing working trees: %w", err)
	}
	trees, err := b.parseWorkTreeOutput(res)
	if err != nil {
		return nil, fmt.Errorf("error listing repository trees: %w", err)
	}
	return b.filterNonBarePaths(trees), nil
}

func (b *bareRepo) parseWorkTreeOutput(output []byte) ([]workTreeInfo, error) {
	var trees []workTreeInfo
	var current *workTreeInfo

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, " ", 2)

		key := parts[0]
		value := ""
		if len(parts) > 1 {
			value = parts[1]
		}

		switch key {
		case "worktree":
			if current != nil {
				trees = append(trees, *current)
			}
			current = &workTreeInfo{Path: value}
		case "HEAD":
			if current != nil {
				current.HEAD = value
			}
		case "branch":
			if current != nil {
				current.Branch = value
			}
		case "bare":
			if current != nil {
				current.Bare = true
			}
		case "detached":
			if current != nil {
				current.Detached = true
			}
		}
	}

	if current != nil {
		trees = append(trees, *current)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning worktree output: %w", err)
	}

	return trees, nil
}

func (b *bareRepo) filterNonBarePaths(trees []workTreeInfo) []string {
	var paths []string
	for _, info := range trees {
		if !info.Bare {
			paths = append(paths, info.Path)
		}
	}
	return paths
}
