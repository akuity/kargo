package datacenter

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/gitprovider"
)

// mockClient implements ClientWithResponsesInterface for testing.
type mockClient struct {
	getPRsFunc func(
		ctx context.Context,
		projectKey, repoSlug string,
		params *GetPullRequestsParams,
		reqEditors ...RequestEditorFn,
	) (*GetPullRequestsResponse, error)

	createPRFunc func(
		ctx context.Context,
		projectKey, repoSlug string,
		body CreatePullRequestJSONRequestBody,
		reqEditors ...RequestEditorFn,
	) (*CreatePullRequestResponse, error)

	getPRFunc func(
		ctx context.Context,
		projectKey, repoSlug string,
		pullRequestId int,
		reqEditors ...RequestEditorFn,
	) (*GetPullRequestResponse, error)

	mergePRFunc func(
		ctx context.Context,
		projectKey, repoSlug string,
		pullRequestId int,
		params *MergePullRequestParams,
		body MergePullRequestJSONRequestBody,
		reqEditors ...RequestEditorFn,
	) (*MergePullRequestResponse, error)
}

func (m *mockClient) GetCommitWithResponse(
	_ context.Context,
	_, _, _ string,
	_ ...RequestEditorFn,
) (*GetCommitResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockClient) GetPullRequestsWithResponse(
	ctx context.Context,
	projectKey, repoSlug string,
	params *GetPullRequestsParams,
	reqEditors ...RequestEditorFn,
) (*GetPullRequestsResponse, error) {
	return m.getPRsFunc(ctx, projectKey, repoSlug, params, reqEditors...)
}

func (m *mockClient) CreatePullRequestWithBodyWithResponse(
	_ context.Context,
	_, _ string,
	_ string,
	_ io.Reader,
	_ ...RequestEditorFn,
) (*CreatePullRequestResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockClient) CreatePullRequestWithResponse(
	ctx context.Context,
	projectKey, repoSlug string,
	body CreatePullRequestJSONRequestBody,
	reqEditors ...RequestEditorFn,
) (*CreatePullRequestResponse, error) {
	return m.createPRFunc(ctx, projectKey, repoSlug, body, reqEditors...)
}

func (m *mockClient) GetPullRequestWithResponse(
	ctx context.Context,
	projectKey, repoSlug string,
	pullRequestId int,
	reqEditors ...RequestEditorFn,
) (*GetPullRequestResponse, error) {
	return m.getPRFunc(ctx, projectKey, repoSlug, pullRequestId, reqEditors...)
}

func (m *mockClient) MergePullRequestWithBodyWithResponse(
	_ context.Context,
	_, _ string,
	_ int,
	_ *MergePullRequestParams,
	_ string,
	_ io.Reader,
	_ ...RequestEditorFn,
) (*MergePullRequestResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockClient) MergePullRequestWithResponse(
	ctx context.Context,
	projectKey, repoSlug string,
	pullRequestId int,
	params *MergePullRequestParams,
	body MergePullRequestJSONRequestBody,
	reqEditors ...RequestEditorFn,
) (*MergePullRequestResponse, error) {
	return m.mergePRFunc(ctx, projectKey, repoSlug, pullRequestId, params, body, reqEditors...)
}

func intPtr(i int) *int                                     { return &i }
func strPtr(s string) *string                               { return &s }
func boolPtr(b bool) *bool                                  { return &b }
func statePtr(s RestPullRequestState) *RestPullRequestState { return &s }

// makePR builds a minimal RestPullRequest for use in test cases.
func makePR(id int, state RestPullRequestState) *RestPullRequest {
	return &RestPullRequest{
		Id:      &id,
		State:   statePtr(state),
		Version: intPtr(1),
	}
}

func Test_registration(t *testing.T) {
	t.Parallel()

	t.Run("predicate matches self-hosted bitbucket hostname", func(t *testing.T) {
		t.Parallel()
		assert.True(t, registration.Predicate("https://bitbucket.example.com/projects/PROJ/repos/repo"))
	})

	t.Run("predicate matches subdomain of bitbucket", func(t *testing.T) {
		t.Parallel()
		assert.True(t, registration.Predicate("https://git.bitbucket.corp.io/projects/PROJ/repos/repo"))
	})

	t.Run("predicate does not match bitbucket.org (Cloud)", func(t *testing.T) {
		t.Parallel()
		assert.False(t, registration.Predicate("https://bitbucket.org/owner/repo"))
	})

	t.Run("predicate does not match other providers", func(t *testing.T) {
		t.Parallel()
		assert.False(t, registration.Predicate("https://github.com/owner/repo"))
	})

	t.Run("predicate handles invalid URLs", func(t *testing.T) {
		t.Parallel()
		assert.False(t, registration.Predicate("://invalid-url"))
	})

	t.Run("NewProvider factory works", func(t *testing.T) {
		t.Parallel()
		p, err := registration.NewProvider(
			"https://bitbucket.example.com/projects/PROJ/repos/repo",
			nil,
		)
		assert.NoError(t, err)
		assert.NotNil(t, p)
	})
}

