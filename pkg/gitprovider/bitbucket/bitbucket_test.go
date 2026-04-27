package bitbucket

import (
	"errors"
	"testing"

	"github.com/ktrysmt/go-bitbucket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/gitprovider"
)

type mockPullRequestClient struct {
	createFunc    func(opt *bitbucket.PullRequestsOptions) (any, error)
	getsFunc      func(opt *bitbucket.PullRequestsOptions) (any, error)
	getFunc       func(opt *bitbucket.PullRequestsOptions) (any, error)
	getCommitFunc func(opt *bitbucket.CommitsOptions) (any, error)
	mergeFunc     func(opt *bitbucket.PullRequestsOptions) (any, error)
}

func (m *mockPullRequestClient) Create(
	opt *bitbucket.PullRequestsOptions,
) (any, error) {
	return m.createFunc(opt)
}

func (m *mockPullRequestClient) Gets(
	opt *bitbucket.PullRequestsOptions,
) (any, error) {
	return m.getsFunc(opt)
}

func (m *mockPullRequestClient) Get(
	opt *bitbucket.PullRequestsOptions,
) (any, error) {
	return m.getFunc(opt)
}

func (m *mockPullRequestClient) GetCommit(opt *bitbucket.CommitsOptions) (any, error) {
	return m.getCommitFunc(opt)
}

func (m *mockPullRequestClient) Merge(
	opt *bitbucket.PullRequestsOptions,
) (any, error) {
	return m.mergeFunc(opt)
}

func TestNewProvider(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		provider, err := NewProvider("https://bitbucket.org/owner/repo", &gitprovider.Options{Token: "token"})
		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("successful creation with nil options", func(t *testing.T) {
		provider, err := NewProvider("https://bitbucket.org/owner/repo", nil)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("error with invalid URL", func(t *testing.T) {
		provider, err := NewProvider("://invalid-url", &gitprovider.Options{Token: "token"})
		assert.Error(t, err)
		assert.Nil(t, provider)
	})

	t.Run("error with unsupported host", func(t *testing.T) {
		provider, err := NewProvider("https://not-bitbucket.org/owner/repo", &gitprovider.Options{Token: "token"})
		assert.Error(t, err)
		assert.Nil(t, provider)
	})

	t.Run("error with invalid path", func(t *testing.T) {
		provider, err := NewProvider("https://bitbucket.org/invalid-path", &gitprovider.Options{Token: "token"})
		assert.Error(t, err)
		assert.Nil(t, provider)
	})
}

func TestCreatePullRequest(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			createFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
				return map[string]any{
					"id":    int64(1),
					"state": prStateOpen,
					"links": map[string]any{
						"html": map[string]any{
							"href": "https://bitbucket.org/owner/repo/pull-requests/1",
						},
					},
					"source": map[string]any{
						"branch": map[string]any{
							"name": "feature-branch",
						},
						"commit": map[string]any{
							"hash": "abcdef1234567890",
						},
					},
					"destination": map[string]any{
						"branch": map[string]any{
							"name": "main",
						},
					},
					"created_on": "2023-01-01T12:00:00Z",
				}, nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		opts := &gitprovider.CreatePullRequestOpts{
			Title:       "Test PR",
			Description: "PR description",
			Head:        "feature-branch",
			Base:        "main",
		}
		pr, err := provider.CreatePullRequest(ctx, opts)
		assert.NoError(t, err)
		assert.NotNil(t, pr)
		assert.Equal(t, int64(1), pr.Number)
		assert.Equal(t, "https://bitbucket.org/owner/repo/pull-requests/1", pr.URL)
		assert.Equal(t, "abcdef1234567890", pr.HeadSHA)
		assert.True(t, pr.Open)
		assert.False(t, pr.Merged)
	})

	t.Run("successful creation with nil options", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			createFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
				return map[string]any{
					"id":    int64(1),
					"state": prStateOpen,
				}, nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		pr, err := provider.CreatePullRequest(ctx, nil)
		assert.NoError(t, err)
		assert.NotNil(t, pr)
		assert.Equal(t, int64(1), pr.Number)
	})

	t.Run("creation with merge commit", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			createFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
				return map[string]any{
					"id":    int64(1),
					"state": prStateOpen,
					"merge_commit": map[string]any{
						"hash": "short123",
					},
				}, nil
			},
			getCommitFunc: func(*bitbucket.CommitsOptions) (any, error) {
				return map[string]any{
					"hash": "full1234567890abcdef",
				}, nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		pr, err := provider.CreatePullRequest(ctx, nil)
		assert.NoError(t, err)
		assert.NotNil(t, pr)
		assert.Equal(t, "full1234567890abcdef", pr.MergeCommitSHA)
	})

	t.Run("error during creation", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			createFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
				return nil, errors.New("creation failed")
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		pr, err := provider.CreatePullRequest(ctx, nil)
		assert.Error(t, err)
		assert.Nil(t, pr)
	})

	t.Run("error converting PR response", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			createFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
				// Return something that can't be properly unmarshaled
				return make(chan int), nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		pr, err := provider.CreatePullRequest(ctx, nil)
		assert.Error(t, err)
		assert.Nil(t, pr)
	})

	t.Run("error getting full commit SHA", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			createFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
				return map[string]any{
					"id":    int64(1),
					"state": prStateOpen,
					"merge_commit": map[string]any{
						"hash": "short123",
					},
				}, nil
			},
			getCommitFunc: func(*bitbucket.CommitsOptions) (any, error) {
				return nil, errors.New("commit fetch failed")
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		pr, err := provider.CreatePullRequest(ctx, nil)
		assert.Error(t, err)
		assert.Nil(t, pr)
	})
}

