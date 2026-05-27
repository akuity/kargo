package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/gitprovider"
)

// mockClient implements ClientWithResponsesInterface for testing.
type mockClient struct {
	getCommitFunc func(
		ctx context.Context,
		workspace, repoSlug, commit string,
		reqEditors ...RequestEditorFn,
	) (*GetRepositoriesWorkspaceRepoSlugCommitCommitResponse, error)

	listPRsFunc func(
		ctx context.Context,
		workspace, repoSlug string,
		params *GetRepositoriesWorkspaceRepoSlugPullrequestsParams,
		reqEditors ...RequestEditorFn,
	) (*GetRepositoriesWorkspaceRepoSlugPullrequestsResponse, error)

	createPRFunc func(
		ctx context.Context,
		workspace, repoSlug string,
		body PostRepositoriesWorkspaceRepoSlugPullrequestsJSONRequestBody,
		reqEditors ...RequestEditorFn,
	) (*PostRepositoriesWorkspaceRepoSlugPullrequestsResponse, error)

	getPRFunc func(
		ctx context.Context,
		workspace, repoSlug string,
		pullRequestId int,
		reqEditors ...RequestEditorFn,
	) (*GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse, error)

	mergePRFunc func(
		ctx context.Context,
		workspace, repoSlug string,
		pullRequestId int,
		params *PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeParams,
		body PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeJSONRequestBody,
		reqEditors ...RequestEditorFn,
	) (*PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeResponse, error)
}

func (m *mockClient) GetRepositoriesWorkspaceRepoSlugCommitCommitWithResponse(
	ctx context.Context,
	workspace, repoSlug, commit string,
	reqEditors ...RequestEditorFn,
) (*GetRepositoriesWorkspaceRepoSlugCommitCommitResponse, error) {
	return m.getCommitFunc(ctx, workspace, repoSlug, commit, reqEditors...)
}

func (m *mockClient) GetRepositoriesWorkspaceRepoSlugPullrequestsWithResponse(
	ctx context.Context,
	workspace, repoSlug string,
	params *GetRepositoriesWorkspaceRepoSlugPullrequestsParams,
	reqEditors ...RequestEditorFn,
) (*GetRepositoriesWorkspaceRepoSlugPullrequestsResponse, error) {
	return m.listPRsFunc(ctx, workspace, repoSlug, params, reqEditors...)
}

func (m *mockClient) PostRepositoriesWorkspaceRepoSlugPullrequestsWithBodyWithResponse(
	_ context.Context,
	_, _ string,
	_ string,
	_ io.Reader,
	_ ...RequestEditorFn,
) (*PostRepositoriesWorkspaceRepoSlugPullrequestsResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockClient) PostRepositoriesWorkspaceRepoSlugPullrequestsWithResponse(
	ctx context.Context,
	workspace, repoSlug string,
	body PostRepositoriesWorkspaceRepoSlugPullrequestsJSONRequestBody,
	reqEditors ...RequestEditorFn,
) (*PostRepositoriesWorkspaceRepoSlugPullrequestsResponse, error) {
	return m.createPRFunc(ctx, workspace, repoSlug, body, reqEditors...)
}

func (m *mockClient) GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdWithResponse(
	ctx context.Context,
	workspace, repoSlug string,
	pullRequestId int,
	reqEditors ...RequestEditorFn,
) (*GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse, error) {
	return m.getPRFunc(ctx, workspace, repoSlug, pullRequestId, reqEditors...)
}

func (m *mockClient) PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeWithBodyWithResponse(
	_ context.Context,
	_, _ string,
	_ int,
	_ *PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeParams,
	_ string,
	_ io.Reader,
	_ ...RequestEditorFn,
) (*PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockClient) PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeWithResponse(
	ctx context.Context,
	workspace, repoSlug string,
	pullRequestId int,
	params *PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeParams,
	body PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeJSONRequestBody,
	reqEditors ...RequestEditorFn,
) (*PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeResponse, error) {
	return m.mergePRFunc(ctx, workspace, repoSlug, pullRequestId, params, body, reqEditors...)
}

// prFromJSON unmarshals a Pullrequest from JSON, for use in test cases.
func prFromJSON(t *testing.T, s string) *Pullrequest {
	t.Helper()
	var pr Pullrequest
	require.NoError(t, json.Unmarshal([]byte(s), &pr))
	return &pr
}

