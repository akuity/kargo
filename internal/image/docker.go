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
	tag := ""
	digest := ""

	// Extract tag or digest
	switch r := parsed.(type) {
	case name.Tag:
		tag = r.TagStr()
	case name.Digest:
		digest = r.DigestStr()
	}

	// Normalize registry: drop docker.io/index.docker.io
	if reg == "docker.io" || reg == "index.docker.io" {
		reg = ""
		// Drop "library/" prefix for official images
		repo = strings.TrimPrefix(repo, "library/")
	}

	// Compose normalized reference
	var sb strings.Builder
	if reg != "" {
		sb.WriteString(reg)
		sb.WriteString("/")
	}
	sb.WriteString(repo)
	if tag != "" && tag != "latest" {
		sb.WriteString(":")
		sb.WriteString(tag)
	}
	if digest != "" {
		sb.WriteString("@")
		sb.WriteString(digest)
	}
	return sb.String(), nil
}
