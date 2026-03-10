package builtin

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/promotion"
	builtinx "github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// mockWorkTree is a minimal mock of git.WorkTree for testing.
type mockWorkTree struct {
	git.WorkTree
	url             string
	dir             string
	homeDir         string
	currentBranchFn func() (string, error)
	lastCommitIDFn  func() (string, error)
	pushFn          func(*git.PushOptions) error
}

func (m *mockWorkTree) URL() string     { return m.url }
func (m *mockWorkTree) Dir() string     { return m.dir }
func (m *mockWorkTree) HomeDir() string { return m.homeDir }

func (m *mockWorkTree) CurrentBranch() (string, error) {
	return m.currentBranchFn()
}

func (m *mockWorkTree) LastCommitID() (string, error) {
	return m.lastCommitIDFn()
}

func (m *mockWorkTree) Push(opts *git.PushOptions) error {
	return m.pushFn(opts)
}

func Test_githubVerifiedPusher_convert(t *testing.T) {
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
			name: "just generateTargetBranch is true",
			config: promotion.Config{
				"path":                 "/fake/path",
				"generateTargetBranch": true,
			},
		},
		{
			name: "generateTargetBranch is true and targetBranch is empty",
			config: promotion.Config{
				"path":                 "/fake/path",
				"generateTargetBranch": true,
				"targetBranch":         "",
			},
		},
		{
			name: "generateTargetBranch is true and targetBranch is specified",
			config: promotion.Config{
				"path":                 "/fake/path",
				"generateTargetBranch": true,
				"targetBranch":         "main",
			},
			expectedProblems: []string{
				"(root): Must validate one and only one schema",
			},
		},
		{
			name: "targetBranch is specified",
			config: promotion.Config{
				"path":         "/fake/path",
				"targetBranch": "main",
			},
		},
		{
			name: "neither generateTargetBranch nor targetBranch",
			config: promotion.Config{
				"path": "/fake/path",
			},
		},
		{
			name: "insecureSkipTLSVerify is true",
			config: promotion.Config{
				"path":                  "/fake/path",
				"insecureSkipTLSVerify": true,
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
			name: "force is false",
			config: promotion.Config{
				"path":  "/fake/path",
				"force": false,
			},
		},
		{
			name: "unknown field",
			config: promotion.Config{
				"path":    "/fake/path",
				"unknown": "value",
			},
			expectedProblems: []string{
				"Additional property unknown is not allowed",
			},
		},
	}
	r := newGitHubVerifiedPusher(
		promotion.StepRunnerCapabilities{},
		githubVerifiedPusherConfig{},
	)
	runner, ok := r.(*githubVerifiedPusher)
	require.True(t, ok)
	runValidationTests(t, runner.convert, tests)
}

// mockGitHubVerifiedPushClient is a mock implementation of
// githubVerifiedPushClient for testing.
type mockGitHubVerifiedPushClient struct {
	compareCommitsFn func(
		ctx context.Context,
		owner, repo, base, head string,
		opts *github.ListOptions,
	) (*github.CommitsComparison, *github.Response, error)
	createCommitFn func(
		ctx context.Context,
		owner, repo string,
		commit github.Commit,
		opts *github.CreateCommitOptions,
	) (*github.Commit, *github.Response, error)
	createRefFn func(
		ctx context.Context,
		owner, repo string,
		ref github.CreateRef,
	) (*github.Reference, *github.Response, error)
	getRefFn func(
		ctx context.Context,
		owner, repo, ref string,
	) (*github.Reference, *github.Response, error)
	updateRefFn func(
		ctx context.Context,
		owner, repo, ref string,
		updateRef github.UpdateRef,
	) (*github.Reference, *github.Response, error)
	deleteRefFn func(
		ctx context.Context,
		owner, repo, ref string,
	) (*github.Response, error)
}

func (m *mockGitHubVerifiedPushClient) CompareCommits(
	ctx context.Context,
	owner, repo, base, head string,
	opts *github.ListOptions,
) (*github.CommitsComparison, *github.Response, error) {
	return m.compareCommitsFn(ctx, owner, repo, base, head, opts)
}

func (m *mockGitHubVerifiedPushClient) CreateCommit(
	ctx context.Context,
	owner, repo string,
	commit github.Commit,
	opts *github.CreateCommitOptions,
) (*github.Commit, *github.Response, error) {
	return m.createCommitFn(ctx, owner, repo, commit, opts)
}

func (m *mockGitHubVerifiedPushClient) CreateRef(
	ctx context.Context,
	owner, repo string,
	ref github.CreateRef,
) (*github.Reference, *github.Response, error) {
	return m.createRefFn(ctx, owner, repo, ref)
}

func (m *mockGitHubVerifiedPushClient) GetRef(
	ctx context.Context,
	owner, repo, ref string,
) (*github.Reference, *github.Response, error) {
	return m.getRefFn(ctx, owner, repo, ref)
}

func (m *mockGitHubVerifiedPushClient) UpdateRef(
	ctx context.Context,
	owner, repo, ref string,
	updateRef github.UpdateRef,
) (*github.Reference, *github.Response, error) {
	return m.updateRefFn(ctx, owner, repo, ref, updateRef)
}

func (m *mockGitHubVerifiedPushClient) DeleteRef(
	ctx context.Context,
	owner, repo, ref string,
) (*github.Response, error) {
	return m.deleteRefFn(ctx, owner, repo, ref)
}

