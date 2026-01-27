package urls

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
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
	repo = strings.TrimSpace(strings.ToLower(repo))
	repo = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '\ufeff' || r == '\u200B' || r == '\u00A0' || r == '\n' || r == '\r' || r == '\t' {
			return -1 // Remove the character
		}
		return r
	}, repo)

	if repo == "" {
		return repo
	}

	// HTTP/S URLs
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") {
		repoURL, err := getURL(repo)
		if err != nil {
			return repo
		}
		repoURL.User = nil // Remove user info if there is any
		repoURL.Path = strings.TrimSuffix(repoURL.Path, "/")
		repoURL.Path = strings.TrimSuffix(repoURL.Path, ".git")
		return repoURL.String()
	}

	// URLS of the form ssh://[user@]host.xz[:port][/path/to/repo[.git][/]]
	if strings.HasPrefix(repo, "ssh://") {
		repoURL, err := getURL(repo)
		if err != nil {
			return repo
		}
		repoURL.Path = strings.TrimSuffix(repoURL.Path, "/")
		repoURL.Path = strings.TrimSuffix(repoURL.Path, ".git")
		return repoURL.String()
	}

	// URLS of the form [user@]host.xz[:path/to/repo[.git][/]]
	matches := scpSyntaxRegex.FindStringSubmatch(repo)
	if len(matches) != 2 && len(matches) != 3 {
		// This URL doesn't appear to be in a format we recognize
		return repo
	}
	userHost := matches[1]
	var path string
	if len(matches) == 3 {
		path = matches[2]
	}
	pathURL, err := url.Parse(path)
	if err != nil {
		return repo
	}
	decodedPath, err := url.PathUnescape(pathURL.Path)
	if err == nil {
		pathURL.Path = decodedPath
	}
	pathURL.Path = strings.TrimSuffix(pathURL.Path, "/")
	pathURL.Path = strings.TrimSuffix(pathURL.Path, ".git")
	if pathURL.Path == "" {
		return fmt.Sprintf("ssh://%s", userHost)
	}
	return fmt.Sprintf("ssh://%s/%s", userHost, pathURL.String())
}

func getURL(repo string) (*url.URL, error) {
	repoURL, err := url.Parse(repo)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}
	if len(repoURL.Query()) > 0 {
		// Query parameters are not permitted
		return nil, fmt.Errorf("URL contains %d query parameters; not permitted", len(repoURL.Query()))
	}
	decodedPath, err := url.PathUnescape(repoURL.Path)
	if err == nil {
		repoURL.Path = decodedPath
	}
	return repoURL, nil
}