func TestGetPullRequest(t *testing.T) {
	t.Run("successful retrieval", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getFunc: func(opt *bitbucket.PullRequestsOptions) (any, error) {
				assert.Equal(t, "1", opt.ID)
				return map[string]any{
					"id":    int64(1),
					"state": prStateOpen,
					"links": map[string]any{
						"html": map[string]any{
							"href": "https://bitbucket.org/owner/repo/pull-requests/1",
						},
					},
					"source": map[string]any{
						"branch": map[string]any{
							"name": "feature-branch",
						},
						"commit": map[string]any{
							"hash": "abcdef1234567890",
						},
					},
					"destination": map[string]any{
						"branch": map[string]any{
							"name": "main",
						},
					},
					"created_on": "2023-01-01T12:00:00Z",
				}, nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		pr, err := provider.GetPullRequest(ctx, 1)
		assert.NoError(t, err)
		assert.NotNil(t, pr)
		assert.Equal(t, int64(1), pr.Number)
		assert.Equal(t, "https://bitbucket.org/owner/repo/pull-requests/1", pr.URL)
		assert.Equal(t, "abcdef1234567890", pr.HeadSHA)
		assert.True(t, pr.Open)
		assert.False(t, pr.Merged)
		assert.NotNil(t, pr.CreatedAt)
	})

	t.Run("retrieval of merged PR", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
				return map[string]any{
					"id":    int64(1),
					"state": prStateMerged,
					"merge_commit": map[string]any{
						"hash": "short123",
					},
				}, nil
			},
			getCommitFunc: func(*bitbucket.CommitsOptions) (any, error) {
				return map[string]any{
					"hash": "full1234567890abcdef",
				}, nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		pr, err := provider.GetPullRequest(ctx, 1)
		assert.NoError(t, err)
		assert.NotNil(t, pr)
		assert.False(t, pr.Open)
		assert.True(t, pr.Merged)
		assert.Equal(t, "full1234567890abcdef", pr.MergeCommitSHA)
	})

	t.Run("error during retrieval", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
				return nil, errors.New("retrieval failed")
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		pr, err := provider.GetPullRequest(ctx, 1)
		assert.Error(t, err)
		assert.Nil(t, pr)
	})

	t.Run("error converting PR response", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
				// Return something that can't be properly unmarshaled
				return make(chan int), nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		pr, err := provider.GetPullRequest(ctx, 1)
		assert.Error(t, err)
		assert.Nil(t, pr)
	})
}

