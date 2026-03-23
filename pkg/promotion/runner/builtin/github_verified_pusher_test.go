package builtin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/credentials"
	kargogithub "github.com/akuity/kargo/pkg/github"
	"github.com/akuity/kargo/pkg/promotion"
	builtinx "github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func policyPtr(p builtinx.PullPolicy) *builtinx.PullPolicy { return &p }

// mockWorkTree is a minimal mock of git.WorkTree for testing.
type mockWorkTree struct {
	git.WorkTree
	url                       string
	dir                       string
	homeDir                   string
	currentBranchFn           func() (string, error)
	lastCommitIDFn            func() (string, error)
	pullMergeFn               func(string) error
	pullRebaseFn              func(string) error
	pushFn                    func(*git.PushOptions) error
	forcePullFn               func(string) error
	commitSignatureStatusesFn func(ids []string) (map[string]git.CommitSignatureInfo, error)
	importGPGKeyFn func(keyData string) (string, error)
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

func (m *mockWorkTree) PullMerge(branch string) error {
	if m.pullMergeFn == nil {
		return nil
	}
	return m.pullMergeFn(branch)
}

func (m *mockWorkTree) PullRebase(branch string) error {
	if m.pullRebaseFn == nil {
		return nil
	}
	return m.pullRebaseFn(branch)
}

func (m *mockWorkTree) Push(opts *git.PushOptions) error {
	return m.pushFn(opts)
}

func (m *mockWorkTree) ForcePull(branch string) error {
	if m.forcePullFn == nil {
		return nil
	}
	return m.forcePullFn(branch)
}

func (m *mockWorkTree) CommitSignatureStatuses(
	ids []string,
) (map[string]git.CommitSignatureInfo, error) {
	if m.commitSignatureStatusesFn == nil {
		return nil, nil
	}
	return m.commitSignatureStatusesFn(ids)
}

func (m *mockWorkTree) ImportGPGKey(keyData string) (string, error) {
	if m.importGPGKeyFn == nil {
		return "", nil
	}
	return m.importGPGKeyFn(keyData)
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
		git.User{},
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
	if m.deleteRefFn == nil {
		return nil, nil
	}
	return m.deleteRefFn(ctx, owner, repo, ref)
}

func Test_githubVerifiedPusher_compareRemote(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		client githubVerifiedPushClient
		force  bool
		assert func(*testing.T, *comparisonResult, error)
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
			assert: func(t *testing.T, _ *comparisonResult, err error) {
				t.Helper()
				require.ErrorContains(t, err, "API error")
			},
		},
		{
			name: "ahead returns commits and parentSHA",
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					treeSHA := "tree1"
					return &github.CommitsComparison{
						Status:       ptr.To("ahead"),
						AheadBy:      ptr.To(1),
						TotalCommits: ptr.To(1),
						Commits: []*github.RepositoryCommit{{
							SHA: ptr.To("commit1"),
							Commit: &github.Commit{
								Message: ptr.To("msg"),
								Tree:    &github.Tree{SHA: &treeSHA},
							},
						}},
					}, nil, nil
				},
			},
			assert: func(
				t *testing.T, cmp *comparisonResult, err error,
			) {
				t.Helper()
				require.NoError(t, err)
				require.Nil(t, cmp.earlyResult)
				require.Equal(t, "target-head", cmp.parentSHA)
				require.Len(t, cmp.commits, 1)
			},
		},
		{
			name: "identical returns early skip",
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
				t *testing.T, cmp *comparisonResult, err error,
			) {
				t.Helper()
				require.NoError(t, err)
				require.NotNil(t, cmp.earlyResult)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusSkipped,
					cmp.earlyResult.Status,
				)
			},
		},
		{
			name: "diverged without force is terminal",
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
			force: false,
			assert: func(
				t *testing.T, _ *comparisonResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.True(t, promotion.IsTerminal(err))
			},
		},
		{
			name: "diverged with force uses merge base",
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					treeSHA := "tree1"
					return &github.CommitsComparison{
						Status: ptr.To("diverged"),
						MergeBaseCommit: &github.RepositoryCommit{
							SHA: ptr.To("merge-base-sha"),
						},
						Commits: []*github.RepositoryCommit{{
							SHA: ptr.To("c1"),
							Commit: &github.Commit{
								Message: ptr.To("msg"),
								Tree:    &github.Tree{SHA: &treeSHA},
							},
						}},
					}, nil, nil
				},
			},
			force: true,
			assert: func(
				t *testing.T, cmp *comparisonResult, err error,
			) {
				t.Helper()
				require.NoError(t, err)
				require.Nil(t, cmp.earlyResult)
				require.Equal(t, "merge-base-sha", cmp.parentSHA)
				require.Len(t, cmp.commits, 1)
			},
		},
		{
			name: "too many revisions is terminal",
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
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
				t *testing.T, _ *comparisonResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.True(t, promotion.IsTerminal(err))
				require.ErrorContains(t, err, "exceeds the maximum")
			},
		},
		{
			name: "empty commits returns skip",
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
				t *testing.T, cmp *comparisonResult, err error,
			) {
				t.Helper()
				require.NoError(t, err)
				require.NotNil(t, cmp.earlyResult)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusSkipped,
					cmp.earlyResult.Status,
				)
			},
		},
		{
			name: "unknown status is terminal",
			client: &mockGitHubVerifiedPushClient{
				compareCommitsFn: func(
					_ context.Context,
					_, _, _, _ string,
					_ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return &github.CommitsComparison{
						Status: ptr.To("unknown-status"),
					}, nil, nil
				},
			},
			assert: func(
				t *testing.T, _ *comparisonResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.True(t, promotion.IsTerminal(err))
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := &githubVerifiedPusher{
				cfg: githubVerifiedPusherConfig{MaxRevisions: 10},
			}
			cmp, err := g.compareRemote(
				context.Background(),
				tc.client,
				"owner", "repo", "main",
				"target-head", "local-head",
				tc.force,
				&mockWorkTree{url: "https://github.com/o/r"},
			)
			tc.assert(t, cmp, err)
		})
	}
}

