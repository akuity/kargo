package external

import (
	"encoding/json"
	"fmt"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"

	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/io"
	"github.com/akuity/kargo/internal/logging"
)

func quayHandler(
	c client.Client,
	namespace string,
) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.LoggerFromContext(ctx).WithValues("path", r.URL.Path)
		ctx = logging.ContextWithLogger(ctx, logger)
		logger.Debug("identifying source repository")

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

		payload := struct {
			// format: https://quay.io/repository/mynamespace/repository
			RepoWebURL string `json:"homepage"`
		}{}

		if err = json.Unmarshal(b, &payload); err != nil {
			logger.Error(err, "failed to unmarshal request body")
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					fmt.Errorf("failed to unmarshal request body: %w", err),
					http.StatusBadRequest,
				),
			)
			return
		}

		if payload.RepoWebURL == "" {
			logger.Debug("missing repository web URL in request body")
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					fmt.Errorf("missing repository web URL in request body"),
					http.StatusBadRequest,
				),
			)
			return
		}

		logger = logger.WithValues("repoWebURL", payload.RepoWebURL)
		ctx = logging.ContextWithLogger(ctx, logger)
		result, err := refreshWarehouses(ctx, c, namespace, payload.RepoWebURL)
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
	})
}