func TestListPullRequests(t *testing.T) {
	t.Run("list open PRs by default", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getsFunc: func(opt *bitbucket.PullRequestsOptions) (any, error) {
				assert.Equal(t, []string{prStateOpen}, opt.States)
				return map[string]any{"values": []any{
					map[string]any{"id": int64(1), "state": prStateOpen},
					map[string]any{"id": int64(2), "state": prStateOpen},
				}}, nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		prs, err := provider.ListPullRequests(ctx, nil)
		assert.NoError(t, err)
		assert.Len(t, prs, 2)
	})

	t.Run("list all PRs", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getsFunc: func(opt *bitbucket.PullRequestsOptions) (any, error) {
				assert.Contains(t, opt.States, prStateOpen)
				assert.Contains(t, opt.States, prStateMerged)
				assert.Contains(t, opt.States, prStateDeclined)
				assert.Contains(t, opt.States, prStateSuperseded)
				return map[string]any{"values": []any{
					map[string]any{"id": int64(1), "state": prStateOpen},
					map[string]any{"id": int64(2), "state": prStateMerged},
					map[string]any{"id": int64(3), "state": prStateDeclined},
					map[string]any{"id": int64(4), "state": prStateSuperseded},
				}}, nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		prs, err := provider.ListPullRequests(ctx, &gitprovider.ListPullRequestOptions{
			State: gitprovider.PullRequestStateAny,
		})
		assert.NoError(t, err)
		assert.Len(t, prs, 4)
	})

	t.Run("list closed PRs", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getsFunc: func(opt *bitbucket.PullRequestsOptions) (any, error) {
				assert.Contains(t, opt.States, prStateMerged)
				assert.Contains(t, opt.States, prStateDeclined)
				assert.Contains(t, opt.States, prStateSuperseded)
				assert.NotContains(t, opt.States, prStateOpen)
				return map[string]any{"values": []any{
					map[string]any{"id": int64(2), "state": prStateMerged},
					map[string]any{"id": int64(3), "state": prStateDeclined},
					map[string]any{"id": int64(4), "state": prStateSuperseded},
				}}, nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		prs, err := provider.ListPullRequests(ctx, &gitprovider.ListPullRequestOptions{
			State: gitprovider.PullRequestStateClosed,
		})
		assert.NoError(t, err)
		assert.Len(t, prs, 3)
	})

	t.Run("filter by head branch", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getsFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
				return map[string]any{"values": []any{
					map[string]any{
						"id":    int64(1),
						"state": prStateOpen,
						"source": map[string]any{
							"branch": map[string]any{
								"name": "feature-1",
							},
							"commit": map[string]any{
								"hash": "hash1",
							},
						},
						"destination": map[string]any{
							"branch": map[string]any{
								"name": "main",
							},
						},
					},
					map[string]any{
						"id":    int64(2),
						"state": prStateOpen,
						"source": map[string]any{
							"branch": map[string]any{
								"name": "feature-2",
							},
							"commit": map[string]any{
								"hash": "hash2",
							},
						},
						"destination": map[string]any{
							"branch": map[string]any{
								"name": "main",
							},
						},
					},
				}}, nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		prs, err := provider.ListPullRequests(ctx, &gitprovider.ListPullRequestOptions{
			HeadBranch: "feature-1",
		})
		assert.NoError(t, err)
		assert.Len(t, prs, 1)
		assert.Equal(t, int64(1), prs[0].Number)
	})

	t.Run("filter by base branch", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getsFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
				return map[string]any{"values": []any{
					map[string]any{
						"id":    int64(1),
						"state": prStateOpen,
						"source": map[string]any{
							"branch": map[string]any{
								"name": "feature-1",
							},
							"commit": map[string]any{
								"hash": "hash1",
							},
						},
						"destination": map[string]any{
							"branch": map[string]any{
								"name": "main",
							},
						},
					},
					map[string]any{
						"id":    int64(2),
						"state": prStateOpen,
						"source": map[string]any{
							"branch": map[string]any{
								"name": "feature-2",
							},
							"commit": map[string]any{
								"hash": "hash2",
							},
						},
						"destination": map[string]any{
							"branch": map[string]any{
								"name": "dev",
							},
						},
					},
				}}, nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		prs, err := provider.ListPullRequests(ctx, &gitprovider.ListPullRequestOptions{
			BaseBranch: "dev",
		})
		assert.NoError(t, err)
		assert.Len(t, prs, 1)
		assert.Equal(t, int64(2), prs[0].Number)
	})

	t.Run("filter by head commit", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getsFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
				return map[string]any{"values": []any{
					map[string]any{
						"id":    int64(1),
						"state": prStateOpen,
						"source": map[string]any{
							"branch": map[string]any{
								"name": "feature-1",
							},
							"commit": map[string]any{
								"hash": "specific-hash",
							},
						},
					},
					map[string]any{
						"id":    int64(2),
						"state": prStateOpen,
						"source": map[string]any{
							"branch": map[string]any{
								"name": "feature-2",
							},
							"commit": map[string]any{
								"hash": "other-hash",
							},
						},
					},
				}}, nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		prs, err := provider.ListPullRequests(ctx, &gitprovider.ListPullRequestOptions{
			HeadCommit: "specific-hash",
		})
		assert.NoError(t, err)
		assert.Len(t, prs, 1)
		assert.Equal(t, int64(1), prs[0].Number)
	})

	t.Run("PR with merge commit", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getsFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
				return map[string]any{"values": []any{
					map[string]any{
						"id":    int64(1),
						"state": prStateMerged,
						"merge_commit": map[string]any{
							"hash": "short123",
						},
					},
				}}, nil
			},
			getCommitFunc: func(*bitbucket.CommitsOptions) (any, error) {
				return map[string]any{
					"hash": "full1234567890abcdef",
				}, nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		prs, err := provider.ListPullRequests(ctx, nil)
		assert.NoError(t, err)
		assert.Len(t, prs, 1)
		assert.Equal(t, "full1234567890abcdef", prs[0].MergeCommitSHA)
	})

	t.Run("error during list", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getsFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
				return nil, errors.New("list failed")
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		prs, err := provider.ListPullRequests(ctx, nil)
		assert.Error(t, err)
		assert.Nil(t, prs)
	})

	t.Run("invalid response format", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getsFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
				return "not a map", nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		prs, err := provider.ListPullRequests(ctx, nil)
		assert.Error(t, err)
		assert.Nil(t, prs)
	})

	t.Run("missing values field", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getsFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
				return map[string]any{}, nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		prs, err := provider.ListPullRequests(ctx, nil)
		assert.Error(t, err)
		assert.Nil(t, prs)
	})

	t.Run("invalid values type", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getsFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
				return map[string]any{"values": "not an array"}, nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		prs, err := provider.ListPullRequests(ctx, nil)
		assert.Error(t, err)
		assert.Nil(t, prs)
	})

	t.Run("invalid state", func(t *testing.T) {
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
		}

		ctx := t.Context()
		prs, err := provider.ListPullRequests(ctx, &gitprovider.ListPullRequestOptions{
			State: "invalid-state",
		})
		assert.Error(t, err)
		assert.Nil(t, prs)
	})
}

