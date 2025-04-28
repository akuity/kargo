package git

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/internal/types"
)

func TestBareRepo(t *testing.T) {
	testRepoCreds := RepoCredentials{
		Username: "fake-username",
		Password: "fake-password",
	}

	// This will be something to opt into because on some OSes, this will lead
	// to keychain-related prompts.
	var useAuth bool
	if useAuthStr := os.Getenv("TEST_GIT_CLIENT_WITH_AUTH"); useAuthStr != "" {
		useAuth = types.MustParseBool(useAuthStr)
	}
	service := gitkit.New(
		gitkit.Config{
			Dir:        t.TempDir(),
			AutoCreate: true,
			Auth:       useAuth,
		},
	)
	require.NoError(t, service.Setup())
	service.AuthFunc =
		func(cred gitkit.Credential, _ *gitkit.Request) (bool, error) {
			return cred.Username == testRepoCreds.Username &&
				cred.Password == testRepoCreds.Password, nil
		}
	server := httptest.NewServer(service)
	defer server.Close()

	testRepoURL := fmt.Sprintf("%s/test.git", server.URL)

	setupRep, err := Clone(
		testRepoURL,
		&ClientOptions{
			Credentials: &testRepoCreds,
		},
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, setupRep)
	defer setupRep.Close()
	err = os.WriteFile(fmt.Sprintf("%s/%s", setupRep.Dir(), "test.txt"), []byte("foo"), 0600)
	require.NoError(t, err)
	err = setupRep.AddAllAndCommit(fmt.Sprintf("initial commit %s", uuid.NewString()), nil)
	require.NoError(t, err)
	err = setupRep.Push(nil)
	require.NoError(t, err)
	err = setupRep.Close()
	require.NoError(t, err)

	rep, err := CloneBare(
		testRepoURL,
		&ClientOptions{
			Credentials: &testRepoCreds,
		},
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, rep)
	defer rep.Close()
	r, ok := rep.(*bareRepo)
	require.True(t, ok)

	t.Run("can clone", func(t *testing.T) {
		var repoURL *url.URL
		repoURL, err = url.Parse(r.url)
		require.NoError(t, err)
		repoURL.User = nil
		require.Equal(t, testRepoURL, repoURL.String())
		require.NotEmpty(t, r.homeDir)
		var fi os.FileInfo
		fi, err = os.Stat(r.homeDir)
		require.NoError(t, err)
		require.True(t, fi.IsDir())
		require.NotEmpty(t, r.dir)
		fi, err = os.Stat(r.dir)
		require.NoError(t, err)
		require.True(t, fi.IsDir())
	})

	t.Run("can get the repo url", func(t *testing.T) {
		require.Equal(t, r.url, r.URL())
	})

	t.Run("can get the home dir", func(t *testing.T) {
		require.Equal(t, r.homeDir, r.HomeDir())
	})

	t.Run("can get the working tree dir", func(t *testing.T) {
		require.Equal(t, r.dir, r.Dir())
	})

	workingTreePath := filepath.Join(rep.HomeDir(), "working-tree")
	workTree, err := rep.AddWorkTree(
		workingTreePath,
		// "master" is still the default branch name for a new repository unless
		// you configure it otherwise.
		&AddWorkTreeOptions{Ref: "master"},
	)

	require.NoError(t, err)
	defer workTree.Close()

	t.Run("add a working tree", func(t *testing.T) {
		require.NoError(t, err)
		require.Equal(t, workingTreePath, workTree.Dir())
		_, err = os.Stat(workTree.Dir())
		require.NoError(t, err)
	})

	t.Run("can list working trees", func(t *testing.T) {
		var workTrees []WorkTree
		workTrees, err = rep.WorkTrees()
		require.NoError(t, err)
		require.Len(t, workTrees, 1)
		require.Equal(t, workTree, workTrees[0])
	})

	t.Run("can remove a working tree", func(t *testing.T) {
		err = rep.RemoveWorkTree(workTree.Dir())
		require.NoError(t, err)
		trees, err := rep.WorkTrees()
		require.NoError(t, err)
		require.Len(t, trees, 0)
		_, err = os.Stat(workTree.Dir())
		require.True(t, os.IsNotExist(err))
	})

	t.Run("can load an existing repo", func(t *testing.T) {
		existingRepo, err := LoadBareRepo(
			rep.Dir(),
			&LoadBareRepoOptions{
				Credentials: &testRepoCreds,
			},
		)
		require.NoError(t, err)
		require.Equal(t, rep, existingRepo)
	})

	t.Run("can close repo", func(t *testing.T) {
		require.NoError(t, rep.Close())
		_, err := os.Stat(rep.HomeDir())
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
	})
}