func Test_githubVerifiedPusher_signAndUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		client       githubVerifiedPushClient
		targetBranch string
		createBranch bool
		force        bool
		assert       func(*testing.T, promotion.StepResult, error)
	}{
		{
			name: "compare API error",
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return nil, nil, fmt.Errorf("API error")
				},
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.Contains(t, err.Error(), "error comparing")
				require.Equal(
					t,
					kargoapi.PromotionStepStatusErrored, result.Status,
				)
			},
		},
		{
			name: "identical commits",
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status: ptr.To("identical"),
					}, nil, nil
				},
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusSkipped, result.Status,
				)
				require.Equal(t, "abc123", result.Output["commit"])
				require.Equal(t, "main", result.Output["branch"])
			},
		},
		{
			name: "diverged commits without force",
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status: ptr.To("diverged"),
					}, nil, nil
				},
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.True(t, promotion.IsTerminal(err))
				require.Contains(
					t, err.Error(), "target branch may have diverged",
				)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusFailed, result.Status,
				)
			},
		},
		{
			name: "empty commits list",
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status:  ptr.To("ahead"),
						Commits: []*github.RepositoryCommit{},
					}, nil, nil
				},
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusSkipped, result.Status,
				)
			},
		},
		{
			name: "too many revisions",
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					// Create 11 commits (exceeds default max of 10)
					commits := make(
						[]*github.RepositoryCommit, 11,
					)
					for i := range commits {
						commits[i] = &github.RepositoryCommit{}
					}
					return &github.CommitsComparison{
						Status:  ptr.To("ahead"),
						Commits: commits,
					}, nil, nil
				},
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.True(t, promotion.IsTerminal(err))
				require.Contains(t, err.Error(), "exceeds the maximum")
				require.Equal(
					t,
					kargoapi.PromotionStepStatusFailed, result.Status,
				)
			},
		},
		{
			name: "missing tree information",
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status: ptr.To("ahead"),
						Commits: []*github.RepositoryCommit{
							{Commit: &github.Commit{}},
						},
					}, nil, nil
				},
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.Contains(
					t, err.Error(), "missing tree information",
				)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusErrored, result.Status,
				)
			},
		},
		{
			name: "create commit error",
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status: ptr.To("ahead"),
						Commits: []*github.RepositoryCommit{{
							SHA: ptr.To("orig-sha"),
							Commit: &github.Commit{
								Message: ptr.To("test commit"),
								Tree:    &github.Tree{SHA: ptr.To("tree-sha")},
							},
						}},
					}, nil, nil
				},
				createCommitFn: func(
					_ context.Context,
					_, _ string,
					_ github.Commit,
					_ *github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					return nil, nil, fmt.Errorf("create error")
				},
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.Contains(
					t, err.Error(), "error creating revision",
				)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusErrored, result.Status,
				)
			},
		},
		{
			name: "update ref error",
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status: ptr.To("ahead"),
						Commits: []*github.RepositoryCommit{{
							SHA: ptr.To("orig-sha"),
							Commit: &github.Commit{
								Message: ptr.To("test commit"),
								Tree:    &github.Tree{SHA: ptr.To("tree-sha")},
							},
						}},
					}, nil, nil
				},
				createCommitFn: func(
					_ context.Context,
					_, _ string,
					_ github.Commit,
					_ *github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					return &github.Commit{
						SHA: ptr.To("signed-sha"),
					}, nil, nil
				},
				updateRefFn: func(
					_ context.Context,
					_, _, _ string,
					_ github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					return nil, nil, fmt.Errorf("ref update error")
				},
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.Contains(t, err.Error(), "error updating ref")
				require.Equal(
					t,
					kargoapi.PromotionStepStatusErrored, result.Status,
				)
			},
		},
		{
			name: "successful signing of single commit",
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status: ptr.To("ahead"),
						Commits: []*github.RepositoryCommit{{
							SHA: ptr.To("orig-sha"),
							Commit: &github.Commit{
								Message: ptr.To("test commit"),
								Tree:    &github.Tree{SHA: ptr.To("tree-sha")},
							},
						}},
					}, nil, nil
				},
				createCommitFn: func(
					_ context.Context,
					_, _ string,
					_ github.Commit,
					_ *github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					return &github.Commit{
						SHA: ptr.To("signed-sha"),
					}, nil, nil
				},
				updateRefFn: func(
					_ context.Context,
					_, _, _ string,
					_ github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					return &github.Reference{}, nil, nil
				},
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusSucceeded, result.Status,
				)
				require.Equal(t, "signed-sha", result.Output["commit"])
				require.Equal(t, "main", result.Output["branch"])
				commitURL, ok := result.Output["commitURL"].(string)
				require.True(t, ok)
				require.Contains(t, commitURL, "signed-sha")
			},
		},
		{
			name: "create ref error for new branch",
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status: ptr.To("ahead"),
						Commits: []*github.RepositoryCommit{{
							SHA: ptr.To("orig-sha"),
							Commit: &github.Commit{
								Message: ptr.To("test commit"),
								Tree:    &github.Tree{SHA: ptr.To("tree-sha")},
							},
						}},
					}, nil, nil
				},
				createCommitFn: func(
					_ context.Context,
					_, _ string,
					_ github.Commit,
					_ *github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					return &github.Commit{
						SHA: ptr.To("signed-sha"),
					}, nil, nil
				},
				createRefFn: func(
					_ context.Context,
					_, _ string,
					_ github.CreateRef,
				) (*github.Reference, *github.Response, error) {
					return nil, nil, fmt.Errorf("ref create error")
				},
			},
			createBranch: true,
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.Contains(t, err.Error(), "error creating ref")
				require.Equal(
					t,
					kargoapi.PromotionStepStatusErrored, result.Status,
				)
			},
		},
		{
			name: "successful create of new branch",
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status: ptr.To("ahead"),
						Commits: []*github.RepositoryCommit{{
							SHA: ptr.To("orig-sha"),
							Commit: &github.Commit{
								Message: ptr.To("test commit"),
								Tree:    &github.Tree{SHA: ptr.To("tree-sha")},
							},
						}},
					}, nil, nil
				},
				createCommitFn: func(
					_ context.Context,
					_, _ string,
					_ github.Commit,
					_ *github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					return &github.Commit{
						SHA: ptr.To("signed-sha"),
					}, nil, nil
				},
				createRefFn: func(
					_ context.Context,
					_, _ string,
					ref github.CreateRef,
				) (*github.Reference, *github.Response, error) {
					require.Equal(
						t,
						"refs/heads/kargo/promotion/test",
						ref.Ref,
					)
					require.Equal(t, "signed-sha", ref.SHA)
					return &github.Reference{}, nil, nil
				},
			},
			createBranch: true,
			targetBranch: "kargo/promotion/test",
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusSucceeded, result.Status,
				)
				require.Equal(t, "signed-sha", result.Output["commit"])
				require.Equal(
					t,
					"kargo/promotion/test",
					result.Output["branch"],
				)
			},
		},
		{
			name:  "force push with diverged branches",
			force: true,
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status: ptr.To("diverged"),
						MergeBaseCommit: &github.RepositoryCommit{
							SHA: ptr.To("merge-base-sha"),
						},
						Commits: []*github.RepositoryCommit{{
							SHA: ptr.To("orig-sha"),
							Commit: &github.Commit{
								Message: ptr.To("test commit"),
								Tree:    &github.Tree{SHA: ptr.To("tree-sha")},
							},
						}},
					}, nil, nil
				},
				createCommitFn: func(
					_ context.Context,
					_, _ string,
					commit github.Commit,
					_ *github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					// Verify the parent is the merge base, not targetHead.
					require.Equal(
						t, "merge-base-sha",
						commit.Parents[0].GetSHA(),
					)
					return &github.Commit{
						SHA: ptr.To("signed-sha"),
					}, nil, nil
				},
				updateRefFn: func(
					_ context.Context,
					_, _, _ string,
					ref github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					require.Equal(t, "signed-sha", ref.SHA)
					require.NotNil(t, ref.Force)
					require.True(t, *ref.Force)
					return &github.Reference{}, nil, nil
				},
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusSucceeded, result.Status,
				)
				require.Equal(t, "signed-sha", result.Output["commit"])
			},
		},
		{
			name:  "force push with diverged branches missing merge base",
			force: true,
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status: ptr.To("diverged"),
					}, nil, nil
				},
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.Contains(
					t, err.Error(), "cannot determine merge base",
				)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusErrored, result.Status,
				)
			},
		},
		{
			name:  "force push with behind status",
			force: true,
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status:  ptr.To("behind"),
						Commits: []*github.RepositoryCommit{},
					}, nil, nil
				},
				updateRefFn: func(
					_ context.Context,
					_, _, _ string,
					ref github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					// For behind+force, the ref should be updated
					// to localHead ("def456").
					require.Equal(t, "def456", ref.SHA)
					require.NotNil(t, ref.Force)
					require.True(t, *ref.Force)
					return &github.Reference{}, nil, nil
				},
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusSucceeded, result.Status,
				)
				require.Equal(t, "def456", result.Output["commit"])
			},
		},
		{
			name: "behind status without force",
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status: ptr.To("behind"),
					}, nil, nil
				},
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.True(t, promotion.IsTerminal(err))
				require.Contains(
					t, err.Error(), "target branch may have diverged",
				)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusFailed, result.Status,
				)
			},
		},
		{
			name:  "non-force update ref passes force=false",
			force: false,
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status: ptr.To("ahead"),
						Commits: []*github.RepositoryCommit{{
							SHA: ptr.To("orig-sha"),
							Commit: &github.Commit{
								Message: ptr.To("test commit"),
								Tree:    &github.Tree{SHA: ptr.To("tree-sha")},
							},
						}},
					}, nil, nil
				},
				createCommitFn: func(
					_ context.Context,
					_, _ string,
					_ github.Commit,
					_ *github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					return &github.Commit{
						SHA: ptr.To("signed-sha"),
					}, nil, nil
				},
				updateRefFn: func(
					_ context.Context,
					_, _, _ string,
					ref github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					require.NotNil(t, ref.Force)
					require.False(t, *ref.Force)
					return &github.Reference{}, nil, nil
				},
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusSucceeded, result.Status,
				)
			},
		},
		{
			name: "co-authored-by added for non-system author",
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status: ptr.To("ahead"),
						Commits: []*github.RepositoryCommit{{
							SHA: ptr.To("orig-sha"),
							Commit: &github.Commit{
								Message: ptr.To("test commit"),
								Tree:    &github.Tree{SHA: ptr.To("tree-sha")},
								Author: &github.CommitAuthor{
									Name:  ptr.To("Alice"),
									Email: ptr.To("alice@example.com"),
								},
							},
						}},
					}, nil, nil
				},
				createCommitFn: func(
					_ context.Context,
					_, _ string,
					commit github.Commit,
					_ *github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					// Verify: no Author/Committer (App signs)
					require.Nil(t, commit.Author)
					require.Nil(t, commit.Committer)
					// Verify: Co-authored-by trailer added
					require.Contains(
						t, commit.GetMessage(),
						"Co-authored-by: Alice <alice@example.com>",
					)
					return &github.Commit{
						SHA: ptr.To("signed-sha"),
					}, nil, nil
				},
				updateRefFn: func(
					_ context.Context,
					_, _, _ string,
					_ github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					return &github.Reference{}, nil, nil
				},
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusSucceeded, result.Status,
				)
			},
		},
		{
			name: "no co-authored-by for system author",
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status: ptr.To("ahead"),
						Commits: []*github.RepositoryCommit{{
							SHA: ptr.To("orig-sha"),
							Commit: &github.Commit{
								Message: ptr.To("test commit"),
								Tree:    &github.Tree{SHA: ptr.To("tree-sha")},
								Author: &github.CommitAuthor{
									Name:  ptr.To("Kargo"),
									Email: ptr.To("no-reply@kargo.io"),
								},
							},
						}},
					}, nil, nil
				},
				createCommitFn: func(
					_ context.Context,
					_, _ string,
					commit github.Commit,
					_ *github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					// Verify: no Author/Committer (App signs)
					require.Nil(t, commit.Author)
					require.Nil(t, commit.Committer)
					// Verify: no Co-authored-by trailer
					require.NotContains(
						t, commit.GetMessage(), "Co-authored-by",
					)
					return &github.Commit{
						SHA: ptr.To("signed-sha"),
					}, nil, nil
				},
				updateRefFn: func(
					_ context.Context,
					_, _, _ string,
					_ github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					return &github.Reference{}, nil, nil
				},
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusSucceeded, result.Status,
				)
			},
		},
		{
			name: "successful signing of multiple commits",
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status: ptr.To("ahead"),
						Commits: []*github.RepositoryCommit{
							{
								SHA: ptr.To("orig-sha-1"),
								Commit: &github.Commit{
									Message: ptr.To("first commit"),
									Tree: &github.Tree{
										SHA: ptr.To("tree-sha-1"),
									},
								},
							},
							{
								SHA: ptr.To("orig-sha-2"),
								Commit: &github.Commit{
									Message: ptr.To("second commit"),
									Tree: &github.Tree{
										SHA: ptr.To("tree-sha-2"),
									},
								},
							},
						},
					}, nil, nil
				},
				createCommitFn: func() func(
					context.Context, string, string,
					github.Commit, *github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					callCount := 0
					return func(
						_ context.Context,
						_, _ string,
						commit github.Commit,
						_ *github.CreateCommitOptions,
					) (*github.Commit, *github.Response, error) {
						callCount++
						sha := fmt.Sprintf("signed-sha-%d", callCount)
						// Verify parent chaining
						if callCount == 1 {
							require.Equal(
								t, "abc123",
								commit.Parents[0].GetSHA(),
							)
						} else {
							require.Equal(
								t, "signed-sha-1",
								commit.Parents[0].GetSHA(),
							)
						}
						return &github.Commit{
							SHA: ptr.To(sha),
						}, nil, nil
					}
				}(),
				updateRefFn: func(
					_ context.Context,
					_, _, _ string,
					ref github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					require.Equal(t, "signed-sha-2", ref.SHA)
					return &github.Reference{}, nil, nil
				},
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusSucceeded, result.Status,
				)
				require.Equal(
					t, "signed-sha-2", result.Output["commit"],
				)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := &githubVerifiedPusher{
				cfg: githubVerifiedPusherConfig{MaxRevisions: 10},
				gitUser: git.User{
					Name:  git.DefaultUsername,
					Email: git.DefaultEmail,
				},
			}
			targetBranch := tc.targetBranch
			if targetBranch == "" {
				targetBranch = "main"
			}
			result, err := g.signAndUpdate(
				context.Background(),
				tc.client,
				"owner", "repo",
				targetBranch, tc.createBranch, tc.force,
				"abc123",
				"def456",
				&mockWorkTree{url: "https://github.com/owner/repo"},
			)
			tc.assert(t, result, err)
		})
	}
}

