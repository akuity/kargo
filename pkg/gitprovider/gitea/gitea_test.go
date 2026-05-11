package gitea

import (
	"errors"
	"testing"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/akuity/kargo/pkg/gitprovider"
)

const testRepoOwner = "akuity"
const testRepoName = "kargo"

func TestParseGiteaURL(t *testing.T) {
	testCases := []struct {
		url            string
		expectedScheme string
		expectedHost   string
		expectedOwner  string
		expectedRepo   string
		errExpected    bool
	}{
		{
			url:         "not-a-url",
			errExpected: true,
		},
		{
			url:         "https://git.domain.com/akuity",
			errExpected: true,
		},
		{
			url:            "https://git.domain.com/akuity/kargo",
			expectedScheme: "https",
			expectedHost:   "git.domain.com",
			expectedOwner:  "akuity",
			expectedRepo:   "kargo",
		},
		{
			url:            "https://git.domain.com/akuity/kargo.git",
			expectedScheme: "https",
			expectedHost:   "git.domain.com",
			expectedOwner:  "akuity",
			expectedRepo:   "kargo",
		},
		{
			url:            "git@git.domain.com:akuity/kargo",
			expectedScheme: "https",
			expectedHost:   "git.domain.com",
			expectedOwner:  "akuity",
			expectedRepo:   "kargo",
		},
		{
			url:            "http://git.example.com:8080/akuity/kargo",
			expectedScheme: "http",
			expectedHost:   "git.example.com:8080",
			expectedOwner:  "akuity",
			expectedRepo:   "kargo",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.url, func(t *testing.T) {
			scheme, host, owner, repo, err := parseRepoURL(testCase.url)
			if testCase.errExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, testCase.expectedScheme, scheme)
				require.Equal(t, testCase.expectedHost, host)
				require.Equal(t, testCase.expectedOwner, owner)
				require.Equal(t, testCase.expectedRepo, repo)
			}
		})
	}
}

type mockGiteaClient struct {
	mock.Mock
	newPr              gitea.CreatePullRequestOption
	pr                 *gitea.PullRequest
	owner              string
	repo               string
	issueLabelsOptions gitea.IssueLabelsOption
	listOpts           gitea.ListPullRequestsOptions
}

func (m *mockGiteaClient) ListRepoPullRequests(
	owner string,
	repo string,
	opts gitea.ListPullRequestsOptions,
) ([]*gitea.PullRequest, *gitea.Response, error) {
	args := m.Called(owner, repo, opts)
	m.owner = owner
	m.repo = repo
	m.listOpts = opts
	prs, ok := args.Get(0).([]*gitea.PullRequest)
	if !ok {
		return nil, nil, args.Error(2)
	}
	resp, ok := args.Get(1).(*gitea.Response)
	if !ok {
		return prs, nil, args.Error(2)
	}
	return prs, resp, args.Error(2)
}

func (m *mockGiteaClient) GetPullRequest(
	owner string,
	repo string,
	number int64,
) (*gitea.PullRequest, *gitea.Response, error) {
	args := m.Called(owner, repo, number)
	m.owner = owner
	m.repo = repo
	pr, ok := args.Get(0).(*gitea.PullRequest)
	if !ok {
		return nil, nil, args.Error(2)
	}
	resp, ok := args.Get(1).(*gitea.Response)
	if !ok {
		return pr, nil, args.Error(2)
	}
	return pr, resp, args.Error(2)
}

func (m *mockGiteaClient) AddIssueLabels(
	owner string,
	repo string,
	number int64,
	opts gitea.IssueLabelsOption,
) ([]*gitea.Label, *gitea.Response, error) {
	args := m.Called(owner, repo, number, opts)
	m.issueLabelsOptions = opts
	labelsResp, ok := args.Get(0).([]*gitea.Label)
	if !ok {
		return nil, nil, args.Error(2)
	}
	resp, ok := args.Get(1).(*gitea.Response)
	if !ok {
		return labelsResp, nil, args.Error(2)
	}
	return labelsResp, resp, args.Error(2)
}

func (m *mockGiteaClient) MergePullRequest(
	owner string,
	repo string,
	number int64,
	opts gitea.MergePullRequestOption,
) (bool, *gitea.Response, error) {
	args := m.Called(owner, repo, number, opts)
	resp, _ := args.Get(1).(*gitea.Response)
	return args.Bool(0), resp, args.Error(2)
}

