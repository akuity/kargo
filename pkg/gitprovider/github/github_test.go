package github

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/gitprovider"
)

const (
	testRepoOwner = "akuity"
	testRepoName  = "kargo"
)

func TestRegistrationPredicate(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "Standard GitHub URL",
			url:      "https://github.com/akuity/kargo",
			expected: true,
		},
		{
			name:     "GitHub Enterprise (ghe.com)",
			url:      "https://my-org.ghe.com/repo/project",
			expected: true, // This would fail before your fix!
		},
		{
			name:     "GitLab URL (Should fail)",
			url:      "https://gitlab.com/owner/repo",
			expected: false,
		},
		{
			name:     "Random Domain (Should fail)",
			url:      "https://google.com",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Accessing the unexported 'registration' variable directly
			// because we are in package 'github'
			result := registration.Predicate(tc.url)
			require.Equal(t, tc.expected, result)
		})
	}
}

type mockGithubClient struct {
	mock.Mock
	pr       *github.PullRequest
	owner    string
	repo     string
	newPr    *github.NewPullRequest
	labels   []string
	listOpts *github.PullRequestListOptions
}

func (m *mockGithubClient) ListPullRequests(
	ctx context.Context,
	owner string,
	repo string,
	opts *github.PullRequestListOptions,
) ([]*github.PullRequest, *github.Response, error) {
	args := m.Called(ctx, owner, repo, opts)
	m.owner = owner
	m.repo = repo
	m.listOpts = opts
	prs, ok := args.Get(0).([]*github.PullRequest)
	if !ok {
		return nil, nil, args.Error(2)
	}
	resp, ok := args.Get(1).(*github.Response)
	if !ok {
		return prs, nil, args.Error(2)
	}
	return prs, resp, args.Error(2)
}

func (m *mockGithubClient) GetPullRequests(
	ctx context.Context,
	owner string,
	repo string,
	number int,
) (*github.PullRequest, *github.Response, error) {
	args := m.Called(ctx, owner, repo, number)
	m.owner = owner
	m.repo = repo
	pr, ok := args.Get(0).(*github.PullRequest)
	if !ok {
		return nil, nil, args.Error(2)
	}
	resp, ok := args.Get(1).(*github.Response)
	if !ok {
		return pr, nil, args.Error(2)
	}
	return pr, resp, args.Error(2)
}

func (m *mockGithubClient) MergePullRequest(
	ctx context.Context,
	owner string,
	repo string,
	number int,
	commitMessage string,
	options *github.PullRequestOptions,
) (*github.PullRequestMergeResult, *github.Response, error) {
	args := m.Called(ctx, owner, repo, number, commitMessage, options)
	result, ok := args.Get(0).(*github.PullRequestMergeResult)
	if !ok {
		return nil, nil, args.Error(2)
	}
	resp, ok := args.Get(1).(*github.Response)
	if !ok {
		return result, nil, args.Error(2)
	}
	return result, resp, args.Error(2)
}

func (m *mockGithubClient) AddLabelsToIssue(
	ctx context.Context,
	owner string,
	repo string,
	number int,
	labels []string,
) ([]*github.Label, *github.Response, error) {
	args := m.Called(ctx, owner, repo, number, labels)
	m.labels = labels
	labelsResp, ok := args.Get(0).([]*github.Label)
	if !ok {
		return nil, nil, args.Error(2)
	}
	resp, ok := args.Get(1).(*github.Response)
	if !ok {
		return labelsResp, nil, args.Error(2)
	}
	return labelsResp, resp, args.Error(2)
}

func (m *mockGithubClient) CreatePullRequest(
	ctx context.Context,
	owner string,
	repo string,
	pull *github.NewPullRequest,
) (*github.PullRequest, *github.Response, error) {
	args := m.Called(ctx, owner, repo, pull)
	m.owner = owner
	m.repo = repo
	m.newPr = pull

	pr, ok := args.Get(0).(*github.PullRequest)
	if !ok {
		return nil, nil, args.Error(2)
	}
	resp, ok := args.Get(1).(*github.Response)
	if !ok {
		return pr, nil, args.Error(2)
	}
	return pr, resp, args.Error(2)
}

