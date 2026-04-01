package builtin

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_gitTagger_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name: "path not specified",
			config: promotion.Config{
				"tag":     "v1.0.0",
				"message": "hello",
			},
			expectedProblems: []string{
				"(root): path is required",
			},
		},
		{
			name: "path is empty string",
			config: promotion.Config{
				"path":    "",
				"tag":     "v1.0.0",
				"message": "hello",
			},
			expectedProblems: []string{
				"path: String length must be greater than or equal to 1",
			},
		},
		{
			name: "tag not specified",
			config: promotion.Config{
				"path":    "/tmp/foo",
				"message": "hello",
			},
			expectedProblems: []string{
				"(root): tag is required",
			},
		},
		{
			name: "tag is empty string",
			config: promotion.Config{
				"path":    "/tmp/foo",
				"tag":     "",
				"message": "hello",
			},
			expectedProblems: []string{
				"tag: String length must be greater than or equal to 1",
			},
		},
		{
			name: "message not specified",
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
				"path":    "/tmp/foo",
				"tag":     "v1.0.0",
				"message": "",
			},
			expectedProblems: []string{
				"message: String length must be greater than or equal to 1",
			},
		},
		{
			name: "valid config",
			config: promotion.Config{
				"path":    "/tmp/foo",
				"tag":     "v1.0.0",
				"message": "hello",
			},
		},
	}

	r := newGitTagger(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*gitTagTagger)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_gitTagger_run(t *testing.T) {
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
	// gitCloner might have so we can verify gitTagger's ability to reload the
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

	// Write a file. It will be gitTagger's job to tag the current commit.
	err = os.WriteFile(
		filepath.Join(workTree.Dir(), "test.txt"),
		[]byte("foo"), 0600,
	)
	require.NoError(t, err)

	// Commit the file
	err = workTree.AddAll()
	require.NoError(t, err)
	err = workTree.Commit("Initial commit", nil)
	require.NoError(t, err)

	// Now we can proceed to test gitTagger...
	r := newGitTagger(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*gitTagTagger)
	require.True(t, ok)

	// Test creating a tag
	res, err := runner.run(
		t.Context(),
		&promotion.StepContext{WorkDir: workDir},
		builtin.GitTagConfig{
			Path: "master",
			Tag:  "v1.0.0",
		},
	)
	// Verify
	require.NoError(t, err)
	require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
	require.NoError(t, workTree.Checkout("v1.0.0"))
	expectedCommit, err := workTree.LastCommitID()
	require.NoError(t, err)
	actualCommit, ok := res.Output[stateKeyCommit]
	require.True(t, ok)
	require.Equal(t, expectedCommit, actualCommit)
}
