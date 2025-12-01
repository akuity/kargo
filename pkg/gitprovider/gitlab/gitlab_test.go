package gitlab

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/akuity/kargo/pkg/gitprovider"
)

const testProjectName = "group/project"

type mockGitLabClient struct {
	mr         *gitlab.MergeRequest
	createOpts *gitlab.CreateMergeRequestOptions
	listOpts   *gitlab.ListProjectMergeRequestsOptions
	pid        any
	getMRFunc  func(
		pid any, mergeRequest int64, opt *gitlab.GetMergeRequestsOptions,
		options ...gitlab.RequestOptionFunc,
	) (*gitlab.MergeRequest, *gitlab.Response, error)
	acceptMRFunc func(pid any, mergeRequest int64, opt *gitlab.AcceptMergeRequestOptions,
		options ...gitlab.RequestOptionFunc,
	) (*gitlab.MergeRequest, *gitlab.Response, error)
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
) ([]*gitlab.BasicMergeRequest, *gitlab.Response, error) {
	m.pid = pid
	m.listOpts = opt
	return []*gitlab.BasicMergeRequest{&m.mr.BasicMergeRequest}, nil, nil
}

func (m *mockGitLabClient) GetMergeRequest(
	pid any,
	mergeRequest int64,
	opt *gitlab.GetMergeRequestsOptions,
	options ...gitlab.RequestOptionFunc,
) (*gitlab.MergeRequest, *gitlab.Response, error) {
	m.pid = pid
	if m.getMRFunc != nil {
		return m.getMRFunc(pid, mergeRequest, opt, options...)
	}
	return m.mr, nil, nil
}

func (m *mockGitLabClient) AcceptMergeRequest(
	pid any,
	mergeRequest int64,
	opt *gitlab.AcceptMergeRequestOptions,
	options ...gitlab.RequestOptionFunc,
) (*gitlab.MergeRequest, *gitlab.Response, error) {
	m.pid = pid
	if m.acceptMRFunc != nil {
		return m.acceptMRFunc(pid, mergeRequest, opt, options...)
	}
	return m.mr, nil, nil
}