func TestCreatePullRequestWithLabels(t *testing.T) {
	opts := gitprovider.CreatePullRequestOpts{
		Head:        "feature-branch",
		Base:        "main",
		Title:       "title",
		Description: "desc",
		Labels:      []string{"label1", "label2"},
	}

	// set up mock
	mockClient := &mockGithubClient{
		pr: &github.PullRequest{
			Number:         github.Ptr(42),
			MergeCommitSHA: github.Ptr("sha"),
			State:          github.Ptr("open"),
			URL:            github.Ptr("url"),
		},
	}
	mockClient.
		On("CreatePullRequest", t.Context(), testRepoOwner, testRepoName, mock.Anything).
		Return(
			&github.PullRequest{
				Number: mockClient.pr.Number,
				Head: &github.PullRequestBranch{
					Ref: github.Ptr(opts.Head),
				},
				Base: &github.PullRequestBranch{
					Ref: github.Ptr(opts.Base),
				},
				Title:          github.Ptr(opts.Title),
				Body:           github.Ptr(opts.Description),
				MergeCommitSHA: mockClient.pr.MergeCommitSHA,
				State:          mockClient.pr.State,
				HTMLURL:        mockClient.pr.URL,
			},
			&github.Response{},
			nil,
		)
	mockClient.
		On("AddLabelsToIssue", t.Context(), testRepoOwner, testRepoName, *mockClient.pr.Number, mock.Anything).
		Return(
			[]*github.Label{},
			&github.Response{},
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
	require.Equal(t, opts.Head, *mockClient.newPr.Head)
	require.Equal(t, opts.Base, *mockClient.newPr.Base)
	require.Equal(t, opts.Title, *mockClient.newPr.Title,
		"Expected title in new PR request to match title from options")
	require.Equal(t, opts.Description, *mockClient.newPr.Body,
		"Expected body in new PR request to match description from options")
	require.ElementsMatch(t, opts.Labels, mockClient.labels,
		"Expected labels passed to GitHub client to match labels from options")

	require.Equal(t, int64(*mockClient.pr.Number), pr.Number,
		"Expected PR number in returned object to match what was returned by GitHub")
	require.Equal(t, *mockClient.pr.MergeCommitSHA, pr.MergeCommitSHA)
	require.Equal(t, *mockClient.pr.URL, pr.URL)
	require.True(t, pr.Open)
}

func TestGetPullRequest(t *testing.T) {
	// set up mock
	mockClient := &mockGithubClient{
		pr: &github.PullRequest{
			Number:         github.Ptr(42),
			MergeCommitSHA: github.Ptr("sha"),
			State:          github.Ptr("open"),
			URL:            github.Ptr("url"),
		},
	}
	mockClient.
		On("GetPullRequests", t.Context(), testRepoOwner, testRepoName, *mockClient.pr.Number).
		Return(
			&github.PullRequest{
				Number: mockClient.pr.Number,
				Head: &github.PullRequestBranch{
					Ref: github.Ptr("head"),
				},
				MergeCommitSHA: mockClient.pr.MergeCommitSHA,
				State:          mockClient.pr.State,
				HTMLURL:        mockClient.pr.URL,
			},
			&github.Response{},
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
	require.Equal(t, int64(*mockClient.pr.Number), pr.Number,
		"Expected PR number in returned object to match what was returned by GitHub")
	require.Equal(t, *mockClient.pr.MergeCommitSHA, pr.MergeCommitSHA)
	require.Equal(t, *mockClient.pr.URL, pr.URL)
	require.True(t, pr.Open)
}

func TestListPullRequests(t *testing.T) {
	opts := gitprovider.ListPullRequestOptions{
		State:      gitprovider.PullRequestStateAny,
		HeadBranch: "head",
		BaseBranch: "base",
	}

	// set up mock
	mockClient := &mockGithubClient{
		pr: &github.PullRequest{
			Number:         github.Ptr(42),
			MergeCommitSHA: github.Ptr("sha"),
			State:          github.Ptr("open"),
			URL:            github.Ptr("url"),
		},
	}
	mockClient.
		On("ListPullRequests", t.Context(), testRepoOwner, testRepoName, &github.PullRequestListOptions{
			State:     "all",
			Head:      opts.HeadBranch,
			Base:      opts.BaseBranch,
			Sort:      "",
			Direction: "",
			ListOptions: github.ListOptions{
				Page:    0,
				PerPage: 100,
			},
		}).
		Return(
			[]*github.PullRequest{{
				Number: mockClient.pr.Number,
				Head: &github.PullRequestBranch{
					Ref: github.Ptr("head"),
				},
				MergeCommitSHA: mockClient.pr.MergeCommitSHA,
				State:          mockClient.pr.State,
				HTMLURL:        mockClient.pr.URL,
			}},
			&github.Response{},
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
	require.Equal(t, opts.HeadBranch, mockClient.listOpts.Head)
	require.Equal(t, opts.BaseBranch, mockClient.listOpts.Base)

	require.Equal(t, int64(*mockClient.pr.Number), prs[0].Number)
	require.Equal(t, *mockClient.pr.MergeCommitSHA, prs[0].MergeCommitSHA)
	require.Equal(t, *mockClient.pr.URL, prs[0].URL)
	require.True(t, prs[0].Open)
}

func TestMergePullRequest(t *testing.T) {
	tests := []struct {
		name               string
		prNumber           int64
		mergeOpts          *gitprovider.MergePullRequestOpts
		setupMock          func(*mockGithubClient)
		expectedMerged     bool
		expectError        bool
		errorContains      string
		expectMergeOptions *github.PullRequestOptions
	}{
		{
			name:     "error getting initial PR state",
			prNumber: 999,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(999)).
					Return(nil, nil, errors.New("get PR failed"))
			},
			expectError:   true,
			errorContains: "error getting pull request",
		},
		{
			name:     "nil PR returned from initial get",
			prNumber: 404,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(404)).
					Return(nil, &github.Response{}, nil)
			},
			expectError:   true,
			errorContains: "pull request 404 not found",
		},
		{
			name:     "PR already merged",
			prNumber: 123,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(123)).
					Return(&github.PullRequest{
						Number:         github.Ptr(123),
						State:          github.Ptr("closed"),
						Merged:         github.Ptr(true),
						HTMLURL:        github.Ptr("https://github.com/akuity/kargo/pull/123"),
						MergeCommitSHA: github.Ptr("merge_sha"),
						Head: &github.PullRequestBranch{
							SHA: github.Ptr("head_sha"),
						},
						MergedAt: &github.Timestamp{Time: time.Now()},
					}, &github.Response{}, nil)
			},
			expectedMerged: true,
		},
		{
			name:     "PR closed but not merged",
			prNumber: 456,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(456)).
					Return(&github.PullRequest{
						Number:  github.Ptr(456),
						State:   github.Ptr("closed"),
						Merged:  github.Ptr(false),
						HTMLURL: github.Ptr("https://github.com/akuity/kargo/pull/456"),
						Head:    &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
					}, &github.Response{}, nil)
			},
			expectError:   true,
			errorContains: "closed but not merged",
		},
		{
			name:     "unknown mergeability",
			prNumber: 444,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(444)).
					Return(&github.PullRequest{
						Number:    github.Ptr(444),
						State:     github.Ptr("open"),
						Merged:    github.Ptr(false),
						Mergeable: nil,
						Head:      &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:   github.Ptr("https://github.com/akuity/kargo/pull/444"),
					}, &github.Response{}, nil)
			},
			expectError: false,
		},
		{
			name:     "PR not ready to merge",
			prNumber: 789,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(789)).
					Return(&github.PullRequest{
						Number:    github.Ptr(789),
						State:     github.Ptr("open"),
						Merged:    github.Ptr(false),
						Mergeable: github.Ptr(false),
						Head:      &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:   github.Ptr("https://github.com/akuity/kargo/pull/789"),
					}, &github.Response{}, nil)
			},
		},
		{
			name:     "mergeable_state clean proceeds to merge",
			prNumber: 201,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(201)).
					Return(&github.PullRequest{
						Number:         github.Ptr(201),
						State:          github.Ptr("open"),
						Merged:         github.Ptr(false),
						Mergeable:      github.Ptr(true),
						MergeableState: github.Ptr("clean"),
						Head:           &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:        github.Ptr("https://github.com/akuity/kargo/pull/201"),
					}, &github.Response{}, nil).Once()

				m.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(201), "",
					mock.AnythingOfType("*github.PullRequestOptions")).
					Return(&github.PullRequestMergeResult{
						SHA:    github.Ptr("merge_sha"),
						Merged: github.Ptr(true),
					}, &github.Response{}, nil)

				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(201)).
					Return(&github.PullRequest{
						Number:         github.Ptr(201),
						State:          github.Ptr("closed"),
						Merged:         github.Ptr(true),
						MergeCommitSHA: github.Ptr("merge_sha"),
						Head:           &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:        github.Ptr("https://github.com/akuity/kargo/pull/201"),
						MergedAt:       &github.Timestamp{Time: time.Now()},
					}, &github.Response{}, nil).Once()
			},
			expectedMerged: true,
		},
		{
			name:     "mergeable_state unknown is not ready",
			prNumber: 202,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(202)).
					Return(&github.PullRequest{
						Number:         github.Ptr(202),
						State:          github.Ptr("open"),
						Merged:         github.Ptr(false),
						Mergeable:      nil,
						MergeableState: github.Ptr("unknown"),
						Head:           &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:        github.Ptr("https://github.com/akuity/kargo/pull/202"),
					}, &github.Response{}, nil)
			},
			expectError: false,
		},
		{
			name:     "mergeable_state draft is not ready",
			prNumber: 203,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(203)).
					Return(&github.PullRequest{
						Number:         github.Ptr(203),
						State:          github.Ptr("open"),
						Merged:         github.Ptr(false),
						Draft:          github.Ptr(true),
						Mergeable:      github.Ptr(true),
						MergeableState: github.Ptr("draft"),
						Head:           &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:        github.Ptr("https://github.com/akuity/kargo/pull/203"),
					}, &github.Response{}, nil)
			},
			expectError: false,
		},
		{
			// mergeable_state describes the repository's rules, not the caller's
			// permissions. A caller authorized to bypass them merges a blocked PR,
			// so the merge must be attempted rather than pre-empted.
			name:     "mergeable_state blocked merges for a bypass-authorized caller",
			prNumber: 205,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(205)).
					Return(&github.PullRequest{
						Number:         github.Ptr(205),
						State:          github.Ptr("open"),
						Merged:         github.Ptr(false),
						Mergeable:      github.Ptr(true),
						MergeableState: github.Ptr("blocked"),
						Head:           &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:        github.Ptr("https://github.com/akuity/kargo/pull/205"),
					}, &github.Response{}, nil).Once()

				m.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(205), "",
					mock.AnythingOfType("*github.PullRequestOptions")).
					Return(&github.PullRequestMergeResult{
						SHA:    github.Ptr("merge_sha"),
						Merged: github.Ptr(true),
					}, &github.Response{}, nil)

				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(205)).
					Return(&github.PullRequest{
						Number:         github.Ptr(205),
						State:          github.Ptr("closed"),
						Merged:         github.Ptr(true),
						MergeCommitSHA: github.Ptr("merge_sha"),
						Head:           &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:        github.Ptr("https://github.com/akuity/kargo/pull/205"),
						MergedAt:       &github.Timestamp{Time: time.Now()},
					}, &github.Response{}, nil).Once()
			},
			expectedMerged: true,
		},
		{
			// The caller is not authorized to bypass the rules. The block may still
			// clear on its own (a review lands, a check passes), so this is
			// not-ready rather than an error.
			name:     "mergeable_state blocked rejected by GitHub is not ready",
			prNumber: 206,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(206)).
					Return(&github.PullRequest{
						Number:         github.Ptr(206),
						State:          github.Ptr("open"),
						Merged:         github.Ptr(false),
						Mergeable:      github.Ptr(true),
						MergeableState: github.Ptr("blocked"),
						Head:           &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:        github.Ptr("https://github.com/akuity/kargo/pull/206"),
					}, &github.Response{}, nil).Once()

				m.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(206), "",
					mock.AnythingOfType("*github.PullRequestOptions")).
					Return(nil, nil, &github.ErrorResponse{
						Response: &http.Response{StatusCode: http.StatusMethodNotAllowed},
						// Real message observed from GitHub when a required status
						// check has not yet reported.
						Message: `Required status check "ci" is expected.`,
					})
			},
			expectedMerged: false,
		},
		{
			// A disabled merge method is permanent even for a blocked PR, so it must
			// not be mistaken for the rejection above.
			name:      "mergeable_state blocked with disabled merge method is terminal",
			prNumber:  207,
			mergeOpts: &gitprovider.MergePullRequestOpts{MergeMethod: "squash"},
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(207)).
					Return(&github.PullRequest{
						Number:         github.Ptr(207),
						State:          github.Ptr("open"),
						Merged:         github.Ptr(false),
						Mergeable:      github.Ptr(true),
						MergeableState: github.Ptr("blocked"),
						Head:           &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:        github.Ptr("https://github.com/akuity/kargo/pull/207"),
					}, &github.Response{}, nil).Once()

				m.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(207), "",
					mock.AnythingOfType("*github.PullRequestOptions")).
					Return(nil, nil, &github.ErrorResponse{
						Response: &http.Response{StatusCode: http.StatusMethodNotAllowed},
						Message:  "Squash merges are not allowed on this repository.",
					})
			},
			expectError:   true,
			errorContains: "Squash merges are not allowed",
		},
		{
			// A branch that is behind its base is blocked by a strict required
			// check, which a bypass-authorized caller may also override.
			name:     "mergeable_state behind attempts the merge",
			prNumber: 208,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(208)).
					Return(&github.PullRequest{
						Number:         github.Ptr(208),
						State:          github.Ptr("open"),
						Merged:         github.Ptr(false),
						Mergeable:      github.Ptr(true),
						MergeableState: github.Ptr("behind"),
						Head:           &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:        github.Ptr("https://github.com/akuity/kargo/pull/208"),
					}, &github.Response{}, nil).Once()

				m.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(208), "",
					mock.AnythingOfType("*github.PullRequestOptions")).
					Return(nil, nil, &github.ErrorResponse{
						Response: &http.Response{StatusCode: http.StatusMethodNotAllowed},
						Message:  "Base branch was modified. Review and try the merge again.",
					})
			},
			expectedMerged: false,
		},
		{
			name:     "mergeable_state dirty fails with conflict",
			prNumber: 204,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(204)).
					Return(&github.PullRequest{
						Number:         github.Ptr(204),
						State:          github.Ptr("open"),
						Merged:         github.Ptr(false),
						Mergeable:      github.Ptr(false),
						MergeableState: github.Ptr("dirty"),
						Head:           &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:        github.Ptr("https://github.com/akuity/kargo/pull/204"),
					}, &github.Response{}, nil)
			},
			expectError:   true,
			errorContains: "has conflicts and cannot be merged",
		},
		{
			name:     "merge call fails",
			prNumber: 555,
			setupMock: func(m *mockGithubClient) {
				// Get PR first
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(555)).
					Return(&github.PullRequest{
						Number:    github.Ptr(555),
						State:     github.Ptr("open"),
						Merged:    github.Ptr(false),
						Mergeable: github.Ptr(true),
						Head:      &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:   github.Ptr("https://github.com/akuity/kargo/pull/555"),
					}, &github.Response{}, nil).Once()

				// Merge call fails
				m.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(555), "",
					mock.AnythingOfType("*github.PullRequestOptions")).
					Return(nil, nil, errors.New("merge failed"))
			},
			expectError:   true,
			errorContains: "error merging pull request",
		},
		{
			name:     "merge call returns 405 base branch modified is not ready",
			prNumber: 405,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(405)).
					Return(&github.PullRequest{
						Number:    github.Ptr(405),
						State:     github.Ptr("open"),
						Merged:    github.Ptr(false),
						Mergeable: github.Ptr(true),
						Head:      &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:   github.Ptr("https://github.com/akuity/kargo/pull/405"),
					}, &github.Response{}, nil).Once()

				// The base branch moved between the mergeability check and this
				// merge call; GitHub returns a 405. The provider treats it as
				// not-ready (no error, not merged) so the caller retries.
				m.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(405), "",
					mock.AnythingOfType("*github.PullRequestOptions")).
					Return(nil, nil, &github.ErrorResponse{
						Response: &http.Response{StatusCode: http.StatusMethodNotAllowed},
						Message:  "Base branch was modified. Review and try the merge again.",
					})
			},
			expectedMerged: false,
		},
		{
			name:     "merge call returns 405 for disabled merge method is terminal",
			prNumber: 406,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(406)).
					Return(&github.PullRequest{
						Number:    github.Ptr(406),
						State:     github.Ptr("open"),
						Merged:    github.Ptr(false),
						Mergeable: github.Ptr(true),
						Head:      &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:   github.Ptr("https://github.com/akuity/kargo/pull/406"),
					}, &github.Response{}, nil).Once()

				// The PR is mergeable, but the configured merge method is disabled
				// on the repo. GitHub returns a 405 with a different message. This
				// is permanent: the provider must surface it as an error rather
				// than treating it as not-ready (which would loop forever under
				// wait=true).
				m.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(406), "",
					mock.AnythingOfType("*github.PullRequestOptions")).
					Return(nil, nil, &github.ErrorResponse{
						Response: &http.Response{StatusCode: http.StatusMethodNotAllowed},
						// Real message observed from GitHub when the configured
						// merge method is disabled on the repo.
						Message: "Squash merges are not allowed on this repository.",
					})
			},
			expectError:   true,
			errorContains: "Squash merges are not allowed",
		},
		{
			name:     "merge call returns 405 not mergeable is not ready",
			prNumber: 407,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(407)).
					Return(&github.PullRequest{
						Number:         github.Ptr(407),
						State:          github.Ptr("open"),
						Merged:         github.Ptr(false),
						Mergeable:      github.Ptr(true),
						MergeableState: github.Ptr("clean"),
						Head:           &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:        github.Ptr("https://github.com/akuity/kargo/pull/407"),
					}, &github.Response{}, nil).Once()

				// GitHub's own view of the PR was stale or has since changed. Its
				// generic rejection is recomputed as checks report and reviews land,
				// so the provider treats it as not-ready rather than as an error.
				m.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(407), "",
					mock.AnythingOfType("*github.PullRequestOptions")).
					Return(nil, nil, &github.ErrorResponse{
						Response: &http.Response{StatusCode: http.StatusMethodNotAllowed},
						Message:  "Pull Request is not mergeable",
					})
			},
			expectedMerged: false,
		},
		{
			name:     "nil merge result",
			prNumber: 333,
			setupMock: func(m *mockGithubClient) {
				// Get PR first
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(333)).
					Return(&github.PullRequest{
						Number:    github.Ptr(333),
						State:     github.Ptr("open"),
						Merged:    github.Ptr(false),
						Mergeable: github.Ptr(true),
						Head:      &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:   github.Ptr("https://github.com/akuity/kargo/pull/333"),
					}, &github.Response{}, nil).Once()

				// Merge returns nil result
				m.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(333), "",
					mock.AnythingOfType("*github.PullRequestOptions")).
					Return(nil, &github.Response{}, nil)
			},
			expectError:   true,
			errorContains: "unexpected nil merge result",
		},
		{
			name:     "get PR after merge fails",
			prNumber: 666,
			setupMock: func(m *mockGithubClient) {
				// First Get PR returns mergeable
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(666)).
					Return(&github.PullRequest{
						Number:    github.Ptr(666),
						State:     github.Ptr("open"),
						Merged:    github.Ptr(false),
						Mergeable: github.Ptr(true),
						Head:      &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:   github.Ptr("https://github.com/akuity/kargo/pull/666"),
					}, &github.Response{}, nil).Once()

				// Merge succeeds
				m.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(666), "",
					mock.AnythingOfType("*github.PullRequestOptions")).
					Return(&github.PullRequestMergeResult{
						SHA:     github.Ptr("merge_sha"),
						Merged:  github.Ptr(true),
						Message: github.Ptr("Pull Request successfully merged"),
					}, &github.Response{}, nil)

				// Second Get PR fails
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(666)).
					Return(nil, nil, errors.New("get PR failed")).Once()
			},
			expectError:   true,
			errorContains: "error getting pull request 666 after merge",
		},
		{
			name:     "nil PR returned after merge",
			prNumber: 888,
			setupMock: func(m *mockGithubClient) {
				// First Get PR returns mergeable
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(888)).
					Return(&github.PullRequest{
						Number:    github.Ptr(888),
						State:     github.Ptr("open"),
						Merged:    github.Ptr(false),
						Mergeable: github.Ptr(true),
						Head:      &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:   github.Ptr("https://github.com/akuity/kargo/pull/888"),
					}, &github.Response{}, nil).Once()

				// Merge succeeds
				m.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(888), "",
					mock.AnythingOfType("*github.PullRequestOptions")).
					Return(&github.PullRequestMergeResult{
						SHA:     github.Ptr("merge_sha"),
						Merged:  github.Ptr(true),
						Message: github.Ptr("Pull Request successfully merged"),
					}, &github.Response{}, nil)

				// Second Get PR returns nil
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(888)).
					Return(nil, &github.Response{}, nil).Once()
			},
			expectError:   true,
			errorContains: "unexpected nil pull request after merge",
		},
		{
			name:     "successful merge",
			prNumber: 777,
			setupMock: func(m *mockGithubClient) {
				// First Get PR
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(777)).
					Return(&github.PullRequest{
						Number:    github.Ptr(777),
						State:     github.Ptr("open"),
						Merged:    github.Ptr(false),
						Mergeable: github.Ptr(true),
						Head:      &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:   github.Ptr("https://github.com/akuity/kargo/pull/777"),
					}, &github.Response{}, nil).Once()

				// Merge
				m.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(777), "",
					mock.AnythingOfType("*github.PullRequestOptions")).
					Return(&github.PullRequestMergeResult{
						SHA:     github.Ptr("merge_sha"),
						Merged:  github.Ptr(true),
						Message: github.Ptr("Pull Request successfully merged"),
					}, &github.Response{}, nil)

				// Second Get PR returns merged
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(777)).
					Return(&github.PullRequest{
						Number:         github.Ptr(777),
						State:          github.Ptr("closed"),
						Merged:         github.Ptr(true),
						MergeCommitSHA: github.Ptr("merge_sha"),
						Head:           &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:        github.Ptr("https://github.com/akuity/kargo/pull/777"),
						MergedAt:       &github.Timestamp{Time: time.Now()},
					}, &github.Response{}, nil).Once()
			},
			expectedMerged: true,
		},
		{
			name:      "successful merge with explicit method",
			prNumber:  100,
			mergeOpts: &gitprovider.MergePullRequestOpts{MergeMethod: "squash"},
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(100)).
					Return(&github.PullRequest{
						Number:    github.Ptr(100),
						State:     github.Ptr("open"),
						Merged:    github.Ptr(false),
						Mergeable: github.Ptr(true),
						Head:      &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:   github.Ptr("https://github.com/akuity/kargo/pull/100"),
					}, &github.Response{}, nil).Once()

				m.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(100), "",
					mock.MatchedBy(func(opts *github.PullRequestOptions) bool {
						return opts.MergeMethod == "squash"
					})).
					Return(&github.PullRequestMergeResult{
						SHA:     github.Ptr("squash_sha"),
						Merged:  github.Ptr(true),
						Message: github.Ptr("Pull Request successfully merged"),
					}, &github.Response{}, nil)

				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(100)).
					Return(&github.PullRequest{
						Number:         github.Ptr(100),
						State:          github.Ptr("closed"),
						Merged:         github.Ptr(true),
						MergeCommitSHA: github.Ptr("squash_sha"),
						Head:           &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:        github.Ptr("https://github.com/akuity/kargo/pull/100"),
						MergedAt:       &github.Timestamp{Time: time.Now()},
					}, &github.Response{}, nil).Once()
			},
			expectedMerged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockGithubClient{}
			p := provider{
				owner:  testRepoOwner,
				repo:   testRepoName,
				client: mockClient,
			}

			tt.setupMock(mockClient)

			pr, merged, err := p.MergePullRequest(
				t.Context(),
				tt.prNumber,
				tt.mergeOpts,
			)
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

