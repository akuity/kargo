package external

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/io"
	"github.com/akuity/kargo/internal/logging"
	"github.com/go-playground/webhooks/v6/bitbucket"
	"hash"
	body "io"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	bitbucketEventTypeHeader   = "X-Event-Key"
	bitbucketCloudHookIDHeader = "X-Hook-UUID"
	bitbucketSignatureHeader   = "X-Hub-Signature"
)

// getUUID fetches the hook UUID from the bitbucketCloudHookIDHeader header
func getUUID(r *http.Request) string {
	UUID := r.Header.Get(bitbucketCloudHookIDHeader)
	return UUID
}

// ValidateSignature validates the signature for the given payload.
// signature is the GitHub hash signature delivered in the X-Hub-Signature header.
// payload is the JSON payload sent by Bitbucket Webhooks.
func ValidateSignature(signature string, payload, secretKey []byte) error {
	messageMAC, hashFunc, err := messageMAC(signature)
	if err != nil {
		return err
	}
	if !checkMAC(payload, messageMAC, secretKey, hashFunc) {
		return errors.New("payload signature check failed")
	}
	return nil
}

// bitbucketHandler handles push events for bitbucket.
// After the request has been authenticated,
// the kubeclient is queried for all warehouses that contain a subscription
// to the repo in question. Those warehouses are then patched with a special
// annotation that signals down stream logic to refresh the warehouse.
func bitbucketHandler(c client.Client, namespace, secretName string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hook, _ := bitbucket.New(bitbucket.Options.UUID(getUUID(r)))
		ctx := r.Context()
		logger := logging.LoggerFromContext(ctx).WithValues("path", r.URL.Path)
		ctx = logging.ContextWithLogger(ctx, logger)
		logger.Debug("retrieving secret", "secret-name", secretName)
		var secret corev1.Secret
		err := c.Get(ctx, client.ObjectKey{
			Name:      secretName,
			Namespace: namespace,
		}, &secret,
		)
		if err != nil {
			logger.Error(err, "failed to get bitbucket secret")
			xhttp.WriteErrorJSON(w, errors.New("configuration error"))
			return
		}
		BitbucketSecret, exists := secret.Data[kargoapi.WebhookReceiverSecretKeyBitbucket]
		if !exists {
			logger.Error(
				errors.New("invalid secret data"),
				"no value for target key",
				"target-key", kargoapi.WebhookReceiverSecretKeyGithub,
			)
			xhttp.WriteErrorJSON(w, errors.New("configuration error"))
			return
		}
		logger.Debug("identifying source repository")
		eventType := r.Header.Get(bitbucketEventTypeHeader)
		switch eventType {
		case "repo:push":
		default:
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					fmt.Errorf("event type %s is not supported", eventType),
					http.StatusNotImplemented,
				),
			)
			return
		}

		const maxBytes = 2 << 20 // 2MB
		b, err := io.LimitRead(r.Body, maxBytes)
		if err != nil {
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					fmt.Errorf("failed to read request body: %w", err),
					http.StatusRequestEntityTooLarge,
				),
			)
			return
		}

		// Reset r.Body so the parser can read the payload
		r.Body = body.NopCloser(bytes.NewReader(b))

		sig := r.Header.Get(bitbucketSignatureHeader)
		if sig == "" {
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					errors.New("missing signature"),
					http.StatusUnauthorized,
				),
			)
			return
		}

		if err = ValidateSignature(sig, b, BitbucketSecret); err != nil {
			logger.Error(err, "failed to validate signature")
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					errors.New("unauthorized"),
					http.StatusUnauthorized,
				),
			)
			return
		}

		payload, err := hook.Parse(r, bitbucket.RepoPushEvent)
		if err != nil {
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					fmt.Errorf("failed to parse webhook event: %w", err),
					http.StatusBadRequest,
				),
			)
			return
		}

		switch payload := payload.(type) {
		case bitbucket.RepoPushPayload:
			repoWebURL := payload.Repository.Links.HTML.Href
			logger = logger.WithValues("repoWebURL", repoWebURL)
			ctx = logging.ContextWithLogger(ctx, logger)
			result, err := refreshWarehouses(ctx, c, namespace, repoWebURL)
			if err != nil {
				xhttp.WriteErrorJSON(w,
					xhttp.Error(err, http.StatusInternalServerError),
				)
				return
			}

			logger.Debug("execution complete",
				"successes", result.successes,
				"failures", result.failures,
			)

			if result.failures > 0 {
				xhttp.WriteResponseJSON(w,
					http.StatusInternalServerError,
					map[string]string{
						"error": fmt.Sprintf("failed to refresh %d of %d warehouses",
							result.failures,
							result.successes+result.failures,
						),
					},
				)
				return
			}

			xhttp.WriteResponseJSON(w,
				http.StatusOK,
				map[string]string{
					"msg": fmt.Sprintf("refreshed %d warehouse(s)",
						result.successes,
					),
				},
			)
		}
	})
}

// messageMAC returns the hex-decoded HMAC tag from the signature and its
// corresponding hash function.
func messageMAC(signature string) ([]byte, func() hash.Hash, error) {
	if signature == "" {
		return nil, nil, errors.New("missing signature")
	}
	sigParts := strings.SplitN(signature, "=", 2)
	if len(sigParts) != 2 {
		return nil, nil, fmt.Errorf("error parsing signature %q", signature)
	}

	var hashFunc func() hash.Hash
	switch sigParts[0] {
	case "sha1":
		hashFunc = sha1.New
	case "sha256":
		hashFunc = sha256.New
	case "sha512":
		hashFunc = sha512.New
	default:
		return nil, nil, fmt.Errorf("unknown hash type prefix: %q", sigParts[0])
	}

	buf, err := hex.DecodeString(sigParts[1])
	if err != nil {
		return nil, nil, fmt.Errorf("error decoding signature %q: %v", signature, err)
	}
	return buf, hashFunc, nil
}

// checkMAC reports whether messageMAC is a valid HMAC tag for message.
func checkMAC(message, messageMAC, key []byte, hashFunc func() hash.Hash) bool {
	expectedMAC := genMAC(message, key, hashFunc)
	return hmac.Equal(messageMAC, expectedMAC)
}

// genMAC generates the HMAC signature for a message provided the secret key
// and hashFunc.
func genMAC(message, key []byte, hashFunc func() hash.Hash) []byte {
	mac := hmac.New(hashFunc, key)
	mac.Write(message)
	return mac.Sum(nil)
}
