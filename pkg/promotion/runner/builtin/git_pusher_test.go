package builtin

import (
	"fmt"
	"math"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/gitprovider"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_gitPusher_convert(t *testing.T) {
	tests := []validationTestCase{
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
			name: "generateTargetBranch, targetBranch, and tag not specified",
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
		{
			name: "force is true",
			config: promotion.Config{ // Should be completely valid
				"path":  "/fake/path",
				"force": true,
			},
		},
		{
			name: "force is false",
			config: promotion.Config{ // Should be completely valid
				"path":  "/fake/path",
				"force": false,
			},
		},
		{
			name: "tag and generateTargetBranch both specified",
			config: promotion.Config{
				"path":                 "/fake/path",
				"generateTargetBranch": true,
				"tag":                  "v1.0.0",
			},
			expectedProblems: []string{
				"(root): Must validate one and only one schema",
			},
		},
		{
			name: "tag and targetBranch both specified",
			config: promotion.Config{
				"path":         "/fake/path",
				"targetBranch": "fake-branch",
				"tag":          "v1.0.0",
			},
			expectedProblems: []string{
				"(root): Must validate one and only one schema",
			},
		},
	}

	r := newGitPusher(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*gitPushPusher)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_gitPusher_run(t *testing.T) {
	t.Run("push commit to generated branch", func(t *testing.T) {
		withGitPusherTestSuite(t, func(suite *gitPusherTestSuite) {
			t.Parallel()
			suite.addFileAndCommit(t, "test.txt", "foo", "Initial commit")
			result := suite.pushCommit(t)
			branchName, ok := result.Output[stateKeyBranch]
			require.True(t, ok)
			require.Equal(t, "kargo/promotion/fake-promotion", branchName)
			expectedCommit, err := suite.workingTree.LastCommitID()
			require.NoError(t, err)
			actualCommit, ok := result.Output[stateKeyCommit]
			require.True(t, ok)
			require.Equal(t, expectedCommit, actualCommit)
			expectedCommitURL := fmt.Sprintf("%s/commit/%s", suite.testRepoURL, expectedCommit)
			actualCommitURL := result.Output[stateKeyCommitURL]
			require.Equal(t, expectedCommitURL, actualCommitURL)
		})
	})
	t.Run("push tag", func(t *testing.T) {
		withGitPusherTestSuite(t, func(suite *gitPusherTestSuite) {
			t.Parallel()
			// working tree needs to have a commit to successfully create a tag
			suite.addFileAndCommit(t, "test.txt", "foo", "Initial commit")
			_ = suite.pushCommit(t)
			suite.createTag(t, "v1.0.0")
			result := suite.pushTag(t)
			actualTag, ok := result.Output[stateKeyTag]
			require.True(t, ok)
			require.Equal(t, "v1.0.0", actualTag)
		})
	})
}

type gitPusherTestSuite struct {
	testRepoURL         string
	fakeGitProviderName string
	runner              *gitPushPusher
	workDir             string
	workingTree         git.WorkTree
}

func (s *gitPusherTestSuite) addFileAndCommit(t *testing.T, filename, content, msg string) {
	t.Helper()
	require.NoError(t,
		os.WriteFile(
			filepath.Join(s.workingTree.Dir(), filename),
			[]byte(content), 0600,
		),
	)
	require.NoError(t,
		s.workingTree.AddAllAndCommit(msg, nil),
	)
}

func (s *gitPusherTestSuite) pushCommit(t *testing.T) promotion.StepResult {
	t.Helper()
	result, err := s.runner.run(
		t.Context(),
		&promotion.StepContext{
			Project:   "fake-project",
			Stage:     "fake-stage",
			Promotion: "fake-promotion",
			WorkDir:   s.workDir,
		},
		builtin.GitPushConfig{
			Path:                 "master",
			GenerateTargetBranch: true,
			Provider:             ptr.To(builtin.Provider(s.fakeGitProviderName)),
		},
	)
	require.NoError(t, err)
	return result
}

func (s *gitPusherTestSuite) createTag(t *testing.T, tag string) {
	t.Helper()
	require.NoError(t,
		s.workingTree.CreateTag(tag),
	)
}

func (s *gitPusherTestSuite) pushTag(t *testing.T) promotion.StepResult {
	t.Helper()
	result, err := s.runner.run(
		t.Context(),
		&promotion.StepContext{
			Project:   "fake-project",
			Stage:     "fake-stage",
			Promotion: "fake-promotion",
			WorkDir:   s.workDir,
		},
		builtin.GitPushConfig{
			Path: "master",
			Tag:  "v1.0.0",
		},
	)
	require.NoError(t, err)
	return result
}

type gitPusherTestFn func(suite *gitPusherTestSuite)

// withGitPusherTestSuite is a pre-test hook that sets up a unique:
// - test Git server
// - gitPushPusher instance
// - working directory
// - working tree with a local clone of the test Git server's repository
// - git provider
//
// it then collects those components for use in testFn.
// It is safe to run in parallel with other tests that also use withGitPusherTestSuite.
func withGitPusherTestSuite(t *testing.T, testFn gitPusherTestFn) {
	// server
	workDir := t.TempDir()
	service := gitkit.New(
		gitkit.Config{
			Dir:        workDir,
			AutoCreate: true,
		},
	)
	require.NoError(t, service.Setup())
	server := httptest.NewServer(service)
	defer server.Close()

	// pusher
	r := newGitPusher(promotion.StepRunnerCapabilities{
		CredsDB: &credentials.FakeDB{},
	})
	runner, ok := r.(*gitPushPusher)
	require.True(t, ok)
	require.NotNil(t, runner.branchMus)

	// working tree
	testRepoURL := fmt.Sprintf("%s/test.git", server.URL)
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

	// Set up a fake git provider
	// Cannot register multiple providers with the same name, so this takes
	// care of that problem
	uniqueFakeGitProviderName := uuid.NewString()
	gitprovider.Register(
		uniqueFakeGitProviderName,
		gitprovider.Registration{
			Predicate: func(repoURL string) bool {
				return repoURL == uniqueFakeGitProviderName
			},
			NewProvider: func(
				string,
				*gitprovider.Options,
			) (gitprovider.Interface, error) {
				return &gitprovider.Fake{
					GetCommitURLFn: func(
						repoURL string,
						sha string,
					) (string, error) {
						return fmt.Sprintf("%s/commit/%s", repoURL, sha), nil
					},
				}, nil
			},
		},
	)
	testFn(&gitPusherTestSuite{
		testRepoURL:         testRepoURL,
		fakeGitProviderName: uniqueFakeGitProviderName,
		runner:              runner,
		workDir:             workDir,
		workingTree:         workTree,
	})
}