// TestMergePullRequestNotMergeableThenDirty is a sanity check that isTransientMerge405
// handles merge conflict PRs properly. A conflict can reach the merge call
// behind a mergeable_state that GitHub has not finished recomputing, and the first attempt reports
// not-ready, but the caller re-reads the PR on the next attempt. So the conflict then fails
// terminally instead of being retried indefinitely.
func TestMergePullRequestNotMergeableThenDirty(t *testing.T) {
	mockClient := &mockGithubClient{}
	p := provider{
		owner:  testRepoOwner,
		repo:   testRepoName,
		client: mockClient,
	}

	// First attempt: GitHub still reports the state it computed before the
	// conflicting change landed, so the gate proceeds to the merge call, which
	// GitHub rejects.
	mockClient.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(500)).
		Return(&github.PullRequest{
			Number:         new(500),
			State:          new("open"),
			Merged:         new(false),
			Mergeable:      new(true),
			MergeableState: new("clean"),
			Head:           &github.PullRequestBranch{SHA: new("head_sha")},
			HTMLURL:        new("https://github.com/akuity/kargo/pull/500"),
		}, &github.Response{}, nil).Once()
	mockClient.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(500), "",
		mock.AnythingOfType("*github.PullRequestOptions")).
		Return(nil, nil, &github.ErrorResponse{
			Response: &http.Response{StatusCode: http.StatusMethodNotAllowed},
			Message:  "Pull Request is not mergeable",
		}).Once()

	pr, merged, err := p.MergePullRequest(t.Context(), 500, nil)
	require.NoError(t, err)
	require.False(t, merged)
	require.Nil(t, pr)

	// Second attempt: GitHub has finished recomputing and reports the conflict.
	mockClient.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(500)).
		Return(&github.PullRequest{
			Number:         new(500),
			State:          new("open"),
			Merged:         new(false),
			Mergeable:      new(false),
			MergeableState: new("dirty"),
			Head:           &github.PullRequestBranch{SHA: new("head_sha")},
			HTMLURL:        new("https://github.com/akuity/kargo/pull/500"),
		}, &github.Response{}, nil).Once()

	pr, merged, err = p.MergePullRequest(t.Context(), 500, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "has conflicts and cannot be merged")
	require.False(t, merged)
	require.Nil(t, pr)

	// The conflict must be caught by the gate, without a second merge attempt.
	mockClient.AssertExpectations(t)
	mockClient.AssertNumberOfCalls(t, "MergePullRequest", 1)
}

