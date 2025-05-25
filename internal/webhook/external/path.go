package external

import (
	"crypto/sha256"
	"fmt"
)

func GenerateWebhookPath(project, provider, token string) string {
	input := []byte(project + provider + token)
	h := sha256.New()
	h.Write(input)
	return fmt.Sprintf("/%x", h.Sum(nil))
}
