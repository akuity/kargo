package external

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const testSigningKey = "mysupersecrettoken"

func sign(content []byte) string {
	mac := hmac.New(sha256.New, []byte(testSigningKey))
	_, _ = mac.Write(content)
	return fmt.Sprintf("sha256=%s",
		hex.EncodeToString(mac.Sum(nil)),
	)
}
