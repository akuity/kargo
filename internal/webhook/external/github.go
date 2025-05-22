package external

import (
	"errors"
	"fmt"
	"net/http"

	gh "github.com/google/go-github/v71/github"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/logging"
)

// githubHandler handles push events for github.
// After the request has been authenticated,
// the kubeclient is queried for all warehouses that contain a subscription
// to the repo in question. Those warehouses are then patched with a special
// annotation that signals down stream logic to refresh the warehouse.
func githubHandler(c client.Client, namespace, token string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.LoggerFromContext(ctx)
		logger.Debug("identifying source repository")

		eventType := r.Header.Get("X-GitHub-Event")
		if eventType != "push" {
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					fmt.Errorf("only push events are supported"),
					http.StatusNotImplemented,
				),
			)
			return
		}

		const maxBytes = 2 << 20 // 2MB
		bodyBytes, err := xhttp.LimitRead(r.Body, maxBytes)
		if err != nil {
			xhttp.WriteErrorJSON(w, err)
			return
		}

		signature := r.Header.Get(gh.SHA256SignatureHeader)
		if signature == "" {
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					errors.New("missing signature"),
					http.StatusUnauthorized,
				),
			)
			return
		}

		if err = gh.ValidateSignature(
			signature,
			bodyBytes,
			[]byte(token),
		); err != nil {
			logger.Error(err, "failed to validate signature")
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					errors.New("unauthorized"),
					http.StatusUnauthorized,
				),
			)
			return
		}

		e, err := gh.ParseWebHook(eventType, bodyBytes)
		if err != nil {
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					fmt.Errorf("failed to parse webhook event: %w", err),
					http.StatusBadRequest,
				),
			)
			return
		}

		pe, ok := e.(*gh.PushEvent)
		if !ok {
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					fmt.Errorf("only push events are supported"),
					http.StatusNotImplemented,
				),
			)
			return
		}

		repo := *pe.Repo.HTMLURL
		logger.Debug("source repository retrieved", "name", repo)
		ctx = logging.ContextWithLogger(ctx, logger)
		result, err := refreshWarehouses(ctx, c, namespace, repo)
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
				"msg": fmt.Sprintf("refreshed %d warehouses",
					result.successes,
				),
			},
		)
	})
}
