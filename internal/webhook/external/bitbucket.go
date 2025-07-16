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
	"github.com/akuity/kargo/internal/logging"
)

const (
	bitbucket = "bitbucket"

	bitbucketSecretDataKey = "secret"

	bitbucketEventHeader     = "X-Event-Key"
	bitbucketSignatureHeader = "X-Hub-Signature"

	bitbucketPushEvent = "repo:push"
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
		return nil, fmt.Errorf("missing data key %q for Bitbucket WebhookReceiver", bitbucketSecretDataKey)
	}
	return []string{string(secretValue)}, nil
}

// getHandler implements WebhookReceiver.
func (b *bitbucketWebhookReceiver) getHandler(requestBody []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger := logging.LoggerFromContext(ctx)

		signingKey, ok := b.secretData[bitbucketSecretDataKey]
		if !ok {
			xhttp.WriteErrorJSON(w, nil)
			return
		}

		eventType := r.Header.Get(bitbucketEventHeader)
		switch eventType {
		case bitbucketPushEvent:
		default:
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					fmt.Errorf("event type %s is not supported", eventType),
					http.StatusBadRequest,
				),
			)
			return
		}

		sig := r.Header.Get(bitbucketSignatureHeader)
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
		if err := gh.ValidateSignature(sig, requestBody, signingKey); err != nil {
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
		if err := json.Unmarshal(requestBody, &payload); err != nil {
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

		refreshWarehouses(ctx, w, b.client, b.project, repoURL)
	})
}