func (m *mockGiteaClient) CreatePullRequest(
	owner string,
	repo string,
	opts gitea.CreatePullRequestOption,
) (*gitea.PullRequest, *gitea.Response, error) {
	args := m.Called(owner, repo, opts)
	m.owner = owner
	m.repo = repo
	m.newPr = opts

	pr, ok := args.Get(0).(*gitea.PullRequest)
	if !ok {
		return nil, nil, args.Error(2)
	}
	resp, ok := args.Get(1).(*gitea.Response)
	if !ok {
		return pr, nil, args.Error(2)
	}
	return pr, resp, args.Error(2)
}

func TestCreatePullRequest(t *testing.T) {
	opts := gitprovider.CreatePullRequestOpts{
		Head:        "feature-branch",
		Base:        "main",
		Title:       "title",
		Description: "desc",
	}

	// set up mock
	mockClient := &mockGiteaClient{
		pr: &gitea.PullRequest{
			Index: int64(42),
			State: gitea.StateOpen,
			Head: &gitea.PRBranchInfo{
				Sha: "HeadSha",
			},
			Base: &gitea.PRBranchInfo{
				Sha: "BaseSha",
			},
			HTMLURL:        "http://localhost:8080",
			MergedCommitID: ptr.To("2994fd93"),
			HasMerged:      false,
		},
	}
	mockClient.
		On("CreatePullRequest", testRepoOwner, testRepoName, mock.Anything).
		Return(
			&gitea.PullRequest{
				Index: int64(42),
				State: gitea.StateOpen,
				Head: &gitea.PRBranchInfo{
					Sha: "HeadSha",
				},
				Base: &gitea.PRBranchInfo{
					Sha: "BaseSha",
				},
				HTMLURL:        "http://localhost:8080",
				MergedCommitID: ptr.To("BaseSha"),
				HasMerged:      false,
				Created:        &time.Time{},
			},
			&gitea.Response{},
			nil,
		)

	// call the code we are testing
	g := provider{
		owner:  testRepoOwner,
		repo:   testRepoName,
		client: mockClient,
	}
	pr, err := g.CreatePullRequest(t.Context(), &opts)

	// assert that the expectations were met
	mockClient.AssertExpectations(t)

	// other assertions
	require.NoError(t, err)
	require.Equal(t, testRepoOwner, mockClient.owner)
	require.Equal(t, testRepoName, mockClient.repo)
	require.Equal(t, opts.Head, mockClient.newPr.Head)
	require.Equal(t, opts.Base, mockClient.newPr.Base)
	require.Equal(t, opts.Title, mockClient.newPr.Title,
		"Expected title in new PR request to match title from options")
	require.Equal(t, opts.Description, mockClient.newPr.Body,
		"Expected body in new PR request to match description from options")

	require.Equal(t, mockClient.pr.Index, pr.Number,
		"Expected PR number in returned object to match what was returned by gitea")
	require.Equal(t, mockClient.pr.Base.Sha, pr.MergeCommitSHA)
	require.Equal(t, mockClient.pr.HTMLURL, pr.URL)
	require.True(t, pr.Open)
}

func TestGetPullRequest(t *testing.T) {
	// set up mock
	mockClient := &mockGiteaClient{
		pr: &gitea.PullRequest{
			Index: int64(42),
			State: gitea.StateOpen,
			Head: &gitea.PRBranchInfo{
				Sha: "HeadSha",
			},
			Base: &gitea.PRBranchInfo{
				Sha: "BaseSha",
			},
			HTMLURL:        "http://localhost:8080",
			MergedCommitID: ptr.To("2994fd93"),
			HasMerged:      false,
			Created:        &time.Time{},
		},
	}

	mockClient.
		On("GetPullRequest", testRepoOwner, testRepoName, mockClient.pr.Index).
		Return(
			&gitea.PullRequest{
				Index: int64(42),
				State: gitea.StateOpen,
				Head: &gitea.PRBranchInfo{
					Sha: "HeadSha",
				},
				Base: &gitea.PRBranchInfo{
					Sha: "BaseSha",
				},
				HTMLURL:        "http://localhost:8080",
				MergedCommitID: ptr.To("BaseSha"),
				HasMerged:      false,
			},
			&gitea.Response{},
			nil,
		)

	// call the code we are testing
	g := provider{
		owner:  testRepoOwner,
		repo:   testRepoName,
		client: mockClient,
	}
	pr, err := g.GetPullRequest(t.Context(), 42)

	// assert that the expectations were met
	mockClient.AssertExpectations(t)

	// other assertions
	require.NoError(t, err)
	require.Equal(t, testRepoOwner, mockClient.owner)
	require.Equal(t, testRepoName, mockClient.repo)
	require.Equal(t, mockClient.pr.Index, pr.Number,
		"Expected PR number in returned object to match what was returned by gitea")
	require.Equal(t, mockClient.pr.Base.Sha, pr.MergeCommitSHA)
	require.Equal(t, mockClient.pr.HTMLURL, pr.URL)
	require.True(t, pr.Open)
}