func TestIsTransientMerge405(t *testing.T) {
	tests := []struct {
		name          string
		msg           string
		policyBlocked bool
		expected      bool
	}{
		{
			name:     "base branch modified",
			msg:      "Base branch was modified. Review and try the merge again.",
			expected: true,
		},
		{
			name:     "not mergeable",
			msg:      "Pull Request is not mergeable",
			expected: true,
		},
		{
			name:          "disabled squash merges",
			msg:           "Squash merges are not allowed on this repository",
			policyBlocked: true,
			expected:      false,
		},
		{
			name:          "disabled merge commits",
			msg:           "Merge commits are not allowed on this repository.",
			policyBlocked: true,
			expected:      false,
		},
		{
			name:          "disabled rebase merges",
			msg:           "Rebase merges are not allowed",
			policyBlocked: true,
			expected:      false,
		},
		{
			// A ruleset restricting the merge methods permitted for the branch
			// words this differently from the repository-level settings above.
			name:          "merge method restricted by a ruleset",
			msg:           "The selected merge method (squash) is not allowed",
			policyBlocked: true,
			expected:      false,
		},
		{
			name:          "unsatisfied branch protection rule",
			msg:           `Required status check "ci" is expected.`,
			policyBlocked: true,
			expected:      true,
		},
		{
			name:     "unrecognized message without a policy block",
			msg:      `Required status check "ci" is expected.`,
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, isTransientMerge405(tt.msg, tt.policyBlocked))
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
			repoURL:           "ssh://git@github.com/akuity/kargo.git",
			sha:               "sha",
			expectedCommitURL: "https://github.com/akuity/kargo/commit/sha",
		},
		{
			repoURL:           "git@github.com:akuity/kargo.git",
			sha:               "sha",
			expectedCommitURL: "https://github.com/akuity/kargo/commit/sha",
		},
		{
			repoURL:           "https://username@github.com/akuity/kargo",
			sha:               "sha",
			expectedCommitURL: "https://github.com/akuity/kargo/commit/sha",
		},
		{
			repoURL:           "http://github.com/akuity/kargo.git",
			sha:               "sha",
			expectedCommitURL: "https://github.com/akuity/kargo/commit/sha",
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
