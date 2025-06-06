package external

import (
	"errors"
	"fmt"
	"net/http"

	gh "github.com/google/go-github/v71/github"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/io"
	"github.com/akuity/kargo/internal/logging"
)

// githubHandler handles push events for github.
// After the request has been authenticated,
// the kubeclient is queried for all warehouses that contain a subscription
// to the repo in question. Those warehouses are then patched with a special
// annotation that signals down stream logic to refresh the warehouse.
func githubHandler(
	c client.Client,
	namespace string,
	secretName string,
) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.LoggerFromContext(ctx).WithValues("path", r.URL.Path)
		ctx = logging.ContextWithLogger(ctx, logger)
		logger.Debug("retrieving secret", "secret-name", secretName)
		var secret corev1.Secret
		err := c.Get(ctx,
			client.ObjectKey{
				Name:      secretName,
				Namespace: namespace,
			},
			&secret,
		)
		if err != nil {
			logger.Error(err, "failed to get github secret")
			xhttp.WriteErrorJSON(w, errors.New("configuration error"))
			return
		}
		token, ok := secret.Data[kargoapi.WebhookReceiverSecretKeyGithub]
		if !ok {
			logger.Error(
				errors.New("invalid secret data"),
				"no value for target key",
				"target-key", kargoapi.WebhookReceiverSecretKeyGithub,
			)
			xhttp.WriteErrorJSON(w, errors.New("configuration error"))
			return
		}
		logger.Debug("identifying source repository")

		// TODO(fuskovic): eventually switch on event type to perform
		// different actions (e.g. refresh Promotion on PR merge)
		eventType := r.Header.Get("X-GitHub-Event")
		switch eventType {
		case "ping", "push", "package":
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

		sig := r.Header.Get(gh.SHA256SignatureHeader)
		if sig == "" {
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					errors.New("missing signature"),
					http.StatusUnauthorized,
				),
			)
			return
		}

		if err = gh.ValidateSignature(sig, b, token); err != nil {
			logger.Error(err, "failed to validate signature")
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					errors.New("unauthorized"),
					http.StatusUnauthorized,
				),
			)
			return
		}

		e, err := gh.ParseWebHook(eventType, b)
		if err != nil {
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					fmt.Errorf("failed to parse webhook event: %w", err),
					http.StatusBadRequest,
				),
			)
			return
		}

		switch e := e.(type) {
		case *gh.PingEvent:
			repoWebURL := e.GetRepo().GetHTMLURL()
			logger.Debug("received ping event", "repo", repoWebURL)
			xhttp.WriteResponseJSON(w,
				http.StatusOK,
				map[string]string{
					"msg": fmt.Sprintf(
						"ping event received, webhook is configured correctly for %s",
						repoWebURL,
					),
				},
			)
		case *gh.PushEvent:
			repoWebURL := e.GetRepo().GetHTMLURL()
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
		case *gh.PackageEvent:
			name := e.GetPackage().Name
			ns := e.GetPackage().Namespace
			// construct the package URL that will be used by the warehouse
			// based on name and namespace field from the package payload
			// https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry
			packageURL := fmt.Sprintf("ghcr.io/%v/%v", *ns, *name)
			logger = logger.WithValues("packageURL", packageURL)
			ctx = logging.ContextWithLogger(ctx, logger)
			result, err := refreshWarehouses(ctx, c, namespace, packageURL)
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