func TestNewProvider(t *testing.T) {
	t.Parallel()

	t.Run("successful creation with token", func(t *testing.T) {
		t.Parallel()
		p, err := NewProvider(
			"https://bitbucket.example.com/projects/PROJ/repos/myrepo",
			&gitprovider.Options{Token: "token"},
		)
		assert.NoError(t, err)
		assert.NotNil(t, p)
	})

	t.Run("successful creation with nil options", func(t *testing.T) {
		t.Parallel()
		p, err := NewProvider(
			"https://bitbucket.example.com/projects/PROJ/repos/myrepo",
			nil,
		)
		assert.NoError(t, err)
		assert.NotNil(t, p)
	})

	t.Run("error with invalid URL path", func(t *testing.T) {
		t.Parallel()
		p, err := NewProvider(
			"https://bitbucket.example.com/invalid",
			&gitprovider.Options{},
		)
		assert.Error(t, err)
		assert.Nil(t, p)
	})
}

func TestCreatePullRequest(t *testing.T) {
	t.Parallel()

	t.Run("successful creation", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			createPRFunc: func(
				_ context.Context,
				projectKey, repoSlug string,
				body CreatePullRequestJSONRequestBody,
				_ ...RequestEditorFn,
			) (*CreatePullRequestResponse, error) {
				assert.Equal(t, "PROJ", projectKey)
				assert.Equal(t, "myrepo", repoSlug)
				assert.Equal(t, "Test PR", body.Title)
				assert.Equal(t, "refs/heads/feature", body.FromRef.Id)
				assert.Equal(t, "refs/heads/main", body.ToRef.Id)
				require.NotNil(t, body.Description)
				assert.Equal(t, "PR description", *body.Description)

				prURL := "https://bitbucket.example.com/projects/PROJ/repos/myrepo/pull-requests/1"
				return &CreatePullRequestResponse{
					JSON201: &RestPullRequest{
						Id:    intPtr(1),
						State: statePtr(RestPullRequestStateOPEN),
						Links: &RestPullRequestLinks{
							Self: &[]RestHref{{Href: &prURL}},
						},
						FromRef: &RestRef{
							LatestCommit: strPtr("abc123"),
						},
					},
				}, nil
			},
		}
		p := &provider{
			projectKey: "PROJ",
			repoSlug:   "myrepo",
			client:     mc,
		}
		pr, err := p.CreatePullRequest(t.Context(), &gitprovider.CreatePullRequestOpts{
			Title:       "Test PR",
			Description: "PR description",
			Head:        "feature",
			Base:        "main",
		})
		require.NoError(t, err)
		require.NotNil(t, pr)
		assert.Equal(t, int64(1), pr.Number)
		assert.Equal(t, "https://bitbucket.example.com/projects/PROJ/repos/myrepo/pull-requests/1", pr.URL)
		assert.Equal(t, "abc123", pr.HeadSHA)
		assert.True(t, pr.Open)
		assert.False(t, pr.Merged)
	})

	t.Run("successful creation with nil options", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			createPRFunc: func(
				_ context.Context,
				_, _ string,
				body CreatePullRequestJSONRequestBody,
				_ ...RequestEditorFn,
			) (*CreatePullRequestResponse, error) {
				assert.Nil(t, body.Description) // no description set when empty
				return &CreatePullRequestResponse{
					JSON201: makePR(42, RestPullRequestStateOPEN),
				}, nil
			},
		}
		p := &provider{client: mc}
		pr, err := p.CreatePullRequest(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, pr)
		assert.Equal(t, int64(42), pr.Number)
	})

	t.Run("error from API", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			createPRFunc: func(
				_ context.Context,
				_, _ string,
				_ CreatePullRequestJSONRequestBody,
				_ ...RequestEditorFn,
			) (*CreatePullRequestResponse, error) {
				return nil, errors.New("network error")
			},
		}
		p := &provider{client: mc}
		pr, err := p.CreatePullRequest(t.Context(), nil)
		assert.Error(t, err)
		assert.Nil(t, pr)
	})

	t.Run("unexpected non-201 response", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			createPRFunc: func(
				_ context.Context,
				_, _ string,
				_ CreatePullRequestJSONRequestBody,
				_ ...RequestEditorFn,
			) (*CreatePullRequestResponse, error) {
				return &CreatePullRequestResponse{}, nil // JSON201 is nil
			},
		}
		p := &provider{client: mc}
		pr, err := p.CreatePullRequest(t.Context(), nil)
		assert.Error(t, err)
		assert.Nil(t, pr)
	})
}

