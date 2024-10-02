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

	"github.com/akuity/kargo/internal/controller/git"
)

func Test_gitTreeOverwriter_validate(t *testing.T) {
	testCases := []struct {
		name             string
		config           Config
		expectedProblems []string
	}{
		{
			name:   "inPath not specified",
			config: Config{},
			expectedProblems: []string{
				"(root): inPath is required",
			},
		},
		{
			name: "inPath is empty string",
			config: Config{
				"inPath": "",
			},
			expectedProblems: []string{
				"inPath: String length must be greater than or equal to 1",
			},
		},
		{
			name:   "outPath not specified",
			config: Config{},
			expectedProblems: []string{
				"(root): outPath is required",
			},
		},
		{
			name: "outPath is empty string",
			config: Config{
				"outPath": "",
			},
			expectedProblems: []string{
				"outPath: String length must be greater than or equal to 1",
			},
		},
	}

	r := newGitTreeOverwriter()
	runner, ok := r.(*gitTreeOverwriter)
	require.True(t, ok)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := runner.validate(testCase.config)
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

func Test_gitTreeOverwriter_runPromotionStep(t *testing.T) {
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
	defer repo.Close()
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

	// Write another file to a different directory. This will be the source
	// directory for the gitTreeOverwriter.
	srcDir := filepath.Join(workDir, "src")
	err = os.Mkdir(srcDir, 0700)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(srcDir, "new.txt"), []byte("bar"), 0600)
	require.NoError(t, err)

	r := newGitTreeOverwriter()
	runner, ok := r.(*gitTreeOverwriter)
	require.True(t, ok)

	res, err := runner.runPromotionStep(
		context.Background(),
		&PromotionStepContext{
			Project: "fake-project",
			Stage:   "fake-stage",
			WorkDir: workDir,
		},
		GitOverwriteConfig{
			InPath:  "src",
			OutPath: "master",
		},
	)
	require.NoError(t, err)
	require.Equal(t, PromotionStatusSucceeded, res.Status)

	// Make sure old files are gone
	_, err = os.Stat(filepath.Join(workTree.Dir(), "original.txt"))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
	// Make sure new files are present
	_, err = os.Stat(filepath.Join(workTree.Dir(), "new.txt"))
	require.NoError(t, err)
	// Make sure the .git directory is still there
	_, err = os.Stat(filepath.Join(workTree.Dir(), ".git"))
	require.NoError(t, err)
}
