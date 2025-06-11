package image

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	DefaultDockerHost      = "docker.io"
	DefaultDockerNamespace = "library"
	DefaultDockerTag       = "latest"
)

var (
	RepoNameComponentRegexp = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*$`)
)

// NormalizeRef normalizes Docker image references of the following forms:
//
//   - [docker.io/][namespace/]repo[:tag]
//   - [docker.io/][namespace/]repo[@digest]
//
// This is useful for the purposes of comparison and also in cases where a
// canonical representation of a Docker Hub image reference is needed. Any reference
// that cannot be normalized will return an error.
//
// Examples:
//
//	"nginx"                    -> "docker.io/library/nginx:latest"
//	"user/repo:v1.0"           -> "docker.io/user/repo:v1.0"
//	"docker.io/library/nginx"  -> "docker.io/library/nginx:latest"
//	"nginx@sha256:..."         -> "docker.io/library/nginx@sha256:..."
func NormalizeRef(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", errors.New("empty image reference")
	}
	ref = strings.ToLower(ref)

	// Remove leading docker.io/ if present
	if strings.HasPrefix(ref, DefaultDockerHost+"/") {
		ref = strings.TrimPrefix(ref, DefaultDockerHost+"/")
	}

	// Extract digest if present
	var digest string
	if at := strings.LastIndex(ref, "@"); at != -1 {
		digest = ref[at:]
		ref = ref[:at]
		matched, _ := regexp.MatchString(`^@sha256:[a-f0-9]{64}$`, digest)
		if !matched {
			return "", fmt.Errorf("invalid digest format: %q", digest)
		}
	}

	// Extract tag if present (only if no digest)
	var tag string
	if digest == "" {
		if colon := strings.LastIndex(ref, ":"); colon != -1 && colon > strings.LastIndex(ref, "/") {
			tag = ref[colon+1:]
			ref = ref[:colon]
			if tag == "" {
				return "", errors.New("image reference has a colon but no tag")
			}
		} else {
			tag = DefaultDockerTag
		}
	}

	// Normalize path: always at least namespace/repo
	parts := strings.Split(ref, "/")
	if len(parts) == 1 {
		parts = []string{DefaultDockerNamespace, parts[0]}
	}
	for _, part := range parts {
		if !RepoNameComponentRegexp.MatchString(part) {
			return "", fmt.Errorf("invalid repository name component: %q", part)
		}
	}
	path := strings.Join(parts, "/")

	if digest != "" {
		return fmt.Sprintf("%s/%s%s", DefaultDockerHost, path, digest), nil
	}
	return fmt.Sprintf("%s/%s:%s", DefaultDockerHost, path, tag), nil
}
