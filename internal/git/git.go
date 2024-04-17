package git

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var scpSyntaxRegex = regexp.MustCompile(`^((?:[\w-]+@)?[\w-]+(?:\.[\w-]+)*)(?::(.*))?$`)

// NormalizeGitURL normalizes a Git URL for purposes of comparison.
func NormalizeGitURL(repo string) string {
	origRepo := repo
	repo = strings.ToLower(repo)

	// HTTP/S URLs
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") {
		repoURL, err := url.Parse(repo)
		if err != nil {
			panic(fmt.Errorf("error normalizing HTTP/S URL %s: %w", origRepo, err))
		}
		if len(repoURL.Query()) > 0 {
			panic(fmt.Errorf(
				"error normalizing HTTP/S URL %s: query parameters are not permitted",
				origRepo,
			))
		}
		repoURL.User = nil // Remove user info if there is any
		repoURL.Path = strings.TrimSuffix(repoURL.Path, "/")
		repoURL.Path = strings.TrimSuffix(repoURL.Path, ".git")
		return repoURL.String()
	}

	// URLS of the form ssh://[user@]host.xz[:port][/path/to/repo[.git][/]]
	if strings.HasPrefix(repo, "ssh://") {
		// repo = strings.TrimPrefix(repo, "ssh://")
		repoURL, err := url.Parse(repo)
		if err != nil {
			panic(fmt.Errorf("error normalizing SSH URL %s: %w", origRepo, err))
		}
		if len(repoURL.Query()) > 0 {
			panic(fmt.Errorf(
				"error SSH URL %s: query parameters are not permitted",
				origRepo,
			))
		}
		repoURL.Path = strings.TrimSuffix(repoURL.Path, "/")
		repoURL.Path = strings.TrimSuffix(repoURL.Path, ".git")
		return repoURL.String()
	}

	// URLS of the form [user@]host.xz[:path/to/repo[.git][/]]
	matches := scpSyntaxRegex.FindStringSubmatch(repo)
	if len(matches) != 2 && len(matches) != 3 {
		panic(fmt.Errorf(
			"error normalizing URL: %s does not appear to be a valid HTTP/S, SSH, or SCP-style URL",
			origRepo,
		))
	}
	userHost := matches[1]
	var path string
	if len(matches) == 3 {
		path = matches[2]
	}
	pathURL, err := url.Parse(path)
	if err != nil {
		panic(fmt.Errorf("error normalizing SCP-style URL %s: %w", origRepo, err))
	}
	pathURL.Path = strings.TrimSuffix(pathURL.Path, "/")
	pathURL.Path = strings.TrimSuffix(pathURL.Path, ".git")
	if pathURL.Path == "" {
		return fmt.Sprintf("ssh://%s", userHost)
	}
	return fmt.Sprintf("ssh://%s/%s", userHost, pathURL.String())
}
