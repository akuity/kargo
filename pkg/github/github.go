package github

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"

	ghlib "github.com/google/go-github/v76/github"
	"github.com/hashicorp/go-cleanhttp"

	"github.com/akuity/kargo/pkg/urls"
)

// ParseRepoURL extracts the scheme, host, owner, and repo name from a Git
// repository URL.
func ParseRepoURL(
	repoURL string,
) (scheme, host, owner, repo string, err error) {
	u, err := url.Parse(urls.NormalizeGit(repoURL))
	if err != nil {
		return "", "", "", "",
			fmt.Errorf("error parsing repository URL %q: %w", repoURL, err)
	}
	scheme = u.Scheme
	if scheme != "https" && scheme != "http" {
		scheme = "https"
	}
	host = u.Host
	path := strings.TrimPrefix(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return "", "", "", "", fmt.Errorf(
			"could not extract repository owner and name from URL %q",
			repoURL,
		)
	}
	return scheme, host, parts[0], parts[1], nil
}

// NewClient creates an authenticated GitHub API client for the given
// repository. It returns the owner and repo name extracted from the URL.
func NewClient(
	repoURL, token string,
	insecureSkipTLSVerify bool,
) (*ghlib.Client, string, string, error) {
	scheme, host, owner, repo, err := ParseRepoURL(repoURL)
	if err != nil {
		return nil, "", "", err
	}

	httpClient := cleanhttp.DefaultClient()
	if insecureSkipTLSVerify {
		transport := cleanhttp.DefaultTransport()
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
		httpClient.Transport = transport
	}

	client := ghlib.NewClient(httpClient)
	if host != "github.com" {
		baseURL := fmt.Sprintf("%s://%s", scheme, host)
		client, err = client.WithEnterpriseURLs(baseURL, baseURL)
		if err != nil {
			return nil, "", "", fmt.Errorf(
				"error configuring GitHub Enterprise URLs: %w", err,
			)
		}
	}
	if token != "" {
		client = client.WithAuthToken(token)
	}

	return client, owner, repo, nil
}

// BuildCommitURL constructs a human-readable commit URL from a repository URL
// and commit SHA.
func BuildCommitURL(repoURL, sha string) string {
	_, host, _, _, err := ParseRepoURL(repoURL)
	if err != nil {
		return ""
	}
	u, err := url.Parse(urls.NormalizeGit(repoURL))
	if err != nil {
		return ""
	}
	return fmt.Sprintf("https://%s%s/commit/%s", host, u.Path, sha)
}
