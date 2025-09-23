package urls

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var scpSyntaxRegex = regexp.MustCompile(`^((?:[\w-]+@)?[\w-]+(?:\.[\w-]+)*)(?::(.*))?$`)

// NormalizeGit normalizes Git URLs of the following forms:
//
//   - http[s]://[proxy-user:proxy-pass@]host.xz[:port][/path/to/repo[.git][/]]
//   - ssh://[user@]host.xz[:port][/path/to/repo[.git][/]]
//   - [user@]host.xz[:path/to/repo[.git][/]]
//
// This is useful for the purposes of comparison and also in cases where a
// canonical representation of a Git URL is needed. Any URL that cannot be
// normalized will be returned as-is.
func NormalizeGit(repo string) string {
	origRepo := repo
	repo = strings.ToLower(repo)

	// HTTP/S URLs
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") {
		repoURL, err := url.Parse(repo)
		if err != nil {
			return origRepo
		}
		if len(repoURL.Query()) > 0 {
			// Query parameters are not permitted
			return origRepo
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
			return origRepo
		}
		if len(repoURL.Query()) > 0 {
			// Query parameters are not permitted
			return origRepo
		}
		repoURL.Path = strings.TrimSuffix(repoURL.Path, "/")
		repoURL.Path = strings.TrimSuffix(repoURL.Path, ".git")
		return repoURL.String()
	}

	// URLS of the form [user@]host.xz[:path/to/repo[.git][/]]
	matches := scpSyntaxRegex.FindStringSubmatch(repo)
	if len(matches) != 2 && len(matches) != 3 {
		// This URL doesn't appear to be in a format we recognize
		return origRepo
	}
	userHost := matches[1]
	var path string
	if len(matches) == 3 {
		path = matches[2]
	}
	pathURL, err := url.Parse(path)
	if err != nil {
		return origRepo
	}
	pathURL.Path = strings.TrimSuffix(pathURL.Path, "/")
	pathURL.Path = strings.TrimSuffix(pathURL.Path, ".git")
	if pathURL.Path == "" {
		return fmt.Sprintf("ssh://%s", userHost)
	}
	return fmt.Sprintf("ssh://%s/%s", userHost, pathURL.String())
}
