package image

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

// NormalizeRef returns a short, canonical Docker reference for comparison and display.
// E.g. "docker.io/library/nginx:latest" -> "nginx"
func NormalizeURL(repoURL string) (string, error) {
	parsed, err := name.ParseReference(repoURL, name.WeakValidation)
	if err != nil {
		return "", fmt.Errorf("invalid image reference: %w", err)
	}

	reg := parsed.Context().Registry.Name()
	repo := parsed.Context().RepositoryStr()

	// Normalize registry: drop docker.io/index.docker.io
	if reg == "docker.io" || reg == "index.docker.io" {
		reg = ""
		// Drop "library/" prefix for official images
		repo = strings.TrimPrefix(repo, "library/")
	}

	// Compose normalized reference (no tag or digest)
	var sb strings.Builder
	if reg != "" {
		sb.WriteString(reg)
		sb.WriteString("/")
	}
	sb.WriteString(repo)
	return sb.String(), nil
}
