package builtin

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
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_gitCommitter_validate(t *testing.T) {
	testCases := []struct {
		name             string
		config           promotion.Config
		expectedProblems []string
	}{
		{
			name: "path not specified",
			config: promotion.Config{
				"message": "fake commit message",
			},
			expectedProblems: []string{
				"(root): path is required",
			},
		},
		{
			name: "path is empty string",
			config: promotion.Config{
				"path": "",
			},
			expectedProblems: []string{
				"path: String length must be greater than or equal to 1",
			},
		},
		{
			name: "message is not specified",
			config: promotion.Config{
				"path": "/tmp/foo",
			},
			expectedProblems: []string{
				"(root): message is required",
			},
		},
		{
			name: "message is empty string",
			config: promotion.Config{
				"message": "",
			},
			expectedProblems: []string{
				"message: String length must be greater than or equal to 1",
			},
		},
		{
			name: "author is not specified",
			config: promotion.Config{ // Should be completely valid
				"path":    "/tmp/foo",
				"message": "fake commit message",
			},
		},
		{
			name: "author email is not specified",
			config: promotion.Config{ // If author is specified, email must be specified
				"author":  promotion.Config{
					"name": "Tony Stark",
				},
				"path":    "/tmp/foo",
				"message": "fake commit message",
			},
			expectedProblems: []string{
				"invalid git-commit config: author: email is required",
			},
		},
		{
			name: "author email is empty string",
			config: promotion.Config{
				"author": promotion.Config{
					"name":  "Tony Stark",
					"email": "",
				},
				"path":    "/tmp/foo",
				"message": "fake commit message",
			},
			expectedProblems: []string{
				"invalid git-commit config: author.email: Does not match format 'email'",
			},
		},
		{
			name: "author name is not specified",
			config: promotion.Config{ // If author is specified, name must be specified
				"author":  promotion.Config{
					"email": "example@example.com",
				},
				"path":    "/tmp/foo",
				"message": "fake commit message",
			},
			expectedProblems: []string{
				"invalid git-commit config: author: name is required",
			},
		},
		{
			name: "author name is empty string",
			config: promotion.Config{
				"author": promotion.Config{
					"name": "",
					"email": "example@example.com",
				},
				"path":    "/tmp/foo",
				"message": "fake commit message",
			},
			expectedProblems: []string{
				"invalid git-commit config: author.name: String length must be greater than or equal to 1",
			},
		},
		{
			name: "author signingKey is empty string",
			config: promotion.Config{
				"author": promotion.Config{
					"name":       "Tony Stark",
					"email":      "tony@starkindustries.com",
					"signingKey": "",
				},
				"path":    "/tmp/foo",
				"message": "fake commit message",
			},
			// No expected problems because signingKey is optional
		},
		{
			name: "author signingKey is missing",
			config: promotion.Config{
				"author": promotion.Config{
					"name":  "Tony Stark",
					"email": "tony@starkindustries.com",
					// signingKey is absent
				},
				"path":    "/tmp/foo",
				"message": "fake commit message",
			},
			// No expected problems because signingKey is optional
		},
		{
			name: "valid kitchen sink",
			config: promotion.Config{
				"author": promotion.Config{
					"email": "tony@starkindustries.com",
					"name":  "Tony Stark",
					"signingKey": "valid-signing-key",
				},
				"path":    "/tmp/foo",
				"message": "fake commit message",
			},
		},
	}

	r := newGitCommitter()
	runner, ok := r.(*gitCommitter)
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

func Test_gitCommitter_run(t *testing.T) {
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

	// Finagle a local bare repo and working tree into place the way that the
	// gitCloner might have so we can verify gitCommitter's ability to reload the
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
	// `git worktree add` doesn't give much control over the branch name when you
	// create an orphaned working tree, so we have to follow up with this to make
	// the branch name look like what we wanted. gitCloner does this internally as
	// well.
	err = workTree.CreateOrphanedBranch("master")
	require.NoError(t, err)

	// Write a file. It will be gitCommitter's job to commit it.
	err = os.WriteFile(filepath.Join(workTree.Dir(), "test.txt"), []byte("foo"), 0600)
	require.NoError(t, err)

	// Now we can proceed to test gitCommitter...

	r := newGitCommitter()
	runner, ok := r.(*gitCommitter)
	require.True(t, ok)

	stepCtx := &promotion.StepContext{
		WorkDir: workDir,
	}

	res, err := runner.run(
		context.Background(),
		stepCtx,
		builtin.GitCommitConfig{
			Path:    "master",
			Message: "Initial commit",
		},
	)
	require.NoError(t, err)
	require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
	expectedCommit, err := workTree.LastCommitID()
	require.NoError(t, err)
	actualCommit, ok := res.Output[stateKeyCommit]
	require.True(t, ok)
	require.Equal(t, expectedCommit, actualCommit)
	lastCommitMsg, err := workTree.CommitMessage("HEAD")
	require.NoError(t, err)
	require.Equal(t, "Initial commit\n", lastCommitMsg)
}