func TestListPullRequests(t *testing.T) {
	opts := gitprovider.ListPullRequestOptions{
		State:      gitprovider.PullRequestStateAny,
		HeadBranch: "head",
		BaseBranch: "base",
	}

	// set up mock
	mockClient := &mockGiteaClient{
		pr: &gitea.PullRequest{
			Index: int64(42),
			State: gitea.StateOpen,
			Head: &gitea.PRBranchInfo{
				Sha: "HeadSha",
			},
			Base: &gitea.PRBranchInfo{
				Sha: "BaseSha",
			},
			HTMLURL:        "http://localhost:8080",
			MergedCommitID: ptr.To("BaseSha"),
			HasMerged:      false,
		},
	}
	mockClient.
		On("ListRepoPullRequests", testRepoOwner, testRepoName, gitea.ListPullRequestsOptions{
			State: "all",
			ListOptions: gitea.ListOptions{
				Page: 0,
			},
		}).
		Return(
			[]*gitea.PullRequest{{
				Index: int64(42),
				State: gitea.StateOpen,
				Head: &gitea.PRBranchInfo{
					Sha: "HeadSha",
				},
				Base: &gitea.PRBranchInfo{
					Sha: "BaseSha",
				},
				HTMLURL:        "http://localhost:8080",
				MergedCommitID: ptr.To("BaseSha"),
				HasMerged:      false,
				Created:        &time.Time{},
			}},
			&gitea.Response{},
			nil,
		)

	// call the code we are testing
	g := provider{
		owner:  testRepoOwner,
		repo:   testRepoName,
		client: mockClient,
	}

	prs, err := g.ListPullRequests(t.Context(), &opts)
	require.NoError(t, err)

	require.Equal(t, testRepoOwner, mockClient.owner)
	require.Equal(t, testRepoName, mockClient.repo)

	require.Equal(t, mockClient.pr.Index, prs[0].Number)
	require.Equal(t, mockClient.pr.Base.Sha, prs[0].MergeCommitSHA)
	require.Equal(t, mockClient.pr.HTMLURL, prs[0].URL)
	require.True(t, prs[0].Open)
}

