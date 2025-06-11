package external

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	gh "github.com/google/go-github/v71/github"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/io"
	"github.com/akuity/kargo/internal/logging"
)

const (
	bitbucket                    = "bitbucket"
	bitbucketSecretDataKey       = "secret"
	bitbucketPushEvent           = "repo:push"
	bitbucketWebhookBodyMaxBytes = 2 << 20 // 2MB
)

func init() {
	registry.register(
		bitbucket,
		webhookReceiverRegistration{
			predicate: func(cfg kargoapi.WebhookReceiverConfig) bool {
				return cfg.Bitbucket != nil
			},
			factory: newBitbucketWebhookReceiver,
		},
	)
}

// bitbucketWebhookReceiver is an implementation of WebhookReceiver that handles
// inbound webhooks from Bitbucket.
type bitbucketWebhookReceiver struct {
	*baseWebhookReceiver
}

// newBitbucketWebhookReceiver returns a new instance of
// bitbucketWebhookReceiver.
func newBitbucketWebhookReceiver(
	c client.Client,
	project string,
	cfg kargoapi.WebhookReceiverConfig,
) WebhookReceiver {
	return &bitbucketWebhookReceiver{
		baseWebhookReceiver: &baseWebhookReceiver{
			client:     c,
			project:    project,
			secretName: cfg.Bitbucket.SecretRef.Name,
		},
	}
}

// getReceiverType implements WebhookReceiver.
func (b *bitbucketWebhookReceiver) getReceiverType() string {
	return bitbucket
}

// getSecretValues implements WebhookReceiver.
func (b *bitbucketWebhookReceiver) getSecretValues(
	secretData map[string][]byte,
) ([]string, error) {
	secretValue, ok := secretData[bitbucketSecretDataKey]
	if !ok {
		return nil,
			errors.New("Secret data is not valid for a Bitbucket WebhookReceiver")
	}
	return []string{string(secretValue)}, nil
}

// GetHandler implements WebhookReceiver.
func (b *bitbucketWebhookReceiver) GetHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger := logging.LoggerFromContext(ctx)

		signingKey, ok := b.secretData[bitbucketSecretDataKey]
		if !ok {
			xhttp.WriteErrorJSON(w, nil)
			return
		}

		eventType := r.Header.Get("X-Event-Key")
		switch eventType {
		case bitbucketPushEvent:
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

		// Early check of Content-Length if available
		if contentLength := r.ContentLength; contentLength > bitbucketWebhookBodyMaxBytes {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					fmt.Errorf("content exceeds limit of %d bytes", bitbucketWebhookBodyMaxBytes),
					http.StatusRequestEntityTooLarge,
				),
			)
			return
		}

		body, err := io.LimitRead(r.Body, bitbucketWebhookBodyMaxBytes)
		if err != nil {
			if errors.Is(err, &io.BodyTooLargeError{}) {
				xhttp.WriteErrorJSON(
					w,
					xhttp.Error(err, http.StatusRequestEntityTooLarge),
				)
				return
			}
			xhttp.WriteErrorJSON(w, err)
			return
		}

		sig := r.Header.Get("X-Hub-Signature")
		if sig == "" {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("missing signature"), http.StatusUnauthorized),
			)
			return
		}

		// Note: github.com/google/go-github/v71/github has a great implementation
		// of HMAC signature validation that isn't GitHub-specific, so we've opted
		// to use it here for Bitbucket as well.
		if err = gh.ValidateSignature(sig, body, signingKey); err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("unauthorized"), http.StatusUnauthorized),
			)
			return
		}

		payload := struct {
			Repository struct {
				Links struct {
					HTML struct {
						Href string `json:"href"`
					} `json:"html"`
				} `json:"links"`
			} `json:"repository"`
		}{}
		if err = json.Unmarshal(body, &payload); err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("invalid request body"), http.StatusBadRequest),
			)
			return
		}

		// Note: It may seem peculiar to obtain the repo URL from the payload's
		// repository.links.html.href field, however, there is no better option.
		//
		// A naive option would be combining https://bitbucket.org/ with the value
		// of the payload's repository.full_name field, but that does not hold up
		// for events originating from Bitbucket Data Center, which will utilize a
		// custom domain name.
		//
		// A slightly better approach would be to parse the protocol and hostname
		// from the value of the payload's repository.links.html.href field and
		// combine that with the value of the payload's repository.full_name field,
		// however, in all (currently known) cases, that yields the same result as
		// simply using the value of the repository.links.html.href field directly.
		//
		// TODO(krancour): There are very likely some yet-to-be-identified edge
		// cases where this choice does not hold up.
		repoURL := payload.Repository.Links.HTML.Href

		logger = logger.WithValues("repoURL", repoURL)
		ctx = logging.ContextWithLogger(ctx, logger)

		result, err := refreshWarehouses(ctx, b.client, b.project, repoURL)
		if err != nil {
			xhttp.WriteErrorJSON(w, err)
			return
		}
		if result.failures > 0 {
			xhttp.WriteResponseJSON(
				w,
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
		xhttp.WriteResponseJSON(
			w,
			http.StatusOK,
			map[string]string{
				"msg": fmt.Sprintf("refreshed %d warehouse(s)", result.successes),
			},
		)
	})
}