func Test_githubVerifiedPusher_newGitHubClient(t *testing.T) {
	testCases := []struct {
		name      string
		repoURL   string
		assertErr func(*testing.T, error)
	}{
		{
			name:    "valid GitHub URL",
			repoURL: "https://github.com/owner/repo",
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name:    "valid GitHub URL with .git suffix",
			repoURL: "https://github.com/owner/repo.git",
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name:    "invalid URL with wrong path segments",
			repoURL: "https://github.com/owner",
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				require.Contains(
					t, err.Error(),
					"could not extract repository owner and name",
				)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := &githubVerifiedPusher{}
			_, _, _, err := g.newGitHubClient(
				tc.repoURL, "token", false,
			)
			tc.assertErr(t, err)
		})
	}
}

func Test_parseGitHubRepoURL(t *testing.T) {
	testCases := []struct {
		name          string
		repoURL       string
		expectedOwner string
		expectedRepo  string
		expectedHost  string
		expectErr     bool
	}{
		{
			name:          "standard GitHub URL",
			repoURL:       "https://github.com/owner/repo",
			expectedOwner: "owner",
			expectedRepo:  "repo",
			expectedHost:  "github.com",
		},
		{
			name:          "URL with .git suffix",
			repoURL:       "https://github.com/owner/repo.git",
			expectedOwner: "owner",
			expectedRepo:  "repo",
			expectedHost:  "github.com",
		},
		{
			name:          "GitHub Enterprise URL",
			repoURL:       "https://github.example.com/owner/repo",
			expectedOwner: "owner",
			expectedRepo:  "repo",
			expectedHost:  "github.example.com",
		},
		{
			name:      "invalid URL with wrong path segments",
			repoURL:   "https://github.com/owner",
			expectErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, host, owner, repo, err := parseGitHubRepoURL(tc.repoURL)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expectedOwner, owner)
			require.Equal(t, tc.expectedRepo, repo)
			require.Equal(t, tc.expectedHost, host)
		})
	}
}

