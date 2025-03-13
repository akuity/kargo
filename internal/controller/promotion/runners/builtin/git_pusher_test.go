package builtin

import (
	"context"
	"fmt"
	"math"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/controller/promotion"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/pkg/x/promotion/runners/builtin"
)

func Test_gitPusher_validate(t *testing.T) {
	testCases := []struct {
		name             string
		config           promotion.Config
		expectedProblems []string
	}{
		{
			name:   "path not specified",
			config: promotion.Config{},
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
			name: "maxAttempts < 1",
			config: promotion.Config{
				"maxAttempts": 0,
			},
			expectedProblems: []string{
				"maxAttempts: Must be greater than or equal to 1",
			},
		},
		{
			name: fmt.Sprintf("maxAttempts > %d", math.MaxInt32),
			config: promotion.Config{
				"maxAttempts": math.MaxInt32 + 1,
			},
			expectedProblems: []string{
				fmt.Sprintf("maxAttempts: Must be less than or equal to %.9e", float64(math.MaxInt32)),
			},
		},
		{
			name: "just generateTargetBranch is true",
			config: promotion.Config{ // Should be completely valid
				"path":                 "/fake/path",
				"generateTargetBranch": true,
			},
		},
		{
			name: "generateTargetBranch is true and targetBranch is empty string",
			config: promotion.Config{ // Should be completely valid
				"path":                 "/fake/path",
				"generateTargetBranch": true,
				"targetBranch":         "",
			},
		},
		{
			name: "generateTargetBranch is true and targetBranch is specified",
			// These are meant to be mutually exclusive.
			config: promotion.Config{
				"path":                 "/fake/path",
				"generateTargetBranch": true,
				"targetBranch":         "fake-branch",
			},
			expectedProblems: []string{
				"(root): Must validate one and only one schema",
			},
		},
		{
			name: "generateTargetBranch not specified and targetBranch not specified",
			config: promotion.Config{ // Should be completely valid
				"path": "/fake/path",
			},
		},
		{
			name: "generateTargetBranch not specified and targetBranch is empty string",
			config: promotion.Config{ // Should be completely valid
				"path":         "/fake/path",
				"targetBranch": "",
			},
		},
		{
			name: "generateTargetBranch not specified and targetBranch is specified",
			config: promotion.Config{ // Should be completely valid
				"path":         "/fake/path",
				"targetBranch": "fake-branch",
			},
		},
		{
			name: "just generateTargetBranch is false",
			config: promotion.Config{ // Should be completely valid
				"path":                 "/fake/path",
				"generateTargetBranch": false,
			},
		},
		{
			name: "generateTargetBranch is false and targetBranch is empty string",
			config: promotion.Config{ // Should be completely valid
				"path":                 "/fake/path",
				"generateTargetBranch": false,
				"targetBranch":         "",
			},
		},
		{
			name: "generateTargetBranch is false and targetBranch is specified",
			config: promotion.Config{ // Should be completely valid
				"path":         "/fake/path",
				"targetBranch": "fake-branch",
			},
		},
	}

	r := newGitPusher(nil)
	runner, ok := r.(*gitPushPusher)
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

func Test_gitPusher_run(t *testing.T) {
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
	// `git worktree add` doesn't give much control over the branch name when you
	// create an orphaned working tree, so we have to follow up with this to make
	// the branch name look like what we wanted. gitCloner does this internally as
	// well.
	err = workTree.CreateOrphanedBranch("master")
	require.NoError(t, err)

	// Write a file.
	err = os.WriteFile(filepath.Join(workTree.Dir(), "test.txt"), []byte("foo"), 0600)
	require.NoError(t, err)

	// Commit the changes similarly to how gitCommitter would
	// have. It will be gitPushStepRunner's job to push this commit.
	err = workTree.AddAllAndCommit("Initial commit")
	require.NoError(t, err)

	// Now we can proceed to test gitPusher...

	r := newGitPusher(&credentials.FakeDB{})
	runner, ok := r.(*gitPushPusher)
	require.True(t, ok)
	require.NotNil(t, runner.branchMus)

	res, err := runner.run(
		context.Background(),
		&promotion.StepContext{
			Project:   "fake-project",
			Stage:     "fake-stage",
			Promotion: "fake-promotion",
			WorkDir:   workDir,
		},
		builtin.GitPushConfig{
			Path:                 "master",
			GenerateTargetBranch: true,
		},
	)
	require.NoError(t, err)
	branchName, ok := res.Output[stateKeyBranch]
	require.True(t, ok)
	require.Equal(t, "kargo/promotion/fake-promotion", branchName)
	expectedCommit, err := workTree.LastCommitID()
	require.NoError(t, err)
	actualCommit, ok := res.Output[stateKeyCommit]
	require.True(t, ok)
	require.Equal(t, expectedCommit, actualCommit)
}
