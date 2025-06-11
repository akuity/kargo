package external

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/io"
	"github.com/akuity/kargo/internal/logging"
)

const (
	dockerhub                    = "dockerhub"
	dockerhubSecretDataKey       = "secret"
	dockerhubWebhookBodyMaxBytes = 2 << 20 // 2MB
)

var (
	repoNameComponentRegexp = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*$`)
)

func init() {
	registry.register(
		dockerhub,
		webhookReceiverRegistration{
			predicate: func(cfg kargoapi.WebhookReceiverConfig) bool {
				return cfg.DockerHub != nil
			},
			factory: newDockerHubWebhookReceiver,
		},
	)
}

// dockerhubWebhookReceiver is an implementation of WebhookReceiver that handles
// inbound webhooks from Docker Hub.
type dockerhubWebhookReceiver struct {
	*baseWebhookReceiver
}

// newDockerHubWebhookReceiver returns a new instance of
// dockerhubWebhookReceiver.
func newDockerHubWebhookReceiver(
	c client.Client,
	project string,
	cfg kargoapi.WebhookReceiverConfig,
) WebhookReceiver {
	return &dockerhubWebhookReceiver{
		baseWebhookReceiver: &baseWebhookReceiver{
			client:     c,
			project:    project,
			secretName: cfg.GitHub.SecretRef.Name,
		},
	}
}

// getReceiverType implements WebhookReceiver.
func (d *dockerhubWebhookReceiver) getReceiverType() string {
	return dockerhub
}

// getSecretValues implements WebhookReceiver.
func (d *dockerhubWebhookReceiver) getSecretValues(
	secretData map[string][]byte,
) ([]string, error) {
	secretValue, ok := secretData[dockerhubSecretDataKey]
	if !ok {
		return nil,
			errors.New("Secret data is not valid for a Docker Hub WebhookReceiver")
	}
	return []string{string(secretValue)}, nil
}

// normalizeDockerImageRef normalizes Docker image references of the following forms:
//
//   - [docker.io/][namespace/]repo[:tag]
//   - [docker.io/][namespace/]repo[@digest]
//
// This is useful for the purposes of comparison and also in cases where a
// canonical representation of a Docker Hub image reference is needed. Any reference
// that cannot be normalized will return an error.
//
// Examples:
//
//	"nginx"                    -> "docker.io/library/nginx:latest"
//	"user/repo:v1.0"           -> "docker.io/user/repo:v1.0"
//	"docker.io/library/nginx"  -> "docker.io/library/nginx:latest"
//	"nginx@sha256:..."         -> "docker.io/library/nginx@sha256:..."
func normalizeDockerImageRef(ref string) (string, error) {
	const (
		defaultHost      = "docker.io"
		defaultNamespace = "library"
		defaultTag       = "latest"
	)

	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", errors.New("empty image reference")
	}
	ref = strings.ToLower(ref)

	// Remove leading docker.io/ if present
	if strings.HasPrefix(ref, defaultHost+"/") {
		ref = strings.TrimPrefix(ref, defaultHost+"/")
	}

	// Extract digest if present
	var digest string
	if at := strings.LastIndex(ref, "@"); at != -1 {
		digest = ref[at:]
		ref = ref[:at]
		// Validate digest format: must be @sha256:<64 hex>
		matched, _ := regexp.MatchString(`^@sha256:[a-f0-9]{64}$`, digest)
		if !matched {
			return "", fmt.Errorf("invalid digest format: %q", digest)
		}
	}

	// Extract tag if present (only if no digest)
	var tag string
	if digest == "" {
		if colon := strings.LastIndex(ref, ":"); colon != -1 && colon > strings.LastIndex(ref, "/") {
			tag = ref[colon+1:]
			ref = ref[:colon]
			if tag == "" {
				return "", errors.New("image reference has a colon but no tag")
			}
		} else {
			tag = defaultTag
		}
	}

	// Normalize path: always at least namespace/repo
	parts := strings.Split(ref, "/")
	if len(parts) == 1 {
		parts = []string{defaultNamespace, parts[0]}
	}
	// Validate path parts
	for _, part := range parts {
		if !repoNameComponentRegexp.MatchString(part) {
			return "", fmt.Errorf("invalid repository name component: %q", part)
		}
	}
	path := strings.Join(parts, "/")

	if digest != "" {
		return fmt.Sprintf("%s/%s%s", defaultHost, path, digest), nil
	}
	return fmt.Sprintf("%s/%s:%s", defaultHost, path, tag), nil
}

// GetHandler implements WebhookReceiver.
func (d *dockerhubWebhookReceiver) GetHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger := logging.LoggerFromContext(ctx)

		// Early check of Content-Length if available
		if contentLength := r.ContentLength; contentLength > dockerhubWebhookBodyMaxBytes {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					fmt.Errorf("content exceeds limit of %d bytes", dockerhubWebhookBodyMaxBytes),
					http.StatusRequestEntityTooLarge,
				),
			)
			return
		}

		body, err := io.LimitRead(r.Body, dockerhubWebhookBodyMaxBytes)
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

		payload := struct {
			Repository struct {
				RepoName string `json:"repo_name"`
			} `json:"repository"`
		}{}

		if err = json.Unmarshal(body, &payload); err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("invalid request body"), http.StatusBadRequest),
			)
			return
		}

		// Normalize the repo name
		repoURL, err := normalizeDockerImageRef(payload.Repository.RepoName)
		if err != nil {
			xhttp.WriteErrorJSON(w, err)
			return
		}

		logger = logger.WithValues("repoURL", repoURL)
		ctx = logging.ContextWithLogger(ctx, logger)

		result, err := refreshWarehouses(ctx, d.client, d.project, repoURL)
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