func Test_buildCommitURL(t *testing.T) {
	testCases := []struct {
		name     string
		repoURL  string
		sha      string
		expected string
	}{
		{
			name:     "standard GitHub URL",
			repoURL:  "https://github.com/owner/repo",
			sha:      "abc123",
			expected: "https://github.com/owner/repo/commit/abc123",
		},
		{
			name:     "URL with .git suffix",
			repoURL:  "https://github.com/owner/repo.git",
			sha:      "abc123",
			expected: "https://github.com/owner/repo/commit/abc123",
		},
		{
			name:     "GitHub Enterprise URL",
			repoURL:  "https://github.example.com/owner/repo",
			sha:      "def456",
			expected: "https://github.example.com/owner/repo/commit/def456",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildCommitURL(tc.repoURL, tc.sha)
			require.Equal(t, tc.expected, result)
		})
	}
}

func Test_githubVerifiedPusher_cleanupStagingRef(t *testing.T) {
	testCases := []struct {
		name      string
		deleteErr error
	}{
		{
			name:      "successful cleanup",
			deleteErr: nil,
		},
		{
			name:      "cleanup failure is non-fatal",
			deleteErr: fmt.Errorf("delete error"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			g := &githubVerifiedPusher{}
			client := &mockGitHubVerifiedPushClient{
				deleteRefFn: func(
					_ context.Context,
					_, _, _ string,
				) (*github.Response, error) {
					return nil, tc.deleteErr
				},
			}
			// Should not panic or return error — cleanup is best-effort.
			g.cleanupStagingRef(
				context.Background(),
				client, "owner", "repo",
				"refs/kargo/staging/test-promo",
			)
		})
	}
}

