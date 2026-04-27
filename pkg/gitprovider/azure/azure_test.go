package azure

import (
	"context"
	"errors"
	"testing"

	adogit "github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/akuity/kargo/pkg/gitprovider"
)

type mockAzureGitClient struct {
	getRepositoryFn func(
		context.Context,
		adogit.GetRepositoryArgs,
	) (*adogit.GitRepository, error)
	createPullRequestFn func(
		context.Context, adogit.CreatePullRequestArgs,
	) (*adogit.GitPullRequest, error)
	getPullRequestFn func(
		context.Context, adogit.GetPullRequestArgs,
	) (*adogit.GitPullRequest, error)
	getPullRequestsFn func(
		context.Context, adogit.GetPullRequestsArgs,
	) (*[]adogit.GitPullRequest, error)
	updatePullRequestFn func(
		context.Context, adogit.UpdatePullRequestArgs,
	) (*adogit.GitPullRequest, error)
}

func (m *mockAzureGitClient) GetRepository(
	ctx context.Context, args adogit.GetRepositoryArgs,
) (*adogit.GitRepository, error) {
	return m.getRepositoryFn(ctx, args)
}

func (m *mockAzureGitClient) CreatePullRequest(
	ctx context.Context, args adogit.CreatePullRequestArgs,
) (*adogit.GitPullRequest, error) {
	return m.createPullRequestFn(ctx, args)
}

func (m *mockAzureGitClient) GetPullRequest(
	ctx context.Context, args adogit.GetPullRequestArgs,
) (*adogit.GitPullRequest, error) {
	return m.getPullRequestFn(ctx, args)
}

func (m *mockAzureGitClient) GetPullRequests(
	ctx context.Context, args adogit.GetPullRequestsArgs,
) (*[]adogit.GitPullRequest, error) {
	return m.getPullRequestsFn(ctx, args)
}

func (m *mockAzureGitClient) UpdatePullRequest(
	ctx context.Context, args adogit.UpdatePullRequestArgs,
) (*adogit.GitPullRequest, error) {
	return m.updatePullRequestFn(ctx, args)
}

