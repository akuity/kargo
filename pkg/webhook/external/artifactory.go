package external

import (
	"context"
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
)

var artifactoryValidImageTypes = []string{
	artifactoryDockerDomain,
	artifactoryChartImageType,
}

func init() {
	defaultWebhookReceiverRegistry.MustRegister(
		webhookReceiverRegistration{
			Predicate: func(_ context.Context, cfg kargoapi.WebhookReceiverConfig) (bool, error) {
				return cfg.Artifactory != nil, nil
			},
			Value: newArtifactoryWebhookReceiver,
		},
	)
}

// artifactoryWebhookReceiver is an implementation of WebhookReceiver that
// handles inbound webhooks from Jfrog Artifactory.
type artifactoryWebhookReceiver struct {
	*baseWebhookReceiver
	virtualRepoName string
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
		virtualRepoName: cfg.Artifactory.VirtualRepoName,
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
		_, _ = mac.Write(requestBody)
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

		if !slices.Contains(artifactoryValidImageTypes, payload.Data.ImageType) {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					fmt.Errorf("unsupported image type %q", payload.Data.ImageType),
					http.StatusNotImplemented,
				),
			)
			return
		}

		originURL, err := url.Parse(payload.Origin)
		if err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					fmt.Errorf("invalid value %q in payload's jpd_origin field", payload.Origin),
					http.StatusBadRequest,
				),
			)
			return
		}

		pathSections := strings.Split(payload.Data.Path, "/")
		if len(pathSections) < 2 {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					fmt.Errorf("invalid value %q in payload's data.path field", payload.Data.Path),
					http.StatusBadRequest,
				),
			)
			return
		}

		repoName := payload.Data.RepoKey
		if a.virtualRepoName != "" {
			repoName = a.virtualRepoName
		}

		repoURL := strings.Join(
			append(
				[]string{originURL.Host, repoName},
				pathSections[:len(pathSections)-2]...,
			),
			"/",
		)

		switch payload.Data.ImageType {
		case artifactoryDockerDomain:
			repoURL = urls.NormalizeImage(repoURL)
		case artifactoryChartImageType:
			repoURL = urls.NormalizeChart(repoURL)
		default:
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					fmt.Errorf("unsupported image type %q", payload.Data.ImageType),
					http.StatusNotImplemented,
				),
			)
			return
		}
		logger = logger.WithValues("repoURL", repoURL)
		ctx = logging.ContextWithLogger(ctx, logger)
		refreshWarehouses(ctx, w, a.client, a.project, []string{repoURL}, payload.Data.Tag)
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