func Test_bareRepo_parseWorkTreeOutput(t *testing.T) {
	tests := []struct {
		name       string
		input      []byte
		assertions func(*testing.T, []workTreeInfo, error)
	}{
		{
			name: "single worktree",
			input: []byte(`worktree /path/to/worktree
HEAD abcdef1234567890
branch main
`),
			assertions: func(t *testing.T, result []workTreeInfo, err error) {
				assert.NoError(t, err)
				assert.Len(t, result, 1)
				assert.Equal(t, result, []workTreeInfo{
					{Path: "/path/to/worktree", HEAD: "abcdef1234567890", Branch: "main"},
				})
			},
		},
		{
			name: "multiple worktrees",
			input: []byte(`worktree /path/to/worktree1
HEAD abcdef1234567890
branch main

worktree /path/to/worktree2
HEAD fedcba9876543210
branch feature
bare
detached
`),
			assertions: func(t *testing.T, result []workTreeInfo, err error) {
				assert.NoError(t, err)
				assert.Len(t, result, 2)
				assert.Equal(t, result, []workTreeInfo{
					{
						Path:   "/path/to/worktree1",
						HEAD:   "abcdef1234567890",
						Branch: "main",
					},
					{
						Path:     "/path/to/worktree2",
						HEAD:     "fedcba9876543210",
						Branch:   "feature",
						Bare:     true,
						Detached: true,
					},
				})
			},
		},
		{
			name:  "empty input",
			input: []byte(``),
			assertions: func(t *testing.T, result []workTreeInfo, err error) {
				assert.NoError(t, err)
				assert.Empty(t, result)
			},
		},
		{
			name: "incomplete worktree info",
			input: []byte(`worktree /path/to/incomplete
HEAD
branch

worktree /path/to/complete
HEAD abcdef1234567890
branch main
`),
			assertions: func(t *testing.T, result []workTreeInfo, err error) {
				assert.NoError(t, err)
				assert.Len(t, result, 2)
				assert.Equal(t, result, []workTreeInfo{
					{Path: "/path/to/incomplete"},
					{
						Path:   "/path/to/complete",
						HEAD:   "abcdef1234567890",
						Branch: "main",
					},
				})
			},
		},
		{
			name: "invalid input",
			input: []byte(`invalid input
not a worktree
`),
			assertions: func(t *testing.T, result []workTreeInfo, err error) {
				assert.NoError(t, err)
				assert.Empty(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bareRepo{}
			result, err := b.parseWorkTreeOutput(tt.input)
			tt.assertions(t, result, err)
		})
	}
}

func Test_bareRepo_filterNonBarePaths(t *testing.T) {
	tests := []struct {
		name       string
		input      []workTreeInfo
		assertions func(*testing.T, []string)
	}{
		{
			name: "mixed bare and non-bare worktrees",
			input: []workTreeInfo{
				{Path: "/path/to/worktree1", Bare: false},
				{Path: "/path/to/worktree2", Bare: true},
				{Path: "/path/to/worktree3", Bare: false},
			},
			assertions: func(t *testing.T, result []string) {
				assert.Len(t, result, 2)
				assert.Equal(t, result, []string{"/path/to/worktree1", "/path/to/worktree3"})
			},
		},
		{
			name: "all non-bare worktrees",
			input: []workTreeInfo{
				{Path: "/path/to/worktree1", Bare: false},
				{Path: "/path/to/worktree2", Bare: false},
			},
			assertions: func(t *testing.T, result []string) {
				assert.Len(t, result, 2)
				assert.Equal(t, result, []string{"/path/to/worktree1", "/path/to/worktree2"})
			},
		},
		{
			name: "all bare worktrees",
			input: []workTreeInfo{
				{Path: "/path/to/worktree1", Bare: true},
				{Path: "/path/to/worktree2", Bare: true},
			},
			assertions: func(t *testing.T, result []string) {
				assert.Empty(t, result)
			},
		},
		{
			name:  "empty input",
			input: []workTreeInfo{},
			assertions: func(t *testing.T, result []string) {
				assert.Empty(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bareRepo{}
			result := b.filterNonBarePaths(tt.input)
			tt.assertions(t, result)
		})
	}
}