func TestMergePullRequest(t *testing.T) {
	testCases := []struct {
		name           string
		prNumber       int64
		mergeOpts      *gitprovider.MergePullRequestOpts
		mockClient     *mockAzureGitClient
		expectedMerged bool
		expectError    bool
		errorContains  string
	}{
		{
			name:     "error getting PR",
			prNumber: 999,
			mockClient: &mockAzureGitClient{
				getPullRequestFn: func(
					context.Context, adogit.GetPullRequestArgs,
				) (*adogit.GitPullRequest, error) {
					return nil, errors.New("get PR failed")
				},
			},
			expectError:   true,
			errorContains: "error getting pull request",
		},
		{
			name:     "nil PR returned",
			prNumber: 404,
			mockClient: &mockAzureGitClient{
				getPullRequestFn: func(
					context.Context, adogit.GetPullRequestArgs,
				) (*adogit.GitPullRequest, error) {
					return nil, nil
				},
			},
			expectError:   true,
			errorContains: "pull request 404 not found",
		},
		{
			name:     "PR already completed",
			prNumber: 123,
			mockClient: &mockAzureGitClient{
				getPullRequestFn: func(
					context.Context, adogit.GetPullRequestArgs,
				) (*adogit.GitPullRequest, error) {
					return &adogit.GitPullRequest{
						PullRequestId: ptr.To(123),
						Status:        ptr.To(adogit.PullRequestStatusValues.Completed),
						Repository: &adogit.GitRepository{
							WebUrl: ptr.To("https://dev.azure.com/org/project/_git/repo"),
						},
						Url: ptr.To("https://dev.azure.com/org/project/_git/repo/pullrequest/123"),
						LastMergeSourceCommit: &adogit.GitCommitRef{
							CommitId: ptr.To("head_sha"),
						},
						LastMergeCommit: &adogit.GitCommitRef{
							CommitId: ptr.To("merge_sha"),
						},
					}, nil
				},
			},
			expectedMerged: true,
		},
		{
			name:     "PR abandoned",
			prNumber: 456,
			mockClient: &mockAzureGitClient{
				getPullRequestFn: func(
					context.Context, adogit.GetPullRequestArgs,
				) (*adogit.GitPullRequest, error) {
					return &adogit.GitPullRequest{
						PullRequestId: ptr.To(456),
						Status:        ptr.To(adogit.PullRequestStatusValues.Abandoned),
					}, nil
				},
			},
			expectError:   true,
			errorContains: "is abandoned",
		},
		{
			name:     "PR is draft",
			prNumber: 333,
			mockClient: &mockAzureGitClient{
				getPullRequestFn: func(
					context.Context, adogit.GetPullRequestArgs,
				) (*adogit.GitPullRequest, error) {
					return &adogit.GitPullRequest{
						PullRequestId: ptr.To(333),
						Status:        ptr.To(adogit.PullRequestStatusValues.Active),
						IsDraft:       ptr.To(true),
						MergeStatus: ptr.To(
							adogit.PullRequestAsyncStatusValues.Succeeded,
						),
					}, nil
				},
			},
		},
		{
			name:     "PR not ready to merge",
			prNumber: 444,
			mockClient: &mockAzureGitClient{
				getPullRequestFn: func(
					context.Context, adogit.GetPullRequestArgs,
				) (*adogit.GitPullRequest, error) {
					return &adogit.GitPullRequest{
						PullRequestId: ptr.To(444),
						Status:        ptr.To(adogit.PullRequestStatusValues.Active),
						MergeStatus: ptr.To(
							adogit.PullRequestAsyncStatusValues.Conflicts,
						),
					}, nil
				},
			},
		},
		{
			name:     "unknown status",
			prNumber: 555,
			mockClient: &mockAzureGitClient{
				getPullRequestFn: func(
					context.Context, adogit.GetPullRequestArgs,
				) (*adogit.GitPullRequest, error) {
					return &adogit.GitPullRequest{
						PullRequestId: ptr.To(555),
						Status:        ptr.To(adogit.PullRequestStatusValues.NotSet),
					}, nil
				},
			},
		},
		{
			name:      "unsupported merge method",
			prNumber:  100,
			mergeOpts: &gitprovider.MergePullRequestOpts{MergeMethod: "bogus"},
			mockClient: &mockAzureGitClient{
				getPullRequestFn: func(
					_ context.Context, _ adogit.GetPullRequestArgs,
				) (*adogit.GitPullRequest, error) {
					return &adogit.GitPullRequest{
						PullRequestId: ptr.To(100),
						Status:        ptr.To(adogit.PullRequestStatusValues.Active),
						MergeStatus: ptr.To(
							adogit.PullRequestAsyncStatusValues.Succeeded,
						),
						LastMergeSourceCommit: &adogit.GitCommitRef{
							CommitId: ptr.To("head_sha"),
						},
					}, nil
				},
			},
			expectError:   true,
			errorContains: `unsupported merge method "bogus"`,
		},
		{
			name:     "merge operation fails",
			prNumber: 888,
			mockClient: &mockAzureGitClient{
				getPullRequestFn: func(
					context.Context, adogit.GetPullRequestArgs,
				) (*adogit.GitPullRequest, error) {
					return &adogit.GitPullRequest{
						PullRequestId: ptr.To(888),
						Status:        ptr.To(adogit.PullRequestStatusValues.Active),
						MergeStatus: ptr.To(
							adogit.PullRequestAsyncStatusValues.Succeeded,
						),
						LastMergeSourceCommit: &adogit.GitCommitRef{
							CommitId: ptr.To("head_sha"),
						},
					}, nil
				},
				updatePullRequestFn: func(
					context.Context, adogit.UpdatePullRequestArgs,
				) (*adogit.GitPullRequest, error) {
					return nil, errors.New("merge failed")
				},
			},
			expectError:   true,
			errorContains: "error merging pull request",
		},
		{
			name:     "nil response after merge",
			prNumber: 777,
			mockClient: &mockAzureGitClient{
				getPullRequestFn: func(
					context.Context, adogit.GetPullRequestArgs,
				) (*adogit.GitPullRequest, error) {
					return &adogit.GitPullRequest{
						PullRequestId: ptr.To(777),
						Status:        ptr.To(adogit.PullRequestStatusValues.Active),
						MergeStatus: ptr.To(
							adogit.PullRequestAsyncStatusValues.Succeeded,
						),
						LastMergeSourceCommit: &adogit.GitCommitRef{
							CommitId: ptr.To("head_sha"),
						},
					}, nil
				},
				updatePullRequestFn: func(
					context.Context, adogit.UpdatePullRequestArgs,
				) (*adogit.GitPullRequest, error) {
					return nil, nil
				},
			},
			expectError:   true,
			errorContains: "unexpected nil response after merging",
		},
		{
			name:     "successful merge",
			prNumber: 1234,
			mockClient: func() *mockAzureGitClient {
				calls := 0
				return &mockAzureGitClient{
					getPullRequestFn: func(
						_ context.Context, _ adogit.GetPullRequestArgs,
					) (*adogit.GitPullRequest, error) {
						calls++
						if calls == 1 {
							return &adogit.GitPullRequest{
								PullRequestId: ptr.To(1234),
								Status:        ptr.To(adogit.PullRequestStatusValues.Active),
								MergeStatus: ptr.To(
									adogit.PullRequestAsyncStatusValues.Succeeded,
								),
								Url: ptr.To("https://dev.azure.com/org/project/_git/repo/pullrequest/1234"),
								LastMergeSourceCommit: &adogit.GitCommitRef{
									CommitId: ptr.To("head_sha"),
								},
							}, nil
						}
						return &adogit.GitPullRequest{
							PullRequestId: ptr.To(1234),
							Status:        ptr.To(adogit.PullRequestStatusValues.Completed),
							Url:           ptr.To("https://dev.azure.com/org/project/_git/repo/pullrequest/1234"),
							LastMergeSourceCommit: &adogit.GitCommitRef{
								CommitId: ptr.To("head_sha"),
							},
							LastMergeCommit: &adogit.GitCommitRef{
								CommitId: ptr.To("merge_sha"),
							},
						}, nil
					},
					updatePullRequestFn: func(
						context.Context, adogit.UpdatePullRequestArgs,
					) (*adogit.GitPullRequest, error) {
						return &adogit.GitPullRequest{
							PullRequestId: ptr.To(1234),
							Status:        ptr.To(adogit.PullRequestStatusValues.Active),
							Url:           ptr.To("https://dev.azure.com/org/project/_git/repo/pullrequest/1234"),
							LastMergeSourceCommit: &adogit.GitCommitRef{
								CommitId: ptr.To("head_sha"),
							},
						}, nil
					},
				}
			}(),
			expectedMerged: true,
		},
		{
			name:      "successful merge with explicit method",
			prNumber:  100,
			mergeOpts: &gitprovider.MergePullRequestOpts{MergeMethod: "squash"},
			mockClient: func() *mockAzureGitClient {
				calls := 0
				return &mockAzureGitClient{
					getPullRequestFn: func(
						_ context.Context, _ adogit.GetPullRequestArgs,
					) (*adogit.GitPullRequest, error) {
						calls++
						if calls == 1 {
							return &adogit.GitPullRequest{
								PullRequestId: ptr.To(100),
								Status:        ptr.To(adogit.PullRequestStatusValues.Active),
								MergeStatus: ptr.To(
									adogit.PullRequestAsyncStatusValues.Succeeded,
								),
								Url: ptr.To("https://dev.azure.com/org/project/_git/repo/pullrequest/100"),
								LastMergeSourceCommit: &adogit.GitCommitRef{
									CommitId: ptr.To("head_sha"),
								},
							}, nil
						}
						return &adogit.GitPullRequest{
							PullRequestId: ptr.To(100),
							Status:        ptr.To(adogit.PullRequestStatusValues.Completed),
							Url:           ptr.To("https://dev.azure.com/org/project/_git/repo/pullrequest/100"),
							LastMergeSourceCommit: &adogit.GitCommitRef{
								CommitId: ptr.To("head_sha"),
							},
							LastMergeCommit: &adogit.GitCommitRef{
								CommitId: ptr.To("squash_sha"),
							},
						}, nil
					},
					updatePullRequestFn: func(
						_ context.Context, args adogit.UpdatePullRequestArgs,
					) (*adogit.GitPullRequest, error) {
						require.NotNil(t, args.GitPullRequestToUpdate.CompletionOptions)
						require.Equal(t,
							adogit.GitPullRequestMergeStrategyValues.Squash,
							*args.GitPullRequestToUpdate.CompletionOptions.MergeStrategy,
						)
						return &adogit.GitPullRequest{
							PullRequestId: ptr.To(100),
							Status:        ptr.To(adogit.PullRequestStatusValues.Active),
							Url:           ptr.To("https://dev.azure.com/org/project/_git/repo/pullrequest/100"),
							LastMergeSourceCommit: &adogit.GitCommitRef{
								CommitId: ptr.To("head_sha"),
							},
						}, nil
					},
				}
			}(),
			expectedMerged: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := &provider{
				org:     "org",
				project: "project",
				repo:    "repo",
				client:  tc.mockClient,
			}

			pr, merged, err := p.MergePullRequest(t.Context(), tc.prNumber, tc.mergeOpts)

			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorContains)
				require.False(t, merged)
				require.Nil(t, pr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectedMerged, merged)
			if tc.expectedMerged {
				require.NotNil(t, pr)
				require.Equal(t, tc.prNumber, pr.Number)
			}
		})
	}
}