func TestGetPullRequest(t *testing.T) {
	t.Parallel()

	t.Run("successful retrieval of open PR", func(t *testing.T) {
		t.Parallel()
		prURL := "https://bitbucket.example.com/projects/PROJ/repos/myrepo/pull-requests/5"
		createdMs := int64(1672574400000) // 2023-01-01T12:00:00Z
		mc := &mockClient{
			getPRFunc: func(
				_ context.Context,
				projectKey, repoSlug string,
				pullRequestId int,
				_ ...RequestEditorFn,
			) (*GetPullRequestResponse, error) {
				assert.Equal(t, "PROJ", projectKey)
				assert.Equal(t, "myrepo", repoSlug)
				assert.Equal(t, 5, pullRequestId)
				return &GetPullRequestResponse{
					JSON200: &RestPullRequest{
						Id:          intPtr(5),
						State:       statePtr(RestPullRequestStateOPEN),
						Version:     intPtr(2),
						CreatedDate: &createdMs,
						Links:       &RestPullRequestLinks{Self: &[]RestHref{{Href: &prURL}}},
						FromRef:     &RestRef{LatestCommit: strPtr("deadbeef")},
					},
				}, nil
			},
		}
		p := &provider{
			projectKey: "PROJ",
			repoSlug:   "myrepo",
			client:     mc,
		}
		pr, err := p.GetPullRequest(t.Context(), 5)
		require.NoError(t, err)
		require.NotNil(t, pr)
		assert.Equal(t, int64(5), pr.Number)
		assert.Equal(t, prURL, pr.URL)
		assert.Equal(t, "deadbeef", pr.HeadSHA)
		assert.True(t, pr.Open)
		assert.False(t, pr.Merged)
		require.NotNil(t, pr.CreatedAt)
		assert.Equal(t, time.UnixMilli(createdMs).UTC(), *pr.CreatedAt)
	})

	t.Run("successful retrieval of merged PR", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRFunc: func(
				_ context.Context,
				_, _ string,
				_ int,
				_ ...RequestEditorFn,
			) (*GetPullRequestResponse, error) {
				return &GetPullRequestResponse{
					JSON200: makePR(3, RestPullRequestStateMERGED),
				}, nil
			},
		}
		p := &provider{client: mc}
		pr, err := p.GetPullRequest(t.Context(), 3)
		require.NoError(t, err)
		require.NotNil(t, pr)
		assert.False(t, pr.Open)
		assert.True(t, pr.Merged)
		assert.Equal(t, "", pr.MergeCommitSHA) // Data Center does not include merge commit SHA
	})

	t.Run("error from API", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRFunc: func(
				_ context.Context,
				_, _ string,
				_ int,
				_ ...RequestEditorFn,
			) (*GetPullRequestResponse, error) {
				return nil, errors.New("not found")
			},
		}
		p := &provider{client: mc}
		pr, err := p.GetPullRequest(t.Context(), 1)
		assert.Error(t, err)
		assert.Nil(t, pr)
	})

	t.Run("unexpected non-200 response", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRFunc: func(
				_ context.Context,
				_, _ string,
				_ int,
				_ ...RequestEditorFn,
			) (*GetPullRequestResponse, error) {
				return &GetPullRequestResponse{}, nil // JSON200 is nil
			},
		}
		p := &provider{client: mc}
		pr, err := p.GetPullRequest(t.Context(), 1)
		assert.Error(t, err)
		assert.Nil(t, pr)
	})
}

