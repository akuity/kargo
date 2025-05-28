package external

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

func GenerateWebhookPath(project, provider, token string) string {
	input := []byte(project + token)
	h := sha256.New()
	h.Write(input)
	return fmt.Sprintf("/webhook/%s/%x", strings.ToLower(provider), h.Sum(nil))
}
