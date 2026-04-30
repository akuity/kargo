package builtin

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/go-github/v76/github"
	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_githubPusher_convert(t *testing.T) {
	testCases := []validationTestCase{
		{
			name:             "path not specified",
			config:           promotion.Config{},
			expectedProblems: []string{"(root): path is required"},
		},
		{
			name:   "path is empty string",
			config: promotion.Config{"path": ""},
			expectedProblems: []string{
				"path: String length must be greater than or equal to 1",
			},
		},
		{
			name:   "maxAttempts < 1",
			config: promotion.Config{"maxAttempts": 0},
			expectedProblems: []string{
				"maxAttempts: Must be greater than or equal to 1",
			},
		},
		{
			name:   fmt.Sprintf("maxAttempts > %d", math.MaxInt32),
			config: promotion.Config{"maxAttempts": math.MaxInt32 + 1},
			expectedProblems: []string{
				fmt.Sprintf(
					"maxAttempts: Must be less than or equal to %.9e",
					float64(math.MaxInt32),
				),
			},
		},
		{
			name: "generateTargetBranch is true",
			config: promotion.Config{
				"path":                 "/fake/path",
				"generateTargetBranch": true,
			},
		},
		{
			name: "generateTargetBranch is true and targetBranch is specified",
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
			name:   "only path specified",
			config: promotion.Config{"path": "/fake/path"},
		},
		{
			name: "targetBranch specified",
			config: promotion.Config{
				"path":         "/fake/path",
				"targetBranch": "fake-branch",
			},
		},
		{
			name: "force is true",
			config: promotion.Config{
				"path":  "/fake/path",
				"force": true,
			},
		},
		{
			name: "insecureSkipTLSVerify is true",
			config: promotion.Config{
				"path":                  "/fake/path",
				"insecureSkipTLSVerify": true,
			},
		},
	}

	pusher := newGitHubPusher(
		promotion.StepRunnerCapabilities{},
		githubPusherConfig{},
	)
	runner, ok := pusher.(*githubPusher)
	require.True(t, ok)

	runValidationTests(t, runner.convert, testCases)
}