func Test_githubVerifiedPusher_replayCommits(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name      string
		pusher    *githubVerifiedPusher
		client    githubVerifiedPushClient
		commits   []*github.RepositoryCommit
		parentSHA string
		assert    func(*testing.T, string, error)
	}{
		{
			name:   "missing tree information",
			pusher: &githubVerifiedPusher{},
			client: &mockGitHubVerifiedPushClient{},
			commits: []*github.RepositoryCommit{{
				Commit: &github.Commit{},
			}},
			parentSHA: "parent",
			assert: func(t *testing.T, _ string, err error) {
				t.Helper()
				require.ErrorContains(
					t, err, "missing tree information",
				)
			},
		},
		{
			name:   "create commit API error",
			pusher: &githubVerifiedPusher{},
			client: &mockGitHubVerifiedPushClient{
				createCommitFn: func(
					_ context.Context,
					_, _ string,
					_ github.Commit,
					_ *github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					return nil, nil, fmt.Errorf("API error")
				},
			},
			commits: []*github.RepositoryCommit{{
				SHA: ptr.To("orig1"),
				Commit: &github.Commit{
					Message: ptr.To("msg"),
					Tree:    &github.Tree{SHA: ptr.To("tree1")},
				},
			}},
			parentSHA: "parent",
			assert: func(t *testing.T, _ string, err error) {
				t.Helper()
				require.ErrorContains(t, err, "API error")
			},
		},
		{
			name:   "single commit replayed successfully",
			pusher: &githubVerifiedPusher{},
			client: &mockGitHubVerifiedPushClient{
				createCommitFn: func(
					_ context.Context,
					_, _ string,
					commit github.Commit,
					_ *github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					// Verify parent is the seed parentSHA.
					require.Len(t, commit.Parents, 1)
					require.Equal(
						t, "parent", commit.Parents[0].GetSHA(),
					)
					return &github.Commit{
						SHA: ptr.To("signed1"),
					}, nil, nil
				},
			},
			commits: []*github.RepositoryCommit{{
				SHA: ptr.To("orig1"),
				Commit: &github.Commit{
					Message: ptr.To("msg"),
					Tree:    &github.Tree{SHA: ptr.To("tree1")},
				},
			}},
			parentSHA: "parent",
			assert: func(t *testing.T, sha string, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(t, "signed1", sha)
			},
		},
		{
			name:   "multiple commits chain parents correctly",
			pusher: &githubVerifiedPusher{},
			client: func() githubVerifiedPushClient {
				callCount := 0
				return &mockGitHubVerifiedPushClient{
					createCommitFn: func(
						_ context.Context,
						_, _ string,
						commit github.Commit,
						_ *github.CreateCommitOptions,
					) (*github.Commit, *github.Response, error) {
						callCount++
						if callCount == 1 {
							require.Equal(
								t, "parent",
								commit.Parents[0].GetSHA(),
							)
							return &github.Commit{
								SHA: ptr.To("signed1"),
							}, nil, nil
						}
						require.Equal(
							t, "signed1",
							commit.Parents[0].GetSHA(),
						)
						return &github.Commit{
							SHA: ptr.To("signed2"),
						}, nil, nil
					},
				}
			}(),
			commits: []*github.RepositoryCommit{
				{
					SHA: ptr.To("orig1"),
					Commit: &github.Commit{
						Message: ptr.To("first"),
						Tree:    &github.Tree{SHA: ptr.To("t1")},
					},
				},
				{
					SHA: ptr.To("orig2"),
					Commit: &github.Commit{
						Message: ptr.To("second"),
						Tree:    &github.Tree{SHA: ptr.To("t2")},
					},
				},
			},
			parentSHA: "parent",
			assert: func(t *testing.T, sha string, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(t, "signed2", sha)
			},
		},
		{
			name:   "app-authored commit omits author",
			pusher: &githubVerifiedPusher{},
			client: &mockGitHubVerifiedPushClient{
				createCommitFn: func(
					_ context.Context,
					_, _ string,
					commit github.Commit,
					_ *github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					// App-authored: Author/Committer should be nil
					// so the GitHub App signs.
					require.Nil(t, commit.Author)
					require.Nil(t, commit.Committer)
					return &github.Commit{
						SHA: ptr.To("signed1"),
					}, nil, nil
				},
			},
			commits: []*github.RepositoryCommit{{
				SHA: ptr.To("orig1"),
				Commit: &github.Commit{
					Message: ptr.To("msg"),
					Tree:    &github.Tree{SHA: ptr.To("tree1")},
					Author: &github.CommitAuthor{
						Name:  ptr.To("Kargo"),
						Email: ptr.To("no-reply@kargo.io"),
					},
					Committer: &github.CommitAuthor{
						Name:  ptr.To("Kargo"),
						Email: ptr.To("no-reply@kargo.io"),
					},
				},
			}},
			parentSHA: "parent",
			assert: func(t *testing.T, sha string, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(t, "signed1", sha)
			},
		},
		{
			name:   "non-app commit preserves author",
			pusher: &githubVerifiedPusher{},
			client: &mockGitHubVerifiedPushClient{
				createCommitFn: func(
					_ context.Context,
					_, _ string,
					commit github.Commit,
					_ *github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					// Non-app: Author/Committer preserved.
					require.NotNil(t, commit.Author)
					require.Equal(
						t, "Custom", commit.Author.GetName(),
					)
					require.NotNil(t, commit.Committer)
					require.Equal(
						t, "Custom", commit.Committer.GetName(),
					)
					return &github.Commit{
						SHA: ptr.To("signed1"),
					}, nil, nil
				},
			},
			commits: []*github.RepositoryCommit{{
				SHA: ptr.To("orig1"),
				Commit: &github.Commit{
					Message: ptr.To("msg"),
					Tree:    &github.Tree{SHA: ptr.To("tree1")},
					Author: &github.CommitAuthor{
						Name:  ptr.To("Custom"),
						Email: ptr.To("custom@test.com"),
					},
					Committer: &github.CommitAuthor{
						Name:  ptr.To("Custom"),
						Email: ptr.To("custom@test.com"),
					},
				},
			}},
			parentSHA: "parent",
			assert: func(t *testing.T, sha string, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(t, "signed1", sha)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sha, err := tc.pusher.replayCommits(
				context.Background(),
				tc.client,
				"owner", "repo",
				tc.commits,
				tc.parentSHA,
				"Kargo", "no-reply@kargo.io", "",
				nil,
			)
			tc.assert(t, sha, err)
		})
	}
}

