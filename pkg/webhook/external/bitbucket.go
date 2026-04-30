package external

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	gh "github.com/google/go-github/v76/github"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	xhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/urls"
)

const (
	bitbucket = "bitbucket"

	bitbucketSecretDataKey = "secret"

	bitbucketEventHeader     = "X-Event-Key"
	bitbucketSignatureHeader = "X-Hub-Signature"

	// bitbucketPushEvent is the event Bitbucket Cloud sends when a branch
	// receives new commits.  Its body is represented by the
	// bitbucketPushEventBody struct.
	// See https://support.atlassian.com/bitbucket-cloud/docs/event-payloads/#Push
	bitbucketPushEvent = "repo:push"
	// bitbucketRefsChangedEvent is the event Bitbucket Server (Data Center)
	// sends when a branch receives new commits. Its body is represented by the
	// bitbucketRefsChangedEventBody struct.
	// See https://confluence.atlassian.com/bitbucketserver/event-payload-938025882.html#Eventpayload-repo-push
	bitbucketRefsChangedEvent = "repo:refs_changed"
	// bitbucketPRFulfilledEvent is the event Bitbucket Cloud sends when a pull
	// request is merged.
	// See https://support.atlassian.com/bitbucket-cloud/docs/event-payloads/#Merged
	bitbucketPRFulfilledEvent = "pullrequest:fulfilled"
	// bitbucketPRRejectedEvent is the event Bitbucket Cloud sends when a pull
	// request is declined (closed without merge).
	// See https://support.atlassian.com/bitbucket-cloud/docs/event-payloads/#Declined
	bitbucketPRRejectedEvent = "pullrequest:rejected"
	// bitbucketPRMergedEvent is the event Bitbucket Server (Data Center) sends
	// when a pull request is merged.
	// See https://confluence.atlassian.com/bitbucketserver/event-payload-938025882.html#Eventpayload-pr:merged
	bitbucketPRMergedEvent = "pr:merged"
	// bitbucketPRDeclinedEvent is the event Bitbucket Server (Data Center) sends
	// when a pull request is declined (closed without merge).
	// See https://confluence.atlassian.com/bitbucketserver/event-payload-938025882.html#Eventpayload-pr:declined
	bitbucketPRDeclinedEvent = "pr:declined"
)

func init() {
	defaultWebhookReceiverRegistry.MustRegister(
		webhookReceiverRegistration{
			Predicate: func(_ context.Context, cfg kargoapi.WebhookReceiverConfig) (bool, error) {
				return cfg.Bitbucket != nil, nil
			},
			Value: newBitbucketWebhookReceiver,
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

		signingKey, ok := b.secretData[bitbucketSecretDataKey]
		if !ok {
			xhttp.WriteErrorJSON(w, nil)
			return
		}

		eventType := r.Header.Get(bitbucketEventHeader)
		var handleEvent func(context.Context, http.ResponseWriter, []byte)
		switch eventType {
		case bitbucketPushEvent:
			handleEvent = b.handlePushEvent
		case bitbucketRefsChangedEvent:
			handleEvent = b.handleRefsChangedEvent
		case bitbucketPRFulfilledEvent, bitbucketPRRejectedEvent:
			handleEvent = b.handlePRFulfilledEvent
		case bitbucketPRMergedEvent, bitbucketPRDeclinedEvent:
			handleEvent = b.handlePRMergedEvent
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

		// Note: github.com/google/go-github/v76/github has a great implementation
		// of HMAC signature validation that isn't GitHub-specific, so we've opted
		// to use it here for Bitbucket as well.
		if err := gh.ValidateSignature(sig, requestBody, signingKey); err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("unauthorized"), http.StatusUnauthorized),
			)
			return
		}

		handleEvent(ctx, w, requestBody)
	})
}

// handlePushEvent handles Bitbucket Cloud repo:push events.
// See https://support.atlassian.com/bitbucket-cloud/docs/event-payloads/#Push
func (b *bitbucketWebhookReceiver) handlePushEvent(
	ctx context.Context,
	w http.ResponseWriter,
	requestBody []byte,
) {
	payload := &bitbucketPushEventBody{}
	if err := json.Unmarshal(requestBody, payload); err != nil {
		xhttp.WriteErrorJSON(
			w,
			xhttp.Error(errors.New("invalid request body"), http.StatusBadRequest),
		)
		return
	}
	repoURLs := payload.getRepoURLs()
	refs := payload.getRefs()
	logger := logging.LoggerFromContext(ctx).WithValues(
		"repoURLs", repoURLs,
		"refs", refs,
	)
	ctx = logging.ContextWithLogger(ctx, logger)
	refreshWarehouses(ctx, w, b.client, b.project, repoURLs, nil, refs...)
}

// handleRefsChangedEvent handles Bitbucket Server (Data Center) repo:refs_changed
// events.
// See https://confluence.atlassian.com/bitbucketserver/event-payload-938025882.html#Eventpayload-repo-push
func (b *bitbucketWebhookReceiver) handleRefsChangedEvent(
	ctx context.Context,
	w http.ResponseWriter,
	requestBody []byte,
) {
	payload := &bitbucketRefsChangedEventBody{}
	if err := json.Unmarshal(requestBody, payload); err != nil {
		xhttp.WriteErrorJSON(
			w,
			xhttp.Error(errors.New("invalid request body"), http.StatusBadRequest),
		)
		return
	}
	repoURLs := payload.getRepoURLs()
	refs := payload.getRefs()
	logger := logging.LoggerFromContext(ctx).WithValues(
		"repoURLs", repoURLs,
		"refs", refs,
	)
	ctx = logging.ContextWithLogger(ctx, logger)
	refreshWarehouses(ctx, w, b.client, b.project, repoURLs, nil, refs...)
}

