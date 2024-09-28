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

func Test_gitCommitter_validate(t *testing.T) {
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
		{
			name:   "neither message nor messageFromSteps is specified",
			config: Config{},
			expectedProblems: []string{
				"(root): Must validate one and only one schema",
			},
		},
		{
			name: "both message and messageFromSteps are specified",
			config: Config{
				"message":          "fake commit message",
				"messageFromSteps": []string{"fake-step-alias"},
			},
			expectedProblems: []string{
				"(root): Must validate one and only one schema",
			},
		},
		{
			name: "message is empty string",
			config: Config{
				"message": "",
			},
			expectedProblems: []string{
				"message: String length must be greater than or equal to 1",
			},
		},
		{
			name: "messageFromSteps is empty array",
			config: Config{
				"messageFromSteps": []string{},
			},
			expectedProblems: []string{
				"messageFromSteps: Array must have at least 1 items",
			},
		},
		{
			name: "messageFromSteps array contains an empty string",
			config: Config{
				"messageFromSteps": []string{""},
			},
			expectedProblems: []string{
				"messageFromSteps.0: String length must be greater than or equal to 1",
			},
		},
		{
			name: "author is not specified",
			config: Config{ // Should be completely valid
				"path":    "/tmp/foo",
				"message": "fake commit message",
			},
		},
		{
			name: "author email is not specified",
			config: Config{ // Should be completely valid
				"author":  Config{},
				"path":    "/tmp/foo",
				"message": "fake commit message",
			},
		},
		{
			name: "author email is empty string",
			config: Config{ // Should be completely valid
				"author": Config{
					"email": "",
				},
				"path":    "/tmp/foo",
				"message": "fake commit message",
			},
		},
		{
			name: "author name is not specified",
			config: Config{ // Should be completely valid
				"author":  Config{},
				"path":    "/tmp/foo",
				"message": "fake commit message",
			},
		},
		{
			name: "author name is empty string",
			config: Config{ // Should be completely valid
				"author": Config{
					"name": "",
				},
				"path":    "/tmp/foo",
				"message": "fake commit message",
			},
		},
		{
			name: "valid kitchen sink",
			config: Config{
				"author": Config{
					"email": "tony@starkindustries.com",
					"name":  "Tony Stark",
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

func Test_gitCommitter_runPromotionStep(t *testing.T) {
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

	stepCtx := &PromotionStepContext{
		WorkDir: workDir,
	}

	res, err := runner.runPromotionStep(
		context.Background(),
		stepCtx,
		GitCommitConfig{
			Path:    "master",
			Message: "Initial commit",
		},
	)
	require.NoError(t, err)
	require.Equal(t, PromotionStatusSuccess, res.Status)
	expectedCommit, err := workTree.LastCommitID()
	require.NoError(t, err)
	actualCommit, ok := res.Output[commitKey]
	require.True(t, ok)
	require.Equal(t, expectedCommit, actualCommit)
	lastCommitMsg, err := workTree.CommitMessage("HEAD")
	require.NoError(t, err)
	require.Equal(t, "Initial commit", lastCommitMsg)
}

func Test_gitCommitter_buildCommitMessage(t *testing.T) {
	testCases := []struct {
		name        string
		sharedState State
		cfg         GitCommitConfig
		assertions  func(t *testing.T, msg string, err error)
	}{
		{
			name: "message is specified",
			cfg:  GitCommitConfig{Message: "fake commit message"},
			assertions: func(t *testing.T, msg string, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake commit message", msg)
			},
		},
		{
			name:        "no output from step with alias",
			sharedState: State{},
			cfg:         GitCommitConfig{MessageFromSteps: []string{"fake-step-alias"}},
			assertions: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "no output found from step with alias")
			},
		},
		{
			name: "unexpected value type from step with alias",
			sharedState: State{
				"fake-step-alias": "not a State",
			},
			cfg: GitCommitConfig{MessageFromSteps: []string{"fake-step-alias"}},
			assertions: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "output from step with alias")
				require.ErrorContains(t, err, "is not a map[string]any")
			},
		},
		{
			name: "output from step with alias does not contain a commit message",
			sharedState: State{
				"fake-step-alias": map[string]any{},
			},
			cfg: GitCommitConfig{MessageFromSteps: []string{"fake-step-alias"}},
			assertions: func(t *testing.T, _ string, err error) {
				require.ErrorContains(
					t, err, "no commit message found in output from step with alias",
				)
			},
		},
		{
			name: "output from step with alias contain a commit message that isn't a string",
			sharedState: State{
				"fake-step-alias": map[string]any{
					"commitMessage": 42,
				},
			},
			cfg: GitCommitConfig{MessageFromSteps: []string{"fake-step-alias"}},
			assertions: func(t *testing.T, _ string, err error) {
				require.ErrorContains(
					t, err, "commit message in output from step with alias",
				)
				require.ErrorContains(t, err, "is not a string")
			},
		},
		{
			name: "successful message construction",
			sharedState: State{
				"fake-step-alias": map[string]any{
					"commitMessage": "part one",
				},
				"another-fake-step-alias": map[string]any{
					"commitMessage": "part two",
				},
			},
			cfg: GitCommitConfig{
				MessageFromSteps: []string{
					"fake-step-alias",
					"another-fake-step-alias",
				},
			},
			assertions: func(t *testing.T, msg string, err error) {
				require.NoError(t, err)
				require.Contains(t, msg, "Kargo applied multiple changes")
				require.Contains(t, msg, "part one")
				require.Contains(t, msg, "part two")
			},
		},
	}

	r := newGitCommitter()
	runner, ok := r.(*gitCommitter)
	require.True(t, ok)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			commitMsg, err := runner.buildCommitMessage(
				testCase.sharedState,
				testCase.cfg,
			)
			testCase.assertions(t, commitMsg, err)
		})
	}
}