func Test_githubPusher_run(t *testing.T) {
	// Set up a test Git server in-process so that LoadWorkTree succeeds for
	// test cases that get past the initial path validation.
	service := gitkit.New(
		gitkit.Config{
			Dir:        t.TempDir(),
			AutoCreate: true,
		},
	)
	require.NoError(t, service.Setup())
	server := httptest.NewServer(service)
	defer server.Close()
	testRepoURL := fmt.Sprintf("%s/test.git", server.URL)

	// Set up a real working tree that LoadWorkTree can load.
	workDir := t.TempDir()
	repo, err := git.CloneBare(
		testRepoURL,
		nil,
		&git.BareCloneOptions{BaseDir: workDir},
	)
	require.NoError(t, err)
	defer repo.Close()
	workTreePath := filepath.Join(workDir, "main")
	workTree, err := repo.AddWorkTree(
		workTreePath,
		&git.AddWorkTreeOptions{Orphan: true},
	)
	require.NoError(t, err)
	require.NoError(t, workTree.CreateOrphanedBranch("main"))
	require.NoError(t, os.WriteFile(
		filepath.Join(workTree.Dir(), "test.txt"),
		[]byte("foo"),
		0o600,
	))
	require.NoError(t, workTree.AddAllAndCommit("initial commit", nil))

	stepCtx := &promotion.StepContext{WorkDir: workDir}

	testCases := []struct {
		name   string
		runner *githubPusher
		cfg    builtin.GitHubPushConfig
		assert func(*testing.T, promotion.StepResult, error)
	}{
		{
			name:   "error loading work tree",
			runner: &githubPusher{cfg: githubPusherConfig{}},
			cfg: builtin.GitHubPushConfig{
				Path: "nonexistent", // Invalid path forces a failure
			},
			assert: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, "error loading working tree")
				require.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
			},
		},
		{
			name: "error getting credentials",
			runner: &githubPusher{
				cfg: githubPusherConfig{},
				credsDB: &credentials.FakeDB{
					GetFn: func(
						context.Context,
						string,
						credentials.Type,
						string,
					) (*credentials.Credentials, error) {
						return nil, errors.New("something went wrong")
					},
				},
			},
			cfg: builtin.GitHubPushConfig{Path: "main"},
			assert: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, "error getting credentials")
				require.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
			},
		},
		{
			name: "no credentials found",
			runner: &githubPusher{
				cfg: githubPusherConfig{},
				credsDB: &credentials.FakeDB{
					GetFn: func(
						context.Context,
						string,
						credentials.Type,
						string,
					) (*credentials.Credentials, error) {
						return nil, nil
					},
				},
			},
			cfg: builtin.GitHubPushConfig{Path: "main"},
			assert: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, "no credentials found")
				require.True(t, promotion.IsTerminal(err))
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
			},
		},
		{
			name: "SSH key only",
			runner: &githubPusher{
				cfg: githubPusherConfig{},
				credsDB: &credentials.FakeDB{
					GetFn: func(
						context.Context,
						string,
						credentials.Type,
						string,
					) (*credentials.Credentials, error) {
						return &credentials.Credentials{SSHPrivateKey: "fake-ssh-key"}, nil
					},
				},
			},
			cfg: builtin.GitHubPushConfig{Path: "main"},
			assert: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, "found SSH key")
				require.True(t, promotion.IsTerminal(err))
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
			},
		},
		{
			name: "credentials with neither token nor SSH key",
			runner: &githubPusher{
				cfg: githubPusherConfig{},
				credsDB: &credentials.FakeDB{
					GetFn: func(
						context.Context,
						string,
						credentials.Type,
						string,
					) (*credentials.Credentials, error) {
						return &credentials.Credentials{}, nil
					},
				},
			},
			cfg: builtin.GitHubPushConfig{Path: "main"},
			assert: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, "missing a password/token")
				require.True(t, promotion.IsTerminal(err))
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
			},
		},
		{
			name: "valid credentials, fails at URL parsing",
			runner: &githubPusher{
				cfg: githubPusherConfig{},
				credsDB: &credentials.FakeDB{
					GetFn: func(
						context.Context,
						string,
						credentials.Type,
						string,
					) (*credentials.Credentials, error) {
						return &credentials.Credentials{Password: "fake-token"}, nil
					},
				},
			},
			cfg: builtin.GitHubPushConfig{Path: "main"},
			assert: func(t *testing.T, res promotion.StepResult, err error) {
				// The test repo URL is not GitHub-shaped, so parsing
				// owner/repo fails. This proves we got past credential
				// handling and work tree reload successfully.
				require.ErrorContains(t, err, "error parsing repository URL")
				require.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
			},
		},
		// Test cases beyond this point exercise the push() method and retry
		// loop, which are covered only by integration tests.
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			res, err := testCase.runner.run(
				context.Background(),
				stepCtx,
				testCase.cfg,
			)
			testCase.assert(t, res, err)
		})
	}
}