// handlePRFulfilledEvent handles Bitbucket Cloud pullrequest:fulfilled events,
// which are sent when a pull request is merged.
// See https://support.atlassian.com/bitbucket-cloud/docs/event-payloads/#Merged
func (b *bitbucketWebhookReceiver) handlePRFulfilledEvent(
	ctx context.Context,
	w http.ResponseWriter,
	requestBody []byte,
) {
	payload := struct {
		PullRequest struct {
			Links struct {
				HTML struct {
					Href string `json:"href"`
				} `json:"html"`
			} `json:"links"`
		} `json:"pullrequest"`
	}{}
	if err := json.Unmarshal(requestBody, &payload); err != nil {
		xhttp.WriteErrorJSON(
			w,
			xhttp.Error(errors.New("invalid request body"), http.StatusBadRequest),
		)
		return
	}
	prURL := payload.PullRequest.Links.HTML.Href
	logger := logging.LoggerFromContext(ctx).WithValues("prURL", prURL)
	ctx = logging.ContextWithLogger(ctx, logger)
	refreshPromotionsByPrURL(ctx, w, b.client, b.project, prURL)
}

// handlePRMergedEvent handles Bitbucket Server (Data Center) pr:merged events,
// which are sent when a pull request is merged.
// See https://confluence.atlassian.com/bitbucketserver/event-payload-938025882.html#Eventpayload-pr:merged
func (b *bitbucketWebhookReceiver) handlePRMergedEvent(
	ctx context.Context,
	w http.ResponseWriter,
	requestBody []byte,
) {
	payload := struct {
		PullRequest struct {
			Links struct {
				Self []struct {
					Href string `json:"href"`
				} `json:"self"`
			} `json:"links"`
		} `json:"pullRequest"`
	}{}
	if err := json.Unmarshal(requestBody, &payload); err != nil {
		xhttp.WriteErrorJSON(
			w,
			xhttp.Error(errors.New("invalid request body"), http.StatusBadRequest),
		)
		return
	}
	var prURL string
	if len(payload.PullRequest.Links.Self) > 0 {
		prURL = payload.PullRequest.Links.Self[0].Href
	}
	logger := logging.LoggerFromContext(ctx).WithValues("prURL", prURL)
	ctx = logging.ContextWithLogger(ctx, logger)
	refreshPromotionsByPrURL(ctx, w, b.client, b.project, prURL)
}

// bitbucketPushEventBody represents the payload Bitbucket Cloud sends for
// "repo:push" events.
// See https://support.atlassian.com/bitbucket-cloud/docs/event-payloads/#Push
type bitbucketPushEventBody struct {
	Actor struct {
		Name         string `json:"name"`
		EmailAddress string `json:"emailAddress"`
	} `json:"actor"`
	Push struct {
		Changes []struct {
			New struct {
				Type string `json:"type"`
				Name string `json:"name"`
			} `json:"new"`
		} `json:"changes"`
	} `json:"push"`
	Repository struct {
		Links struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
	} `json:"repository"`
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
func (b bitbucketPushEventBody) getRepoURLs() []string {
	return []string{b.Repository.Links.HTML.Href}
}

// getRefs extracts all references mentioned by the repo:push event.
// See https://support.atlassian.com/bitbucket-cloud/docs/event-payloads/#Push
//
// Bitbucket Cloud sends bare names (e.g. "main") in change.new.name, unlike
// GitHub/GitLab which send full refs. We qualify them here so that downstream
// MatchesRef checks (which require refs/heads/ or refs/tags/ prefixes) work.
func (b bitbucketPushEventBody) getRefs() []string {
	var qualifiers []string
	for _, change := range b.Push.Changes {
		if change.New.Name == "" {
			continue // branch/tag deletion — new is null
		}
		if change.New.Type == "tag" {
			qualifiers = append(qualifiers, "refs/tags/"+change.New.Name)
		} else {
			qualifiers = append(qualifiers, "refs/heads/"+change.New.Name)
		}
	}
	return qualifiers
}

// bitbucketRefsChangedEventBody represents the payload Bitbucket Server (Data
// Center) sends for "repo:refs_changed" events.
// See https://confluence.atlassian.com/bitbucketserver/event-payload-938025882.html
// That documentation states: "The following payloads contain some of the
// common entity types – User, Repository,  Comment, and Pull Request – which
// have consistent representations in all the payloads where they appear.
type bitbucketRefsChangedEventBody struct {
	Actor struct {
		Name         string `json:"name"`
		EmailAddress string `json:"emailAddress"`
	} `json:"actor"`
	Changes []struct {
		Ref struct {
			ID string `json:"id"`
		} `json:"ref"`
	} `json:"changes"`
	Repository struct {
		Links struct {
			Clone []struct {
				Href string `json:"href"`
			} `json:"clone"`
		} `json:"links"`
	} `json:"repository"`
}

// getRepoURLs returns a set of normalized repoURLs from the event.
func (b bitbucketRefsChangedEventBody) getRepoURLs() []string {
	repoURLs := []string{}
	for _, link := range b.Repository.Links.Clone {
		repoURLs = append(repoURLs, urls.NormalizeGit(link.Href))
	}
	return repoURLs
}

func (b bitbucketRefsChangedEventBody) getRefs() []string {
	var qualifiers []string
	for _, change := range b.Changes {
		qualifiers = append(qualifiers, change.Ref.ID)
	}
	return qualifiers
}
