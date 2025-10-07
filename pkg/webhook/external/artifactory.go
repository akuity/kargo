package external

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	xhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/urls"
)

const (
	artifactoryDockerDomain    = "docker"
	artifactoryPushedEventType = "pushed"
	artifactoryChartImageType  = "oci"
	artifactoryAuthHeader      = "X-Jfrog-Event-Auth"
	artifactorySecretDataKey   = "secret-token"
	artifactory                = "artifactory"
	artifactoryRepoURLHeader   = "X-Kargo-Repo-URLs"
)

var validImageTypes = []string{
	artifactoryDockerDomain,
	artifactoryChartImageType,
}

func init() {
	registry.register(
		artifactory,
		webhookReceiverRegistration{
			predicate: func(cfg kargoapi.WebhookReceiverConfig) bool {
				return cfg.Artifactory != nil
			},
			factory: newArtifactoryWebhookReceiver,
		},
	)
}

// artifactoryWebhookReceiver is an implementation of WebhookReceiver that
// handles inbound webhooks from Jfrog Artifactory.
type artifactoryWebhookReceiver struct {
	*baseWebhookReceiver
}

// newArtifactoryWebhookReceiver returns a new instance of
// artifactoryWebhookReceiver.
func newArtifactoryWebhookReceiver(
	c client.Client,
	project string,
	cfg kargoapi.WebhookReceiverConfig,
) WebhookReceiver {
	return &artifactoryWebhookReceiver{
		baseWebhookReceiver: &baseWebhookReceiver{
			client:     c,
			project:    project,
			secretName: cfg.Artifactory.SecretRef.Name,
		},
	}
}

// getReceiverType implements WebhookReceiver.
func (a *artifactoryWebhookReceiver) getReceiverType() string {
	return artifactory
}

// getSecretValues implements WebhookReceiver.
func (a *artifactoryWebhookReceiver) getSecretValues(
	secretData map[string][]byte,
) ([]string, error) {
	secretValue, ok := secretData[artifactorySecretDataKey]
	if !ok {
		return nil,
			errors.New("secret data is not valid for a Artifactory WebhookReceiver")
	}
	return []string{string(secretValue)}, nil
}

// getHandler implements WebhookReceiver.
func (a *artifactoryWebhookReceiver) getHandler(requestBody []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.LoggerFromContext(ctx)

		token, ok := a.secretData[artifactorySecretDataKey]
		if !ok {
			xhttp.WriteErrorJSON(w, nil)
			return
		}

		sig := r.Header.Get(artifactoryAuthHeader)
		if sig == "" {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("missing signature"), http.StatusUnauthorized),
			)
			return
		}

		mac := hmac.New(sha256.New, token)
		mac.Write(requestBody)
		computedSig := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(sig), []byte(computedSig)) {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					errors.New("unauthorized"),
					http.StatusUnauthorized,
				),
			)
			return
		}

		var payload artifactoryEvent
		if err := json.Unmarshal(requestBody, &payload); err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("invalid request body"), http.StatusBadRequest),
			)
			return
		}

		logger = logger.WithValues(
			"path", payload.Data.Path,
			"eventType", payload.EventType,
			"domain", payload.Domain,
			"repoKey", payload.Data.RepoKey,
			"imageName", payload.Data.ImageName,
			"imageType", payload.Data.ImageType,
			"origin", payload.Origin,
			"tag", payload.Data.Tag,
		)

		logger.Info("unmarshalled Artifactory webhook payload")

		if payload.EventType != artifactoryPushedEventType || payload.Domain != artifactoryDockerDomain {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					errors.New("event type must be 'pushed' and domain must be 'docker'"),
					http.StatusNotImplemented,
				),
			)
			return
		}

		if !slices.Contains(validImageTypes, payload.Data.ImageType) {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					fmt.Errorf("unsupported image type %q", payload.Data.ImageType),
					http.StatusNotImplemented,
				),
			)
			return
		}

		var repoURLs []string
		if repoURLsHeader := r.Header.Get(artifactoryRepoURLHeader); repoURLsHeader != "" {
			repoURLs = strings.Split(repoURLsHeader, ",")
			for _, repoURL := range repoURLs {
				if _, err := url.Parse(repoURL); err != nil {
					xhttp.WriteErrorJSON(
						w,
						xhttp.Error(
							fmt.Errorf("invalid repo URL %q in %s header", repoURL, artifactoryRepoURLHeader),
							http.StatusBadRequest,
						),
					)
					return
				}
			}
		} else {
			originURL, err := url.Parse(payload.Origin)
			if err != nil {
				xhttp.WriteErrorJSON(
					w,
					xhttp.Error(errors.New("invalid request body"), http.StatusBadRequest),
				)
				return
			}

			pathSections := strings.Split(payload.Data.Path, "/")
			if len(pathSections) < 2 {
				xhttp.WriteErrorJSON(
					w,
					xhttp.Error(errors.New("invalid path"), http.StatusBadRequest),
				)
				return
			}

			repoURL := strings.Join(
				append(
					[]string{originURL.Host, payload.Data.RepoKey},
					pathSections[:len(pathSections)-2]...,
				),
				"/",
			)
			repoURLs = append(repoURLs, repoURL)
		}

		for i, repoURL := range repoURLs {
			switch payload.Data.ImageType {
			case artifactoryDockerDomain:
				repoURLs[i] = urls.NormalizeImage(repoURL)
			case artifactoryChartImageType:
				repoURLs[i] = urls.NormalizeChart(repoURL)
				// no default case because this is already checked above
			}
		}
		logger = logger.WithValues("repoURLs", repoURLs)
		ctx = logging.ContextWithLogger(ctx, logger)
		refreshWarehouses(ctx, w, a.client, a.project, repoURLs, payload.Data.Tag)
	})
}

type artifactoryEvent struct {
	Domain    string               `json:"domain"`
	EventType string               `json:"event_type"`
	Data      artifactoryEventData `json:"data"`
	Origin    string               `json:"jpd_origin"`
}

type artifactoryEventData struct {
	Path      string `json:"path"`
	RepoKey   string `json:"repo_key"`
	ImageName string `json:"image_name"`
	ImageType string `json:"image_type"`
	Tag       string `json:"tag"`
}