func TestMergePullRequest(t *testing.T) {
	tests := []struct {
		name           string
		prNumber       int64
		mergeOpts      *gitprovider.MergePullRequestOpts
		setupMock      func(*mockGiteaClient)
		expectedMerged bool
		expectError    bool
		errorContains  string
	}{
		{
			name:     "error getting initial PR state",
			prNumber: 999,
			setupMock: func(m *mockGiteaClient) {
				m.On("GetPullRequest", testRepoOwner, testRepoName, int64(999)).
					Return(nil, nil, errors.New("get PR failed"))
			},
			expectError:   true,
			errorContains: "error getting pull request",
		},
		{
			name:     "nil PR returned from initial get",
			prNumber: 404,
			setupMock: func(m *mockGiteaClient) {
				m.On("GetPullRequest", testRepoOwner, testRepoName, int64(404)).
					Return(nil, &gitea.Response{}, nil)
			},
			expectError:   true,
			errorContains: "pull request 404 not found",
		},
		{
			name:     "PR already merged",
			prNumber: 123,
			setupMock: func(m *mockGiteaClient) {
				m.On("GetPullRequest", testRepoOwner, testRepoName, int64(123)).
					Return(&gitea.PullRequest{
						Index:     123,
						State:     gitea.StateClosed,
						HasMerged: true,
						Head:      &gitea.PRBranchInfo{Sha: "head_sha"},
						URL:       "https://gitea.com/akuity/kargo/pulls/123",
					}, &gitea.Response{}, nil)
			},
			expectedMerged: true,
		},
		{
			name:     "PR not in open state",
			prNumber: 456,
			setupMock: func(m *mockGiteaClient) {
				m.On("GetPullRequest", testRepoOwner, testRepoName, int64(456)).
					Return(&gitea.PullRequest{
						Index:     456,
						State:     gitea.StateClosed,
						HasMerged: false,
					}, &gitea.Response{}, nil)
			},
			expectError:   true,
			errorContains: "closed but not merged",
		},
		{
			name:     "PR not mergeable",
			prNumber: 333,
			setupMock: func(m *mockGiteaClient) {
				m.On("GetPullRequest", testRepoOwner, testRepoName, int64(333)).
					Return(&gitea.PullRequest{
						Index:     333,
						State:     gitea.StateOpen,
						Mergeable: false,
					}, &gitea.Response{}, nil)
			},
		},
		{
			name:     "merge operation fails",
			prNumber: 888,
			setupMock: func(m *mockGiteaClient) {
				m.On("GetPullRequest", testRepoOwner, testRepoName, int64(888)).
					Return(&gitea.PullRequest{
						Index:     888,
						State:     gitea.StateOpen,
						Mergeable: true,
					}, &gitea.Response{}, nil).Once()

				m.On("MergePullRequest", testRepoOwner, testRepoName, int64(888),
					mock.AnythingOfType("gitea.MergePullRequestOption")).
					Return(false, nil, errors.New("merge failed"))
			},
			expectError:   true,
			errorContains: "error merging pull request",
		},
		{
			name:     "merge rejected",
			prNumber: 555,
			setupMock: func(m *mockGiteaClient) {
				m.On("GetPullRequest", testRepoOwner, testRepoName, int64(555)).
					Return(&gitea.PullRequest{
						Index:     555,
						State:     gitea.StateOpen,
						Mergeable: true,
					}, &gitea.Response{}, nil).Once()

				m.On("MergePullRequest", testRepoOwner, testRepoName, int64(555),
					mock.AnythingOfType("gitea.MergePullRequestOption")).
					Return(false, &gitea.Response{}, nil)
			},
			expectError:   true,
			errorContains: "merge rejected",
		},
		{
			name:     "get PR after merge fails",
			prNumber: 777,
			setupMock: func(m *mockGiteaClient) {
				m.On("GetPullRequest", testRepoOwner, testRepoName, int64(777)).
					Return(&gitea.PullRequest{
						Index:     777,
						State:     gitea.StateOpen,
						Mergeable: true,
					}, &gitea.Response{}, nil).Once()

				m.On("MergePullRequest", testRepoOwner, testRepoName, int64(777),
					mock.AnythingOfType("gitea.MergePullRequestOption")).
					Return(true, &gitea.Response{}, nil)

				m.On("GetPullRequest", testRepoOwner, testRepoName, int64(777)).
					Return(nil, nil, errors.New("get PR failed")).Once()
			},
			expectError:   true,
			errorContains: "error fetching PR",
		},
		{
			name:     "nil PR returned after merge",
			prNumber: 666,
			setupMock: func(m *mockGiteaClient) {
				m.On("GetPullRequest", testRepoOwner, testRepoName, int64(666)).
					Return(&gitea.PullRequest{
						Index:     666,
						State:     gitea.StateOpen,
						Mergeable: true,
					}, &gitea.Response{}, nil).Once()

				m.On("MergePullRequest", testRepoOwner, testRepoName, int64(666),
					mock.AnythingOfType("gitea.MergePullRequestOption")).
					Return(true, &gitea.Response{}, nil)

				m.On("GetPullRequest", testRepoOwner, testRepoName, int64(666)).
					Return(nil, &gitea.Response{}, nil).Once()
			},
			expectError:   true,
			errorContains: "unexpected nil PR after merge",
		},
		{
			name:      "successful merge with explicit method",
			prNumber:  100,
			mergeOpts: &gitprovider.MergePullRequestOpts{MergeMethod: "squash"},
			setupMock: func(m *mockGiteaClient) {
				m.On("GetPullRequest", testRepoOwner, testRepoName, int64(100)).
					Return(&gitea.PullRequest{
						Index:     100,
						State:     gitea.StateOpen,
						Mergeable: true,
						Head:      &gitea.PRBranchInfo{Sha: "head_sha"},
						URL:       "https://gitea.com/akuity/kargo/pulls/100",
					}, &gitea.Response{}, nil).Once()

				m.On("MergePullRequest", testRepoOwner, testRepoName, int64(100),
					mock.MatchedBy(func(opts gitea.MergePullRequestOption) bool {
						return opts.Style == gitea.MergeStyleSquash
					})).
					Return(true, &gitea.Response{}, nil)

				m.On("GetPullRequest", testRepoOwner, testRepoName, int64(100)).
					Return(&gitea.PullRequest{
						Index:     100,
						State:     gitea.StateClosed,
						HasMerged: true,
						Head:      &gitea.PRBranchInfo{Sha: "head_sha"},
						URL:       "https://gitea.com/akuity/kargo/pulls/100",
					}, &gitea.Response{}, nil).Once()
			},
			expectedMerged: true,
		},
		{
			name:      "unsupported merge method",
			prNumber:  100,
			mergeOpts: &gitprovider.MergePullRequestOpts{MergeMethod: "bogus"},
			setupMock: func(m *mockGiteaClient) {
				m.On("GetPullRequest", testRepoOwner, testRepoName, int64(100)).
					Return(&gitea.PullRequest{
						Index:     100,
						State:     gitea.StateOpen,
						Mergeable: true,
						Head:      &gitea.PRBranchInfo{Sha: "head_sha"},
						URL:       "https://gitea.com/akuity/kargo/pulls/100",
					}, &gitea.Response{}, nil).Once()
			},
			expectError:   true,
			errorContains: `unsupported merge method "bogus"`,
		},
		{
			name:     "successful merge",
			prNumber: 1234,
			setupMock: func(m *mockGiteaClient) {
				m.On("GetPullRequest", testRepoOwner, testRepoName, int64(1234)).
					Return(&gitea.PullRequest{
						Index:     1234,
						State:     gitea.StateOpen,
						Mergeable: true,
						Head:      &gitea.PRBranchInfo{Sha: "head_sha"},
						URL:       "https://gitea.com/akuity/kargo/pulls/1234",
					}, &gitea.Response{}, nil).Once()

				m.On("MergePullRequest", testRepoOwner, testRepoName,
					int64(1234), mock.AnythingOfType("gitea.MergePullRequestOption")).
					Return(true, &gitea.Response{}, nil)

				m.On("GetPullRequest", testRepoOwner, testRepoName, int64(1234)).
					Return(&gitea.PullRequest{
						Index:     1234,
						State:     gitea.StateClosed,
						HasMerged: true,
						Head:      &gitea.PRBranchInfo{Sha: "head_sha"},
						URL:       "https://gitea.com/akuity/kargo/pulls/1234",
					}, &gitea.Response{}, nil).Once()
			},
			expectedMerged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockGiteaClient{}
			p := provider{
				owner:  testRepoOwner,
				repo:   testRepoName,
				client: mockClient,
			}

			tt.setupMock(mockClient)

			pr, merged, err := p.MergePullRequest(t.Context(), tt.prNumber, tt.mergeOpts)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorContains)
				require.False(t, merged)
				require.Nil(t, pr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedMerged, merged)
				if pr != nil {
					require.Equal(t, tt.prNumber, pr.Number)
				}
			}

			mockClient.AssertExpectations(t)
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
			repoURL:           "http://gitea.com/akuity/kargo",
			sha:               "sha",
			expectedCommitURL: "https://gitea.com/akuity/kargo/commit/sha",
		},
		{
			repoURL:           "ssh://git@gitea.com/akuity/kargo",
			sha:               "sha",
			expectedCommitURL: "https://gitea.com/akuity/kargo/commit/sha",
		},
		{
			repoURL:           "git@gitea.com:akuity/kargo",
			sha:               "sha",
			expectedCommitURL: "https://gitea.com/akuity/kargo/commit/sha",
		},
		{
			repoURL:           "git@custom.host.com:akuity/kargo",
			sha:               "sha",
			expectedCommitURL: "https://custom.host.com/akuity/kargo/commit/sha",
		},
		{
			repoURL:           "http://custom.host.com/akuity/kargo",
			sha:               "sha",
			expectedCommitURL: "https://custom.host.com/akuity/kargo/commit/sha",
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