func Test_githubPusher_push(t *testing.T) {
	const testRepoURL = "https://github.com/owner/repo"
	testCases := []struct {
		name       string
		workTree   git.WorkTree
		client     githubPushClient
		cfg        githubPushConfig
		assertions func(*testing.T, error)
	}{
		{
			name: "error integrating remote changes",
			workTree: &git.MockRepo{
				URLFn: func() string {
					return testRepoURL
				},
				IntegrateRemoteChangesFn: func(*git.IntegrationOptions) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error integrating remote changes")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error pushing to remote staging ref",
			workTree: &git.MockRepo{
				URLFn: func() string {
					return testRepoURL
				},
				IntegrateRemoteChangesFn: func(*git.IntegrationOptions) error {
					return nil
				},
				PushFn: func(*git.PushOptions) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error pushing to staging ref")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error getting last commit ID",
			workTree: &git.MockRepo{
				URLFn: func() string {
					return testRepoURL
				},
				IntegrateRemoteChangesFn: func(*git.IntegrationOptions) error {
					return nil
				},
				PushFn: func(*git.PushOptions) error {
					return nil
				},
				LastCommitIDFn: func() (string, error) {
					return "", errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error getting source HEAD")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error checking for target branch existence",
			workTree: &git.MockRepo{
				URLFn: func() string {
					return testRepoURL
				},
				IntegrateRemoteChangesFn: func(*git.IntegrationOptions) error {
					return nil
				},
				PushFn: func(*git.PushOptions) error {
					return nil
				},
				LastCommitIDFn: func() (string, error) {
					return "fake-source-head", nil
				},
				RemoteBranchExistsFn: func(string) (bool, error) {
					return false, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error checking if remote branch")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "remote target branch does not exist; error getting current branch name",
			workTree: &git.MockRepo{
				URLFn: func() string {
					return testRepoURL
				},
				IntegrateRemoteChangesFn: func(*git.IntegrationOptions) error {
					return nil
				},
				PushFn: func(*git.PushOptions) error {
					return nil
				},
				LastCommitIDFn: func() (string, error) {
					return "fake-source-head", nil
				},
				RemoteBranchExistsFn: func(string) (bool, error) {
					return false, nil
				},
				CurrentBranchFn: func() (string, error) {
					return "", errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error getting current branch name")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error getting base ref from API",
			workTree: &git.MockRepo{
				URLFn: func() string {
					return testRepoURL
				},
				IntegrateRemoteChangesFn: func(*git.IntegrationOptions) error {
					return nil
				},
				PushFn: func(*git.PushOptions) error {
					return nil
				},
				LastCommitIDFn: func() (string, error) {
					return "fake-source-head", nil
				},
				RemoteBranchExistsFn: func(string) (bool, error) {
					return true, nil
				},
			},
			client: &mockGitHubPushClient{
				getRefFn: func(
					context.Context,
					string,
					string,
					string,
				) (*github.Reference, *github.Response, error) {
					return nil, nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error getting ref")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error comparing commits",
			workTree: &git.MockRepo{
				URLFn: func() string {
					return testRepoURL
				},
				IntegrateRemoteChangesFn: func(*git.IntegrationOptions) error {
					return nil
				},
				PushFn: func(*git.PushOptions) error {
					return nil
				},
				LastCommitIDFn: func() (string, error) {
					return "fake-source-head", nil
				},
				RemoteBranchExistsFn: func(string) (bool, error) {
					return true, nil
				},
			},
			client: &mockGitHubPushClient{
				getRefFn: func(
					context.Context,
					string,
					string,
					string,
				) (*github.Reference, *github.Response, error) {
					return &github.Reference{}, nil, nil
				},
				compareCommitsFn: func(
					context.Context,
					string,
					string,
					string,
					string,
					*github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return nil, nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error comparing commits")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "branches are identical",
			workTree: &git.MockRepo{
				URLFn: func() string {
					return testRepoURL
				},
				IntegrateRemoteChangesFn: func(*git.IntegrationOptions) error {
					return nil
				},
				PushFn: func(*git.PushOptions) error {
					return nil
				},
				LastCommitIDFn: func() (string, error) {
					return "fake-source-head", nil
				},
				RemoteBranchExistsFn: func(string) (bool, error) {
					return true, nil
				},
			},
			client: &mockGitHubPushClient{
				getRefFn: func(
					context.Context,
					string,
					string,
					string,
				) (*github.Reference, *github.Response, error) {
					return &github.Reference{}, nil, nil
				},
				compareCommitsFn: func(
					context.Context,
					string,
					string,
					string,
					string,
					*github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{Status: ptr.To("identical")}, nil, nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "source branch is behind; force is enabled; success via zero-commit replay",
			workTree: &git.MockRepo{
				URLFn: func() string {
					return testRepoURL
				},
				IntegrateRemoteChangesFn: func(*git.IntegrationOptions) error {
					return nil
				},
				PushFn: func(*git.PushOptions) error {
					return nil
				},
				LastCommitIDFn: func() (string, error) {
					return "fake-source-head", nil
				},
				RemoteBranchExistsFn: func(string) (bool, error) {
					return true, nil
				},
			},
			client: &mockGitHubPushClient{
				getRefFn: func(
					context.Context,
					string,
					string,
					string,
				) (*github.Reference, *github.Response, error) {
					return &github.Reference{}, nil, nil
				},
				compareCommitsFn: func(
					context.Context,
					string,
					string,
					string,
					string,
					*github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status:          ptr.To("behind"),
						MergeBaseCommit: &github.RepositoryCommit{SHA: ptr.To("fake-source-head")},
					}, nil, nil
				},
				updateRefFn: func(
					context.Context,
					string,
					string,
					string,
					github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					return nil, nil, nil
				},
			},
			cfg: githubPushConfig{force: true},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "source branch is behind; force is not enabled",
			workTree: &git.MockRepo{
				URLFn: func() string {
					return testRepoURL
				},
				IntegrateRemoteChangesFn: func(*git.IntegrationOptions) error {
					return nil
				},
				PushFn: func(*git.PushOptions) error {
					return nil
				},
				LastCommitIDFn: func() (string, error) {
					return "fake-source-head", nil
				},
				RemoteBranchExistsFn: func(string) (bool, error) {
					return true, nil
				},
			},
			client: &mockGitHubPushClient{
				getRefFn: func(
					context.Context,
					string,
					string,
					string,
				) (*github.Reference, *github.Response, error) {
					return &github.Reference{}, nil, nil
				},
				compareCommitsFn: func(
					context.Context,
					string,
					string,
					string,
					string,
					*github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{Status: ptr.To("behind")}, nil, nil
				},
				updateRefFn: func(
					context.Context,
					string,
					string,
					string,
					github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					return nil, nil, nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "source branch is behind target branch")
				require.ErrorAs(t, err, ptr.To(&retryableError{}))
			},
		},
		{
			name: "branches have diverged; force is not enabled",
			workTree: &git.MockRepo{
				URLFn: func() string {
					return testRepoURL
				},
				IntegrateRemoteChangesFn: func(*git.IntegrationOptions) error {
					return nil
				},
				PushFn: func(*git.PushOptions) error {
					return nil
				},
				LastCommitIDFn: func() (string, error) {
					return "fake-source-head", nil
				},
				RemoteBranchExistsFn: func(string) (bool, error) {
					return true, nil
				},
			},
			client: &mockGitHubPushClient{
				getRefFn: func(
					context.Context,
					string,
					string,
					string,
				) (*github.Reference, *github.Response, error) {
					return &github.Reference{}, nil, nil
				},
				compareCommitsFn: func(
					context.Context,
					string,
					string,
					string,
					string,
					*github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{Status: ptr.To("diverged")}, nil, nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "source and target branches have diverged")
				require.ErrorAs(t, err, ptr.To(&retryableError{}))
			},
		},
		{
			name: "unexpected comparison status",
			workTree: &git.MockRepo{
				URLFn: func() string {
					return testRepoURL
				},
				IntegrateRemoteChangesFn: func(*git.IntegrationOptions) error {
					return nil
				},
				PushFn: func(*git.PushOptions) error {
					return nil
				},
				LastCommitIDFn: func() (string, error) {
					return "fake-source-head", nil
				},
				RemoteBranchExistsFn: func(string) (bool, error) {
					return true, nil
				},
			},
			client: &mockGitHubPushClient{
				getRefFn: func(
					context.Context,
					string,
					string,
					string,
				) (*github.Reference, *github.Response, error) {
					return &github.Reference{}, nil, nil
				},
				compareCommitsFn: func(
					context.Context,
					string,
					string,
					string,
					string,
					*github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{Status: ptr.To("unexpected")}, nil, nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unexpected comparison status")
			},
		},
		{
			name: "source branch is ahead or branches have diverged; commits to replay exceed max",
			workTree: &git.MockRepo{
				URLFn: func() string {
					return testRepoURL
				},
				IntegrateRemoteChangesFn: func(*git.IntegrationOptions) error {
					return nil
				},
				PushFn: func(*git.PushOptions) error {
					return nil
				},
				LastCommitIDFn: func() (string, error) {
					return "fake-source-head", nil
				},
				RemoteBranchExistsFn: func(string) (bool, error) {
					return true, nil
				},
			},
			client: &mockGitHubPushClient{
				getRefFn: func(
					context.Context,
					string,
					string,
					string,
				) (*github.Reference, *github.Response, error) {
					return &github.Reference{}, nil, nil
				},
				compareCommitsFn: func(
					context.Context,
					string,
					string,
					string,
					string,
					*github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status:          ptr.To("ahead"),
						MergeBaseCommit: &github.RepositoryCommit{SHA: ptr.To("fake-merge-base")},
						Commits: []*github.RepositoryCommit{
							{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, // 12 commits
						},
					}, nil, nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "which exceeds the configured maximum")
			},
		},
		{
			name: "source branch is ahead or branches have diverged; error replaying commits",
			workTree: &git.MockRepo{
				URLFn: func() string {
					return testRepoURL
				},
				IntegrateRemoteChangesFn: func(*git.IntegrationOptions) error {
					return nil
				},
				PushFn: func(*git.PushOptions) error {
					return nil
				},
				LastCommitIDFn: func() (string, error) {
					return "fake-source-head", nil
				},
				RemoteBranchExistsFn: func(string) (bool, error) {
					return true, nil
				},
			},
			client: &mockGitHubPushClient{
				getRefFn: func(
					context.Context,
					string,
					string,
					string,
				) (*github.Reference, *github.Response, error) {
					return &github.Reference{}, nil, nil
				},
				compareCommitsFn: func(
					context.Context,
					string,
					string,
					string,
					string,
					*github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status:          ptr.To("ahead"),
						MergeBaseCommit: &github.RepositoryCommit{SHA: ptr.To("fake-merge-base")},
						Commits: []*github.RepositoryCommit{
							{}, // Missing tree information should cause replay to fail
						},
					}, nil, nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error replaying commits via GitHub API")
				require.ErrorContains(t, err, "has missing tree information")
			},
		},
		{
			name: "source branch is ahead or branches have diverged; error updating remote target ref",
			workTree: &git.MockRepo{
				URLFn: func() string {
					return testRepoURL
				},
				IntegrateRemoteChangesFn: func(*git.IntegrationOptions) error {
					return nil
				},
				PushFn: func(*git.PushOptions) error {
					return nil
				},
				LastCommitIDFn: func() (string, error) {
					return "fake-source-head", nil
				},
				RemoteBranchExistsFn: func(string) (bool, error) {
					return true, nil
				},
				GetCommitSignatureInfoFn: func(string) (*git.CommitSignatureInfo, error) {
					return &git.CommitSignatureInfo{}, nil
				},
			},
			client: &mockGitHubPushClient{
				getRefFn: func(
					context.Context,
					string,
					string,
					string,
				) (*github.Reference, *github.Response, error) {
					return &github.Reference{}, nil, nil
				},
				compareCommitsFn: func(
					context.Context,
					string,
					string,
					string,
					string,
					*github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status:          ptr.To("ahead"),
						MergeBaseCommit: &github.RepositoryCommit{SHA: ptr.To("fake-merge-base")},
						Commits: []*github.RepositoryCommit{{
							Commit: &github.Commit{
								Tree: &github.Tree{SHA: ptr.To("fake-commit")},
							},
						}},
					}, nil, nil
				},
				createCommitFn: func(
					context.Context,
					string,
					string,
					github.Commit,
					*github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					return &github.Commit{SHA: ptr.To("new-sha")}, nil, nil
				},
				updateRefFn: func(
					context.Context,
					string,
					string,
					string,
					github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					return nil, nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error updating target ref")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "source branch is ahead or branches have diverged; success",
			workTree: &git.MockRepo{
				URLFn: func() string {
					return testRepoURL
				},
				IntegrateRemoteChangesFn: func(*git.IntegrationOptions) error {
					return nil
				},
				PushFn: func(*git.PushOptions) error {
					return nil
				},
				LastCommitIDFn: func() (string, error) {
					return "fake-source-head", nil
				},
				RemoteBranchExistsFn: func(string) (bool, error) {
					return true, nil
				},
				GetCommitSignatureInfoFn: func(string) (*git.CommitSignatureInfo, error) {
					return &git.CommitSignatureInfo{}, nil
				},
			},
			client: &mockGitHubPushClient{
				getRefFn: func(
					context.Context,
					string,
					string,
					string,
				) (*github.Reference, *github.Response, error) {
					return &github.Reference{}, nil, nil
				},
				compareCommitsFn: func(
					context.Context,
					string,
					string,
					string,
					string,
					*github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status:          ptr.To("ahead"),
						MergeBaseCommit: &github.RepositoryCommit{SHA: ptr.To("fake-merge-base")},
						Commits: []*github.RepositoryCommit{{
							Commit: &github.Commit{
								Tree: &github.Tree{SHA: ptr.To("fake-commit")},
							},
						}},
					}, nil, nil
				},
				createCommitFn: func(
					context.Context,
					string,
					string,
					github.Commit,
					*github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					return &github.Commit{SHA: ptr.To("new-sha")}, nil, nil
				},
				updateRefFn: func(
					context.Context,
					string,
					string,
					string,
					github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					return nil, nil, nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	pusher := &githubPusher{
		cfg:       githubPusherConfig{MaxRevisions: 10},
		branchMus: map[string]*sync.Mutex{},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := pusher.push(
				t.Context(),
				testCase.workTree,
				testCase.client,
				testCase.cfg,
			)
			testCase.assertions(t, err)
		})
	}
}

func Test_githubPusher_replayCommits(t *testing.T) {
	testCases := []struct {
		name       string
		workTree   git.WorkTree
		client     githubPushClient
		commits    []*github.RepositoryCommit
		assertions func(*testing.T, string, error)
	}{
		{
			name: "missing tree information",
			commits: []*github.RepositoryCommit{
				{
					SHA:    github.Ptr("old-sha"),
					Commit: &github.Commit{Message: github.Ptr("test")},
				},
			},
			assertions: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "missing tree information")
			},
		},
		{
			name: "error getting signature info",
			workTree: &git.MockRepo{
				GetCommitSignatureInfoFn: func(string) (*git.CommitSignatureInfo, error) {
					return nil, errors.New("something went wrong")
				},
			},
			commits: []*github.RepositoryCommit{{
				SHA: github.Ptr("old-sha"),
				Commit: &github.Commit{
					Message: github.Ptr("test"),
					Tree:    &github.Tree{SHA: github.Ptr("tree-sha")},
				},
			}},
			assertions: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "error getting signature info")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error creating commit via API",
			workTree: &git.MockRepo{
				GetCommitSignatureInfoFn: func(string) (*git.CommitSignatureInfo, error) {
					return &git.CommitSignatureInfo{Trusted: false}, nil
				},
			},
			client: &mockGitHubPushClient{
				createCommitFn: func(
					context.Context,
					string,
					string,
					github.Commit,
					*github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					return nil, nil, errors.New("something went wrong")
				},
			},
			commits: []*github.RepositoryCommit{
				{
					SHA: github.Ptr("old-sha"),
					Commit: &github.Commit{
						Message: github.Ptr("test"),
						Tree:    &github.Tree{SHA: github.Ptr("tree-sha")},
					},
				},
			},
			assertions: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "error creating commit")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "success; SHA map chains parents across commits",
			workTree: &git.MockRepo{
				GetCommitSignatureInfoFn: func(string) (*git.CommitSignatureInfo, error) {
					return &git.CommitSignatureInfo{Trusted: false}, nil
				},
			},
			client: func() *mockGitHubPushClient {
				callCount := 0
				return &mockGitHubPushClient{
					createCommitFn: func(
						_ context.Context,
						_, _ string,
						commit github.Commit,
						_ *github.CreateCommitOptions,
					) (*github.Commit, *github.Response, error) {
						callCount++
						sha := fmt.Sprintf("new-%d", callCount)
						if callCount == 2 {
							require.Len(t, commit.Parents, 1)
							require.Equal(t, "new-1", commit.Parents[0].GetSHA())
						}
						return &github.Commit{SHA: github.Ptr(sha)}, nil, nil
					},
				}
			}(),
			commits: []*github.RepositoryCommit{
				{
					SHA: github.Ptr("old-1"),
					Commit: &github.Commit{
						Message: github.Ptr("first"),
						Tree:    &github.Tree{SHA: github.Ptr("tree-1")},
					},
					Parents: []*github.Commit{{SHA: github.Ptr("base-sha")}},
				},
				{
					SHA: github.Ptr("old-2"),
					Commit: &github.Commit{
						Message: github.Ptr("second"),
						Tree:    &github.Tree{SHA: github.Ptr("tree-2")},
					},
					Parents: []*github.Commit{{SHA: github.Ptr("old-1")}},
				},
			},
			assertions: func(t *testing.T, sha string, err error) {
				require.NoError(t, err)
				require.Equal(t, "new-2", sha)
			},
		},
	}
	pusher := &githubPusher{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			sha, err := pusher.replayCommits(
				t.Context(),
				testCase.client,
				"owner",
				"repo",
				testCase.commits,
				"base-sha",
				testCase.workTree,
			)
			testCase.assertions(t, sha, err)
		})
	}
}

func Test_githubPusher_buildReplayCommit(t *testing.T) {
	trustedSigInfo := &git.CommitSignatureInfo{
		Trusted:     true,
		SignerName:  "Kargo",
		SignerEmail: "kargo@example.com",
	}
	testCases := []struct {
		name       string
		original   *github.RepositoryCommit
		sigInfo    *git.CommitSignatureInfo
		assertions func(*testing.T, github.Commit)
	}{
		{
			name: "trusted commit with nil author",
			original: &github.RepositoryCommit{
				Commit: &github.Commit{
					Message: github.Ptr("test"),
					Tree:    &github.Tree{SHA: github.Ptr("tree-sha")},
				},
			},
			sigInfo: trustedSigInfo,
			assertions: func(t *testing.T, c github.Commit) {
				require.NotContains(t, c.GetMessage(), "Co-authored-by")
			},
		},
		{
			name: "trusted commit with different author",
			original: &github.RepositoryCommit{
				Commit: &github.Commit{
					Message: github.Ptr("alice's commit"),
					Tree:    &github.Tree{SHA: github.Ptr("tree-sha")},
					Author: &github.CommitAuthor{
						Name:  github.Ptr("Alice"),
						Email: github.Ptr("alice@example.com"),
					},
					Committer: &github.CommitAuthor{
						Name:  &trustedSigInfo.SignerName,
						Email: &trustedSigInfo.SignerEmail,
					},
				},
			},
			sigInfo: trustedSigInfo,
			assertions: func(t *testing.T, c github.Commit) {
				require.Nil(t, c.Author)
				require.Nil(t, c.Committer)
				require.Contains(t, c.GetMessage(), "alice's commit")
				require.Contains(
					t,
					c.GetMessage(),
					"Co-authored-by: Alice <alice@example.com>",
				)
			},
		},
		{
			name: "trusted commit with same author",
			original: &github.RepositoryCommit{
				Commit: &github.Commit{
					Message: github.Ptr("test"),
					Tree:    &github.Tree{SHA: github.Ptr("tree-sha")},
					Author: &github.CommitAuthor{
						Name:  &trustedSigInfo.SignerName,
						Email: &trustedSigInfo.SignerEmail,
					},
					Committer: &github.CommitAuthor{
						Name:  &trustedSigInfo.SignerName,
						Email: &trustedSigInfo.SignerEmail,
					},
				},
			},
			sigInfo: trustedSigInfo,
			assertions: func(t *testing.T, c github.Commit) {
				require.Nil(t, c.Author)
				require.Nil(t, c.Committer)
				require.Equal(t, "test", c.GetMessage())
				require.NotContains(t, c.GetMessage(), "Co-authored-by")
			},
		},
		{
			name: "untrusted commit",
			original: &github.RepositoryCommit{
				Commit: &github.Commit{
					Message: github.Ptr("test"),
					Tree:    &github.Tree{SHA: github.Ptr("tree-sha")},
					Author: &github.CommitAuthor{
						Name:  github.Ptr("Sally"),
						Email: github.Ptr("sally@example.com"),
					},
					Committer: &github.CommitAuthor{
						Name:  github.Ptr("Sally"),
						Email: github.Ptr("sally@example.com"),
					},
				},
			},
			sigInfo: &git.CommitSignatureInfo{Trusted: false},
			assertions: func(t *testing.T, c github.Commit) {
				require.NotNil(t, c.Author)
				require.NotNil(t, c.Committer)
				require.Equal(t, "Sally", c.Author.GetName())
				require.Equal(t, "test", c.GetMessage())
			},
		},
	}
	pusher := &githubPusher{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				pusher.buildReplayCommit(testCase.original, testCase.sigInfo),
			)
		})
	}
}

func Test_githubPusher_resolveParents(t *testing.T) {
	const previousSHA = "aaa111"
	testCases := []struct {
		name       string
		repoCommit *github.RepositoryCommit
		shaMap     map[string]string
		assertions func(*testing.T, []*github.Commit)
	}{
		{
			name:       "no parents",
			repoCommit: &github.RepositoryCommit{},
			shaMap:     map[string]string{},
			assertions: func(t *testing.T, parents []*github.Commit) {
				require.Len(t, parents, 1)
				require.Equal(t, previousSHA, parents[0].GetSHA())
			},
		},
		{
			name: "single parent",
			repoCommit: &github.RepositoryCommit{
				Parents: []*github.Commit{{SHA: github.Ptr("parent-sha")}},
			},
			shaMap: map[string]string{},
			assertions: func(t *testing.T, parents []*github.Commit) {
				require.Len(t, parents, 1)
				require.Equal(t, previousSHA, parents[0].GetSHA())
			},
		},
		{
			name: "multiple, mapped parents",
			repoCommit: &github.RepositoryCommit{
				Parents: []*github.Commit{
					{SHA: github.Ptr("old-a")},
					{SHA: github.Ptr("old-b")},
				},
			},
			shaMap: map[string]string{"old-a": "new-a"},
			assertions: func(t *testing.T, parents []*github.Commit) {
				require.Len(t, parents, 2)
				require.Equal(t, "new-a", parents[0].GetSHA())
				require.Equal(t, "old-b", parents[1].GetSHA())
			},
		},
	}
	pusher := &githubPusher{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				pusher.resolveParents(testCase.repoCommit, testCase.shaMap, previousSHA),
			)
		})
	}
}

func Test_githubPusher_upsertTargetRef(t *testing.T) {
	testCases := []struct {
		name         string
		client       githubPushClient
		targetBranch string
		sha          string
		createBranch bool
		force        bool
		assertions   func(*testing.T, error)
	}{
		{
			name: "non-fast-forward",
			client: &mockGitHubPushClient{
				updateRefFn: func(
					_ context.Context,
					_ string,
					_ string,
					_ string,
					_ github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					return nil, &github.Response{
						Response: &http.Response{
							StatusCode: http.StatusUnprocessableEntity,
						},
					}, fmt.Errorf("update failed")
				},
			},
			targetBranch: "main",
			sha:          "abc123",
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				var re *retryableError
				require.ErrorAs(t, err, &re)
			},
		},
		{
			name: "success creating new branch",
			client: &mockGitHubPushClient{
				createRefFn: func(
					_ context.Context,
					_, _ string,
					ref github.CreateRef,
				) (*github.Reference, *github.Response, error) {
					if ref.Ref != "refs/heads/new-branch" || ref.SHA != "abc123" {
						return nil, nil, fmt.Errorf("unexpected ref")
					}
					return &github.Reference{}, nil, nil
				},
			},
			targetBranch: "new-branch",
			sha:          "abc123",
			createBranch: true,
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "success updating existing branch",
			client: &mockGitHubPushClient{
				updateRefFn: func(
					_ context.Context,
					_ string,
					_ string,
					ref string,
					updateRef github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					if ref != "heads/main" || updateRef.SHA != "abc123" {
						return nil, nil, fmt.Errorf("unexpected ref")
					}
					return &github.Reference{}, nil, nil
				},
			},
			targetBranch: "main",
			sha:          "abc123",
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	pusher := &githubPusher{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := pusher.upsertTargetRef(
				t.Context(), testCase.client, "owner", "repo",
				testCase.targetBranch, testCase.sha,
				testCase.createBranch, testCase.force,
			)
			testCase.assertions(t, err)
		})
	}
}

func Test_githubPusher_isRetryableError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "retryable error",
			err:      &retryableError{err: fmt.Errorf("conflict")},
			expected: true,
		},
		{
			name:     "non-retryable error",
			err:      fmt.Errorf("some other error"),
			expected: false,
		},
	}
	pusher := &githubPusher{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expected, pusher.isRetryableError(testCase.err))
		})
	}
}