func TestListPullRequests(t *testing.T) {
	t.Parallel()

	open := RestPullRequestStateOPEN
	merged := RestPullRequestStateMERGED
	declined := RestPullRequestStateDECLINED

	t.Run("open state uses OPEN param", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRsFunc: func(
				_ context.Context,
				_, _ string,
				params *GetPullRequestsParams,
				_ ...RequestEditorFn,
			) (*GetPullRequestsResponse, error) {
				require.NotNil(t, params.State)
				assert.Equal(t, GetPullRequestsParamsStateOPEN, *params.State)
				return &GetPullRequestsResponse{
					JSON200: &RestPullRequestPage{
						Values: &[]RestPullRequest{
							*makePR(1, RestPullRequestStateOPEN),
						},
					},
				}, nil
			},
		}
		p := &provider{client: mc}
		prs, err := p.ListPullRequests(t.Context(), &gitprovider.ListPullRequestOptions{
			State: gitprovider.PullRequestStateOpen,
		})
		require.NoError(t, err)
		assert.Len(t, prs, 1)
	})

	t.Run("nil options defaults to open state", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRsFunc: func(
				_ context.Context,
				_, _ string,
				params *GetPullRequestsParams,
				_ ...RequestEditorFn,
			) (*GetPullRequestsResponse, error) {
				require.NotNil(t, params.State)
				assert.Equal(t, GetPullRequestsParamsStateOPEN, *params.State)
				return &GetPullRequestsResponse{
					JSON200: &RestPullRequestPage{Values: &[]RestPullRequest{}},
				}, nil
			},
		}
		p := &provider{client: mc}
		prs, err := p.ListPullRequests(t.Context(), nil)
		require.NoError(t, err)
		assert.Empty(t, prs)
	})

	t.Run("closed state uses ALL and filters out OPEN PRs", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRsFunc: func(
				_ context.Context,
				_, _ string,
				params *GetPullRequestsParams,
				_ ...RequestEditorFn,
			) (*GetPullRequestsResponse, error) {
				require.NotNil(t, params.State)
				assert.Equal(t, GetPullRequestsParamsStateALL, *params.State)
				return &GetPullRequestsResponse{
					JSON200: &RestPullRequestPage{
						Values: &[]RestPullRequest{
							{Id: intPtr(1), State: &open},
							{Id: intPtr(2), State: &merged},
							{Id: intPtr(3), State: &declined},
						},
					},
				}, nil
			},
		}
		p := &provider{client: mc}
		prs, err := p.ListPullRequests(t.Context(), &gitprovider.ListPullRequestOptions{
			State: gitprovider.PullRequestStateClosed,
		})
		require.NoError(t, err)
		// OPEN PR should be excluded; MERGED and DECLINED included
		assert.Len(t, prs, 2)
	})

	t.Run("any state uses ALL and returns everything", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRsFunc: func(
				_ context.Context,
				_, _ string,
				params *GetPullRequestsParams,
				_ ...RequestEditorFn,
			) (*GetPullRequestsResponse, error) {
				require.NotNil(t, params.State)
				assert.Equal(t, GetPullRequestsParamsStateALL, *params.State)
				return &GetPullRequestsResponse{
					JSON200: &RestPullRequestPage{
						Values: &[]RestPullRequest{
							{Id: intPtr(1), State: &open},
							{Id: intPtr(2), State: &merged},
						},
					},
				}, nil
			},
		}
		p := &provider{client: mc}
		prs, err := p.ListPullRequests(t.Context(), &gitprovider.ListPullRequestOptions{
			State: gitprovider.PullRequestStateAny,
		})
		require.NoError(t, err)
		assert.Len(t, prs, 2)
	})

	t.Run("filters by head branch", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRsFunc: func(
				_ context.Context,
				_, _ string,
				_ *GetPullRequestsParams,
				_ ...RequestEditorFn,
			) (*GetPullRequestsResponse, error) {
				return &GetPullRequestsResponse{
					JSON200: &RestPullRequestPage{
						Values: &[]RestPullRequest{
							{Id: intPtr(1), State: &open, FromRef: &RestRef{DisplayId: strPtr("feature")}},
							{Id: intPtr(2), State: &open, FromRef: &RestRef{DisplayId: strPtr("other")}},
						},
					},
				}, nil
			},
		}
		p := &provider{client: mc}
		prs, err := p.ListPullRequests(t.Context(), &gitprovider.ListPullRequestOptions{
			State:      gitprovider.PullRequestStateOpen,
			HeadBranch: "feature",
		})
		require.NoError(t, err)
		require.Len(t, prs, 1)
		assert.Equal(t, int64(1), prs[0].Number)
	})

	t.Run("filters by base branch", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRsFunc: func(
				_ context.Context,
				_, _ string,
				_ *GetPullRequestsParams,
				_ ...RequestEditorFn,
			) (*GetPullRequestsResponse, error) {
				return &GetPullRequestsResponse{
					JSON200: &RestPullRequestPage{
						Values: &[]RestPullRequest{
							{Id: intPtr(1), State: &open, ToRef: &RestRef{DisplayId: strPtr("main")}},
							{Id: intPtr(2), State: &open, ToRef: &RestRef{DisplayId: strPtr("dev")}},
						},
					},
				}, nil
			},
		}
		p := &provider{client: mc}
		prs, err := p.ListPullRequests(t.Context(), &gitprovider.ListPullRequestOptions{
			State:      gitprovider.PullRequestStateOpen,
			BaseBranch: "main",
		})
		require.NoError(t, err)
		require.Len(t, prs, 1)
		assert.Equal(t, int64(1), prs[0].Number)
	})

	t.Run("filters by head commit", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRsFunc: func(
				_ context.Context,
				_, _ string,
				_ *GetPullRequestsParams,
				_ ...RequestEditorFn,
			) (*GetPullRequestsResponse, error) {
				return &GetPullRequestsResponse{
					JSON200: &RestPullRequestPage{
						Values: &[]RestPullRequest{
							{Id: intPtr(1), State: &open, FromRef: &RestRef{LatestCommit: strPtr("abc123")}},
							{Id: intPtr(2), State: &open, FromRef: &RestRef{LatestCommit: strPtr("def456")}},
						},
					},
				}, nil
			},
		}
		p := &provider{client: mc}
		prs, err := p.ListPullRequests(t.Context(), &gitprovider.ListPullRequestOptions{
			State:      gitprovider.PullRequestStateOpen,
			HeadCommit: "abc123",
		})
		require.NoError(t, err)
		require.Len(t, prs, 1)
		assert.Equal(t, int64(1), prs[0].Number)
	})

	t.Run("nil values returns empty slice", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRsFunc: func(
				_ context.Context,
				_, _ string,
				_ *GetPullRequestsParams,
				_ ...RequestEditorFn,
			) (*GetPullRequestsResponse, error) {
				return &GetPullRequestsResponse{
					JSON200: &RestPullRequestPage{Values: nil},
				}, nil
			},
		}
		p := &provider{client: mc}
		prs, err := p.ListPullRequests(t.Context(), nil)
		require.NoError(t, err)
		assert.Empty(t, prs)
	})

	t.Run("error from API", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRsFunc: func(
				_ context.Context,
				_, _ string,
				_ *GetPullRequestsParams,
				_ ...RequestEditorFn,
			) (*GetPullRequestsResponse, error) {
				return nil, errors.New("server error")
			},
		}
		p := &provider{client: mc}
		prs, err := p.ListPullRequests(t.Context(), nil)
		assert.Error(t, err)
		assert.Nil(t, prs)
	})

	t.Run("unexpected non-200 response", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRsFunc: func(
				_ context.Context,
				_, _ string,
				_ *GetPullRequestsParams,
				_ ...RequestEditorFn,
			) (*GetPullRequestsResponse, error) {
				return &GetPullRequestsResponse{}, nil // JSON200 is nil
			},
		}
		p := &provider{client: mc}
		prs, err := p.ListPullRequests(t.Context(), nil)
		assert.Error(t, err)
		assert.Nil(t, prs)
	})

	t.Run("unknown state returns error", func(t *testing.T) {
		t.Parallel()
		p := &provider{client: &mockClient{}}
		prs, err := p.ListPullRequests(t.Context(), &gitprovider.ListPullRequestOptions{
			State: "bogus",
		})
		assert.Error(t, err)
		assert.Nil(t, prs)
	})
}