func TestMergePullRequest(t *testing.T) {
	testCases := []struct {
		name           string
		prNumber       int64
		mergeOpts      *gitprovider.MergePullRequestOpts
		mockClient     *mockPullRequestClient
		expectedMerged bool
		expectError    bool
		errorContains  string
	}{
		{
			name:     "error getting PR",
			prNumber: 999,
			mockClient: &mockPullRequestClient{
				getFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
					return nil, errors.New("get PR failed")
				},
			},
			expectError:   true,
			errorContains: "error getting pull request",
		},
		{
			name:     "error parsing PR response",
			prNumber: 999,
			mockClient: &mockPullRequestClient{
				getFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
					return "not a valid response", nil
				},
			},
			expectError:   true,
			errorContains: "error parsing pull request response",
		},
		{
			name:     "PR already merged",
			prNumber: 123,
			mockClient: &mockPullRequestClient{
				getFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
					return map[string]any{
						"id":    int64(123),
						"state": prStateMerged,
						"links": map[string]any{
							"html": map[string]any{
								"href": "https://bitbucket.org/owner/repo/pull-requests/123",
							},
						},
						"merge_commit": map[string]any{
							"hash": "merge_sha",
						},
						"source": map[string]any{
							"commit": map[string]any{
								"hash": "head_sha",
							},
						},
					}, nil
				},
			},
			expectedMerged: true,
		},
		{
			name:     "PR declined",
			prNumber: 456,
			mockClient: &mockPullRequestClient{
				getFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
					return map[string]any{
						"id":    int64(456),
						"state": prStateDeclined,
					}, nil
				},
			},
			expectError:   true,
			errorContains: "closed but not merged",
		},
		{
			name:     "PR is draft",
			prNumber: 333,
			mockClient: &mockPullRequestClient{
				getFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
					return map[string]any{
						"id":    int64(333),
						"state": prStateOpen,
						"draft": true,
					}, nil
				},
			},
		},
		{
			name:      "merge method specified",
			prNumber:  100,
			mergeOpts: &gitprovider.MergePullRequestOpts{MergeMethod: "squash"},
			mockClient: &mockPullRequestClient{
				getFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
					return map[string]any{
						"id":    int64(100),
						"state": prStateOpen,
					}, nil
				},
			},
			expectError:   true,
			errorContains: "does not yet support specifying a merge method",
		},
		{
			name:     "merge operation fails",
			prNumber: 888,
			mockClient: &mockPullRequestClient{
				getFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
					return map[string]any{
						"id":    int64(888),
						"state": prStateOpen,
					}, nil
				},
				mergeFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
					return nil, errors.New("merge failed")
				},
			},
			expectError:   true,
			errorContains: "error merging pull request",
		},
		{
			name:     "error parsing merge response",
			prNumber: 777,
			mockClient: &mockPullRequestClient{
				getFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
					return map[string]any{
						"id":    int64(777),
						"state": prStateOpen,
					}, nil
				},
				mergeFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
					return "not a valid response", nil
				},
			},
			expectError:   true,
			errorContains: "error parsing merged pull request response",
		},
		{
			name:     "successful merge",
			prNumber: 1234,
			mockClient: &mockPullRequestClient{
				getFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
					return map[string]any{
						"id":    int64(1234),
						"state": prStateOpen,
						"links": map[string]any{
							"html": map[string]any{
								"href": "https://bitbucket.org/owner/repo/pull-requests/1234",
							},
						},
						"source": map[string]any{
							"commit": map[string]any{
								"hash": "head_sha",
							},
						},
					}, nil
				},
				mergeFunc: func(*bitbucket.PullRequestsOptions) (any, error) {
					return map[string]any{
						"id":    int64(1234),
						"state": prStateMerged,
						"links": map[string]any{
							"html": map[string]any{
								"href": "https://bitbucket.org/owner/repo/pull-requests/1234",
							},
						},
						"merge_commit": map[string]any{
							"hash": "merge_sha",
						},
						"source": map[string]any{
							"commit": map[string]any{
								"hash": "head_sha",
							},
						},
					}, nil
				},
			},
			expectedMerged: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := &provider{
				owner:    "owner",
				repoSlug: "repo",
				client:   tc.mockClient,
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

func TestGetFullCommitSHA(t *testing.T) {
	t.Run("successful retrieval", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getCommitFunc: func(opt *bitbucket.CommitsOptions) (any, error) {
				assert.Equal(t, "short123", opt.Revision)
				return map[string]any{
					"hash": "full1234567890abcdef",
				}, nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		sha, err := provider.getFullCommitSHA(ctx, "short123")
		assert.NoError(t, err)
		assert.Equal(t, "full1234567890abcdef", sha)
	})

	t.Run("empty SHA input", func(t *testing.T) {
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
		}

		ctx := t.Context()
		sha, err := provider.getFullCommitSHA(ctx, "")
		assert.NoError(t, err)
		assert.Equal(t, "", sha)
	})

	t.Run("error during retrieval", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getCommitFunc: func(*bitbucket.CommitsOptions) (any, error) {
				return nil, errors.New("retrieval failed")
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		sha, err := provider.getFullCommitSHA(ctx, "short123")
		assert.Error(t, err)
		assert.Equal(t, "", sha)
	})

	t.Run("invalid response format", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getCommitFunc: func(*bitbucket.CommitsOptions) (any, error) {
				return "not a map", nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		sha, err := provider.getFullCommitSHA(ctx, "short123")
		assert.Error(t, err)
		assert.Equal(t, "", sha)
	})

	t.Run("missing hash field", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getCommitFunc: func(*bitbucket.CommitsOptions) (any, error) {
				return map[string]any{}, nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		sha, err := provider.getFullCommitSHA(ctx, "short123")
		assert.Error(t, err)
		assert.Equal(t, "", sha)
	})

	t.Run("invalid hash type", func(t *testing.T) {
		mockClient := &mockPullRequestClient{
			getCommitFunc: func(*bitbucket.CommitsOptions) (any, error) {
				return map[string]any{
					"hash": 12345, // Not a string
				}, nil
			},
		}
		provider := &provider{
			owner:    "owner",
			repoSlug: "repo",
			client:   mockClient,
		}

		ctx := t.Context()
		sha, err := provider.getFullCommitSHA(ctx, "short123")
		assert.Error(t, err)
		assert.Equal(t, "", sha)
	})
}

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantHost  string
		wantOwner string
		wantSlug  string
		wantErr   bool
	}{
		{
			name:      "valid URL",
			url:       "https://bitbucket.org/owner/repo",
			wantHost:  "bitbucket.org",
			wantOwner: "owner",
			wantSlug:  "repo",
			wantErr:   false,
		},
		{
			name:      "valid URL with trailing slash",
			url:       "https://bitbucket.org/owner/repo/",
			wantHost:  "bitbucket.org",
			wantOwner: "owner",
			wantSlug:  "repo",
			wantErr:   false,
		},
		{
			name:      "valid URL with .git suffix",
			url:       "https://bitbucket.org/owner/repo.git",
			wantHost:  "bitbucket.org",
			wantOwner: "owner",
			wantSlug:  "repo",
			wantErr:   false,
		},
		{
			name:      "valid SSH URL",
			url:       "git@bitbucket.org:owner/repo.git",
			wantHost:  "bitbucket.org",
			wantOwner: "owner",
			wantSlug:  "repo",
			wantErr:   false,
		},
		{
			name:    "invalid URL format",
			url:     "://invalid-url",
			wantErr: true,
		},
		{
			name:    "missing repository name",
			url:     "https://bitbucket.org/owner",
			wantErr: true,
		},
		{
			name:    "too many path segments",
			url:     "https://bitbucket.org/owner/repo/extra",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, owner, slug, err := parseRepoURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantHost, host)
			assert.Equal(t, tt.wantOwner, owner)
			assert.Equal(t, tt.wantSlug, slug)
		})
	}
}

