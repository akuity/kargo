package image

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

// NormalizeURL normalizes image repository URLs. Notably, hostnames docker.io
// and index.docker.io, if present, are dropped. The optional /library prefix
// for official images from Docker Hub, if included, is also dropped. Valid,
// non-Docker Hub repository URLs will be returned unchanged.
//
// This is useful for the purposes of comparison and also in cases where a
// canonical representation of a repository URL is needed. Any URL that cannot
// be normalized will be returned as-is.
func NormalizeURL(repoURL string) string {
	parsed, err := name.ParseReference(repoURL, name.WeakValidation)
	if err != nil {
		return repoURL
	}
	reg := parsed.Context().Registry.Name()
	repo := parsed.Context().RepositoryStr()
	// For all images from Docker Hub, reg will be "index.docker.io" after parsing,
	// except when the input explicitly uses "registry-1.docker.io", which is preserved.
	if reg == "index.docker.io" || reg == "registry-1.docker.io" {
		return strings.TrimPrefix(repo, "library/")
	}
	return fmt.Sprintf("%s/%s", reg, repo)
}
