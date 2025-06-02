package external

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// GenerateWebhookPath generates a unique path for a webhook based on the
// provided webhook receiver name, project name, kind, and secret.
// The path is formatted as "/webhook/{kind}/{hash}" where the hash is
// a SHA-256 hash of the concatenated webhook receiver name, project name, and secret.
func GenerateWebhookPath(name, project, kind, secret string) string {
	// Warning: Changes to this line could alter URLs that existing users may already be using
	input := []byte(name + project + secret)
	h := sha256.New()
	h.Write(input)
	return fmt.Sprintf("/webhook/%s/%x", strings.ToLower(kind), h.Sum(nil))
}