func Test_toBitbucketPR(t *testing.T) {
	t.Run("valid conversion", func(t *testing.T) {
		resp := map[string]any{
			"id":    int64(1),
			"state": prStateOpen,
			"links": map[string]any{
				"html": map[string]any{
					"href": "https://bitbucket.org/owner/repo/pull-requests/1",
				},
			},
			"source": map[string]any{
				"branch": map[string]any{
					"name": "feature-branch",
				},
				"commit": map[string]any{
					"hash": "abcdef1234567890",
				},
			},
			"destination": map[string]any{
				"branch": map[string]any{
					"name": "main",
				},
				"commit": map[string]any{
					"hash": "1234567890abcdef",
				},
			},
			"merge_commit": map[string]any{
				"hash": "merged1234567890",
			},
			"created_on": "2023-01-01T12:00:00Z",
		}

		pr, err := toBitbucketPR(resp)
		require.NoError(t, err)
		assert.Equal(t, int64(1), pr.ID)
		assert.Equal(t, prStateOpen, pr.State)
		assert.Equal(t, "https://bitbucket.org/owner/repo/pull-requests/1", pr.Links.HTML.Href)
		assert.Equal(t, "feature-branch", pr.Source.Branch.Name)
		assert.Equal(t, "abcdef1234567890", pr.Source.Commit.Hash)
		assert.Equal(t, "main", pr.Destination.Branch.Name)
		assert.Equal(t, "1234567890abcdef", pr.Destination.Commit.Hash)
		assert.Equal(t, "merged1234567890", pr.MergeCommit.Hash)
		assert.Equal(t, "2023-01-01T12:00:00Z", pr.CreatedOn)
	})

	t.Run("invalid input type", func(t *testing.T) {
		pr, err := toBitbucketPR(make(chan int))
		assert.Error(t, err)
		assert.Nil(t, pr)
	})
}

