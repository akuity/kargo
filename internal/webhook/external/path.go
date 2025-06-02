package external

import (
	"crypto/sha256"
	"fmt"
	"strings"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// GenerateWebhookPath generates a unique path for a webhook based on the
// provided webhook receiver name, project name, kind, and secret.
// The path is formatted as "/webhook/{kind}/{hash}" where the hash is
// a SHA-256 hash of the concatenated webhook receiver name, project name,
// and secret.
func GenerateWebhookPath(
	name string,
	project string,
	kind kargoapi.WebhookReceiverType,
	secret string,
) string {
	// Warning: Changes to this line could alter URLs that existing users may
	// already be using
	input := []byte(name + project + secret)
	h := sha256.New()
	_, _ = h.Write(input)
	return fmt.Sprintf("/webhook/%s/%x",
		strings.ToLower(string(kind)), h.Sum(nil),
	)
}