func TestMergePullRequest(t *testing.T) {
	t.Parallel()

	t.Run("already merged PR returns early", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRFunc: func(
				_ context.Context,
				_, _ string,
				_ int,
				_ ...RequestEditorFn,
			) (*GetPullRequestResponse, error) {
				return &GetPullRequestResponse{
					JSON200: makePR(1, RestPullRequestStateMERGED),
				}, nil
			},
		}
		p := &provider{client: mc}
		pr, merged, err := p.MergePullRequest(t.Context(), 1, nil)
		require.NoError(t, err)
		assert.True(t, merged)
		require.NotNil(t, pr)
		assert.True(t, pr.Merged)
	})

	t.Run("declined PR returns error", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRFunc: func(
				_ context.Context,
				_, _ string,
				_ int,
				_ ...RequestEditorFn,
			) (*GetPullRequestResponse, error) {
				return &GetPullRequestResponse{
					JSON200: makePR(1, RestPullRequestStateDECLINED),
				}, nil
			},
		}
		p := &provider{client: mc}
		pr, merged, err := p.MergePullRequest(t.Context(), 1, nil)
		assert.Error(t, err)
		assert.False(t, merged)
		assert.Nil(t, pr)
	})

	t.Run("draft PR returns not-ready (nil, false, nil)", func(t *testing.T) {
		t.Parallel()
		pr := makePR(1, RestPullRequestStateOPEN)
		pr.Draft = boolPtr(true)
		mc := &mockClient{
			getPRFunc: func(
				_ context.Context,
				_, _ string,
				_ int,
				_ ...RequestEditorFn,
			) (*GetPullRequestResponse, error) {
				return &GetPullRequestResponse{JSON200: pr}, nil
			},
		}
		p := &provider{client: mc}
		result, merged, err := p.MergePullRequest(t.Context(), 1, nil)
		assert.NoError(t, err)
		assert.False(t, merged)
		assert.Nil(t, result)
	})

	t.Run("successful merge uses version from GET response", func(t *testing.T) {
		t.Parallel()
		getPR := makePR(7, RestPullRequestStateOPEN)
		getPR.Version = intPtr(3)
		mc := &mockClient{
			getPRFunc: func(
				_ context.Context,
				_, _ string,
				pullRequestId int,
				_ ...RequestEditorFn,
			) (*GetPullRequestResponse, error) {
				assert.Equal(t, 7, pullRequestId)
				return &GetPullRequestResponse{JSON200: getPR}, nil
			},
			mergePRFunc: func(
				_ context.Context,
				_, _ string,
				pullRequestId int,
				params *MergePullRequestParams,
				_ MergePullRequestJSONRequestBody,
				_ ...RequestEditorFn,
			) (*MergePullRequestResponse, error) {
				assert.Equal(t, 7, pullRequestId)
				assert.Equal(t, 3, params.Version)
				return &MergePullRequestResponse{
					JSON200: makePR(7, RestPullRequestStateMERGED),
				}, nil
			},
		}
		p := &provider{client: mc}
		pr, merged, err := p.MergePullRequest(t.Context(), 7, nil)
		require.NoError(t, err)
		assert.True(t, merged)
		require.NotNil(t, pr)
		assert.True(t, pr.Merged)
	})

	t.Run("merge with explicit strategy", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRFunc: func(
				_ context.Context,
				_, _ string,
				_ int,
				_ ...RequestEditorFn,
			) (*GetPullRequestResponse, error) {
				return &GetPullRequestResponse{JSON200: makePR(1, RestPullRequestStateOPEN)}, nil
			},
			mergePRFunc: func(
				_ context.Context,
				_, _ string,
				_ int,
				_ *MergePullRequestParams,
				body MergePullRequestJSONRequestBody,
				_ ...RequestEditorFn,
			) (*MergePullRequestResponse, error) {
				require.NotNil(t, body.Strategy)
				require.NotNil(t, body.Strategy.Id)
				assert.Equal(t, Squash, *body.Strategy.Id)
				return &MergePullRequestResponse{
					JSON200: makePR(1, RestPullRequestStateMERGED),
				}, nil
			},
		}
		p := &provider{client: mc}
		pr, merged, err := p.MergePullRequest(t.Context(), 1, &gitprovider.MergePullRequestOpts{
			MergeMethod: "squash",
		})
		require.NoError(t, err)
		assert.True(t, merged)
		assert.NotNil(t, pr)
	})

	t.Run("invalid merge strategy returns error", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRFunc: func(
				_ context.Context,
				_, _ string,
				_ int,
				_ ...RequestEditorFn,
			) (*GetPullRequestResponse, error) {
				return &GetPullRequestResponse{JSON200: makePR(1, RestPullRequestStateOPEN)}, nil
			},
		}
		p := &provider{client: mc}
		pr, merged, err := p.MergePullRequest(t.Context(), 1, &gitprovider.MergePullRequestOpts{
			MergeMethod: "bad-strategy",
		})
		assert.Error(t, err)
		assert.False(t, merged)
		assert.Nil(t, pr)
	})

	t.Run("error getting PR before merge", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRFunc: func(
				_ context.Context,
				_, _ string,
				_ int,
				_ ...RequestEditorFn,
			) (*GetPullRequestResponse, error) {
				return nil, errors.New("network error")
			},
		}
		p := &provider{client: mc}
		pr, merged, err := p.MergePullRequest(t.Context(), 1, nil)
		assert.Error(t, err)
		assert.False(t, merged)
		assert.Nil(t, pr)
	})

	t.Run("non-200 GET response returns error", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRFunc: func(
				_ context.Context,
				_, _ string,
				_ int,
				_ ...RequestEditorFn,
			) (*GetPullRequestResponse, error) {
				return &GetPullRequestResponse{}, nil // JSON200 is nil
			},
		}
		p := &provider{client: mc}
		pr, merged, err := p.MergePullRequest(t.Context(), 1, nil)
		assert.Error(t, err)
		assert.False(t, merged)
		assert.Nil(t, pr)
	})

	t.Run("error during merge API call", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRFunc: func(
				_ context.Context,
				_, _ string,
				_ int,
				_ ...RequestEditorFn,
			) (*GetPullRequestResponse, error) {
				return &GetPullRequestResponse{JSON200: makePR(1, RestPullRequestStateOPEN)}, nil
			},
			mergePRFunc: func(
				_ context.Context,
				_, _ string,
				_ int,
				_ *MergePullRequestParams,
				_ MergePullRequestJSONRequestBody,
				_ ...RequestEditorFn,
			) (*MergePullRequestResponse, error) {
				return nil, errors.New("merge conflict")
			},
		}
		p := &provider{client: mc}
		pr, merged, err := p.MergePullRequest(t.Context(), 1, nil)
		assert.Error(t, err)
		assert.False(t, merged)
		assert.Nil(t, pr)
	})

	t.Run("non-200 merge response returns error", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRFunc: func(
				_ context.Context,
				_, _ string,
				_ int,
				_ ...RequestEditorFn,
			) (*GetPullRequestResponse, error) {
				return &GetPullRequestResponse{JSON200: makePR(1, RestPullRequestStateOPEN)}, nil
			},
			mergePRFunc: func(
				_ context.Context,
				_, _ string,
				_ int,
				_ *MergePullRequestParams,
				_ MergePullRequestJSONRequestBody,
				_ ...RequestEditorFn,
			) (*MergePullRequestResponse, error) {
				return &MergePullRequestResponse{}, nil // JSON200 is nil
			},
		}
		p := &provider{client: mc}
		pr, merged, err := p.MergePullRequest(t.Context(), 1, nil)
		assert.Error(t, err)
		assert.False(t, merged)
		assert.Nil(t, pr)
	})

	t.Run("merge response in non-MERGED state returns error", func(t *testing.T) {
		t.Parallel()
		mc := &mockClient{
			getPRFunc: func(
				_ context.Context,
				_, _ string,
				_ int,
				_ ...RequestEditorFn,
			) (*GetPullRequestResponse, error) {
				return &GetPullRequestResponse{JSON200: makePR(1, RestPullRequestStateOPEN)}, nil
			},
			mergePRFunc: func(
				_ context.Context,
				_, _ string,
				_ int,
				_ *MergePullRequestParams,
				_ MergePullRequestJSONRequestBody,
				_ ...RequestEditorFn,
			) (*MergePullRequestResponse, error) {
				return &MergePullRequestResponse{
					JSON200: makePR(1, RestPullRequestStateOPEN), // Still open — unexpected
				}, nil
			},
		}
		p := &provider{client: mc}
		pr, merged, err := p.MergePullRequest(t.Context(), 1, nil)
		assert.Error(t, err)
		assert.False(t, merged)
		assert.Nil(t, pr)
	})
}

