package bitbucket

import (
	"fmt"
	"net/url"

	"github.com/akuity/kargo/pkg/gitprovider"
	"github.com/akuity/kargo/pkg/gitprovider/bitbucket/cloud"
	"github.com/akuity/kargo/pkg/gitprovider/bitbucket/datacenter"
	"github.com/akuity/kargo/pkg/urls"
)

const ProviderName = "bitbucket"

var registration = gitprovider.Registration{
	Predicate: func(repoURL string) bool {
		u, err := url.Parse(repoURL)
		if err != nil {
			return false
		}
		// For now, only Cloud is supported. Data Center support is forthcoming.
		return u.Hostname() == cloud.Host
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

// NewProvider returns a Bitbucket Cloud or Data Center implementation of
// gitprovider.Interface based on the repository URL host.
func NewProvider(
	repoURL string,
	opts *gitprovider.Options,
) (gitprovider.Interface, error) {
	u, err := url.Parse(urls.NormalizeGit(repoURL))
	if err != nil {
		return nil, fmt.Errorf("parse Bitbucket URL %q: %w", repoURL, err)
	}
	if u.Hostname() == cloud.Host {
		return cloud.NewProvider(repoURL, opts)
	}
	return datacenter.NewProvider(repoURL, opts)
}
