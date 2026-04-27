package github

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/go-github/v76/github"
	"github.com/hashicorp/go-cleanhttp"

	"github.com/akuity/kargo/pkg/urls"
)

// ClientOptions holds the configuration for creating a GitHub API client.
type ClientOptions struct {
	// Token is an authentication token for the GitHub API. If empty, the
	// client will be unauthenticated.
	Token string
	// InsecureSkipTLSVerify controls whether the client skips TLS certificate
	// verification. Intended for GitHub Enterprise instances with self-signed
	// certificates.
	InsecureSkipTLSVerify bool
}

// NewClient creates a GitHub API client configured for the given repository
// URL. For github.com, a standard client is returned. For GitHub Enterprise
// hosts, the client is configured with the appropriate enterprise base URLs.
func NewClient(
	repoURL string,
	opts *ClientOptions,
) (*github.Client, error) {
	if opts == nil {
		opts = &ClientOptions{}
	}

	scheme, host, _, _, err := ParseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}

	httpClient := cleanhttp.DefaultClient()
	if opts.InsecureSkipTLSVerify {
		transport := cleanhttp.DefaultTransport()
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
		httpClient.Transport = transport
	}

	client := github.NewClient(httpClient)

	if host != "github.com" {
		baseURL := fmt.Sprintf("%s://%s", scheme, host)
		// This function call will automatically add correct paths to the
		// base URL.
		client, err = client.WithEnterpriseURLs(baseURL, baseURL)
		if err != nil {
			return nil, err
		}
	}
	if opts.Token != "" {
		client = client.WithAuthToken(opts.Token)
	}

	return client, nil
}

// ParseRepoURL parses a Git repository URL and extracts the URL scheme, host,
// repository owner, and repository name. It handles standard HTTPS URLs, SSH
// URLs, and GitHub Enterprise URLs.
func ParseRepoURL(
	repoURL string,
) (string, string, string, string, error) {
	repoURL = urls.NormalizeGit(repoURL)
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", "", "", "", fmt.Errorf(
			"error parsing github repository URL %q: %w", repoURL, err,
		)
	}

	scheme := u.Scheme
	if scheme != "https" && scheme != "http" {
		scheme = "https"
	}

	path := strings.TrimPrefix(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return "", "", "", "", fmt.Errorf(
			"could not extract repository owner and name from URL %q", u,
		)
	}

	return scheme, u.Host, parts[0], parts[1], nil
}