func Test_toProviderPR(t *testing.T) {
	t.Parallel()

	t.Run("nil input returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, toProviderPR(nil))
	})

	t.Run("open PR", func(t *testing.T) {
		t.Parallel()
		pr := toProviderPR(&RestPullRequest{
			Id:    intPtr(10),
			State: statePtr(RestPullRequestStateOPEN),
		})
		require.NotNil(t, pr)
		assert.Equal(t, int64(10), pr.Number)
		assert.True(t, pr.Open)
		assert.False(t, pr.Merged)
	})

	t.Run("merged PR", func(t *testing.T) {
		t.Parallel()
		pr := toProviderPR(&RestPullRequest{
			Id:    intPtr(11),
			State: statePtr(RestPullRequestStateMERGED),
		})
		require.NotNil(t, pr)
		assert.False(t, pr.Open)
		assert.True(t, pr.Merged)
		assert.Equal(t, "", pr.MergeCommitSHA) // never populated for Data Center
	})

	t.Run("URL from self links", func(t *testing.T) {
		t.Parallel()
		prURL := "https://bitbucket.example.com/pr/1"
		pr := toProviderPR(&RestPullRequest{
			State: statePtr(RestPullRequestStateOPEN),
			Links: &RestPullRequestLinks{
				Self: &[]RestHref{{Href: &prURL}},
			},
		})
		require.NotNil(t, pr)
		assert.Equal(t, prURL, pr.URL)
	})

	t.Run("empty self links gives empty URL", func(t *testing.T) {
		t.Parallel()
		pr := toProviderPR(&RestPullRequest{
			State: statePtr(RestPullRequestStateOPEN),
			Links: &RestPullRequestLinks{Self: &[]RestHref{}},
		})
		require.NotNil(t, pr)
		assert.Equal(t, "", pr.URL)
	})

	t.Run("head SHA from FromRef.LatestCommit", func(t *testing.T) {
		t.Parallel()
		pr := toProviderPR(&RestPullRequest{
			State:   statePtr(RestPullRequestStateOPEN),
			FromRef: &RestRef{LatestCommit: strPtr("sha123")},
		})
		require.NotNil(t, pr)
		assert.Equal(t, "sha123", pr.HeadSHA)
	})

	t.Run("createdAt from unix-ms timestamp", func(t *testing.T) {
		t.Parallel()
		ms := int64(1672574400000)
		pr := toProviderPR(&RestPullRequest{
			State:       statePtr(RestPullRequestStateOPEN),
			CreatedDate: &ms,
		})
		require.NotNil(t, pr)
		require.NotNil(t, pr.CreatedAt)
		assert.Equal(t, time.UnixMilli(ms).UTC(), *pr.CreatedAt)
	})

	t.Run("nil timestamps give nil createdAt", func(t *testing.T) {
		t.Parallel()
		pr := toProviderPR(&RestPullRequest{State: statePtr(RestPullRequestStateOPEN)})
		require.NotNil(t, pr)
		assert.Nil(t, pr.CreatedAt)
	})

	t.Run("object field is set", func(t *testing.T) {
		t.Parallel()
		raw := &RestPullRequest{Id: intPtr(99), State: statePtr(RestPullRequestStateOPEN)}
		pr := toProviderPR(raw)
		require.NotNil(t, pr)
		obj, ok := pr.Object.(*RestPullRequest)
		require.True(t, ok)
		assert.Same(t, raw, obj)
	})
}

func TestParseRepoURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		repoURL         string
		wantBaseURL     string
		wantProjectKey  string
		wantRepoSlug    string
		wantErrContains string
	}{
		{
			// NormalizeGit lowercases the entire path, so "PROJ" → "proj"
			name:           "web UI project URL",
			repoURL:        "https://bitbucket.example.com/projects/PROJ/repos/myrepo",
			wantBaseURL:    "https://bitbucket.example.com",
			wantProjectKey: "proj",
			wantRepoSlug:   "myrepo",
		},
		{
			name:           "web UI user URL",
			repoURL:        "https://bitbucket.example.com/users/alice/repos/myrepo",
			wantBaseURL:    "https://bitbucket.example.com",
			wantProjectKey: "~alice",
			wantRepoSlug:   "myrepo",
		},
		{
			name:           "HTTP clone URL with /scm/ prefix",
			repoURL:        "https://bitbucket.example.com/scm/PROJ/myrepo.git",
			wantBaseURL:    "https://bitbucket.example.com",
			wantProjectKey: "proj", // NormalizeGit lowercases
			wantRepoSlug:   "myrepo",
		},
		{
			name:           "SSH clone URL (two path segments)",
			repoURL:        "ssh://git@bitbucket.example.com/PROJ/myrepo.git",
			wantBaseURL:    "https://bitbucket.example.com",
			wantProjectKey: "proj", // NormalizeGit lowercases
			wantRepoSlug:   "myrepo",
		},
		{
			// NormalizeGit lowercases the entire path, so "PROJ" → "proj"
			name:           "host with explicit port is preserved",
			repoURL:        "https://bitbucket.example.com:7990/projects/PROJ/repos/myrepo",
			wantBaseURL:    "https://bitbucket.example.com:7990",
			wantProjectKey: "proj",
			wantRepoSlug:   "myrepo",
		},
		{
			name:            "invalid URL",
			repoURL:         "://invalid",
			wantErrContains: "parse",
		},
		{
			// 3 segments where first is not "scm" → no matching case → error
			name:            "unrecognized path format",
			repoURL:         "https://bitbucket.example.com/foo/bar/baz",
			wantErrContains: "invalid repository path",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			baseURL, projectKey, repoSlug, err := parseRepoURL(tc.repoURL)
			if tc.wantErrContains != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrContains)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantBaseURL, baseURL)
			assert.Equal(t, tc.wantProjectKey, projectKey)
			assert.Equal(t, tc.wantRepoSlug, repoSlug)
		})
	}
}

