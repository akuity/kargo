package azure

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
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
	getRefsFn func(
		context.Context, adogit.GetRefsArgs,
	) (*adogit.GetRefsResponseValue, error)
	updateRefsFn func(
		context.Context, adogit.UpdateRefsArgs,
	) (*[]adogit.GitRefUpdateResult, error)
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

func (m *mockAzureGitClient) GetRefs(
	ctx context.Context, args adogit.GetRefsArgs,
) (*adogit.GetRefsResponseValue, error) {
	return m.getRefsFn(ctx, args)
}

func (m *mockAzureGitClient) UpdateRefs(
	ctx context.Context, args adogit.UpdateRefsArgs,
) (*[]adogit.GitRefUpdateResult, error) {
	return m.updateRefsFn(ctx, args)
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

func TestCreatePullRequest(t *testing.T) {
	testCases := []struct {
		name          string
		opts          *gitprovider.CreatePullRequestOpts
		mockClient    *mockAzureGitClient
		expectError   bool
		errorContains string
		assert        func(*testing.T, *gitprovider.PullRequest)
	}{
		{
			name: "nil opts does not panic",
			opts: nil,
			mockClient: &mockAzureGitClient{
				getRepositoryFn: func(
					_ context.Context, _ adogit.GetRepositoryArgs,
				) (*adogit.GitRepository, error) {
					id := uuid.New()
					return &adogit.GitRepository{Id: &id}, nil
				},
				createPullRequestFn: func(
					_ context.Context, args adogit.CreatePullRequestArgs,
				) (*adogit.GitPullRequest, error) {
					require.NotNil(t, args.GitPullRequestToCreate)
					return &adogit.GitPullRequest{
						PullRequestId: ptr.To(1),
						Status:        ptr.To(adogit.PullRequestStatusValues.Active),
						Repository: &adogit.GitRepository{
							WebUrl: ptr.To("https://dev.azure.com/org/project/_git/repo"),
						},
						LastMergeSourceCommit: &adogit.GitCommitRef{CommitId: ptr.To("abc")},
					}, nil
				},
			},
			assert: func(t *testing.T, pr *gitprovider.PullRequest) {
				require.NotNil(t, pr)
				require.Equal(t, int64(1), pr.Number)
			},
		},
		{
			name: "error getting repository",
			opts: &gitprovider.CreatePullRequestOpts{Head: "feature", Base: "main", Title: "t"},
			mockClient: &mockAzureGitClient{
				getRepositoryFn: func(
					context.Context, adogit.GetRepositoryArgs,
				) (*adogit.GitRepository, error) {
					return nil, errors.New("not found")
				},
			},
			expectError:   true,
			errorContains: "error getting repository",
		},
		{
			name: "error creating pull request",
			opts: &gitprovider.CreatePullRequestOpts{Head: "feature", Base: "main", Title: "t"},
			mockClient: &mockAzureGitClient{
				getRepositoryFn: func(
					_ context.Context, _ adogit.GetRepositoryArgs,
				) (*adogit.GitRepository, error) {
					id := uuid.New()
					return &adogit.GitRepository{Id: &id}, nil
				},
				createPullRequestFn: func(
					context.Context, adogit.CreatePullRequestArgs,
				) (*adogit.GitPullRequest, error) {
					return nil, errors.New("create failed")
				},
			},
			expectError:   true,
			errorContains: "error creating pull request",
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
			pr, err := p.CreatePullRequest(t.Context(), tc.opts)
			if tc.expectError {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.errorContains)
				require.Nil(t, pr)
				return
			}
			require.NoError(t, err)
			if tc.assert != nil {
				tc.assert(t, pr)
			}
		})
	}
}