func Test_githubVerifiedPusher_isSystemAuthor(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		gitUser git.User
		author  string
		email   string
		expect  bool
	}{
		{
			name: "matches default identity",
			gitUser: git.User{
				Name:  git.DefaultUsername,
				Email: git.DefaultEmail,
			},
			author: "Kargo",
			email:  "no-reply@kargo.io",
			expect: true,
		},
		{
			name: "does not match custom author against defaults",
			gitUser: git.User{
				Name:  git.DefaultUsername,
				Email: git.DefaultEmail,
			},
			author: "Alice",
			email:  "alice@example.com",
			expect: false,
		},
		{
			name: "matches configured system identity",
			gitUser: git.User{
				Name:  "MyKargo",
				Email: "kargo@corp.com",
			},
			author: "MyKargo",
			email:  "kargo@corp.com",
			expect: true,
		},
		{
			name: "does not match when name differs",
			gitUser: git.User{
				Name:  "MyKargo",
				Email: "kargo@corp.com",
			},
			author: "Kargo",
			email:  "kargo@corp.com",
			expect: false,
		},
		{
			name: "does not match when email differs",
			gitUser: git.User{
				Name:  "Kargo",
				Email: "kargo@corp.com",
			},
			author: "Kargo",
			email:  "no-reply@kargo.io",
			expect: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := &githubVerifiedPusher{gitUser: tc.gitUser}
			require.Equal(
				t, tc.expect,
				g.isSystemAuthor(tc.author, tc.email),
			)
		})
	}
}

func Test_appendCoAuthoredBy(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		message string
		coName  string
		coEmail string
		expect  string
	}{
		{
			name:    "simple message",
			message: "fix: something",
			coName:  "Alice",
			coEmail: "alice@example.com",
			expect:  "fix: something\n\nCo-authored-by: Alice <alice@example.com>",
		},
		{
			name:    "message ending with newline",
			message: "fix: something\n",
			coName:  "Alice",
			coEmail: "alice@example.com",
			expect:  "fix: something\n\nCo-authored-by: Alice <alice@example.com>",
		},
		{
			name:    "message ending with double newline",
			message: "fix: something\n\n",
			coName:  "Alice",
			coEmail: "alice@example.com",
			expect:  "fix: something\n\nCo-authored-by: Alice <alice@example.com>",
		},
		{
			name:    "multiline message",
			message: "fix: something\n\nLonger description here.",
			coName:  "Bob",
			coEmail: "bob@example.com",
			expect: "fix: something\n\nLonger description here." +
				"\n\nCo-authored-by: Bob <bob@example.com>",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := appendCoAuthoredBy(tc.message, tc.coName, tc.coEmail)
			require.Equal(t, tc.expect, result)
		})
	}
}

