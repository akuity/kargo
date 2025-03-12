package directives

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/pkg/x/directive/builtin"
)

func Test_gitTreeOverwriter_validate(t *testing.T) {
	testCases := []struct {
		name             string
		config           Config
		expectedProblems []string
	}{
		{
			name:   "path not specified",
			config: Config{},
			expectedProblems: []string{
				"(root): path is required",
			},
		},
		{
			name: "path is empty string",
			config: Config{
				"path": "",
			},
			expectedProblems: []string{
				"path: String length must be greater than or equal to 1",
			},
		},
	}

	p := newGitTreeClearer()
	promoter, ok := p.(*gitTreeClearer)
	require.True(t, ok)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := promoter.validate(testCase.config)
			if len(testCase.expectedProblems) == 0 {
				require.NoError(t, err)
			} else {
				for _, problem := range testCase.expectedProblems {
					require.ErrorContains(t, err, problem)
				}
			}
		})
	}
}

func Test_gitTreeOverwriter_promote(t *testing.T) {
	// Set up a test Git server in-process
	service := gitkit.New(
		gitkit.Config{
			Dir:        t.TempDir(),
			AutoCreate: true,
		},
	)
	require.NoError(t, service.Setup())
	server := httptest.NewServer(service)
	defer server.Close()

	// This is the URL of the "remote" repository
	testRepoURL := fmt.Sprintf("%s/test.git", server.URL)

	workDir := t.TempDir()

	// Finagle a local bare repo and working tree into place the way that
	// gitCloner might have so we can verify gitPusher's ability to reload the
	// working tree from the file system.
	repo, err := git.CloneBare(
		testRepoURL,
		nil,
		&git.BareCloneOptions{
			BaseDir: workDir,
		},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = repo.Close()
	})
	// "master" is still the default branch name for a new repository
	// unless you configure it otherwise.
	workTreePath := filepath.Join(workDir, "master")
	workTree, err := repo.AddWorkTree(
		workTreePath,
		&git.AddWorkTreeOptions{Orphan: true},
	)
	require.NoError(t, err)

	// Write a file. Later, we will expect to see this has been deleted.
	err = os.WriteFile(filepath.Join(workTree.Dir(), "original.txt"), []byte("foo"), 0600)
	require.NoError(t, err)

	promoter := &gitTreeClearer{}

	res, err := promoter.promote(
		context.Background(),
		&PromotionStepContext{
			Project: "fake-project",
			Stage:   "fake-stage",
			WorkDir: workDir,
		},
		builtin.GitClearConfig{
			Path: "master",
		},
	)
	require.NoError(t, err)
	require.Equal(t, kargoapi.PromotionPhaseSucceeded, res.Status)

	// Make sure old files are gone
	_, err = os.Stat(filepath.Join(workTree.Dir(), "original.txt"))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
	// Make sure the .git directory is still there
	_, err = os.Stat(filepath.Join(workTree.Dir(), ".git"))
	require.NoError(t, err)
}
