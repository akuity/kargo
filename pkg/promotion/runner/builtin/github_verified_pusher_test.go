package builtin

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/promotion"
)

// mockWorkTree is a minimal mock of git.WorkTree for testing.
type mockWorkTree struct {
	git.WorkTree
	url                       string
	dir                       string
	homeDir                   string
	commitSignatureStatusesFn func(
		ids []string,
	) (map[string]git.CommitSignatureInfo, error)
}

func (m *mockWorkTree) URL() string     { return m.url }
func (m *mockWorkTree) Dir() string     { return m.dir }
func (m *mockWorkTree) HomeDir() string { return m.homeDir }

func (m *mockWorkTree) CommitSignatureStatuses(
	ids []string,
) (map[string]git.CommitSignatureInfo, error) {
	return m.commitSignatureStatusesFn(ids)
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
	r := newGitHubVerifiedPusher(promotion.StepRunnerCapabilities{})
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
			name: "diverged commits",
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
			}
			targetBranch := tc.targetBranch
			if targetBranch == "" {
				targetBranch = "main"
			}
			result, err := g.signAndUpdate(
				context.Background(),
				tc.client,
				"owner", "repo",
				targetBranch, tc.createBranch,
				"abc123",
				"def456",
				&mockWorkTree{url: "https://github.com/owner/repo"},
			)
			tc.assert(t, result, err)
		})
	}
}

