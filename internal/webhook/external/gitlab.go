package external

import (
	"errors"
	"fmt"
	"net/http"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/io"
	"github.com/akuity/kargo/internal/logging"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gl "gitlab.com/gitlab-org/api/client-go"
)

func gitlabHandler(
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
			logger.Error(err, "failed to get gitlab secret")
			xhttp.WriteErrorJSON(w, errors.New("configuration error"))
			return
		}

		token, ok := secret.Data[kargoapi.WebhookReceiverSecretKeyGitlab]
		if !ok {
			logger.Error(
				errors.New("invalid secret data"),
				"no value for target key",
				"target-key", kargoapi.WebhookReceiverSecretKeyGitlab,
			)
			xhttp.WriteErrorJSON(w, errors.New("configuration error"))
			return
		}


		if string(token) != gl.HookEventToken(r) {
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					errors.New("invalid token"),
					http.StatusUnauthorized,
				),
			)
			return
		}

		eventType := gl.HookEventType(r)
		switch eventType {
		case gl.EventTypePush:
		default:
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					fmt.Errorf("%q is an unsupported event type", eventType),
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

		e, err := gl.ParseWebhook(eventType, b)
		if err != nil {
			logger.Error(err, "failed to parse webhook payload")
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					fmt.Errorf("failed to parse webhook payload: %w", err),
					http.StatusBadRequest,
				),
			)
			return
		}

		switch e := e.(type) {
		case *gl.PushEvent:
			repoWebURL := e.Repository.Homepage
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