func TestListPullRequests(t *testing.T) {
	testCases := []struct {
		name          string
		opts          *gitprovider.ListPullRequestOptions
		mockClient    *mockAzureGitClient
		expectError   bool
		errorContains string
		assert        func(*testing.T, []gitprovider.PullRequest)
	}{
		{
			name: "nil opts defaults to open state",
			opts: nil,
			mockClient: &mockAzureGitClient{
				getPullRequestsFn: func(
					_ context.Context, args adogit.GetPullRequestsArgs,
				) (*[]adogit.GitPullRequest, error) {
					require.Equal(t,
						adogit.PullRequestStatusValues.Active,
						*args.SearchCriteria.Status,
					)
					return &[]adogit.GitPullRequest{}, nil
				},
			},
			assert: func(t *testing.T, prs []gitprovider.PullRequest) {
				require.Empty(t, prs)
			},
		},
		{
			name: "empty state defaults to open",
			opts: &gitprovider.ListPullRequestOptions{State: ""},
			mockClient: &mockAzureGitClient{
				getPullRequestsFn: func(
					_ context.Context, args adogit.GetPullRequestsArgs,
				) (*[]adogit.GitPullRequest, error) {
					require.Equal(t,
						adogit.PullRequestStatusValues.Active,
						*args.SearchCriteria.Status,
					)
					return &[]adogit.GitPullRequest{}, nil
				},
			},
			assert: func(t *testing.T, prs []gitprovider.PullRequest) {
				require.Empty(t, prs)
			},
		},
		{
			name: "nil adoPRs returns nil without panic",
			opts: &gitprovider.ListPullRequestOptions{State: gitprovider.PullRequestStateOpen},
			mockClient: &mockAzureGitClient{
				getPullRequestsFn: func(
					context.Context, adogit.GetPullRequestsArgs,
				) (*[]adogit.GitPullRequest, error) {
					return nil, nil
				},
			},
			assert: func(t *testing.T, prs []gitprovider.PullRequest) {
				require.Nil(t, prs)
			},
		},
		{
			name: "client error is propagated",
			opts: &gitprovider.ListPullRequestOptions{State: gitprovider.PullRequestStateOpen},
			mockClient: &mockAzureGitClient{
				getPullRequestsFn: func(
					context.Context, adogit.GetPullRequestsArgs,
				) (*[]adogit.GitPullRequest, error) {
					return nil, errors.New("list failed")
				},
			},
			expectError:   true,
			errorContains: "list failed",
		},
		{
			name: "returns converted pull requests",
			opts: &gitprovider.ListPullRequestOptions{State: gitprovider.PullRequestStateOpen},
			mockClient: &mockAzureGitClient{
				getPullRequestsFn: func(
					context.Context, adogit.GetPullRequestsArgs,
				) (*[]adogit.GitPullRequest, error) {
					return &[]adogit.GitPullRequest{
						{
							PullRequestId: ptr.To(42),
							Status:        ptr.To(adogit.PullRequestStatusValues.Active),
							Repository: &adogit.GitRepository{
								WebUrl: ptr.To("https://dev.azure.com/org/project/_git/repo"),
							},
							LastMergeSourceCommit: &adogit.GitCommitRef{CommitId: ptr.To("sha1")},
						},
					}, nil
				},
			},
			assert: func(t *testing.T, prs []gitprovider.PullRequest) {
				require.Len(t, prs, 1)
				require.Equal(t, int64(42), prs[0].Number)
			},
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
			prs, err := p.ListPullRequests(t.Context(), tc.opts)
			if tc.expectError {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.errorContains)
				return
			}
			require.NoError(t, err)
			if tc.assert != nil {
				tc.assert(t, prs)
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

func TestDeleteBranch(t *testing.T) {
	const (
		testProject = "project"
		testRepo    = "repo"
		testBranch  = "kargo/promotion/test"
		testRefName = "refs/heads/" + testBranch
		testSHA     = "abc123"
	)

	refsWith := func(names ...string) *adogit.GetRefsResponseValue {
		refs := make([]adogit.GitRef, 0, len(names))
		for _, name := range names {
			refs = append(refs, adogit.GitRef{
				Name:     ptr.To(name),
				ObjectId: ptr.To(testSHA),
			})
		}
		return &adogit.GetRefsResponseValue{Value: refs}
	}

	testCases := []struct {
		name       string
		mockClient *mockAzureGitClient
		assert     func(*testing.T, *mockAzureGitClient, error)
	}{
		{
			name: "error getting refs",
			mockClient: &mockAzureGitClient{
				getRefsFn: func(
					context.Context, adogit.GetRefsArgs,
				) (*adogit.GetRefsResponseValue, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assert: func(t *testing.T, _ *mockAzureGitClient, err error) {
				require.ErrorContains(t, err, "error getting ref for branch")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "branch not found",
			mockClient: &mockAzureGitClient{
				getRefsFn: func(
					context.Context, adogit.GetRefsArgs,
				) (*adogit.GetRefsResponseValue, error) {
					return refsWith(), nil
				},
			},
			assert: func(t *testing.T, _ *mockAzureGitClient, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "only prefix matches found",
			mockClient: &mockAzureGitClient{
				getRefsFn: func(
					_ context.Context, args adogit.GetRefsArgs,
				) (*adogit.GetRefsResponseValue, error) {
					require.Equal(t, "heads/"+testBranch, ptr.Deref(args.Filter, ""))
					return refsWith(testRefName + "-other"), nil
				},
			},
			assert: func(t *testing.T, _ *mockAzureGitClient, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "error updating refs",
			mockClient: &mockAzureGitClient{
				getRefsFn: func(
					context.Context, adogit.GetRefsArgs,
				) (*adogit.GetRefsResponseValue, error) {
					return refsWith(testRefName), nil
				},
				updateRefsFn: func(
					context.Context, adogit.UpdateRefsArgs,
				) (*[]adogit.GitRefUpdateResult, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assert: func(t *testing.T, _ *mockAzureGitClient, err error) {
				require.ErrorContains(t, err, "error deleting branch")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "no result returned",
			mockClient: &mockAzureGitClient{
				getRefsFn: func(
					context.Context, adogit.GetRefsArgs,
				) (*adogit.GetRefsResponseValue, error) {
					return refsWith(testRefName), nil
				},
				updateRefsFn: func(
					context.Context, adogit.UpdateRefsArgs,
				) (*[]adogit.GitRefUpdateResult, error) {
					return &[]adogit.GitRefUpdateResult{}, nil
				},
			},
			assert: func(t *testing.T, _ *mockAzureGitClient, err error) {
				require.ErrorContains(t, err, "no result returned")
			},
		},
		{
			name: "update rejected",
			mockClient: &mockAzureGitClient{
				getRefsFn: func(
					context.Context, adogit.GetRefsArgs,
				) (*adogit.GetRefsResponseValue, error) {
					return refsWith(testRefName), nil
				},
				updateRefsFn: func(
					context.Context, adogit.UpdateRefsArgs,
				) (*[]adogit.GitRefUpdateResult, error) {
					return &[]adogit.GitRefUpdateResult{{
						Success:       ptr.To(false),
						UpdateStatus:  &adogit.GitRefUpdateStatusValues.RejectedByPolicy,
						CustomMessage: ptr.To("branch is protected"),
					}}, nil
				},
			},
			assert: func(t *testing.T, _ *mockAzureGitClient, err error) {
				require.ErrorContains(t, err, "rejectedByPolicy")
				require.ErrorContains(t, err, "branch is protected")
			},
		},
		{
			name: "ref vanished between lookup and update",
			mockClient: &mockAzureGitClient{
				getRefsFn: func(
					context.Context, adogit.GetRefsArgs,
				) (*adogit.GetRefsResponseValue, error) {
					return refsWith(testRefName), nil
				},
				updateRefsFn: func(
					context.Context, adogit.UpdateRefsArgs,
				) (*[]adogit.GitRefUpdateResult, error) {
					return &[]adogit.GitRefUpdateResult{{
						Success:      ptr.To(false),
						UpdateStatus: &adogit.GitRefUpdateStatusValues.SucceededNonExistentRef,
					}}, nil
				},
			},
			assert: func(t *testing.T, _ *mockAzureGitClient, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "branch deleted",
			mockClient: &mockAzureGitClient{
				getRefsFn: func(
					context.Context, adogit.GetRefsArgs,
				) (*adogit.GetRefsResponseValue, error) {
					return refsWith(testRefName), nil
				},
				updateRefsFn: func(
					_ context.Context, args adogit.UpdateRefsArgs,
				) (*[]adogit.GitRefUpdateResult, error) {
					require.Equal(t, testRepo, ptr.Deref(args.RepositoryId, ""))
					require.Equal(t, testProject, ptr.Deref(args.Project, ""))
					require.NotNil(t, args.RefUpdates)
					require.Len(t, *args.RefUpdates, 1)
					update := (*args.RefUpdates)[0]
					require.Equal(t, testRefName, ptr.Deref(update.Name, ""))
					require.Equal(t, testSHA, ptr.Deref(update.OldObjectId, ""))
					require.Equal(t, zeroObjectID, ptr.Deref(update.NewObjectId, ""))
					return &[]adogit.GitRefUpdateResult{{
						Success:      ptr.To(true),
						UpdateStatus: &adogit.GitRefUpdateStatusValues.Succeeded,
					}}, nil
				},
			},
			assert: func(t *testing.T, _ *mockAzureGitClient, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			p := &provider{
				project: testProject,
				repo:    testRepo,
				client:  testCase.mockClient,
			}
			err := p.DeleteBranch(t.Context(), testBranch)
			testCase.assert(t, testCase.mockClient, err)
		})
	}
}
