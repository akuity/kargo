package datacenter

import (
	"context"
	"fmt"

	"github.com/akuity/kargo/pkg/gitprovider"
)

// provider is a Bitbucket Data Center implementation of gitprovider.Interface.
type provider struct{}

// NewProvider returns a Bitbucket Data Center implementation of gitprovider.Interface.
func NewProvider(
	_ string,
	_ *gitprovider.Options,
) (gitprovider.Interface, error) {
	// TODO: Implement Bitbucket Data Center provider.
	return &provider{}, nil
}

func (p *provider) CreatePullRequest(
	_ context.Context,
	_ *gitprovider.CreatePullRequestOpts,
) (*gitprovider.PullRequest, error) {
	return nil, fmt.Errorf("Bitbucket Data Center provider is not yet implemented")
}

func (p *provider) GetPullRequest(
	_ context.Context,
	_ int64,
) (*gitprovider.PullRequest, error) {
	return nil, fmt.Errorf("Bitbucket Data Center provider is not yet implemented")
}

func (p *provider) ListPullRequests(
	_ context.Context,
	_ *gitprovider.ListPullRequestOptions,
) ([]gitprovider.PullRequest, error) {
	return nil, fmt.Errorf("Bitbucket Data Center provider is not yet implemented")
}

func (p *provider) MergePullRequest(
	_ context.Context,
	_ int64,
	_ *gitprovider.MergePullRequestOpts,
) (*gitprovider.PullRequest, bool, error) {
	return nil, false, fmt.Errorf("Bitbucket Data Center provider is not yet implemented")
}

func (p *provider) GetCommitURL(_ string, _ string) (string, error) {
	return "", fmt.Errorf("Bitbucket Data Center provider is not yet implemented")
}