func TestCreatePullRequest(t *testing.T) {
	mockClient := &mockGitLabClient{
		mr: &gitlab.MergeRequest{
			BasicMergeRequest: gitlab.BasicMergeRequest{
				IID:            1,
				MergeCommitSHA: "sha",
				State:          "merged",
				WebURL:         "url",
			},
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

	require.Equal(t, mockClient.mr.IID, pr.Number)
	require.Equal(t, mockClient.mr.MergeCommitSHA, pr.MergeCommitSHA)
	require.Equal(t, mockClient.mr.WebURL, pr.URL)
	require.False(t, pr.Open)
}

func TestGetPullRequest(t *testing.T) {
	mockClient := &mockGitLabClient{
		mr: &gitlab.MergeRequest{
			BasicMergeRequest: gitlab.BasicMergeRequest{
				IID:            1,
				MergeCommitSHA: "sha",
				State:          "merged",
				WebURL:         "url",
			},
		},
	}
	g := provider{
		projectName: testProjectName,
		client:      mockClient,
	}

	pr, err := g.GetPullRequest(context.Background(), 1)

	require.NoError(t, err)
	require.Equal(t, testProjectName, mockClient.pid)
	require.Equal(t, mockClient.mr.IID, pr.Number)
	require.Equal(t, mockClient.mr.MergeCommitSHA, pr.MergeCommitSHA)
	require.Equal(t, mockClient.mr.WebURL, pr.URL)
	require.False(t, pr.Open)
}

func TestListPullRequests(t *testing.T) {
	mockClient := &mockGitLabClient{
		mr: &gitlab.MergeRequest{
			BasicMergeRequest: gitlab.BasicMergeRequest{
				IID:            1,
				MergeCommitSHA: "sha",
				State:          "merged",
				WebURL:         "url",
			},
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

	require.Equal(t, mockClient.mr.IID, prs[0].Number)
	require.Equal(t, mockClient.mr.MergeCommitSHA, prs[0].MergeCommitSHA)
	require.Equal(t, mockClient.mr.WebURL, prs[0].URL)
	require.False(t, prs[0].Open)
}

func TestMergePullRequest(t *testing.T) {
	testCases := []struct {
		name         string
		mockClient   *mockGitLabClient
		id           int64
		expectErr    bool
		expectMerged bool
		expectPR     bool
		errContains  string
	}{
		{
			name: "error getting MR",
			mockClient: func() *mockGitLabClient {
				mc := &mockGitLabClient{}
				mc.getMRFunc = func(
					_ any, _ int64, _ *gitlab.GetMergeRequestsOptions,
					_ ...gitlab.RequestOptionFunc,
				) (*gitlab.MergeRequest, *gitlab.Response, error) {
					return nil, nil, errors.New("MR not found")
				}
				return mc
			}(),
			id:          999,
			expectErr:   true,
			errContains: "error getting merge request",
		},
		{
			name: "nil MR returned from get",
			mockClient: func() *mockGitLabClient {
				mc := &mockGitLabClient{}
				mc.getMRFunc = func(_ any, _ int64, _ *gitlab.GetMergeRequestsOptions,
					_ ...gitlab.RequestOptionFunc,
				) (*gitlab.MergeRequest, *gitlab.Response, error) {
					return nil, &gitlab.Response{}, nil
				}
				return mc
			}(),
			id:          404,
			expectErr:   true,
			errContains: "merge request 404 not found",
		},
		{
			name: "MR already merged",
			mockClient: &mockGitLabClient{
				mr: &gitlab.MergeRequest{
					BasicMergeRequest: gitlab.BasicMergeRequest{
						IID:            123,
						MergeCommitSHA: "sha123",
						State:          "merged",
						WebURL:         "https://gitlab.com/group/project/-/merge_requests/123",
					},
				},
			},
			id:           123,
			expectMerged: true,
			expectPR:     true,
		},
		{
			name: "MR not open",
			mockClient: &mockGitLabClient{
				mr: &gitlab.MergeRequest{
					BasicMergeRequest: gitlab.BasicMergeRequest{
						IID:            456,
						MergeCommitSHA: "sha456",
						State:          "closed",
						WebURL:         "https://gitlab.com/group/project/-/merge_requests/456",
					},
				},
			},
			id:           456,
			expectErr:    true,
			expectMerged: false,
			expectPR:     false,
			errContains:  "closed but not merged",
		},
		{
			name: "MR not ready to merge",
			mockClient: &mockGitLabClient{
				mr: &gitlab.MergeRequest{
					BasicMergeRequest: gitlab.BasicMergeRequest{
						IID:                 333,
						MergeCommitSHA:      "sha333",
						State:               "opened",
						DetailedMergeStatus: "cannot_be_merged",
						WebURL:              "https://gitlab.com/group/project/-/merge_requests/333",
					},
				},
			},
			id:           333,
			expectMerged: false,
			expectPR:     false,
		},
		{
			name: "error accepting MR",
			mockClient: func() *mockGitLabClient {
				mc := &mockGitLabClient{}
				mc.getMRFunc = func(_ any, _ int64, _ *gitlab.GetMergeRequestsOptions,
					_ ...gitlab.RequestOptionFunc,
				) (*gitlab.MergeRequest, *gitlab.Response, error) {
					return &gitlab.MergeRequest{
						BasicMergeRequest: gitlab.BasicMergeRequest{
							IID:                 888,
							State:               "opened",
							DetailedMergeStatus: "mergeable",
							WebURL:              "https://gitlab.com/group/project/-/merge_requests/888",
						},
					}, &gitlab.Response{}, nil
				}
				mc.acceptMRFunc = func(_ any, _ int64, _ *gitlab.AcceptMergeRequestOptions,
					_ ...gitlab.RequestOptionFunc,
				) (*gitlab.MergeRequest, *gitlab.Response, error) {
					return nil, nil, errors.New("merge conflicts")
				}
				return mc
			}(),
			id:          888,
			expectErr:   true,
			errContains: "error merging merge request",
		},
		{
			name: "nil MR returned after merge",
			mockClient: func() *mockGitLabClient {
				mc := &mockGitLabClient{}
				mc.getMRFunc = func(_ any, _ int64, _ *gitlab.GetMergeRequestsOptions,
					_ ...gitlab.RequestOptionFunc,
				) (*gitlab.MergeRequest, *gitlab.Response, error) {
					return &gitlab.MergeRequest{
						BasicMergeRequest: gitlab.BasicMergeRequest{
							IID:                 777,
							State:               "opened",
							DetailedMergeStatus: "mergeable",
							WebURL:              "https://gitlab.com/group/project/-/merge_requests/777",
						},
					}, &gitlab.Response{}, nil
				}
				mc.acceptMRFunc = func(_ any, _ int64, _ *gitlab.AcceptMergeRequestOptions,
					_ ...gitlab.RequestOptionFunc,
				) (*gitlab.MergeRequest, *gitlab.Response, error) {
					return nil, &gitlab.Response{}, nil
				}
				return mc
			}(),
			id:          777,
			expectErr:   true,
			errContains: "unexpected nil merge request after merge",
		},
		{
			name: "successful merge",
			mockClient: func() *mockGitLabClient {
				mc := &mockGitLabClient{}
				mc.getMRFunc = func(_ any, _ int64, _ *gitlab.GetMergeRequestsOptions,
					_ ...gitlab.RequestOptionFunc,
				) (*gitlab.MergeRequest, *gitlab.Response, error) {
					return &gitlab.MergeRequest{
						BasicMergeRequest: gitlab.BasicMergeRequest{
							IID:                 789,
							State:               "opened",
							DetailedMergeStatus: "mergeable",
							WebURL:              "https://gitlab.com/group/project/-/merge_requests/789",
						},
					}, &gitlab.Response{}, nil
				}
				mc.acceptMRFunc = func(_ any, _ int64, _ *gitlab.AcceptMergeRequestOptions,
					_ ...gitlab.RequestOptionFunc,
				) (*gitlab.MergeRequest, *gitlab.Response, error) {
					return &gitlab.MergeRequest{
						BasicMergeRequest: gitlab.BasicMergeRequest{
							IID:            789,
							MergeCommitSHA: "merged_sha789",
							State:          "merged",
							WebURL:         "https://gitlab.com/group/project/-/merge_requests/789",
						},
					}, &gitlab.Response{}, nil
				}
				return mc
			}(),
			id:           789,
			expectMerged: true,
			expectPR:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := provider{
				projectName: testProjectName,
				client:      tc.mockClient,
			}

			pr, merged, err := g.MergePullRequest(context.Background(), tc.id)

			if tc.expectErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
				require.False(t, merged)
				require.Nil(t, pr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectMerged, merged)
			if tc.expectPR {
				require.NotNil(t, pr)
				require.Equal(t, tc.id, pr.Number)
			} else {
				require.Nil(t, pr)
			}
			require.Equal(t, testProjectName, tc.mockClient.pid)
		})
	}
}

func TestParseGitLabURL(t *testing.T) {
	const expectedProjectName = "akuity/kargo"
	testCases := []struct {
		url            string
		expectedScheme string
		expectedHost   string
	}{
		{
			url:            "https://gitlab.com/akuity/kargo",
			expectedScheme: "https",
			expectedHost:   "gitlab.com",
		},
		{
			url:            "https://gitlab.com/akuity/kargo.git",
			expectedScheme: "https",
			expectedHost:   "gitlab.com",
		},
		{
			// This isn't a real URL. It's just to validate that the function can
			// handle GitHub Enterprise URLs.
			url:            "https://gitlab.akuity.io/akuity/kargo.git",
			expectedScheme: "https",
			expectedHost:   "gitlab.akuity.io",
		},
		{
			url:            "ssh://gitlab.com/akuity/kargo.git",
			expectedScheme: "https",
			expectedHost:   "gitlab.com",
		},
		{
			url:            "http://git.example.com/akuity/kargo",
			expectedScheme: "http",
			expectedHost:   "git.example.com",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.url, func(t *testing.T) {
			scheme, host, projectName, err := parseRepoURL(testCase.url)
			require.NoError(t, err)
			require.Equal(t, testCase.expectedScheme, scheme)
			require.Equal(t, testCase.expectedHost, host)
			require.Equal(t, expectedProjectName, projectName)
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
			repoURL:           "ssh://git@gitlab.com/akuity/kargo.git",
			sha:               "sha",
			expectedCommitURL: "https://gitlab.com/akuity/kargo/-/commit/sha",
		},
		{
			repoURL:           "git@gitlab.com:akuity/kargo.git",
			sha:               "sha",
			expectedCommitURL: "https://gitlab.com/akuity/kargo/-/commit/sha",
		},
		{
			repoURL:           "http://gitlab.com/akuity/kargo",
			sha:               "sha",
			expectedCommitURL: "https://gitlab.com/akuity/kargo/-/commit/sha",
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
