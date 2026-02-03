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
	if hasInternalSpaces(repo) {
		return repo
	}
	origRepo := repo
	repo = rmSpaces(origRepo)
	if repo == "" {
		return origRepo
	}

	if hasProtocolPrefix(repo) {
		repoURL, err := safeParseURL(repo)
		if err != nil {
			return origRepo
		}
		// Remove user info for HTTP/S URLs
		if !strings.HasPrefix(repo, "ssh://") {
			repoURL.User = nil
		}
		repoURL.Path = strings.TrimSuffix(repoURL.Path, "/")
		repoURL.Path = strings.TrimSuffix(repoURL.Path, ".git")
		return strings.ToLower(repoURL.String())
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
		return strings.ToLower(fmt.Sprintf("ssh://%s", userHost))
	}
	return strings.ToLower(fmt.Sprintf("ssh://%s/%s", userHost, pathURL.String()))
}

func hasProtocolPrefix(repo string) bool {
	return strings.HasPrefix(repo, "http://") ||
		strings.HasPrefix(repo, "https://") ||
		strings.HasPrefix(repo, "ssh://")
}

func safeParseURL(repo string) (*url.URL, error) {
	repoURL, err := url.Parse(repo)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}
	if len(repoURL.Query()) > 0 {
		// Query parameters are not permitted
		return nil, fmt.Errorf("URL contains %d query parameters; not permitted", len(repoURL.Query()))
	}
	return repoURL, nil
}

// rmSpaces removes all leading, trailing, and internal whitespace characters
// from the given repository string. It also decodes any percent-encoded
// characters in the string before processing.
func rmSpaces(repo string) string {
	return strings.Map(rmRuneFuncfunc, repo)
}

// hasInternalSpaces checks if the given repository URL string contains any
// any non-leading or non-trailing whitespace characters.
func hasInternalSpaces(repo string) bool {
	// First remove leading and trailing spaces and use this to compare.
	// trimmed := strings.TrimSpace(repo)
	// Remove unusual whitespace characters that strings.TrimSpace doesn't remove.
	trimmed := trimSpace(repo)
	return strings.Map(rmRuneFuncfunc, trimmed) != trimmed
}

func trimSpace(repo string) string {
	unespaced, err := url.PathUnescape(repo)
	if err == nil {
		repo = unespaced
	}
	return strings.TrimRightFunc(
		strings.TrimLeftFunc(repo, isSpace),
		isSpace,
	)
}

func isSpace(r rune) bool {
	return unicode.IsSpace(r) || !unicode.IsPrint(r)
}

func rmRuneFuncfunc(r rune) rune {
	if isSpace(r) {
		return -1 // Remove the character
	}
	return r
}
