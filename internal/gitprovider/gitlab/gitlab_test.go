package gitlab

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xanzy/go-gitlab"

	"github.com/akuity/kargo/internal/gitprovider"
)

const testProjectName = "group/project"

type mockGitLabClient struct {
	mr         *gitlab.MergeRequest
	createOpts *gitlab.CreateMergeRequestOptions
	listOpts   *gitlab.ListProjectMergeRequestsOptions
	pid        any
}

func (m *mockGitLabClient) CreateMergeRequest(
	pid any,
	opt *gitlab.CreateMergeRequestOptions,
	_ ...gitlab.RequestOptionFunc,
) (*gitlab.MergeRequest, *gitlab.Response, error) {
	m.pid = pid
	m.createOpts = opt
	return m.mr, nil, nil
}

func (m *mockGitLabClient) ListProjectMergeRequests(
	pid any,
	opt *gitlab.ListProjectMergeRequestsOptions,
	_ ...gitlab.RequestOptionFunc,
) ([]*gitlab.MergeRequest, *gitlab.Response, error) {
	m.pid = pid
	m.listOpts = opt
	return []*gitlab.MergeRequest{m.mr}, nil, nil
}

func (m *mockGitLabClient) GetMergeRequest(
	pid any,
	_ int,
	_ *gitlab.GetMergeRequestsOptions,
	_ ...gitlab.RequestOptionFunc,
) (*gitlab.MergeRequest, *gitlab.Response, error) {
	m.pid = pid
	return m.mr, nil, nil
}

func TestCreatePullRequest(t *testing.T) {
	mockClient := &mockGitLabClient{
		mr: &gitlab.MergeRequest{
			IID:            1,
			MergeCommitSHA: "sha",
			State:          "merged",
			WebURL:         "url",
		},
	}
	g := provider{
		projectName: testProjectName,
		client:      mockClient,
	}

	opts := gitprovider.CreatePullRequestOpts{
		Head:        "",
		Base:        "",
		Title:       "title",
		Description: "desc",
	}
	pr, err := g.CreatePullRequest(context.Background(), &opts)

	require.NoError(t, err)
	require.Equal(t, testProjectName, mockClient.pid)
	require.Equal(t, opts.Head, *mockClient.createOpts.SourceBranch)
	require.Equal(t, opts.Base, *mockClient.createOpts.TargetBranch)
	require.Equal(t, opts.Title, *mockClient.createOpts.Title)
	require.Equal(t, opts.Description, *mockClient.createOpts.Description)

	require.Equal(t, int64(mockClient.mr.IID), pr.Number)
	require.Equal(t, mockClient.mr.MergeCommitSHA, pr.MergeCommitSHA)
	require.Equal(t, mockClient.mr.WebURL, pr.URL)
	require.False(t, pr.Open)
}

func TestGetPullRequest(t *testing.T) {
	mockClient := &mockGitLabClient{
		mr: &gitlab.MergeRequest{
			IID:            1,
			MergeCommitSHA: "sha",
			State:          "merged",
			WebURL:         "url",
		},
	}
	g := provider{
		projectName: testProjectName,
		client:      mockClient,
	}

	pr, err := g.GetPullRequest(context.Background(), 1)

	require.NoError(t, err)
	require.Equal(t, testProjectName, mockClient.pid)
	require.Equal(t, int64(mockClient.mr.IID), pr.Number)
	require.Equal(t, mockClient.mr.MergeCommitSHA, pr.MergeCommitSHA)
	require.Equal(t, mockClient.mr.WebURL, pr.URL)
	require.False(t, pr.Open)
}

func TestListPullRequests(t *testing.T) {
	mockClient := &mockGitLabClient{
		mr: &gitlab.MergeRequest{
			IID:            1,
			MergeCommitSHA: "sha",
			State:          "merged",
			WebURL:         "url",
		},
	}
	g := provider{
		projectName: testProjectName,
		client:      mockClient,
	}

	opts := gitprovider.ListPullRequestOptions{
		State:      gitprovider.PullRequestStateAny,
		HeadBranch: "head",
		BaseBranch: "base",
	}
	prs, err := g.ListPullRequests(context.Background(), &opts)
	require.NoError(t, err)

	require.Equal(t, testProjectName, mockClient.pid)
	require.Equal(t, opts.HeadBranch, *mockClient.listOpts.SourceBranch)
	require.Equal(t, opts.BaseBranch, *mockClient.listOpts.TargetBranch)

	require.Equal(t, int64(mockClient.mr.IID), prs[0].Number)
	require.Equal(t, mockClient.mr.MergeCommitSHA, prs[0].MergeCommitSHA)
	require.Equal(t, mockClient.mr.WebURL, prs[0].URL)
	require.False(t, prs[0].Open)
}

func TestParseGitLabURL(t *testing.T) {
	const expectedProjectName = "akuity/kargo"
	testCases := []struct {
		url          string
		expectedHost string
	}{
		{
			url:          "https://gitlab.com/akuity/kargo",
			expectedHost: "gitlab.com",
		},
		{
			url:          "https://gitlab.com/akuity/kargo.git",
			expectedHost: "gitlab.com",
		},
		{
			// This isn't a real URL. It's just to validate that the function can
			// handle GitHub Enterprise URLs.
			url:          "https://gitlab.akuity.io/akuity/kargo.git",
			expectedHost: "gitlab.akuity.io",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.url, func(t *testing.T) {
			host, projectName, err := parseRepoURL(testCase.url)
			require.NoError(t, err)
			require.Equal(t, testCase.expectedHost, host)
			require.Equal(t, expectedProjectName, projectName)
		})
	}
}