type mockGitHubPushClient struct {
	createCommitFn func(
		context.Context, string, string, github.Commit,
		*github.CreateCommitOptions,
	) (*github.Commit, *github.Response, error)
	compareCommitsFn func(
		context.Context, string, string, string, string,
		*github.ListOptions,
	) (*github.CommitsComparison, *github.Response, error)
	getRefFn func(
		context.Context, string, string, string,
	) (*github.Reference, *github.Response, error)
	createRefFn func(
		context.Context, string, string, github.CreateRef,
	) (*github.Reference, *github.Response, error)
	updateRefFn func(
		context.Context, string, string, string, github.UpdateRef,
	) (*github.Reference, *github.Response, error)
	deleteRefFn func(
		context.Context, string, string, string,
	) (*github.Response, error)
}

func (m *mockGitHubPushClient) CreateCommit(
	ctx context.Context,
	owner string,
	repo string,
	commit github.Commit,
	opts *github.CreateCommitOptions,
) (*github.Commit, *github.Response, error) {
	return m.createCommitFn(ctx, owner, repo, commit, opts)
}

func (m *mockGitHubPushClient) CompareCommits(
	ctx context.Context,
	owner string,
	repo string,
	base string,
	head string,
	opts *github.ListOptions,
) (*github.CommitsComparison, *github.Response, error) {
	return m.compareCommitsFn(ctx, owner, repo, base, head, opts)
}

func (m *mockGitHubPushClient) GetRef(
	ctx context.Context,
	owner string,
	repo string,
	ref string,
) (*github.Reference, *github.Response, error) {
	return m.getRefFn(ctx, owner, repo, ref)
}

func (m *mockGitHubPushClient) CreateRef(
	ctx context.Context,
	owner string,
	repo string,
	ref github.CreateRef,
) (*github.Reference, *github.Response, error) {
	return m.createRefFn(ctx, owner, repo, ref)
}

func (m *mockGitHubPushClient) UpdateRef(
	ctx context.Context,
	owner string,
	repo string,
	ref string,
	updateRef github.UpdateRef,
) (*github.Reference, *github.Response, error) {
	return m.updateRefFn(ctx, owner, repo, ref, updateRef)
}

func (m *mockGitHubPushClient) DeleteRef(
	ctx context.Context,
	owner string,
	repo string,
	ref string,
) (*github.Response, error) {
	return m.deleteRefFn(ctx, owner, repo, ref)
}
