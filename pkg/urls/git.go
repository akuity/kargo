package urls

import (
	"net/url"
	"strings"
)

// NormalizeGit normalizes Git URLs of the following form:
//
//   - http[s]://[proxy-user:proxy-pass@]host.xz[:port][/path/to/repo[.git][/]]
//
// This is useful for the purposes of comparison and also in cases where a
// canonical representation of a Git URL is needed. Any URL that cannot be
// normalized will be returned as-is.
func NormalizeGit(repo string) string {
	origRepo := repo
	repo = SanitizeURL(strings.ToLower(repo))

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

	return origRepo
}