func Test_githubVerifiedPusher_updateTargetRef(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name         string
		client       githubVerifiedPushClient
		createBranch bool
		force        bool
		assert       func(*testing.T, error)
	}{
		{
			name: "create new branch ref",
			client: &mockGitHubVerifiedPushClient{
				createRefFn: func(
					_ context.Context,
					_, _ string,
					ref github.CreateRef,
				) (*github.Reference, *github.Response, error) {
					require.Equal(
						t, "refs/heads/feature", ref.Ref,
					)
					require.Equal(t, "sha123", ref.SHA)
					return &github.Reference{}, nil, nil
				},
			},
			createBranch: true,
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name: "create ref error",
			client: &mockGitHubVerifiedPushClient{
				createRefFn: func(
					_ context.Context,
					_, _ string,
					_ github.CreateRef,
				) (*github.Reference, *github.Response, error) {
					return nil, nil, fmt.Errorf("create failed")
				},
			},
			createBranch: true,
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorContains(t, err, "create failed")
			},
		},
		{
			name: "update existing ref",
			client: &mockGitHubVerifiedPushClient{
				updateRefFn: func(
					_ context.Context,
					_, _, _ string,
					ref github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					require.Equal(t, "sha123", ref.SHA)
					require.False(t, *ref.Force)
					return &github.Reference{}, nil, nil
				},
			},
			createBranch: false,
			force:        false,
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name: "update ref with force",
			client: &mockGitHubVerifiedPushClient{
				updateRefFn: func(
					_ context.Context,
					_, _, _ string,
					ref github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					require.True(t, *ref.Force)
					return &github.Reference{}, nil, nil
				},
			},
			createBranch: false,
			force:        true,
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name: "HTTP 422 returns errRefUpdateConflict",
			client: &mockGitHubVerifiedPushClient{
				updateRefFn: func(
					_ context.Context,
					_, _, _ string,
					_ github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					return nil, nil, &github.ErrorResponse{
						Response: &http.Response{
							StatusCode: http.StatusUnprocessableEntity,
						},
					}
				},
			},
			createBranch: false,
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.True(
					t, errors.Is(err, errRefUpdateConflict),
				)
			},
		},
		{
			name: "other update error passes through",
			client: &mockGitHubVerifiedPushClient{
				updateRefFn: func(
					_ context.Context,
					_, _, _ string,
					_ github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					return nil, nil,
						fmt.Errorf("internal server error")
				},
			},
			createBranch: false,
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorContains(
					t, err, "internal server error",
				)
				require.False(
					t, errors.Is(err, errRefUpdateConflict),
				)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := &githubVerifiedPusher{}
			err := g.updateTargetRef(
				context.Background(),
				tc.client,
				"owner", "repo", "feature", "sha123",
				tc.createBranch, tc.force,
			)
			tc.assert(t, err)
		})
	}
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
			name: "app-authored commit omits author in signAndUpdate",
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
								Committer: &github.CommitAuthor{
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
					Name:  "Kargo",
					Email: "no-reply@kargo.io",
				},
			}
			targetBranch := tc.targetBranch
			if targetBranch == "" {
				targetBranch = "main"
			}
			result, err := g.signAndUpdate(
				context.Background(),
				builtinx.GitHubVerifiedPushConfig{},
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

func Test_parseRepoURL(t *testing.T) {
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
			_, host, owner, repo, err := kargogithub.ParseRepoURL(tc.repoURL)
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
			result := kargogithub.BuildCommitURL(tc.repoURL, tc.sha)
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

func Test_githubVerifiedPusher_isAppAuthored(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name           string
		rc             *github.RepositoryCommit
		appName        string
		appEmail       string
		sigStatuses    map[string]git.CommitSignatureInfo
		appFingerprint string
		expect         bool
	}{
		{
			name: "app-authored no signing",
			rc: &github.RepositoryCommit{
				SHA: ptr.To("abc"),
				Commit: &github.Commit{
					Author: &github.CommitAuthor{
						Name:  ptr.To("Kargo"),
						Email: ptr.To("no-reply@kargo.io"),
					},
					Committer: &github.CommitAuthor{
						Name:  ptr.To("Kargo"),
						Email: ptr.To("no-reply@kargo.io"),
					},
				},
			},
			appName:  "Kargo",
			appEmail: "no-reply@kargo.io",
			expect:   true,
		},
		{
			name: "app-authored with matching fingerprint",
			rc: &github.RepositoryCommit{
				SHA: ptr.To("abc"),
				Commit: &github.Commit{
					Author: &github.CommitAuthor{
						Name:  ptr.To("Kargo"),
						Email: ptr.To("no-reply@kargo.io"),
					},
					Committer: &github.CommitAuthor{
						Name:  ptr.To("Kargo"),
						Email: ptr.To("no-reply@kargo.io"),
					},
				},
			},
			appName:  "Kargo",
			appEmail: "no-reply@kargo.io",
			sigStatuses: map[string]git.CommitSignatureInfo{
				"abc": {Fingerprint: "AAAA1234"},
			},
			appFingerprint: "AAAA1234",
			expect:         true,
		},
		{
			name: "not app-authored different author",
			rc: &github.RepositoryCommit{
				SHA: ptr.To("abc"),
				Commit: &github.Commit{
					Author: &github.CommitAuthor{
						Name:  ptr.To("Alice"),
						Email: ptr.To("alice@example.com"),
					},
					Committer: &github.CommitAuthor{
						Name:  ptr.To("Alice"),
						Email: ptr.To("alice@example.com"),
					},
				},
			},
			appName:  "Kargo",
			appEmail: "no-reply@kargo.io",
			expect:   false,
		},
		{
			name: "not app-authored author!=committer",
			rc: &github.RepositoryCommit{
				SHA: ptr.To("abc"),
				Commit: &github.Commit{
					Author: &github.CommitAuthor{
						Name:  ptr.To("Kargo"),
						Email: ptr.To("no-reply@kargo.io"),
					},
					Committer: &github.CommitAuthor{
						Name:  ptr.To("Someone"),
						Email: ptr.To("someone@example.com"),
					},
				},
			},
			appName:  "Kargo",
			appEmail: "no-reply@kargo.io",
			expect:   false,
		},
		{
			name: "not app-authored fingerprint mismatch",
			rc: &github.RepositoryCommit{
				SHA: ptr.To("abc"),
				Commit: &github.Commit{
					Author: &github.CommitAuthor{
						Name:  ptr.To("Kargo"),
						Email: ptr.To("no-reply@kargo.io"),
					},
					Committer: &github.CommitAuthor{
						Name:  ptr.To("Kargo"),
						Email: ptr.To("no-reply@kargo.io"),
					},
				},
			},
			appName:        "Kargo",
			appEmail:       "no-reply@kargo.io",
			sigStatuses:    map[string]git.CommitSignatureInfo{},
			appFingerprint: "AAAA1234",
			expect:         false,
		},
		{
			name: "nil author",
			rc: &github.RepositoryCommit{
				SHA:    ptr.To("abc"),
				Commit: &github.Commit{},
			},
			appName:  "Kargo",
			appEmail: "no-reply@kargo.io",
			expect:   false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := &githubVerifiedPusher{}
			require.Equal(
				t, tc.expect,
				g.isAppAuthored(
					tc.rc,
					tc.appName, tc.appEmail,
					tc.appFingerprint, tc.sigStatuses,
				),
			)
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
				g.newGitHubClientFn = func(
					_, _ string, _ bool,
				) (string, string, githubVerifiedPushClient, error) {
					return "owner", "repo",
						&mockGitHubVerifiedPushClient{}, nil
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
				g.newGitHubClientFn = func(
					_, _ string, _ bool,
				) (string, string, githubVerifiedPushClient, error) {
					return "owner", "repo",
						&mockGitHubVerifiedPushClient{}, nil
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
		{
			name: "retry on UpdateRef 422 succeeds on second attempt",
			pusher: func() *githubVerifiedPusher {
				g := newTestPusher()
				attempt := 0
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
											Message: ptr.To("test"),
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
								attempt++
								if attempt == 1 {
									return nil, nil,
										&github.ErrorResponse{
											Response: &http.Response{
												StatusCode: http.StatusUnprocessableEntity,
											},
										}
								}
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
			},
		},
		{
			name: "merge conflict during PullRebase is terminal",
			pusher: func() *githubVerifiedPusher {
				g := newTestPusher()
				wt := &mockWorkTree{
					url: "https://github.com/owner/repo",
					currentBranchFn: func() (string, error) {
						return "main", nil
					},
					pullRebaseFn: func(_ string) error {
						return git.ErrMergeConflict
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
						&mockGitHubVerifiedPushClient{}, nil
				}
				return g
			}(),
			cfg: builtinx.GitHubVerifiedPushConfig{
				Path:         "repo",
				TargetBranch: "main",
				PullPolicy:   policyPtr(builtinx.Rebase),
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.True(t, promotion.IsTerminal(err))
				var te *promotion.TerminalError
				require.ErrorAs(t, err, &te)
				require.True(t, git.IsMergeConflict(te.Err))
				require.Equal(
					t,
					kargoapi.PromotionStepStatusFailed,
					result.Status,
				)
			},
		},
		{
			name: "PullRebase skipped when force=true",
			pusher: func() *githubVerifiedPusher {
				g := newTestPusher()
				pullRebaseCalled := false
				wt := &mockWorkTree{
					url: "https://github.com/owner/repo",
					currentBranchFn: func() (string, error) {
						return "main", nil
					},
					pullRebaseFn: func(_ string) error {
						pullRebaseCalled = true
						return nil
					},
					lastCommitIDFn: func() (string, error) {
						require.False(
							t, pullRebaseCalled,
							"PullRebase should not be called "+
								"when force=true",
						)
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
							updateRefFn: func(
								_ context.Context,
								_, _, _ string,
								ref github.UpdateRef,
							) (*github.Reference, *github.Response, error) {
								require.NotNil(t, ref.Force)
								require.True(t, *ref.Force)
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
				Force:        true,
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
			},
		},
		{
			name: "merge conflict during PullMerge is terminal",
			pusher: func() *githubVerifiedPusher {
				g := newTestPusher()
				wt := &mockWorkTree{
					url: "https://github.com/owner/repo",
					currentBranchFn: func() (string, error) {
						return "main", nil
					},
					pullMergeFn: func(_ string) error {
						return git.ErrMergeConflict
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
						&mockGitHubVerifiedPushClient{}, nil
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
				require.True(t, promotion.IsTerminal(err))
				var te *promotion.TerminalError
				require.ErrorAs(t, err, &te)
				require.True(t, git.IsMergeConflict(te.Err))
				require.Equal(
					t,
					kargoapi.PromotionStepStatusFailed,
					result.Status,
				)
			},
		},
		{
			name: "PullMerge skipped when force=true",
			pusher: func() *githubVerifiedPusher {
				g := newTestPusher()
				pullMergeCalled := false
				wt := &mockWorkTree{
					url: "https://github.com/owner/repo",
					currentBranchFn: func() (string, error) {
						return "main", nil
					},
					pullMergeFn: func(_ string) error {
						pullMergeCalled = true
						return nil
					},
					lastCommitIDFn: func() (string, error) {
						require.False(
							t, pullMergeCalled,
							"PullMerge should not be called "+
								"when force=true",
						)
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
							updateRefFn: func(
								_ context.Context,
								_, _, _ string,
								ref github.UpdateRef,
							) (*github.Reference, *github.Response, error) {
								require.NotNil(t, ref.Force)
								require.True(t, *ref.Force)
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
				Force:        true,
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
			},
		},
		{
			name: "FFOnly errors when remote has advanced",
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
									Status: ptr.To("diverged"),
								}, nil, nil
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
				PullPolicy:   policyPtr(builtinx.FFOnly),
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.True(t, promotion.IsTerminal(err))
				require.Contains(
					t, err.Error(),
					"target branch may have diverged",
				)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusFailed,
					result.Status,
				)
			},
		},
		{
			name: "maxAttempts limits retries",
			pusher: func() *githubVerifiedPusher {
				g := newTestPusher()
				attempts := 0
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
							updateRefFn: func(
								_ context.Context,
								_, _, _ string,
								_ github.UpdateRef,
							) (*github.Reference, *github.Response, error) {
								attempts++
								return nil, nil,
									&github.ErrorResponse{
										Response: &http.Response{
											StatusCode: http.StatusUnprocessableEntity,
										},
									}
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
				MaxAttempts:  ptr.To[int64](2),
			},
			assert: func(
				t *testing.T, result promotion.StepResult, err error,
			) {
				t.Helper()
				require.Error(t, err)
				require.True(
					t, errors.Is(err, errRefUpdateConflict),
				)
				require.Equal(
					t,
					kargoapi.PromotionStepStatusErrored,
					result.Status,
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

func Test_isGitHubHTTPStatus(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		err    error
		code   int
		expect bool
	}{
		{
			name: "matching status code",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusUnprocessableEntity,
				},
			},
			code:   http.StatusUnprocessableEntity,
			expect: true,
		},
		{
			name: "non-matching status code",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusInternalServerError,
				},
			},
			code:   http.StatusUnprocessableEntity,
			expect: false,
		},
		{
			name:   "non-GitHub error",
			err:    fmt.Errorf("some error"),
			code:   http.StatusUnprocessableEntity,
			expect: false,
		},
		{
			name:   "nil error",
			err:    nil,
			code:   http.StatusUnprocessableEntity,
			expect: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := &githubVerifiedPusher{}
			assert.Equal(
				t, tc.expect, g.isGitHubHTTPStatus(tc.err, tc.code),
			)
		})
	}
}

func Test_githubVerifiedPusher_resolveParents(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name          string
		rc            *github.RepositoryCommit
		shaMap        map[string]string
		defaultParent string
		assert        func(*testing.T, []*github.Commit)
	}{
		{
			name: "single parent maps through shaMap",
			rc: &github.RepositoryCommit{
				Parents: []*github.Commit{
					{SHA: ptr.To("abc")},
				},
			},
			shaMap:        map[string]string{"abc": "xyz"},
			defaultParent: "fallback",
			assert: func(t *testing.T, parents []*github.Commit) {
				t.Helper()
				require.Len(t, parents, 1)
				require.Equal(t, "xyz", parents[0].GetSHA())
			},
		},
		{
			name: "merge commit preserves multiple parents",
			rc: &github.RepositoryCommit{
				Parents: []*github.Commit{
					{SHA: ptr.To("abc")},
					{SHA: ptr.To("def")},
				},
			},
			shaMap:        map[string]string{"abc": "xyz"},
			defaultParent: "fallback",
			assert: func(t *testing.T, parents []*github.Commit) {
				t.Helper()
				require.Len(t, parents, 2)
				require.Equal(t, "xyz", parents[0].GetSHA())
				require.Equal(t, "def", parents[1].GetSHA())
			},
		},
		{
			name: "no parents falls back to defaultParent",
			rc: &github.RepositoryCommit{
				Parents: []*github.Commit{},
			},
			shaMap:        map[string]string{},
			defaultParent: "fallback",
			assert: func(t *testing.T, parents []*github.Commit) {
				t.Helper()
				require.Len(t, parents, 1)
				require.Equal(
					t, "fallback", parents[0].GetSHA(),
				)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := &githubVerifiedPusher{}
			parents := g.resolveParents(
				tc.rc, tc.shaMap, tc.defaultParent,
			)
			tc.assert(t, parents)
		})
	}
}

func Test_githubVerifiedPusher_pullRemote(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		policy string
		wt     *mockWorkTree
		assert func(*testing.T, error)
	}{
		{
			name:   "Merge calls PullMerge",
			policy: pullPolicyMerge,
			wt: &mockWorkTree{
				pullMergeFn: func(branch string) error {
					require.Equal(t, "main", branch)
					return nil
				},
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name:   "Rebase calls PullRebase",
			policy: pullPolicyRebase,
			wt: &mockWorkTree{
				pullRebaseFn: func(branch string) error {
					require.Equal(t, "main", branch)
					return nil
				},
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name:   "FFOnly does nothing",
			policy: pullPolicyFFOnly,
			wt: &mockWorkTree{
				pullMergeFn: func(_ string) error {
					require.Fail(
						t,
						"PullMerge should not be called "+
							"for FFOnly",
					)
					return nil
				},
				pullRebaseFn: func(_ string) error {
					require.Fail(
						t,
						"PullRebase should not be called "+
							"for FFOnly",
					)
					return nil
				},
			},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name:   "unknown policy returns error",
			policy: "BadPolicy",
			wt:     &mockWorkTree{},
			assert: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				require.Contains(
					t, err.Error(), "unknown pullPolicy",
				)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := &githubVerifiedPusher{}
			err := g.pullRemote(tc.wt, "main", tc.policy)
			tc.assert(t, err)
		})
	}
}

func Test_isRetryableError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		err    error
		expect bool
	}{
		{
			name:   "ref update conflict",
			err:    errRefUpdateConflict,
			expect: true,
		},
		{
			name: "wrapped ref update conflict",
			err: fmt.Errorf(
				"error updating ref: %w", errRefUpdateConflict,
			),
			expect: true,
		},
		{
			name: "non-fast-forward",
			err: fmt.Errorf(
				"error pushing: %w", git.ErrNonFastForward,
			),
			expect: true,
		},
		{
			name:   "unrelated error",
			err:    fmt.Errorf("something else"),
			expect: false,
		},
		{
			name:   "terminal error",
			err:    &promotion.TerminalError{Err: fmt.Errorf("bad")},
			expect: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := &githubVerifiedPusher{}
			assert.Equal(
				t, tc.expect, g.isRetryableError(tc.err),
			)
		})
	}
}