// statesFromEditors applies request editors to a fake request and returns the
// state query params — used to verify which states were passed to the list API.
func statesFromEditors(reqEditors []RequestEditorFn) []string {
	req, _ := http.NewRequest(http.MethodGet, "http://api.bitbucket.org/test", nil)
	for _, ed := range reqEditors {
		_ = ed(context.Background(), req)
	}
	return req.URL.Query()["state"]
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func statePtr(s PullrequestState) *PullrequestState { return &s }

func Test_registration(t *testing.T) {
	t.Run("predicate matches bitbucket.org", func(t *testing.T) {
		assert.True(t, registration.Predicate("https://bitbucket.org/owner/repo"))
	})

	t.Run("predicate does not match self-hosted URLs", func(t *testing.T) {
		assert.False(t, registration.Predicate("https://bitbucket.example.com/projects/PROJ/repos/repo"))
	})

	t.Run("predicate does not match other providers", func(t *testing.T) {
		assert.False(t, registration.Predicate("https://github.com/owner/repo"))
	})

	t.Run("predicate handles invalid URLs", func(t *testing.T) {
		assert.False(t, registration.Predicate("://invalid-url"))
	})

	t.Run("NewProvider factory works", func(t *testing.T) {
		p, err := registration.NewProvider("https://bitbucket.org/owner/repo", nil)
		assert.NoError(t, err)
		assert.NotNil(t, p)
	})
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

	t.Run("error with invalid path", func(t *testing.T) {
		provider, err := NewProvider("https://bitbucket.org/invalid-path", &gitprovider.Options{Token: "token"})
		assert.Error(t, err)
		assert.Nil(t, provider)
	})
}

func TestCreatePullRequest(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		mc := &mockClient{
			createPRFunc: func(
				_ context.Context,
				_, _ string,
				body PostRepositoriesWorkspaceRepoSlugPullrequestsJSONRequestBody,
				_ ...RequestEditorFn,
			) (*PostRepositoriesWorkspaceRepoSlugPullrequestsResponse, error) {
				require.NotNil(t, body.Title)
				assert.Equal(t, "Test PR", *body.Title)
				assert.Equal(t, "PR description", body.AdditionalProperties["description"])
				require.NotNil(t, body.Source)
				require.NotNil(t, body.Source.Branch)
				require.NotNil(t, body.Source.Branch.Name)
				assert.Equal(t, "feature-branch", *body.Source.Branch.Name)
				require.NotNil(t, body.Destination)
				require.NotNil(t, body.Destination.Branch)
				require.NotNil(t, body.Destination.Branch.Name)
				assert.Equal(t, "main", *body.Destination.Branch.Name)
				return &PostRepositoriesWorkspaceRepoSlugPullrequestsResponse{
					JSON201: prFromJSON(t, `{
						"id": 1,
						"state": "OPEN",
						"links": {"html": {"href": "https://bitbucket.org/owner/repo/pull-requests/1"}},
						"source": {
							"branch": {"name": "feature-branch"},
							"commit": {"hash": "abcdef1234567890"}
						},
						"created_on": "2023-01-01T12:00:00Z",
						"type": "pullrequest"
					}`),
				}, nil
			},
		}
		p := &provider{client: mc}
		pr, err := p.CreatePullRequest(t.Context(), &gitprovider.CreatePullRequestOpts{
			Title:       "Test PR",
			Description: "PR description",
			Head:        "feature-branch",
			Base:        "main",
		})
		assert.NoError(t, err)
		require.NotNil(t, pr)
		assert.Equal(t, int64(1), pr.Number)
		assert.Equal(t, "https://bitbucket.org/owner/repo/pull-requests/1", pr.URL)
		assert.Equal(t, "abcdef1234567890", pr.HeadSHA)
		assert.True(t, pr.Open)
		assert.False(t, pr.Merged)
	})

	t.Run("successful creation with nil options", func(t *testing.T) {
		mc := &mockClient{
			createPRFunc: func(
				_ context.Context,
				_, _ string,
				_ PostRepositoriesWorkspaceRepoSlugPullrequestsJSONRequestBody,
				_ ...RequestEditorFn,
			) (*PostRepositoriesWorkspaceRepoSlugPullrequestsResponse, error) {
				return &PostRepositoriesWorkspaceRepoSlugPullrequestsResponse{
					JSON201: &Pullrequest{Id: intPtr(1), State: statePtr(PullrequestStateOPEN)},
				}, nil
			},
		}
		p := &provider{client: mc}
		pr, err := p.CreatePullRequest(t.Context(), nil)
		assert.NoError(t, err)
		require.NotNil(t, pr)
		assert.Equal(t, int64(1), pr.Number)
	})

	t.Run("creation with merge commit resolves full SHA", func(t *testing.T) {
		mc := &mockClient{
			createPRFunc: func(
				_ context.Context,
				_, _ string,
				_ PostRepositoriesWorkspaceRepoSlugPullrequestsJSONRequestBody,
				_ ...RequestEditorFn,
			) (*PostRepositoriesWorkspaceRepoSlugPullrequestsResponse, error) {
				return &PostRepositoriesWorkspaceRepoSlugPullrequestsResponse{
					JSON201: &Pullrequest{
						Id:    intPtr(1),
						State: statePtr(PullrequestStateOPEN),
						MergeCommit: &struct {
							Hash *string `json:"hash,omitempty"`
						}{Hash: strPtr("short123")},
					},
				}, nil
			},
			getCommitFunc: func(
				_ context.Context,
				_, _, commit string,
				_ ...RequestEditorFn,
			) (*GetRepositoriesWorkspaceRepoSlugCommitCommitResponse, error) {
				assert.Equal(t, "short123", commit)
				return &GetRepositoriesWorkspaceRepoSlugCommitCommitResponse{
					JSON200: &Commit{Hash: strPtr("full1234567890abcdef")},
				}, nil
			},
		}
		p := &provider{client: mc}
		pr, err := p.CreatePullRequest(t.Context(), nil)
		assert.NoError(t, err)
		require.NotNil(t, pr)
		assert.Equal(t, "full1234567890abcdef", pr.MergeCommitSHA)
	})

	t.Run("error during creation", func(t *testing.T) {
		mc := &mockClient{
			createPRFunc: func(
				_ context.Context,
				_, _ string,
				_ PostRepositoriesWorkspaceRepoSlugPullrequestsJSONRequestBody,
				_ ...RequestEditorFn,
			) (*PostRepositoriesWorkspaceRepoSlugPullrequestsResponse, error) {
				return nil, errors.New("creation failed")
			},
		}
		p := &provider{client: mc}
		pr, err := p.CreatePullRequest(t.Context(), nil)
		assert.Error(t, err)
		assert.Nil(t, pr)
	})

	t.Run("error getting full commit SHA", func(t *testing.T) {
		mc := &mockClient{
			createPRFunc: func(
				_ context.Context,
				_, _ string,
				_ PostRepositoriesWorkspaceRepoSlugPullrequestsJSONRequestBody,
				_ ...RequestEditorFn,
			) (*PostRepositoriesWorkspaceRepoSlugPullrequestsResponse, error) {
				return &PostRepositoriesWorkspaceRepoSlugPullrequestsResponse{
					JSON201: &Pullrequest{
						Id:    intPtr(1),
						State: statePtr(PullrequestStateOPEN),
						MergeCommit: &struct {
							Hash *string `json:"hash,omitempty"`
						}{Hash: strPtr("short123")},
					},
				}, nil
			},
			getCommitFunc: func(
				_ context.Context,
				_, _, _ string,
				_ ...RequestEditorFn,
			) (*GetRepositoriesWorkspaceRepoSlugCommitCommitResponse, error) {
				return nil, errors.New("commit fetch failed")
			},
		}
		p := &provider{client: mc}
		pr, err := p.CreatePullRequest(t.Context(), nil)
		assert.Error(t, err)
		assert.Nil(t, pr)
	})
}

