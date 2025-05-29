package external

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// GenerateWebhookPath generates a unique path for a webhook based on the
// provided webhook receiver name, project name, kind, and token.
// The path is formatted as "/webhook/{kind}/{hash}" where the hash is
// a SHA-256 hash of the concatenated webhook receiver name, project name, and token.
func GenerateWebhookPath(name, project, kind, token string) string {
	input := []byte(name + project + token)
	h := sha256.New()
	h.Write(input)
	return fmt.Sprintf("/webhook/%s/%x", strings.ToLower(kind), h.Sum(nil))
}