func Test_toProviderPR(t *testing.T) {
	t.Run("valid conversion: open PR", func(t *testing.T) {
		bbPR := &bitbucketPR{
			ID:    1,
			State: prStateOpen,
			Links: struct {
				HTML struct {
					Href string `json:"href"`
				} `json:"html"`
			}{
				HTML: struct {
					Href string `json:"href"`
				}{
					Href: "https://bitbucket.org/owner/repo/pull-requests/1",
				},
			},
			Source: struct {
				Branch struct {
					Name string `json:"name"`
				} `json:"branch"`
				Commit struct {
					Hash string `json:"hash"`
				} `json:"commit"`
			}{
				Branch: struct {
					Name string `json:"name"`
				}{
					Name: "feature-branch",
				},
				Commit: struct {
					Hash string `json:"hash"`
				}{
					Hash: "abcdef1234567890",
				},
			},
			CreatedOn: "2023-01-01T12:00:00Z",
		}

		raw := map[string]any{"id": 1}
		pr := toProviderPR(bbPR, raw)
		require.NotNil(t, pr)
		assert.Equal(t, int64(1), pr.Number)
		assert.Equal(t, "https://bitbucket.org/owner/repo/pull-requests/1", pr.URL)
		assert.Equal(t, "abcdef1234567890", pr.HeadSHA)
		assert.True(t, pr.Open)
		assert.False(t, pr.Merged)
		assert.NotNil(t, pr.CreatedAt)
		assert.Equal(t, raw, pr.Object)
	})

	t.Run("valid conversion: merged PR", func(t *testing.T) {
		bbPR := &bitbucketPR{
			ID:    1,
			State: prStateMerged,
			MergeCommit: struct {
				Hash string `json:"hash"`
			}{
				Hash: "merged1234567890",
			},
			CreatedOn: "2023-01-01T12:00:00Z",
		}

		pr := toProviderPR(bbPR, nil)
		require.NotNil(t, pr)
		assert.False(t, pr.Open)
		assert.True(t, pr.Merged)
		assert.Equal(t, "merged1234567890", pr.MergeCommitSHA)
	})

	t.Run("nil input", func(t *testing.T) {
		pr := toProviderPR(nil, nil)
		assert.Nil(t, pr)
	})

	t.Run("invalid date", func(t *testing.T) {
		bbPR := &bitbucketPR{
			ID:        1,
			CreatedOn: "not-a-date",
		}

		pr := toProviderPR(bbPR, nil)
		require.NotNil(t, pr)
		assert.Nil(t, pr.CreatedAt)
	})
}

