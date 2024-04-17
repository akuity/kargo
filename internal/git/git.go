package git

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// https://regex101.com/r/bJCECT/1
var sshURLRegex = regexp.MustCompile(`^(?:ssh://)?((?:[\w-]+@)[\w-]+(?:\.[\w-]+)*(?::\d+)?)(?::(.*))?`)

// NormalizeGitURL normalizes a git URL for purposes of comparison.
func NormalizeGitURL(repo string) string {
	origRepo := repo
	repo = strings.ToLower(repo)
	matches := sshURLRegex.FindStringSubmatch(repo)
	if len(matches) > 2 { // An ssh URL
		userHost := strings.TrimPrefix(matches[1], "ssh://")
		var path string
		if len(matches) == 3 {
			path = matches[2]
		}
		pathURL, err := url.Parse(path)
		if err != nil {
			panic(fmt.Errorf("error normalizing ssh URL %s: %w", origRepo, err))
		}
		pathURL.Path = strings.TrimSuffix(pathURL.Path, "/")
		pathURL.Path = strings.TrimSuffix(pathURL.Path, ".git")
		if pathURL.Path == "" {
			return userHost
		}
		return fmt.Sprintf("%s:%s", userHost, pathURL.String())
	}
	repoURL, err := url.Parse(repo)
	if err != nil {
		panic(fmt.Errorf("error normalizing http/s URL %s: %w", origRepo, err))
	}
	repoURL.Path = strings.TrimSuffix(repoURL.Path, "/")
	repoURL.Path = strings.TrimSuffix(repoURL.Path, ".git")
	return repoURL.String()
}
