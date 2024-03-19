package gitlab

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xanzy/go-gitlab"

	"github.com/akuity/kargo/internal/gitprovider"
)

type MockGitLabClient struct {
	mr         *gitlab.MergeRequest
	createOpts *gitlab.CreateMergeRequestOptions
	listOpts   *gitlab.ListProjectMergeRequestsOptions
	pid        interface{}
}

func (m *MockGitLabClient) CreateMergeRequest(pid interface{}, opt *gitlab.CreateMergeRequestOptions, options ...gitlab.RequestOptionFunc) (*gitlab.MergeRequest, *gitlab.Response, error) {
	m.pid = pid
	m.createOpts = opt
	return m.mr, nil, nil
}

func (m *MockGitLabClient) ListProjectMergeRequests(pid interface{}, opt *gitlab.ListProjectMergeRequestsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.MergeRequest, *gitlab.Response, error) {
	m.pid = pid
	m.listOpts = opt
	return []*gitlab.MergeRequest{m.mr}, nil, nil
}

func (m *MockGitLabClient) GetMergeRequest(pid interface{}, mergeRequest int, opt *gitlab.GetMergeRequestsOptions, options ...gitlab.RequestOptionFunc) (*gitlab.MergeRequest, *gitlab.Response, error) {
	m.pid = pid
	return m.mr, nil, nil
}

func TestCreatePullRequest(t *testing.T) {
	mockClient := &MockGitLabClient{
		mr: &gitlab.MergeRequest{
			IID:            1,
			MergeCommitSHA: "sha",
			State:          "merged",
			WebURL:         "url",
		},
	}
	g := GitLabProvider{client: &GitLabClient{MergeRequests: mockClient}}

	opts := gitprovider.CreatePullRequestOpts{
		Head:        "",
		Base:        "",
		Title:       "title",
		Description: "desc",
	}
	pr, err := g.CreatePullRequest(context.Background(), "https://gitlab.com/group/project.git", opts)

	require.NoError(t, err)
	require.Equal(t, "group/project", mockClient.pid)
	require.Equal(t, opts.Head, *mockClient.createOpts.SourceBranch)
	require.Equal(t, opts.Base, *mockClient.createOpts.TargetBranch)
	require.Equal(t, opts.Title, *mockClient.createOpts.Title)
	require.Equal(t, opts.Description, *mockClient.createOpts.Description)

	require.Equal(t, int64(mockClient.mr.IID), pr.Number)
	require.Equal(t, mockClient.mr.MergeCommitSHA, pr.MergeCommitSHA)
	require.Equal(t, mockClient.mr.WebURL, pr.URL)
	require.Equal(t, gitprovider.PullRequestStateClosed, pr.State)
}

func TestGetPullRequest(t *testing.T) {
	mockClient := &MockGitLabClient{
		mr: &gitlab.MergeRequest{
			IID:            1,
			MergeCommitSHA: "sha",
			State:          "merged",
			WebURL:         "url",
		},
	}
	g := GitLabProvider{client: &GitLabClient{MergeRequests: mockClient}}

	pr, err := g.GetPullRequest(context.Background(), "https://gitlab.com/group/project.git", 1)

	require.NoError(t, err)
	require.Equal(t, "group/project", mockClient.pid)
	require.Equal(t, int64(mockClient.mr.IID), pr.Number)
	require.Equal(t, mockClient.mr.MergeCommitSHA, pr.MergeCommitSHA)
	require.Equal(t, mockClient.mr.WebURL, pr.URL)
	require.Equal(t, gitprovider.PullRequestStateClosed, pr.State)
}

func TestListPullRequests(t *testing.T) {
	mockClient := &MockGitLabClient{
		mr: &gitlab.MergeRequest{
			IID:            1,
			MergeCommitSHA: "sha",
			State:          "merged",
			WebURL:         "url",
		},
	}
	g := GitLabProvider{client: &GitLabClient{MergeRequests: mockClient}}

	opts := gitprovider.ListPullRequestOpts{
		Head: "head",
		Base: "base",
	}
	prs, err := g.ListPullRequests(context.Background(), "https://gitlab.com/group/project.git", opts)

	require.NoError(t, err)
	require.Equal(t, "group/project", mockClient.pid)
	require.Equal(t, opts.Head, *mockClient.listOpts.SourceBranch)
	require.Equal(t, opts.Base, *mockClient.listOpts.TargetBranch)

	require.Equal(t, int64(mockClient.mr.IID), prs[0].Number)
	require.Equal(t, mockClient.mr.MergeCommitSHA, prs[0].MergeCommitSHA)
	require.Equal(t, mockClient.mr.WebURL, prs[0].URL)
	require.Equal(t, gitprovider.PullRequestStateClosed, prs[0].State)
}

func TestIsPullRequestMerged(t *testing.T) {
	require.True(t, isPullRequestMerged("merged"))
	require.False(t, isPullRequestMerged("closed"))
	require.False(t, isPullRequestMerged("locked"))
	require.False(t, isPullRequestMerged("opened"))
}

func isPullRequestMerged(state string) bool {
	mockClient := &MockGitLabClient{
		mr: &gitlab.MergeRequest{
			IID:            1,
			MergeCommitSHA: "sha",
			State:          state,
			WebURL:         "url",
		},
	}
	g := GitLabProvider{client: &GitLabClient{MergeRequests: mockClient}}
	res, _ := g.IsPullRequestMerged(context.Background(), "https://gitlab.com/group/project.git", 1)
	return res
}