func TestGetPullRequest(t *testing.T) {
	t.Run("successful retrieval", func(t *testing.T) {
		mc := &mockClient{
			getPRFunc: func(
				_ context.Context,
				_, _ string,
				pullRequestId int,
				_ ...RequestEditorFn,
			) (*GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse, error) {
				assert.Equal(t, 1, pullRequestId)
				return &GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse{
					JSON200: prFromJSON(t, `{
						"id": 1,
						"state": "OPEN",
						"links": {"html": {"href": "https://bitbucket.org/owner/repo/pull-requests/1"}},
						"source": {
							"branch": {"name": "feature-branch"},
							"commit": {"hash": "abcdef1234567890"}
						},
						"created_on": "2023-01-01T12:00:00Z",
						"type": "pullrequest"
					}`),
				}, nil
			},
		}
		p := &provider{client: mc}
		pr, err := p.GetPullRequest(t.Context(), 1)
		assert.NoError(t, err)
		require.NotNil(t, pr)
		assert.Equal(t, int64(1), pr.Number)
		assert.Equal(t, "https://bitbucket.org/owner/repo/pull-requests/1", pr.URL)
		assert.Equal(t, "abcdef1234567890", pr.HeadSHA)
		assert.True(t, pr.Open)
		assert.False(t, pr.Merged)
		assert.NotNil(t, pr.CreatedAt)
	})

	t.Run("retrieval of merged PR resolves full SHA", func(t *testing.T) {
		mc := &mockClient{
			getPRFunc: func(
				_ context.Context,
				_, _ string,
				_ int,
				_ ...RequestEditorFn,
			) (*GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse, error) {
				return &GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse{
					JSON200: &Pullrequest{
						Id:    intPtr(1),
						State: statePtr(PullrequestStateMERGED),
						MergeCommit: &struct {
							Hash *string `json:"hash,omitempty"`
						}{Hash: strPtr("short123")},
					},
				}, nil
			},
			getCommitFunc: func(
				_ context.Context,
				_, _, _ string,
				_ ...RequestEditorFn,
			) (*GetRepositoriesWorkspaceRepoSlugCommitCommitResponse, error) {
				return &GetRepositoriesWorkspaceRepoSlugCommitCommitResponse{
					JSON200: &Commit{Hash: strPtr("full1234567890abcdef")},
				}, nil
			},
		}
		p := &provider{client: mc}
		pr, err := p.GetPullRequest(t.Context(), 1)
		assert.NoError(t, err)
		require.NotNil(t, pr)
		assert.False(t, pr.Open)
		assert.True(t, pr.Merged)
		assert.Equal(t, "full1234567890abcdef", pr.MergeCommitSHA)
	})

	t.Run("error during retrieval", func(t *testing.T) {
		mc := &mockClient{
			getPRFunc: func(
				_ context.Context,
				_, _ string,
				_ int,
				_ ...RequestEditorFn,
			) (*GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse, error) {
				return nil, errors.New("retrieval failed")
			},
		}
		p := &provider{client: mc}
		pr, err := p.GetPullRequest(t.Context(), 1)
		assert.Error(t, err)
		assert.Nil(t, pr)
	})
}