func Test_githubVerifiedPusher_run(t *testing.T) {
	t.Parallel()

	// Helper to build a minimal githubVerifiedPusher with the pluggable
	// fields wired. Tests override individual fields as needed.
	newTestPusher := func() *githubVerifiedPusher {
		return &githubVerifiedPusher{
			cfg:       githubVerifiedPusherConfig{MaxRevisions: 10},
			branchMus: map[string]*sync.Mutex{},
		}
	}

	testCases := []struct {
		name   string
		pusher *githubVerifiedPusher
		cfg    builtinx.GitHubVerifiedPushConfig
		assert func(*testing.T, promotion.StepResult, error)
	}{
		{
			name: "LoadWorkTree error on first call",
			pusher: func() *githubVerifiedPusher {
				g := newTestPusher()
				g.loadWorkTreeFn = func(
					_ string, _ *git.LoadWorkTreeOptions,
				) (git.WorkTree, error) {
					return nil, fmt.Errorf("bad worktree")
				}
				return g
			}(),
			cfg: builtinx.GitHubVerifiedPushConfig{Path: "repo"},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.Contains(
					t, err.Error(), "error loading working tree",
				)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusErrored,
					result.Status,
				)
			},
		},
		{
			name: "credentials fetch error",
			pusher: func() *githubVerifiedPusher {
				g := newTestPusher()
				g.loadWorkTreeFn = func(
					_ string, _ *git.LoadWorkTreeOptions,
				) (git.WorkTree, error) {
					return &mockWorkTree{
						url: "https://github.com/owner/repo",
					}, nil
				}
				g.credsDB = &credentials.FakeDB{
					GetFn: func(
						_ context.Context, _ string,
						_ credentials.Type, _ string,
					) (*credentials.Credentials, error) {
						return nil, fmt.Errorf("creds error")
					},
				}
				return g
			}(),
			cfg: builtinx.GitHubVerifiedPushConfig{Path: "repo"},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.Contains(
					t, err.Error(), "error getting credentials",
				)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusErrored,
					result.Status,
				)
			},
		},
		{
			name: "no token",
			pusher: func() *githubVerifiedPusher {
				g := newTestPusher()
				g.loadWorkTreeFn = func(
					_ string, _ *git.LoadWorkTreeOptions,
				) (git.WorkTree, error) {
					return &mockWorkTree{
						url: "https://github.com/owner/repo",
					}, nil
				}
				g.credsDB = &credentials.FakeDB{}
				return g
			}(),
			cfg: builtinx.GitHubVerifiedPushConfig{Path: "repo"},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.True(t, promotion.IsTerminal(err))
				require.Contains(
					t, err.Error(), "no credentials",
				)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusFailed,
					result.Status,
				)
			},
		},
		{
			name: "LoadWorkTree error on second call (with creds)",
			pusher: func() *githubVerifiedPusher {
				g := newTestPusher()
				callCount := 0
				g.loadWorkTreeFn = func(
					_ string, _ *git.LoadWorkTreeOptions,
				) (git.WorkTree, error) {
					callCount++
					if callCount == 1 {
						return &mockWorkTree{
							url: "https://github.com/owner/repo",
						}, nil
					}
					return nil, fmt.Errorf("reload error")
				}
				g.credsDB = &credentials.FakeDB{
					GetFn: func(
						_ context.Context, _ string,
						_ credentials.Type, _ string,
					) (*credentials.Credentials, error) {
						return &credentials.Credentials{
							Username: "x",
							Password: "token123",
						}, nil
					},
				}
				return g
			}(),
			cfg: builtinx.GitHubVerifiedPushConfig{Path: "repo"},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.Contains(
					t, err.Error(), "error loading working tree",
				)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusErrored,
					result.Status,
				)
			},
		},
		{
			name: "CurrentBranch error",
			pusher: func() *githubVerifiedPusher {
				g := newTestPusher()
				wt := &mockWorkTree{
					url: "https://github.com/owner/repo",
					currentBranchFn: func() (string, error) {
						return "", fmt.Errorf("branch error")
					},
				}
				g.loadWorkTreeFn = func(
					_ string, _ *git.LoadWorkTreeOptions,
				) (git.WorkTree, error) {
					return wt, nil
				}
				g.credsDB = &credentials.FakeDB{
					GetFn: func(
						_ context.Context, _ string,
						_ credentials.Type, _ string,
					) (*credentials.Credentials, error) {
						return &credentials.Credentials{
							Password: "token123",
						}, nil
					},
				}
				return g
			}(),
			cfg: builtinx.GitHubVerifiedPushConfig{Path: "repo"},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.Contains(
					t, err.Error(), "error getting current branch",
				)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusErrored,
					result.Status,
				)
			},
		},
		{
			name: "LastCommitID error",
			pusher: func() *githubVerifiedPusher {
				g := newTestPusher()
				wt := &mockWorkTree{
					url: "https://github.com/owner/repo",
					currentBranchFn: func() (string, error) {
						return "main", nil
					},
					lastCommitIDFn: func() (string, error) {
						return "", fmt.Errorf("commit error")
					},
				}
				g.loadWorkTreeFn = func(
					_ string, _ *git.LoadWorkTreeOptions,
				) (git.WorkTree, error) {
					return wt, nil
				}
				g.credsDB = &credentials.FakeDB{
					GetFn: func(
						_ context.Context, _ string,
						_ credentials.Type, _ string,
					) (*credentials.Credentials, error) {
						return &credentials.Credentials{
							Password: "token123",
						}, nil
					},
				}
				return g
			}(),
			cfg: builtinx.GitHubVerifiedPushConfig{Path: "repo"},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.Contains(
					t, err.Error(), "error getting local HEAD",
				)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusErrored,
					result.Status,
				)
			},
		},
		{
			name: "push to staging ref error",
			pusher: func() *githubVerifiedPusher {
				g := newTestPusher()
				wt := &mockWorkTree{
					url: "https://github.com/owner/repo",
					currentBranchFn: func() (string, error) {
						return "main", nil
					},
					lastCommitIDFn: func() (string, error) {
						return "abc123", nil
					},
					pushFn: func(_ *git.PushOptions) error {
						return fmt.Errorf("push error")
					},
				}
				g.loadWorkTreeFn = func(
					_ string, _ *git.LoadWorkTreeOptions,
				) (git.WorkTree, error) {
					return wt, nil
				}
				g.credsDB = &credentials.FakeDB{
					GetFn: func(
						_ context.Context, _ string,
						_ credentials.Type, _ string,
					) (*credentials.Credentials, error) {
						return &credentials.Credentials{
							Password: "token123",
						}, nil
					},
				}
				return g
			}(),
			cfg: builtinx.GitHubVerifiedPushConfig{Path: "repo"},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.Contains(
					t, err.Error(), "error pushing to staging ref",
				)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusErrored,
					result.Status,
				)
			},
		},
		{
			name: "newGitHubClient error",
			pusher: func() *githubVerifiedPusher {
				g := newTestPusher()
				wt := &mockWorkTree{
					url: "https://github.com/owner/repo",
					currentBranchFn: func() (string, error) {
						return "main", nil
					},
					lastCommitIDFn: func() (string, error) {
						return "abc123", nil
					},
					pushFn: func(_ *git.PushOptions) error {
						return nil
					},
				}
				g.loadWorkTreeFn = func(
					_ string, _ *git.LoadWorkTreeOptions,
				) (git.WorkTree, error) {
					return wt, nil
				}
				g.credsDB = &credentials.FakeDB{
					GetFn: func(
						_ context.Context, _ string,
						_ credentials.Type, _ string,
					) (*credentials.Credentials, error) {
						return &credentials.Credentials{
							Password: "token123",
						}, nil
					},
				}
				g.newGitHubClientFn = func(
					_, _ string, _ bool,
				) (string, string, githubVerifiedPushClient, error) {
					return "", "", nil,
						fmt.Errorf("client error")
				}
				return g
			}(),
			cfg: builtinx.GitHubVerifiedPushConfig{Path: "repo"},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.Contains(
					t, err.Error(), "error creating GitHub client",
				)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusErrored,
					result.Status,
				)
			},
		},
		{
			name: "GetRef error (existing branch)",
			pusher: func() *githubVerifiedPusher {
				g := newTestPusher()
				wt := &mockWorkTree{
					url: "https://github.com/owner/repo",
					currentBranchFn: func() (string, error) {
						return "main", nil
					},
					lastCommitIDFn: func() (string, error) {
						return "abc123", nil
					},
					pushFn: func(_ *git.PushOptions) error {
						return nil
					},
				}
				g.loadWorkTreeFn = func(
					_ string, _ *git.LoadWorkTreeOptions,
				) (git.WorkTree, error) {
					return wt, nil
				}
				g.credsDB = &credentials.FakeDB{
					GetFn: func(
						_ context.Context, _ string,
						_ credentials.Type, _ string,
					) (*credentials.Credentials, error) {
						return &credentials.Credentials{
							Password: "token123",
						}, nil
					},
				}
				g.newGitHubClientFn = func(
					_, _ string, _ bool,
				) (string, string, githubVerifiedPushClient, error) {
					return "owner", "repo",
						&mockGitHubVerifiedPushClient{
							getRefFn: func(
								_ context.Context,
								_, _, _ string,
							) (*github.Reference, *github.Response, error) {
								return nil, nil,
									fmt.Errorf("ref error")
							},
							deleteRefFn: func(
								_ context.Context,
								_, _, _ string,
							) (*github.Response, error) {
								return nil, nil
							},
						}, nil
				}
				return g
			}(),
			cfg: builtinx.GitHubVerifiedPushConfig{
				Path:         "repo",
				TargetBranch: "main",
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.Contains(
					t, err.Error(), "error getting ref",
				)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusErrored,
					result.Status,
				)
			},
		},
		{
			name: "GetRef error (generate target branch)",
			pusher: func() *githubVerifiedPusher {
				g := newTestPusher()
				wt := &mockWorkTree{
					url: "https://github.com/owner/repo",
					currentBranchFn: func() (string, error) {
						return "main", nil
					},
					lastCommitIDFn: func() (string, error) {
						return "abc123", nil
					},
					pushFn: func(_ *git.PushOptions) error {
						return nil
					},
				}
				g.loadWorkTreeFn = func(
					_ string, _ *git.LoadWorkTreeOptions,
				) (git.WorkTree, error) {
					return wt, nil
				}
				g.credsDB = &credentials.FakeDB{
					GetFn: func(
						_ context.Context, _ string,
						_ credentials.Type, _ string,
					) (*credentials.Credentials, error) {
						return &credentials.Credentials{
							Password: "token123",
						}, nil
					},
				}
				g.newGitHubClientFn = func(
					_, _ string, _ bool,
				) (string, string, githubVerifiedPushClient, error) {
					return "owner", "repo",
						&mockGitHubVerifiedPushClient{
							getRefFn: func(
								_ context.Context,
								_, _, _ string,
							) (*github.Reference, *github.Response, error) {
								return nil, nil,
									fmt.Errorf("source ref error")
							},
							deleteRefFn: func(
								_ context.Context,
								_, _, _ string,
							) (*github.Response, error) {
								return nil, nil
							},
						}, nil
				}
				return g
			}(),
			cfg: builtinx.GitHubVerifiedPushConfig{
				Path:                 "repo",
				GenerateTargetBranch: true,
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.Contains(
					t, err.Error(),
					"error getting source branch ref",
				)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusErrored,
					result.Status,
				)
			},
		},
		{
			name: "happy path: existing branch delegates to signAndUpdate",
			pusher: func() *githubVerifiedPusher {
				g := newTestPusher()
				wt := &mockWorkTree{
					url: "https://github.com/owner/repo",
					currentBranchFn: func() (string, error) {
						return "main", nil
					},
					lastCommitIDFn: func() (string, error) {
						return "local123", nil
					},
					pushFn: func(_ *git.PushOptions) error {
						return nil
					},
				}
				g.loadWorkTreeFn = func(
					_ string, _ *git.LoadWorkTreeOptions,
				) (git.WorkTree, error) {
					return wt, nil
				}
				g.credsDB = &credentials.FakeDB{
					GetFn: func(
						_ context.Context, _ string,
						_ credentials.Type, _ string,
					) (*credentials.Credentials, error) {
						return &credentials.Credentials{
							Password: "token123",
						}, nil
					},
				}
				treeSHA := "tree456"
				newSHA := "signed789"
				g.newGitHubClientFn = func(
					_, _ string, _ bool,
				) (string, string, githubVerifiedPushClient, error) {
					return "owner", "repo",
						&mockGitHubVerifiedPushClient{
							getRefFn: func(
								_ context.Context,
								_, _, _ string,
							) (*github.Reference, *github.Response, error) {
								return &github.Reference{
									Object: &github.GitObject{
										SHA: ptr.To("target000"),
									},
								}, nil, nil
							},
							compareCommitsFn: func(
								_ context.Context,
								_, _, _, _ string,
								_ *github.ListOptions,
							) (*github.CommitsComparison, *github.Response, error) {
								return &github.CommitsComparison{
									Status:       ptr.To("ahead"),
									AheadBy:      ptr.To(1),
									TotalCommits: ptr.To(1),
									Commits: []*github.RepositoryCommit{{
										SHA: ptr.To("orig111"),
										Commit: &github.Commit{
											Message: ptr.To("test commit"),
											Tree:    &github.Tree{SHA: &treeSHA},
											Author: &github.CommitAuthor{
												Name:  ptr.To("Test"),
												Email: ptr.To("test@test.com"),
											},
										},
									}},
								}, nil, nil
							},
							createCommitFn: func(
								_ context.Context,
								_, _ string,
								_ github.Commit,
								_ *github.CreateCommitOptions,
							) (*github.Commit, *github.Response, error) {
								return &github.Commit{
									SHA: &newSHA,
								}, nil, nil
							},
							updateRefFn: func(
								_ context.Context,
								_, _, _ string,
								_ github.UpdateRef,
							) (*github.Reference, *github.Response, error) {
								return &github.Reference{}, nil, nil
							},
							deleteRefFn: func(
								_ context.Context,
								_, _, _ string,
							) (*github.Response, error) {
								return nil, nil
							},
						}, nil
				}
				return g
			}(),
			cfg: builtinx.GitHubVerifiedPushConfig{
				Path:         "repo",
				TargetBranch: "main",
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusSucceeded,
					result.Status,
				)
				assert.Equal(
					t, "signed789", result.Output[stateKeyCommit],
				)
				assert.Equal(
					t, "main", result.Output[stateKeyBranch],
				)
			},
		},
		{
			name: "happy path: generateTargetBranch",
			pusher: func() *githubVerifiedPusher {
				g := newTestPusher()
				wt := &mockWorkTree{
					url: "https://github.com/owner/repo",
					currentBranchFn: func() (string, error) {
						return "main", nil
					},
					lastCommitIDFn: func() (string, error) {
						return "local123", nil
					},
					pushFn: func(_ *git.PushOptions) error {
						return nil
					},
				}
				g.loadWorkTreeFn = func(
					_ string, _ *git.LoadWorkTreeOptions,
				) (git.WorkTree, error) {
					return wt, nil
				}
				g.credsDB = &credentials.FakeDB{
					GetFn: func(
						_ context.Context, _ string,
						_ credentials.Type, _ string,
					) (*credentials.Credentials, error) {
						return &credentials.Credentials{
							Password: "token123",
						}, nil
					},
				}
				treeSHA := "tree456"
				newSHA := "signed789"
				g.newGitHubClientFn = func(
					_, _ string, _ bool,
				) (string, string, githubVerifiedPushClient, error) {
					return "owner", "repo",
						&mockGitHubVerifiedPushClient{
							getRefFn: func(
								_ context.Context,
								_, _, ref string,
							) (*github.Reference, *github.Response, error) {
								// For generateTargetBranch, GetRef
								// is called on the source branch.
								assert.Equal(
									t, "heads/main", ref,
								)
								return &github.Reference{
									Object: &github.GitObject{
										SHA: ptr.To("target000"),
									},
								}, nil, nil
							},
							compareCommitsFn: func(
								_ context.Context,
								_, _, _, _ string,
								_ *github.ListOptions,
							) (*github.CommitsComparison, *github.Response, error) {
								return &github.CommitsComparison{
									Status:       ptr.To("ahead"),
									AheadBy:      ptr.To(1),
									TotalCommits: ptr.To(1),
									Commits: []*github.RepositoryCommit{{
										SHA: ptr.To("orig111"),
										Commit: &github.Commit{
											Message: ptr.To("test"),
											Tree:    &github.Tree{SHA: &treeSHA},
											Author: &github.CommitAuthor{
												Name:  ptr.To("Test"),
												Email: ptr.To("t@t.com"),
											},
										},
									}},
								}, nil, nil
							},
							createCommitFn: func(
								_ context.Context,
								_, _ string,
								_ github.Commit,
								_ *github.CreateCommitOptions,
							) (*github.Commit, *github.Response, error) {
								return &github.Commit{
									SHA: &newSHA,
								}, nil, nil
							},
							createRefFn: func(
								_ context.Context,
								_, _ string,
								ref github.CreateRef,
							) (*github.Reference, *github.Response, error) {
								assert.Contains(
									t,
									ref.Ref,
									"kargo/promotion/",
								)
								return &github.Reference{}, nil, nil
							},
							deleteRefFn: func(
								_ context.Context,
								_, _, _ string,
							) (*github.Response, error) {
								return nil, nil
							},
						}, nil
				}
				return g
			}(),
			cfg: builtinx.GitHubVerifiedPushConfig{
				Path:                 "repo",
				GenerateTargetBranch: true,
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusSucceeded,
					result.Status,
				)
				assert.Equal(
					t, "signed789", result.Output[stateKeyCommit],
				)
				assert.Contains(
					t,
					result.Output[stateKeyBranch],
					"kargo/promotion/",
				)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := tc.pusher.run(
				context.Background(),
				&promotion.StepContext{
					WorkDir:   t.TempDir(),
					Project:   "test-project",
					Promotion: "test-promo-123",
				},
				tc.cfg,
			)
			tc.assert(t, result, err)
		})
	}
}
