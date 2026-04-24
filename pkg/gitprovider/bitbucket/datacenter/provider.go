package datacenter

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/akuity/kargo/pkg/gitprovider"
)

const (
	// ProviderName is the name used to register the Bitbucket Data Center provider.
	ProviderName = "bitbucket-datacenter"
)

var registration = gitprovider.Registration{
	Predicate: func(repoURL string) bool {
		u, err := url.Parse(repoURL)
		if err != nil {
			return false
		}
		// We assume that any hostname containing "bitbucket" that is not
		// bitbucket.org is a self-hosted Data Center instance. Instances that
		// don't include "bitbucket" in their hostname won't be auto-detected and
		// will require explicit provider configuration via opts.Name.
		host := u.Hostname()
		return strings.Contains(host, "bitbucket") && host != "bitbucket.org"
	},
	NewProvider: func(
		repoURL string,
		opts *gitprovider.Options,
	) (gitprovider.Interface, error) {
		return NewProvider(repoURL, opts)
	},
}

func init() {
	gitprovider.Register(ProviderName, registration)
}

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
