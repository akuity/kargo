package bitbucket

import (
	"context"
	"testing"

	"github.com/akuity/kargo/internal/gitprovider"
	"github.com/ktrysmt/go-bitbucket"
	"github.com/stretchr/testify/assert"
)

type mockPullRequestClient struct {
	createPullRequestFunc func(opt *bitbucket.PullRequestsOptions) (any, error)
	listPullRequestsFunc  func(opt *bitbucket.PullRequestsOptions) (any, error)
	getPullRequestFunc    func(opt *bitbucket.PullRequestsOptions) (any, error)
}

func (m *mockPullRequestClient) CreatePullRequest(opt *bitbucket.PullRequestsOptions) (any, error) {
	return m.createPullRequestFunc(opt)
}

func (m *mockPullRequestClient) ListPullRequests(opt *bitbucket.PullRequestsOptions) (any, error) {
	return m.listPullRequestsFunc(opt)
}

func (m *mockPullRequestClient) GetPullRequest(opt *bitbucket.PullRequestsOptions) (any, error) {
	return m.getPullRequestFunc(opt)
}

func TestNewProvider(t *testing.T) {
	provider, err := NewProvider("https://bitbucket.org/owner/repo", &gitprovider.Options{Token: "token"})
	assert.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestCreatePullRequest(t *testing.T) {
	mockClient := &mockPullRequestClient{
		createPullRequestFunc: func(opt *bitbucket.PullRequestsOptions) (any, error) {
			return map[string]any{"id": int64(1), "state": prStateOpen}, nil
		},
	}
	provider := &provider{
		owner:    "owner",
		repoSlug: "repo",
		client:   mockClient,
	}

	ctx := context.Background()
	opts := &gitprovider.CreatePullRequestOpts{
		Title: "Test PR",
		Head:  "feature-branch",
		Base:  "main",
	}
	pr, err := provider.CreatePullRequest(ctx, opts)
	assert.NoError(t, err)
	assert.NotNil(t, pr)
	assert.Equal(t, int64(1), pr.Number)
}

func TestGetPullRequest(t *testing.T) {
	mockClient := &mockPullRequestClient{
		getPullRequestFunc: func(opt *bitbucket.PullRequestsOptions) (any, error) {
			return map[string]any{"id": int64(1), "state": prStateOpen}, nil
		},
	}
	provider := &provider{
		owner:    "owner",
		repoSlug: "repo",
		client:   mockClient,
	}

	ctx := context.Background()
	pr, err := provider.GetPullRequest(ctx, 1)
	assert.NoError(t, err)
	assert.NotNil(t, pr)
	assert.Equal(t, int64(1), pr.Number)
}

func TestListPullRequests(t *testing.T) {
	mockClient := &mockPullRequestClient{
		listPullRequestsFunc: func(opt *bitbucket.PullRequestsOptions) (any, error) {
			return map[string]any{"values": []any{
				map[string]any{"id": int64(1), "state": prStateOpen},
				map[string]any{"id": int64(2), "state": prStateMerged},
			}}, nil
		},
	}
	provider := &provider{
		owner:    "owner",
		repoSlug: "repo",
		client:   mockClient,
	}

	ctx := context.Background()
	prs, err := provider.ListPullRequests(ctx, &gitprovider.ListPullRequestOptions{State: gitprovider.PullRequestStateAny})
	assert.NoError(t, err)
	assert.Len(t, prs, 2)
	assert.Equal(t, int64(1), prs[0].Number)
	assert.Equal(t, int64(2), prs[1].Number)
}
