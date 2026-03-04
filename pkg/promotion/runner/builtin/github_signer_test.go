package builtin

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_gitHubSigner_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name:   "repoURL not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): repoURL is required",
			},
		},
		{
			name: "repoURL is empty string",
			config: promotion.Config{
				"repoURL": "",
			},
			expectedProblems: []string{
				"repoURL: String length must be greater than or equal to 1",
			},
		},
		{
			name:   "targetBranch not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): targetBranch is required",
			},
		},
		{
			name: "targetBranch is empty string",
			config: promotion.Config{
				"targetBranch": "",
			},
			expectedProblems: []string{
				"targetBranch: String length must be greater than or equal to 1",
			},
		},
		{
			name:   "head not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): head is required",
			},
		},
		{
			name: "head is empty string",
			config: promotion.Config{
				"head": "",
			},
			expectedProblems: []string{
				"head: String length must be greater than or equal to 1",
			},
		},
		{
			name:   "base not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): base is required",
			},
		},
		{
			name: "base is empty string",
			config: promotion.Config{
				"base": "",
			},
			expectedProblems: []string{
				"base: String length must be greater than or equal to 1",
			},
		},
		{
			name: "valid minimal config",
			config: promotion.Config{
				"repoURL":      "https://github.com/example/repo",
				"targetBranch": "main",
				"head":         "def456",
				"base":         "abc123",
			},
		},
		{
			name: "valid full config",
			config: promotion.Config{
				"repoURL":               "https://github.com/example/repo",
				"targetBranch":          "main",
				"head":                  "def456",
				"base":                  "abc123",
				"insecureSkipTLSVerify": true,
			},
		},
	}

	r := newGitHubSigner(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*gitHubSigner)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

// mockGitHubSignClient implements githubSignClient for testing.
type mockGitHubSignClient struct {
	compareCommitsFn func(
		ctx context.Context, owner, repo, base, head string,
		opts *github.ListOptions,
	) (*github.CommitsComparison, *github.Response, error)
	createCommitFn func(
		ctx context.Context, owner, repo string,
		commit github.Commit, opts *github.CreateCommitOptions,
	) (*github.Commit, *github.Response, error)
	updateRefFn func(
		ctx context.Context, owner, repo, ref string,
		updateRef github.UpdateRef,
	) (*github.Reference, *github.Response, error)
}

func (m *mockGitHubSignClient) CompareCommits(
	ctx context.Context, owner, repo, base, head string,
	opts *github.ListOptions,
) (*github.CommitsComparison, *github.Response, error) {
	return m.compareCommitsFn(ctx, owner, repo, base, head, opts)
}

func (m *mockGitHubSignClient) CreateCommit(
	ctx context.Context, owner, repo string,
	commit github.Commit, opts *github.CreateCommitOptions,
) (*github.Commit, *github.Response, error) {
	return m.createCommitFn(ctx, owner, repo, commit, opts)
}

func (m *mockGitHubSignClient) UpdateRef(
	ctx context.Context, owner, repo, ref string,
	updateRef github.UpdateRef,
) (*github.Reference, *github.Response, error) {
	return m.updateRefFn(ctx, owner, repo, ref, updateRef)
}