func Test_githubVerifiedPusher_getPreviousOutput(t *testing.T) {
	testCases := []struct {
		name   string
		state  promotion.State
		alias  string
		assert func(*testing.T, map[string]any, error)
	}{
		{
			name:  "no previous output",
			state: promotion.State{},
			alias: "my-step",
			assert: func(t *testing.T, output map[string]any, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Nil(t, output)
			},
		},
		{
			name: "previous output exists with commit",
			state: promotion.State{
				"my-step": map[string]any{
					"commit": "abc123",
					"branch": "main",
				},
			},
			alias: "my-step",
			assert: func(t *testing.T, output map[string]any, err error) {
				t.Helper()
				require.NoError(t, err)
				require.NotNil(t, output)
				require.Equal(t, "abc123", output["commit"])
			},
		},
		{
			name: "previous output exists without commit",
			state: promotion.State{
				"my-step": map[string]any{
					"branch": "main",
				},
			},
			alias: "my-step",
			assert: func(t *testing.T, output map[string]any, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Nil(t, output)
			},
		},
		{
			name: "previous output is wrong type",
			state: promotion.State{
				"my-step": "not a map",
			},
			alias: "my-step",
			assert: func(t *testing.T, _ map[string]any, err error) {
				t.Helper()
				require.Error(t, err)
				require.Contains(t, err.Error(), "not a map[string]any")
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := &githubVerifiedPusher{}
			output, err := g.getPreviousOutput(
				&promotion.StepContext{
					Alias:       tc.alias,
					SharedState: tc.state,
				},
			)
			tc.assert(t, output, err)
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

func Test_githubVerifiedPusher_buildCommitURL(t *testing.T) {
	g := &githubVerifiedPusher{}
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
			result := g.buildCommitURL(tc.repoURL, tc.sha)
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

func Test_githubVerifiedPusher_verifyCommitSignatures(t *testing.T) {
	testCases := []struct {
		name     string
		gitUser  git.User
		workTree *mockWorkTree
		commits  []*github.RepositoryCommit
		assert   func(*testing.T, map[string]git.CommitSignatureInfo, error)
	}{
		{
			name:     "returns nil map when no signing key is configured",
			workTree: &mockWorkTree{},
			commits: []*github.RepositoryCommit{
				{SHA: ptr.To("abc123")},
			},
			assert: func(
				t *testing.T,
				statuses map[string]git.CommitSignatureInfo,
				err error,
			) {
				require.NoError(t, err)
				require.Nil(t, statuses)
			},
		},
		{
			name:    "CommitSignatureStatuses error",
			gitUser: git.User{SigningKeyPath: "/some/key"},
			workTree: &mockWorkTree{
				commitSignatureStatusesFn: func(
					_ []string,
				) (map[string]git.CommitSignatureInfo, error) {
					return nil, fmt.Errorf("git command failed")
				},
			},
			commits: []*github.RepositoryCommit{
				{SHA: ptr.To("abc123")},
			},
			assert: func(
				t *testing.T,
				_ map[string]git.CommitSignatureInfo,
				err error,
			) {
				require.ErrorContains(t, err, "git command failed")
			},
		},
		{
			name:    "propagates unsigned commits without error",
			gitUser: git.User{SigningKeyPath: "/some/key"},
			workTree: &mockWorkTree{
				commitSignatureStatusesFn: func(
					ids []string,
				) (map[string]git.CommitSignatureInfo, error) {
					result := make(map[string]git.CommitSignatureInfo)
					for _, id := range ids {
						result[id] = git.CommitSignatureInfo{Status: "N"}
					}
					return result, nil
				},
			},
			commits: []*github.RepositoryCommit{
				{SHA: ptr.To("abc123")},
				{SHA: ptr.To("def456")},
			},
			assert: func(
				t *testing.T,
				statuses map[string]git.CommitSignatureInfo,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, statuses)
				require.Equal(t, "N", statuses["abc123"].Status)
				require.Equal(t, "N", statuses["def456"].Status)
			},
		},
		{
			name:    "returns statuses for good signatures",
			gitUser: git.User{SigningKeyPath: "/some/key"},
			workTree: &mockWorkTree{
				commitSignatureStatusesFn: func(
					_ []string,
				) (map[string]git.CommitSignatureInfo, error) {
					return map[string]git.CommitSignatureInfo{
						"abc123": {Status: "G", KeyID: "ABCD1234", Signer: "Kargo"},
					}, nil
				},
			},
			commits: []*github.RepositoryCommit{
				{SHA: ptr.To("abc123")},
			},
			assert: func(
				t *testing.T,
				statuses map[string]git.CommitSignatureInfo,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, statuses)
				require.Equal(t, "G", statuses["abc123"].Status)
				require.Equal(t, "ABCD1234", statuses["abc123"].KeyID)
			},
		},
		{
			name:    "propagates commits with untrusted signatures",
			gitUser: git.User{SigningKeyPath: "/some/key"},
			workTree: &mockWorkTree{
				commitSignatureStatusesFn: func(
					_ []string,
				) (map[string]git.CommitSignatureInfo, error) {
					return map[string]git.CommitSignatureInfo{
						"abc123": {Status: "U", KeyID: "ABCD1234", Signer: "Kargo"},
					}, nil
				},
			},
			commits: []*github.RepositoryCommit{
				{SHA: ptr.To("abc123")},
			},
			assert: func(
				t *testing.T,
				statuses map[string]git.CommitSignatureInfo,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, statuses)
				require.Equal(t, "U", statuses["abc123"].Status)
			},
		},
		{
			name:    "rejects commits with bad signatures",
			gitUser: git.User{SigningKeyPath: "/some/key"},
			workTree: &mockWorkTree{
				commitSignatureStatusesFn: func(
					_ []string,
				) (map[string]git.CommitSignatureInfo, error) {
					return map[string]git.CommitSignatureInfo{
						"abc123": {Status: "B"},
					}, nil
				},
			},
			commits: []*github.RepositoryCommit{
				{SHA: ptr.To("abc123")},
			},
			assert: func(
				t *testing.T,
				_ map[string]git.CommitSignatureInfo,
				err error,
			) {
				require.Error(t, err)
				assert.IsType(t, &promotion.TerminalError{}, err)
				assert.ErrorContains(t, err, "bad GPG signature")
			},
		},
		{
			name:    "propagates commits signed by unknown key",
			gitUser: git.User{SigningKeyPath: "/some/key"},
			workTree: &mockWorkTree{
				commitSignatureStatusesFn: func(
					_ []string,
				) (map[string]git.CommitSignatureInfo, error) {
					return map[string]git.CommitSignatureInfo{
						"abc123": {Status: "E", KeyID: "UNKNOWN1", Signer: "Other"},
					}, nil
				},
			},
			commits: []*github.RepositoryCommit{
				{SHA: ptr.To("abc123")},
			},
			assert: func(
				t *testing.T,
				statuses map[string]git.CommitSignatureInfo,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, statuses)
				require.Equal(t, "E", statuses["abc123"].Status)
			},
		},
		{
			name:    "accepts commits with expired signature",
			gitUser: git.User{SigningKeyPath: "/some/key"},
			workTree: &mockWorkTree{
				commitSignatureStatusesFn: func(
					_ []string,
				) (map[string]git.CommitSignatureInfo, error) {
					return map[string]git.CommitSignatureInfo{
						"abc123": {Status: "X", KeyID: "ABCD1234"},
					}, nil
				},
			},
			commits: []*github.RepositoryCommit{
				{SHA: ptr.To("abc123")},
			},
			assert: func(
				t *testing.T,
				statuses map[string]git.CommitSignatureInfo,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, statuses)
				require.Equal(t, "X", statuses["abc123"].Status)
			},
		},
		{
			name:    "accepts commits with expired key",
			gitUser: git.User{SigningKeyPath: "/some/key"},
			workTree: &mockWorkTree{
				commitSignatureStatusesFn: func(
					_ []string,
				) (map[string]git.CommitSignatureInfo, error) {
					return map[string]git.CommitSignatureInfo{
						"abc123": {Status: "Y", KeyID: "ABCD1234"},
					}, nil
				},
			},
			commits: []*github.RepositoryCommit{
				{SHA: ptr.To("abc123")},
			},
			assert: func(
				t *testing.T,
				statuses map[string]git.CommitSignatureInfo,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, statuses)
				require.Equal(t, "Y", statuses["abc123"].Status)
			},
		},
		{
			name:    "rejects commits with revoked key",
			gitUser: git.User{SigningKeyPath: "/some/key"},
			workTree: &mockWorkTree{
				commitSignatureStatusesFn: func(
					_ []string,
				) (map[string]git.CommitSignatureInfo, error) {
					return map[string]git.CommitSignatureInfo{
						"abc123": {Status: "R"},
					}, nil
				},
			},
			commits: []*github.RepositoryCommit{
				{SHA: ptr.To("abc123")},
			},
			assert: func(
				t *testing.T,
				_ map[string]git.CommitSignatureInfo,
				err error,
			) {
				require.Error(t, err)
				assert.IsType(t, &promotion.TerminalError{}, err)
				assert.ErrorContains(t, err, "revoked key")
			},
		},
		{
			name:    "skips commits with empty SHA",
			gitUser: git.User{SigningKeyPath: "/some/key"},
			workTree: &mockWorkTree{
				commitSignatureStatusesFn: func(
					ids []string,
				) (map[string]git.CommitSignatureInfo, error) {
					require.Equal(t, []string{"def456"}, ids)
					return map[string]git.CommitSignatureInfo{
						"def456": {Status: "G", KeyID: "ABCD1234", Signer: "Kargo"},
					}, nil
				},
			},
			commits: []*github.RepositoryCommit{
				{SHA: ptr.To("")},
				{SHA: ptr.To("def456")},
			},
			assert: func(
				t *testing.T,
				statuses map[string]git.CommitSignatureInfo,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, statuses)
			},
		},
		{
			name:    "mixed signed and unsigned commits",
			gitUser: git.User{SigningKeyPath: "/some/key"},
			workTree: &mockWorkTree{
				commitSignatureStatusesFn: func(
					_ []string,
				) (map[string]git.CommitSignatureInfo, error) {
					return map[string]git.CommitSignatureInfo{
						"signed1":   {Status: "G", KeyID: "ABCD1234", Signer: "Kargo"},
						"unsigned1": {Status: "N"},
					}, nil
				},
			},
			commits: []*github.RepositoryCommit{
				{SHA: ptr.To("signed1")},
				{SHA: ptr.To("unsigned1")},
			},
			assert: func(
				t *testing.T,
				statuses map[string]git.CommitSignatureInfo,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, statuses)
				require.Equal(t, "G", statuses["signed1"].Status)
				require.Equal(t, "N", statuses["unsigned1"].Status)
			},
		},
		{
			name:    "stops at first bad signature",
			gitUser: git.User{SigningKeyPath: "/some/key"},
			workTree: &mockWorkTree{
				commitSignatureStatusesFn: func(
					_ []string,
				) (map[string]git.CommitSignatureInfo, error) {
					return map[string]git.CommitSignatureInfo{
						"bad1":  {Status: "B"},
						"good1": {Status: "G"},
					}, nil
				},
			},
			commits: []*github.RepositoryCommit{
				{SHA: ptr.To("bad1")},
				{SHA: ptr.To("good1")},
			},
			assert: func(
				t *testing.T,
				_ map[string]git.CommitSignatureInfo,
				err error,
			) {
				require.Error(t, err)
				assert.IsType(t, &promotion.TerminalError{}, err)
			},
		},
		{
			name:     "returns nil map for empty commit list",
			gitUser:  git.User{SigningKeyPath: "/some/key"},
			workTree: &mockWorkTree{},
			commits:  []*github.RepositoryCommit{},
			assert: func(
				t *testing.T,
				statuses map[string]git.CommitSignatureInfo,
				err error,
			) {
				require.NoError(t, err)
				require.Nil(t, statuses)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := &githubVerifiedPusher{gitUser: tc.gitUser}
			statuses, err := g.verifyCommitSignatures(
				context.Background(),
				tc.workTree,
				tc.commits,
			)
			tc.assert(t, statuses, err)
		})
	}
}