func TestParseRepoURL(t *testing.T) {
	testCases := []struct {
		name         string
		url          string
		expectedOrg  string
		expectedProj string
		expectedRepo string
		errExpected  bool
	}{
		{
			name:        "invalid URL",
			url:         "not-a-url",
			errExpected: true,
		},
		{
			name:        "unsupported host",
			url:         "https://github.com/org/repo",
			errExpected: true,
		},
		{
			name:        "modern URL with missing parts",
			url:         "https://dev.azure.com/org",
			errExpected: true,
		},
		{
			name:        "legacy URL with missing parts",
			url:         "https://org.visualstudio.com",
			errExpected: true,
		},
		{
			name:         "modern URL format",
			url:          "https://dev.azure.com/myorg/myproject/_git/myrepo",
			expectedOrg:  "myorg",
			expectedProj: "myproject",
			expectedRepo: "myrepo",
			errExpected:  false,
		},
		{
			name:         "modern URL format with .git suffix",
			url:          "https://dev.azure.com/myorg/myproject/_git/myrepo.git",
			expectedOrg:  "myorg",
			expectedProj: "myproject",
			expectedRepo: "myrepo",
			errExpected:  false,
		},
		{
			name:         "legacy URL format",
			url:          "https://myorg.visualstudio.com/myproject/_git/myrepo",
			expectedOrg:  "myorg",
			expectedProj: "myproject",
			expectedRepo: "myrepo",
			errExpected:  false,
		},
		{
			name:         "legacy URL format with .git suffix",
			url:          "https://myorg.visualstudio.com/myproject/_git/myrepo.git",
			expectedOrg:  "myorg",
			expectedProj: "myproject",
			expectedRepo: "myrepo",
			errExpected:  false,
		},
		{
			name:         "modern URL format with dot in repo name",
			url:          "https://dev.azure.com/myorg/myproject/_git/my.repo",
			expectedOrg:  "myorg",
			expectedProj: "myproject",
			expectedRepo: "my.repo",
			errExpected:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			org, proj, repo, err := parseRepoURL(tc.url)
			if tc.errExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedOrg, org)
				require.Equal(t, tc.expectedProj, proj)
				require.Equal(t, tc.expectedRepo, repo)
			}
		})
	}
}

func TestGetCommitURL(t *testing.T) {
	testCases := []struct {
		repoURL           string
		sha               string
		expectedCommitURL string
	}{
		{
			repoURL:           "ssh://git@ssh.dev.azure.com/akuity/_git/kargo",
			sha:               "sha",
			expectedCommitURL: "https://dev.azure.com/akuity/_git/kargo/commit/sha",
		},
		{
			repoURL:           "git@ssh.dev.azure.com:v3/akuity/_git/kargo",
			sha:               "sha",
			expectedCommitURL: "https://dev.azure.com/akuity/_git/kargo/commit/sha",
		},
		{
			repoURL:           "http://dev.azure.com/akuity/_git/kargo",
			sha:               "sha",
			expectedCommitURL: "https://dev.azure.com/akuity/_git/kargo/commit/sha",
		},
	}

	prov := provider{}

	for _, testCase := range testCases {
		t.Run(testCase.repoURL, func(t *testing.T) {
			commitURL, err := prov.GetCommitURL(testCase.repoURL, testCase.sha)
			require.NoError(t, err)
			require.Equal(t, testCase.expectedCommitURL, commitURL)
		})
	}
}