func TestListPullRequests(t *testing.T) {
	t.Run("list open PRs by default", func(t *testing.T) {
		mc := &mockClient{
			listPRsFunc: func(
				_ context.Context,
				_, _ string,
				_ *GetRepositoriesWorkspaceRepoSlugPullrequestsParams,
				reqEditors ...RequestEditorFn,
			) (*GetRepositoriesWorkspaceRepoSlugPullrequestsResponse, error) {
				assert.Equal(t, []string{"OPEN"}, statesFromEditors(reqEditors))
				values := []Pullrequest{
					{Id: intPtr(1), State: statePtr(PullrequestStateOPEN)},
					{Id: intPtr(2), State: statePtr(PullrequestStateOPEN)},
				}
				return &GetRepositoriesWorkspaceRepoSlugPullrequestsResponse{
					JSON200: &PaginatedPullrequests{Values: &values},
				}, nil
			},
		}
		p := &provider{client: mc}
		prs, err := p.ListPullRequests(t.Context(), nil)
		assert.NoError(t, err)
		assert.Len(t, prs, 2)
	})

	t.Run("list all PRs", func(t *testing.T) {
		mc := &mockClient{
			listPRsFunc: func(
				_ context.Context,
				_, _ string,
				_ *GetRepositoriesWorkspaceRepoSlugPullrequestsParams,
				reqEditors ...RequestEditorFn,
			) (*GetRepositoriesWorkspaceRepoSlugPullrequestsResponse, error) {
				states := statesFromEditors(reqEditors)
				assert.Contains(t, states, "OPEN")
				assert.Contains(t, states, "MERGED")
				assert.Contains(t, states, "DECLINED")
				assert.Contains(t, states, "SUPERSEDED")
				values := []Pullrequest{
					{Id: intPtr(1), State: statePtr(PullrequestStateOPEN)},
					{Id: intPtr(2), State: statePtr(PullrequestStateMERGED)},
					{Id: intPtr(3), State: statePtr(PullrequestStateDECLINED)},
					{Id: intPtr(4), State: statePtr(PullrequestStateSUPERSEDED)},
				}
				return &GetRepositoriesWorkspaceRepoSlugPullrequestsResponse{
					JSON200: &PaginatedPullrequests{Values: &values},
				}, nil
			},
		}
		p := &provider{client: mc}
		prs, err := p.ListPullRequests(t.Context(), &gitprovider.ListPullRequestOptions{
			State: gitprovider.PullRequestStateAny,
		})
		assert.NoError(t, err)
		assert.Len(t, prs, 4)
	})

	t.Run("list closed PRs", func(t *testing.T) {
		mc := &mockClient{
			listPRsFunc: func(
				_ context.Context,
				_, _ string,
				_ *GetRepositoriesWorkspaceRepoSlugPullrequestsParams,
				reqEditors ...RequestEditorFn,
			) (*GetRepositoriesWorkspaceRepoSlugPullrequestsResponse, error) {
				states := statesFromEditors(reqEditors)
				assert.Contains(t, states, "MERGED")
				assert.Contains(t, states, "DECLINED")
				assert.Contains(t, states, "SUPERSEDED")
				assert.NotContains(t, states, "OPEN")
				values := []Pullrequest{
					{Id: intPtr(2), State: statePtr(PullrequestStateMERGED)},
					{Id: intPtr(3), State: statePtr(PullrequestStateDECLINED)},
					{Id: intPtr(4), State: statePtr(PullrequestStateSUPERSEDED)},
				}
				return &GetRepositoriesWorkspaceRepoSlugPullrequestsResponse{
					JSON200: &PaginatedPullrequests{Values: &values},
				}, nil
			},
		}
		p := &provider{client: mc}
		prs, err := p.ListPullRequests(t.Context(), &gitprovider.ListPullRequestOptions{
			State: gitprovider.PullRequestStateClosed,
		})
		assert.NoError(t, err)
		assert.Len(t, prs, 3)
	})

	t.Run("filter by head branch", func(t *testing.T) {
		mc := &mockClient{
			listPRsFunc: func(
				_ context.Context,
				_, _ string,
				_ *GetRepositoriesWorkspaceRepoSlugPullrequestsParams,
				_ ...RequestEditorFn,
			) (*GetRepositoriesWorkspaceRepoSlugPullrequestsResponse, error) {
				values := []Pullrequest{
					{
						Id:    intPtr(1),
						State: statePtr(PullrequestStateOPEN),
						Source: &PullrequestEndpoint{Branch: &struct {
							DefaultMergeStrategy *string                                     `json:"default_merge_strategy,omitempty"`
							MergeStrategies      *[]PullrequestEndpointBranchMergeStrategies `json:"merge_strategies,omitempty"`
							Name                 *string                                     `json:"name,omitempty"`
						}{Name: strPtr("feature-1")}},
					},
					{
						Id:    intPtr(2),
						State: statePtr(PullrequestStateOPEN),
						Source: &PullrequestEndpoint{Branch: &struct {
							DefaultMergeStrategy *string                                     `json:"default_merge_strategy,omitempty"`
							MergeStrategies      *[]PullrequestEndpointBranchMergeStrategies `json:"merge_strategies,omitempty"`
							Name                 *string                                     `json:"name,omitempty"`
						}{Name: strPtr("feature-2")}},
					},
				}
				return &GetRepositoriesWorkspaceRepoSlugPullrequestsResponse{
					JSON200: &PaginatedPullrequests{Values: &values},
				}, nil
			},
		}
		p := &provider{client: mc}
		prs, err := p.ListPullRequests(t.Context(), &gitprovider.ListPullRequestOptions{
			HeadBranch: "feature-1",
		})
		assert.NoError(t, err)
		assert.Len(t, prs, 1)
		assert.Equal(t, int64(1), prs[0].Number)
	})

	t.Run("filter by base branch", func(t *testing.T) {
		mc := &mockClient{
			listPRsFunc: func(
				_ context.Context,
				_, _ string,
				_ *GetRepositoriesWorkspaceRepoSlugPullrequestsParams,
				_ ...RequestEditorFn,
			) (*GetRepositoriesWorkspaceRepoSlugPullrequestsResponse, error) {
				values := []Pullrequest{
					{
						Id:    intPtr(1),
						State: statePtr(PullrequestStateOPEN),
						Destination: &PullrequestEndpoint{Branch: &struct {
							DefaultMergeStrategy *string                                     `json:"default_merge_strategy,omitempty"`
							MergeStrategies      *[]PullrequestEndpointBranchMergeStrategies `json:"merge_strategies,omitempty"`
							Name                 *string                                     `json:"name,omitempty"`
						}{Name: strPtr("main")}},
					},
					{
						Id:    intPtr(2),
						State: statePtr(PullrequestStateOPEN),
						Destination: &PullrequestEndpoint{Branch: &struct {
							DefaultMergeStrategy *string                                     `json:"default_merge_strategy,omitempty"`
							MergeStrategies      *[]PullrequestEndpointBranchMergeStrategies `json:"merge_strategies,omitempty"`
							Name                 *string                                     `json:"name,omitempty"`
						}{Name: strPtr("dev")}},
					},
				}
				return &GetRepositoriesWorkspaceRepoSlugPullrequestsResponse{
					JSON200: &PaginatedPullrequests{Values: &values},
				}, nil
			},
		}
		p := &provider{client: mc}
		prs, err := p.ListPullRequests(t.Context(), &gitprovider.ListPullRequestOptions{
			BaseBranch: "dev",
		})
		assert.NoError(t, err)
		assert.Len(t, prs, 1)
		assert.Equal(t, int64(2), prs[0].Number)
	})

	t.Run("filter by head commit", func(t *testing.T) {
		mc := &mockClient{
			listPRsFunc: func(
				_ context.Context,
				_, _ string,
				_ *GetRepositoriesWorkspaceRepoSlugPullrequestsParams,
				_ ...RequestEditorFn,
			) (*GetRepositoriesWorkspaceRepoSlugPullrequestsResponse, error) {
				values := []Pullrequest{
					{
						Id:    intPtr(1),
						State: statePtr(PullrequestStateOPEN),
						Source: &PullrequestEndpoint{
							Commit: &struct {
								Hash *string `json:"hash,omitempty"`
							}{Hash: strPtr("specific-hash")},
						},
					},
					{
						Id:    intPtr(2),
						State: statePtr(PullrequestStateOPEN),
						Source: &PullrequestEndpoint{
							Commit: &struct {
								Hash *string `json:"hash,omitempty"`
							}{Hash: strPtr("other-hash")},
						},
					},
				}
				return &GetRepositoriesWorkspaceRepoSlugPullrequestsResponse{
					JSON200: &PaginatedPullrequests{Values: &values},
				}, nil
			},
		}
		p := &provider{client: mc}
		prs, err := p.ListPullRequests(t.Context(), &gitprovider.ListPullRequestOptions{
			HeadCommit: "specific-hash",
		})
		assert.NoError(t, err)
		assert.Len(t, prs, 1)
		assert.Equal(t, int64(1), prs[0].Number)
	})

	t.Run("PR with merge commit resolves full SHA", func(t *testing.T) {
		mc := &mockClient{
			listPRsFunc: func(
				_ context.Context,
				_, _ string,
				_ *GetRepositoriesWorkspaceRepoSlugPullrequestsParams,
				_ ...RequestEditorFn,
			) (*GetRepositoriesWorkspaceRepoSlugPullrequestsResponse, error) {
				values := []Pullrequest{{
					Id:    intPtr(1),
					State: statePtr(PullrequestStateMERGED),
					MergeCommit: &struct {
						Hash *string `json:"hash,omitempty"`
					}{Hash: strPtr("short123")},
				}}
				return &GetRepositoriesWorkspaceRepoSlugPullrequestsResponse{
					JSON200: &PaginatedPullrequests{Values: &values},
				}, nil
			},
			getCommitFunc: func(
				_ context.Context,
				_, _, _ string,
				_ ...RequestEditorFn,
			) (*GetRepositoriesWorkspaceRepoSlugCommitCommitResponse, error) {
				return &GetRepositoriesWorkspaceRepoSlugCommitCommitResponse{
					JSON200: &Commit{Hash: strPtr("full1234567890abcdef")},
				}, nil
			},
		}
		p := &provider{client: mc}
		prs, err := p.ListPullRequests(t.Context(), nil)
		assert.NoError(t, err)
		assert.Len(t, prs, 1)
		assert.Equal(t, "full1234567890abcdef", prs[0].MergeCommitSHA)
	})

	t.Run("error during list", func(t *testing.T) {
		mc := &mockClient{
			listPRsFunc: func(
				_ context.Context,
				_, _ string,
				_ *GetRepositoriesWorkspaceRepoSlugPullrequestsParams,
				_ ...RequestEditorFn,
			) (*GetRepositoriesWorkspaceRepoSlugPullrequestsResponse, error) {
				return nil, errors.New("list failed")
			},
		}
		p := &provider{client: mc}
		prs, err := p.ListPullRequests(t.Context(), nil)
		assert.Error(t, err)
		assert.Nil(t, prs)
	})

	t.Run("invalid state", func(t *testing.T) {
		p := &provider{client: &mockClient{}}
		prs, err := p.ListPullRequests(t.Context(), &gitprovider.ListPullRequestOptions{
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
		mockClient     *mockClient
		expectedMerged bool
		expectError    bool
		errorContains  string
	}{
		{
			name:     "error getting PR",
			prNumber: 999,
			mockClient: &mockClient{
				getPRFunc: func(
					_ context.Context,
					_, _ string,
					_ int,
					_ ...RequestEditorFn,
				) (*GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse, error) {
					return nil, errors.New("get PR failed")
				},
			},
			expectError:   true,
			errorContains: "error getting pull request",
		},
		{
			name:     "PR already merged",
			prNumber: 123,
			mockClient: &mockClient{
				getPRFunc: func(
					_ context.Context,
					_, _ string,
					_ int,
					_ ...RequestEditorFn,
				) (*GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse, error) {
					return &GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse{
						JSON200: &Pullrequest{
							Id:    intPtr(123),
							State: statePtr(PullrequestStateMERGED),
							MergeCommit: &struct {
								Hash *string `json:"hash,omitempty"`
							}{Hash: strPtr("merge_sha")},
						},
					}, nil
				},
			},
			expectedMerged: true,
		},
		{
			name:     "PR declined",
			prNumber: 456,
			mockClient: &mockClient{
				getPRFunc: func(
					_ context.Context,
					_, _ string,
					_ int,
					_ ...RequestEditorFn,
				) (*GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse, error) {
					return &GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse{
						JSON200: &Pullrequest{Id: intPtr(456), State: statePtr(PullrequestStateDECLINED)},
					}, nil
				},
			},
			expectError:   true,
			errorContains: "closed but not merged",
		},
		{
			name:     "PR is draft",
			prNumber: 333,
			mockClient: &mockClient{
				getPRFunc: func(
					_ context.Context,
					_, _ string,
					_ int,
					_ ...RequestEditorFn,
				) (*GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse, error) {
					isDraft := true
					return &GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse{
						JSON200: &Pullrequest{
							Id:    intPtr(333),
							State: statePtr(PullrequestStateOPEN),
							Draft: &isDraft,
						},
					}, nil
				},
			},
		},
		{
			name:      "unsupported merge strategy",
			prNumber:  100,
			mergeOpts: &gitprovider.MergePullRequestOpts{MergeMethod: "rebase"},
			mockClient: &mockClient{
				getPRFunc: func(
					_ context.Context,
					_, _ string,
					_ int,
					_ ...RequestEditorFn,
				) (*GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse, error) {
					return &GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse{
						JSON200: &Pullrequest{Id: intPtr(100), State: statePtr(PullrequestStateOPEN)},
					}, nil
				},
			},
			expectError:   true,
			errorContains: "unsupported merge strategy",
		},
		{
			name:     "merge operation fails",
			prNumber: 888,
			mockClient: &mockClient{
				getPRFunc: func(
					_ context.Context,
					_, _ string,
					_ int,
					_ ...RequestEditorFn,
				) (*GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse, error) {
					return &GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse{
						JSON200: &Pullrequest{Id: intPtr(888), State: statePtr(PullrequestStateOPEN)},
					}, nil
				},
				mergePRFunc: func(
					_ context.Context,
					_, _ string,
					_ int,
					_ *PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeParams,
					_ PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeJSONRequestBody,
					_ ...RequestEditorFn,
				) (*PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeResponse, error) {
					return nil, errors.New("merge failed")
				},
			},
			expectError:   true,
			errorContains: "error merging pull request",
		},
		{
			name:      "successful merge with strategy",
			prNumber:  200,
			mergeOpts: &gitprovider.MergePullRequestOpts{MergeMethod: "squash"},
			mockClient: &mockClient{
				getPRFunc: func(
					_ context.Context,
					_, _ string,
					_ int,
					_ ...RequestEditorFn,
				) (*GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse, error) {
					return &GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse{
						JSON200: &Pullrequest{Id: intPtr(200), State: statePtr(PullrequestStateOPEN)},
					}, nil
				},
				mergePRFunc: func(
					_ context.Context,
					_, _ string,
					pullRequestId int,
					_ *PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeParams,
					body PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeJSONRequestBody,
					_ ...RequestEditorFn,
				) (*PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeResponse, error) {
					assert.Equal(t, 200, pullRequestId)
					require.NotNil(t, body.MergeStrategy)
					assert.Equal(t, Squash, *body.MergeStrategy)
					return &PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeResponse{
						JSON200: &Pullrequest{Id: intPtr(200), State: statePtr(PullrequestStateMERGED)},
					}, nil
				},
			},
			expectedMerged: true,
		},
		{
			name:     "successful merge",
			prNumber: 1234,
			mockClient: &mockClient{
				getPRFunc: func(
					_ context.Context,
					_, _ string,
					_ int,
					_ ...RequestEditorFn,
				) (*GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse, error) {
					return &GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdResponse{
						JSON200: &Pullrequest{Id: intPtr(1234), State: statePtr(PullrequestStateOPEN)},
					}, nil
				},
				mergePRFunc: func(
					_ context.Context,
					_, _ string,
					_ int,
					_ *PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeParams,
					_ PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeJSONRequestBody,
					_ ...RequestEditorFn,
				) (*PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeResponse, error) {
					return &PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeResponse{
						JSON200: &Pullrequest{
							Id:    intPtr(1234),
							State: statePtr(PullrequestStateMERGED),
							MergeCommit: &struct {
								Hash *string `json:"hash,omitempty"`
							}{Hash: strPtr("merge_sha")},
						},
					}, nil
				},
			},
			expectedMerged: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := &provider{client: tc.mockClient}
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

func TestResolveFullMergeCommitSHA(t *testing.T) {
	t.Run("no-op when merge commit is nil", func(t *testing.T) {
		p := &provider{client: &mockClient{}}
		pr := &Pullrequest{}
		err := p.resolveFullMergeCommitSHA(t.Context(), pr)
		assert.NoError(t, err)
		assert.Nil(t, pr.MergeCommit)
	})

	t.Run("no-op when merge commit hash is empty", func(t *testing.T) {
		p := &provider{client: &mockClient{}}
		pr := &Pullrequest{
			MergeCommit: &struct {
				Hash *string `json:"hash,omitempty"`
			}{Hash: strPtr("")},
		}
		err := p.resolveFullMergeCommitSHA(t.Context(), pr)
		assert.NoError(t, err)
	})

	t.Run("resolves short SHA to full SHA", func(t *testing.T) {
		mc := &mockClient{
			getCommitFunc: func(
				_ context.Context,
				_, _, commit string,
				_ ...RequestEditorFn,
			) (*GetRepositoriesWorkspaceRepoSlugCommitCommitResponse, error) {
				assert.Equal(t, "short123", commit)
				return &GetRepositoriesWorkspaceRepoSlugCommitCommitResponse{
					JSON200: &Commit{Hash: strPtr("full1234567890abcdef")},
				}, nil
			},
		}
		p := &provider{client: mc}
		pr := &Pullrequest{
			MergeCommit: &struct {
				Hash *string `json:"hash,omitempty"`
			}{Hash: strPtr("short123")},
		}
		err := p.resolveFullMergeCommitSHA(t.Context(), pr)
		assert.NoError(t, err)
		require.NotNil(t, pr.MergeCommit.Hash)
		assert.Equal(t, "full1234567890abcdef", *pr.MergeCommit.Hash)
	})

	t.Run("error from getCommit is propagated", func(t *testing.T) {
		mc := &mockClient{
			getCommitFunc: func(
				_ context.Context,
				_, _, _ string,
				_ ...RequestEditorFn,
			) (*GetRepositoriesWorkspaceRepoSlugCommitCommitResponse, error) {
				return nil, errors.New("retrieval failed")
			},
		}
		p := &provider{client: mc}
		pr := &Pullrequest{
			MergeCommit: &struct {
				Hash *string `json:"hash,omitempty"`
			}{Hash: strPtr("short123")},
		}
		err := p.resolveFullMergeCommitSHA(t.Context(), pr)
		assert.Error(t, err)
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
		},
		{
			name:      "valid URL with trailing slash",
			url:       "https://bitbucket.org/owner/repo/",
			wantHost:  "bitbucket.org",
			wantOwner: "owner",
			wantSlug:  "repo",
		},
		{
			name:      "valid URL with .git suffix",
			url:       "https://bitbucket.org/owner/repo.git",
			wantHost:  "bitbucket.org",
			wantOwner: "owner",
			wantSlug:  "repo",
		},
		{
			name:      "valid SSH URL",
			url:       "git@bitbucket.org:owner/repo.git",
			wantHost:  "bitbucket.org",
			wantOwner: "owner",
			wantSlug:  "repo",
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

func Test_toProviderPR(t *testing.T) {
	t.Run("valid conversion: open PR", func(t *testing.T) {
		bbPR := prFromJSON(t, `{
			"id": 1,
			"state": "OPEN",
			"links": {"html": {"href": "https://bitbucket.org/owner/repo/pull-requests/1"}},
			"source": {"commit": {"hash": "abcdef1234567890"}},
			"created_on": "2023-01-01T12:00:00Z",
			"type": "pullrequest"
		}`)
		pr := toProviderPR(bbPR)
		require.NotNil(t, pr)
		assert.Equal(t, int64(1), pr.Number)
		assert.Equal(t, "https://bitbucket.org/owner/repo/pull-requests/1", pr.URL)
		assert.Equal(t, "abcdef1234567890", pr.HeadSHA)
		assert.True(t, pr.Open)
		assert.False(t, pr.Merged)
		assert.NotNil(t, pr.CreatedAt)
		assert.Equal(t, bbPR, pr.Object)
	})

	t.Run("valid conversion: merged PR", func(t *testing.T) {
		bbPR := &Pullrequest{
			Id:    intPtr(1),
			State: statePtr(PullrequestStateMERGED),
			MergeCommit: &struct {
				Hash *string `json:"hash,omitempty"`
			}{Hash: strPtr("merged1234567890")},
		}
		pr := toProviderPR(bbPR)
		require.NotNil(t, pr)
		assert.False(t, pr.Open)
		assert.True(t, pr.Merged)
		assert.Equal(t, "merged1234567890", pr.MergeCommitSHA)
	})

	t.Run("nil input", func(t *testing.T) {
		assert.Nil(t, toProviderPR(nil))
	})

	t.Run("no created_on", func(t *testing.T) {
		pr := toProviderPR(&Pullrequest{Id: intPtr(1)})
		require.NotNil(t, pr)
		assert.Nil(t, pr.CreatedAt)
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

	p := &provider{}
	for _, tc := range testCases {
		t.Run(tc.repoURL, func(t *testing.T) {
			commitURL, err := p.GetCommitURL(tc.repoURL, tc.sha)
			require.NoError(t, err)
			require.Equal(t, tc.expectedCommitURL, commitURL)
		})
	}
}