func Test_registration(t *testing.T) {
	t.Run("predicate matches bitbucket.org URL", func(t *testing.T) {
		result := registration.Predicate("https://bitbucket.org/owner/repo")
		assert.True(t, result)
	})

	t.Run("predicate doesn't match other URLs", func(t *testing.T) {
		result := registration.Predicate("https://github.com/owner/repo")
		assert.False(t, result)
	})

	t.Run("predicate handles invalid URLs", func(t *testing.T) {
		result := registration.Predicate("://invalid-url")
		assert.False(t, result)
	})

	t.Run("NewProvider factory works", func(t *testing.T) {
		provider, err := registration.NewProvider("https://bitbucket.org/owner/repo", nil)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})
}

func TestGetCommitURL(t *testing.T) {
	testCases := []struct {
		repoURL           string
		sha               string
		expectedCommitURL string
	}{
		{
			repoURL:           "ssh://git@bitbucket.org/akuity/kargo.git",
			sha:               "sha",
			expectedCommitURL: "https://bitbucket.org/akuity/kargo/commits/sha",
		},
		{
			repoURL:           "git@bitbucket.org:akuity/kargo.git",
			sha:               "sha",
			expectedCommitURL: "https://bitbucket.org/akuity/kargo/commits/sha",
		},
		{
			repoURL:           "https://username@bitbucket.org/akuity/kargo",
			sha:               "sha",
			expectedCommitURL: "https://bitbucket.org/akuity/kargo/commits/sha",
		},
		{
			repoURL:           "http://bitbucket.org/akuity/kargo.git",
			sha:               "sha",
			expectedCommitURL: "https://bitbucket.org/akuity/kargo/commits/sha",
		},
	}

	prov := &provider{}

	for _, testCase := range testCases {
		t.Run(testCase.repoURL, func(t *testing.T) {
			commitURL, err := prov.GetCommitURL(testCase.repoURL, testCase.sha)
			require.NoError(t, err)
			require.Equal(t, testCase.expectedCommitURL, commitURL)
		})
	}
}