func TestGetCommitURL(t *testing.T) {
	t.Parallel()

	t.Run("regular project commit URL", func(t *testing.T) {
		t.Parallel()
		p := &provider{
			baseURL:    "https://bitbucket.example.com",
			projectKey: "PROJ",
			repoSlug:   "myrepo",
		}
		u, err := p.GetCommitURL("", "abc123")
		require.NoError(t, err)
		assert.Equal(
			t,
			"https://bitbucket.example.com/projects/PROJ/repos/myrepo/commits/abc123",
			u,
		)
	})

	t.Run("personal repo commit URL", func(t *testing.T) {
		t.Parallel()
		p := &provider{
			baseURL:    "https://bitbucket.example.com",
			projectKey: "~alice",
			repoSlug:   "myrepo",
		}
		u, err := p.GetCommitURL("", "deadbeef")
		require.NoError(t, err)
		assert.Equal(
			t,
			"https://bitbucket.example.com/users/alice/repos/myrepo/commits/deadbeef",
			u,
		)
	})

	t.Run("repo URL argument is ignored", func(t *testing.T) {
		t.Parallel()
		p := &provider{
			baseURL:    "https://bitbucket.example.com",
			projectKey: "PROJ",
			repoSlug:   "myrepo",
		}
		u1, _ := p.GetCommitURL("https://any-url.example.com/foo/bar", "sha")
		u2, _ := p.GetCommitURL("", "sha")
		assert.Equal(t, u1, u2)
	})
}