func Test_gitHubSigner_signRevisionRange(t *testing.T) {
	sha := func(s string) *string { return &s }

	tests := []struct {
		name           string
		client         *mockGitHubSignClient
		cfg            builtin.GitHubSignConfig
		maxRevisions   int
		expectedStatus kargoapi.PromotionStepStatus
		expectedOutput map[string]any
		expectedErr    string
	}{
		{
			name: "compare API error",
			client: &mockGitHubSignClient{
				compareCommitsFn: func(
					_ context.Context, _, _, _, _ string, _ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					return nil, nil, fmt.Errorf("API error")
				},
			},
			cfg: builtin.GitHubSignConfig{
				RepoURL:      "https://github.com/org/repo",
				TargetBranch: "main",
				Head:         "def456",
				Base:         "abc123",
			},
			expectedStatus: kargoapi.PromotionStepStatusErrored,
			expectedErr:    "error comparing abc123...def456",
		},
		{
			name: "identical range (nothing to sign)",
			client: &mockGitHubSignClient{
				compareCommitsFn: func(
					_ context.Context, _, _, _, _ string, _ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					status := "identical"
					return &github.CommitsComparison{Status: &status}, nil, nil
				},
			},
			cfg: builtin.GitHubSignConfig{
				RepoURL:      "https://github.com/org/repo",
				TargetBranch: "main",
				Head:         "def456",
				Base:         "abc123",
			},
			expectedStatus: kargoapi.PromotionStepStatusSkipped,
			expectedOutput: map[string]any{
				"commit": "abc123",
				"branch": "main",
			},
		},
		{
			name: "diverged range (error)",
			client: &mockGitHubSignClient{
				compareCommitsFn: func(
					_ context.Context, _, _, _, _ string, _ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					status := "diverged"
					return &github.CommitsComparison{Status: &status}, nil, nil
				},
			},
			cfg: builtin.GitHubSignConfig{
				RepoURL:      "https://github.com/org/repo",
				TargetBranch: "main",
				Head:         "def456",
				Base:         "abc123",
			},
			expectedStatus: kargoapi.PromotionStepStatusFailed,
			expectedErr:    "cannot sign revision range",
		},
		{
			name: "exceeds max revisions",
			client: &mockGitHubSignClient{
				compareCommitsFn: func(
					_ context.Context, _, _, _, _ string, _ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					status := compareStatusAhead
					return &github.CommitsComparison{
						Status: &status,
						Commits: []*github.RepositoryCommit{
							{SHA: sha("c1")},
							{SHA: sha("c2")},
							{SHA: sha("c3")},
						},
					}, nil, nil
				},
			},
			cfg: builtin.GitHubSignConfig{
				RepoURL:      "https://github.com/org/repo",
				TargetBranch: "main",
				Head:         "def456",
				Base:         "abc123",
			},
			maxRevisions:   2,
			expectedStatus: kargoapi.PromotionStepStatusFailed,
			expectedErr:    "exceeds the maximum of 2",
		},
		{
			name: "single revision signed successfully",
			client: &mockGitHubSignClient{
				compareCommitsFn: func(
					_ context.Context, _, _, _, _ string, _ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					status := compareStatusAhead
					msg := "promote: update v1.2.3"
					treeSHA := "tree-sha-1"
					return &github.CommitsComparison{
						Status: &status,
						Commits: []*github.RepositoryCommit{
							{
								SHA: sha("unsigned-1"),
								Commit: &github.Commit{
									Message: &msg,
									Tree:    &github.Tree{SHA: &treeSHA},
								},
							},
						},
					}, nil, nil
				},
				createCommitFn: func(
					_ context.Context, _, _ string,
					commit github.Commit, _ *github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					assert.Equal(t, "promote: update v1.2.3", commit.GetMessage())
					signedSHA := "signed-1"
					return &github.Commit{SHA: &signedSHA}, nil, nil
				},
				updateRefFn: func(
					_ context.Context, _, _, ref string, ur github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					assert.Equal(t, "heads/main", ref)
					assert.Equal(t, "signed-1", ur.SHA)
					assert.False(t, ur.GetForce())
					return &github.Reference{}, nil, nil
				},
			},
			cfg: builtin.GitHubSignConfig{
				RepoURL:      "https://github.com/org/repo",
				TargetBranch: "main",
				Head:         "def456",
				Base:         "abc123",
			},
			expectedStatus: kargoapi.PromotionStepStatusSucceeded,
			expectedOutput: map[string]any{
				"commit":    "signed-1",
				"commitURL": "https://github.com/org/repo/commit/signed-1",
				"branch":    "main",
			},
		},
		{
			name: "multiple revisions signed in order",
			client: &mockGitHubSignClient{
				compareCommitsFn: func(
					_ context.Context, _, _, _, _ string, _ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					status := compareStatusAhead
					msg1 := "commit 1"
					msg2 := "commit 2"
					tree1 := "tree-1"
					tree2 := "tree-2"
					return &github.CommitsComparison{
						Status: &status,
						Commits: []*github.RepositoryCommit{
							{
								SHA: sha("unsigned-1"),
								Commit: &github.Commit{
									Message: &msg1,
									Tree:    &github.Tree{SHA: &tree1},
								},
							},
							{
								SHA: sha("unsigned-2"),
								Commit: &github.Commit{
									Message: &msg2,
									Tree:    &github.Tree{SHA: &tree2},
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
						_ context.Context, _, _ string,
						commit github.Commit, _ *github.CreateCommitOptions,
					) (*github.Commit, *github.Response, error) {
						callCount++
						switch callCount {
						case 1:
							// First revision: parent should be base
							require.Len(t, commit.Parents, 1)
							assert.Equal(t, "abc123", commit.Parents[0].GetSHA())
							assert.Equal(t, "commit 1", commit.GetMessage())
							signedSHA := "signed-1"
							return &github.Commit{SHA: &signedSHA}, nil, nil
						case 2:
							// Second revision: parent should be first signed commit
							require.Len(t, commit.Parents, 1)
							assert.Equal(t, "signed-1", commit.Parents[0].GetSHA())
							assert.Equal(t, "commit 2", commit.GetMessage())
							signedSHA := "signed-2"
							return &github.Commit{SHA: &signedSHA}, nil, nil
						default:
							return nil, nil, fmt.Errorf("unexpected call %d", callCount)
						}
					}
				}(),
				updateRefFn: func(
					_ context.Context, _, _, _ string, ur github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					assert.Equal(t, "signed-2", ur.SHA)
					return &github.Reference{}, nil, nil
				},
			},
			cfg: builtin.GitHubSignConfig{
				RepoURL:      "https://github.com/org/repo",
				TargetBranch: "main",
				Head:         "def456",
				Base:         "abc123",
			},
			expectedStatus: kargoapi.PromotionStepStatusSucceeded,
			expectedOutput: map[string]any{
				"commit":    "signed-2",
				"commitURL": "https://github.com/org/repo/commit/signed-2",
				"branch":    "main",
			},
		},
		{
			name: "create commit API error",
			client: &mockGitHubSignClient{
				compareCommitsFn: func(
					_ context.Context, _, _, _, _ string, _ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					status := compareStatusAhead
					msg := "commit 1"
					treeSHA := "tree-1"
					return &github.CommitsComparison{
						Status: &status,
						Commits: []*github.RepositoryCommit{
							{
								SHA: sha("unsigned-1"),
								Commit: &github.Commit{
									Message: &msg,
									Tree:    &github.Tree{SHA: &treeSHA},
								},
							},
						},
					}, nil, nil
				},
				createCommitFn: func(
					_ context.Context, _, _ string,
					_ github.Commit, _ *github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					return nil, nil, fmt.Errorf("permission denied")
				},
			},
			cfg: builtin.GitHubSignConfig{
				RepoURL:      "https://github.com/org/repo",
				TargetBranch: "main",
				Head:         "def456",
				Base:         "abc123",
			},
			expectedStatus: kargoapi.PromotionStepStatusErrored,
			expectedErr:    "error creating signed revision 1/1",
		},
		{
			name: "update ref API error",
			client: &mockGitHubSignClient{
				compareCommitsFn: func(
					_ context.Context, _, _, _, _ string, _ *github.ListOptions,
				) (*github.CommitsComparison, *github.Response, error) {
					status := compareStatusAhead
					msg := "commit 1"
					treeSHA := "tree-1"
					return &github.CommitsComparison{
						Status: &status,
						Commits: []*github.RepositoryCommit{
							{
								SHA: sha("unsigned-1"),
								Commit: &github.Commit{
									Message: &msg,
									Tree:    &github.Tree{SHA: &treeSHA},
								},
							},
						},
					}, nil, nil
				},
				createCommitFn: func(
					_ context.Context, _, _ string,
					_ github.Commit, _ *github.CreateCommitOptions,
				) (*github.Commit, *github.Response, error) {
					signedSHA := "signed-1"
					return &github.Commit{SHA: &signedSHA}, nil, nil
				},
				updateRefFn: func(
					_ context.Context, _, _, _ string, _ github.UpdateRef,
				) (*github.Reference, *github.Response, error) {
					return nil, nil, fmt.Errorf("branch protected")
				},
			},
			cfg: builtin.GitHubSignConfig{
				RepoURL:      "https://github.com/org/repo",
				TargetBranch: "main",
				Head:         "def456",
				Base:         "abc123",
			},
			expectedStatus: kargoapi.PromotionStepStatusErrored,
			expectedErr:    "error updating ref",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maxRevisions := tt.maxRevisions
			if maxRevisions == 0 {
				maxRevisions = 10
			}
			g := &gitHubSigner{
				cfg: gitHubSignerConfig{MaxRevisions: maxRevisions},
			}
			result, err := g.signRevisionRange(
				context.Background(), tt.client, "org", "repo", tt.cfg,
			)
			assert.Equal(t, tt.expectedStatus, result.Status)
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
			if tt.expectedOutput != nil {
				for k, v := range tt.expectedOutput {
					assert.Equal(t, v, result.Output[k], "output key %q", k)
				}
			}
		})
	}
}

func Test_gitHubSigner_getPreviousOutput(t *testing.T) {
	tests := []struct {
		name           string
		sharedState    promotion.State
		alias          string
		expectedOutput map[string]any
		expectedErr    string
	}{
		{
			name:        "no previous output",
			sharedState: promotion.State{},
			alias:       "sign",
		},
		{
			name: "previous output with commit",
			sharedState: promotion.State{
				"sign": map[string]any{
					"commit":    "signed-abc",
					"commitURL": "https://github.com/org/repo/commit/signed-abc",
					"branch":    "main",
				},
			},
			alias: "sign",
			expectedOutput: map[string]any{
				"commit":    "signed-abc",
				"commitURL": "https://github.com/org/repo/commit/signed-abc",
				"branch":    "main",
			},
		},
		{
			name: "previous output without commit key",
			sharedState: promotion.State{
				"sign": map[string]any{
					"branch": "main",
				},
			},
			alias: "sign",
		},
		{
			name: "previous output is wrong type",
			sharedState: promotion.State{
				"sign": "not a map",
			},
			alias:       "sign",
			expectedErr: "not a map[string]any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &gitHubSigner{}
			stepCtx := &promotion.StepContext{
				Alias:       tt.alias,
				SharedState: tt.sharedState,
			}
			output, err := g.getPreviousOutput(stepCtx)
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
			if tt.expectedOutput != nil {
				require.NotNil(t, output)
				for k, v := range tt.expectedOutput {
					assert.Equal(t, v, output[k])
				}
			} else if err == nil {
				assert.Nil(t, output)
			}
		})
	}
}

func Test_gitHubSigner_buildCommitURL(t *testing.T) {
	g := &gitHubSigner{}
	tests := []struct {
		name     string
		repoURL  string
		sha      string
		expected string
	}{
		{
			name:     "standard GitHub URL",
			repoURL:  "https://github.com/org/repo",
			sha:      "abc123",
			expected: "https://github.com/org/repo/commit/abc123",
		},
		{
			name:     "GitHub URL with .git suffix",
			repoURL:  "https://github.com/org/repo.git",
			sha:      "abc123",
			expected: "https://github.com/org/repo/commit/abc123",
		},
		{
			name:     "GitHub Enterprise URL",
			repoURL:  "https://github.example.com/org/repo",
			sha:      "def456",
			expected: "https://github.example.com/org/repo/commit/def456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.buildCommitURL(tt.repoURL, tt.sha)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_gitHubSigner_newGitHubClient(t *testing.T) {
	g := &gitHubSigner{}

	t.Run("valid GitHub URL", func(t *testing.T) {
		owner, repo, client, err := g.newGitHubClient(
			"https://github.com/myorg/myrepo", "fake-token", false,
		)
		require.NoError(t, err)
		assert.Equal(t, "myorg", owner)
		assert.Equal(t, "myrepo", repo)
		assert.NotNil(t, client)
	})

	t.Run("URL with .git suffix", func(t *testing.T) {
		owner, repo, client, err := g.newGitHubClient(
			"https://github.com/myorg/myrepo.git", "fake-token", false,
		)
		require.NoError(t, err)
		assert.Equal(t, "myorg", owner)
		assert.Equal(t, "myrepo", repo)
		assert.NotNil(t, client)
	})

	t.Run("invalid URL", func(t *testing.T) {
		_, _, _, err := g.newGitHubClient(
			"not-a-valid-url/missing/parts/extra", "fake-token", false,
		)
		require.Error(t, err)
	})
}
