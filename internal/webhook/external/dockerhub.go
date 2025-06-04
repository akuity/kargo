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

// dockerHubHandler handles Docker Hub webhook events.
// It parses the payload to extract the repository name and triggers a refresh
// for matching Warehouses.
func dockerHubHandler(
	c client.Client,
	namespace string,
) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.LoggerFromContext(ctx).WithValues("path", r.URL.Path)
		ctx = logging.ContextWithLogger(ctx, logger)
		logger.Debug("identifying source repository from Docker Hub payload")

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

		if len(b) == 0 {
			logger.Debug("empty request body")
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					fmt.Errorf("empty request body"),
					http.StatusBadRequest,
				),
			)
			return
		}

		payload := BuildPayload{}

		if err = json.Unmarshal(b, &payload); err != nil {
			logger.Error(err, "failed to unmarshal Docker Hub request body")
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					fmt.Errorf("failed to unmarshal request body: %w", err),
					http.StatusBadRequest,
				),
			)
			return
		}

		if payload.Repository.RepoName == "" {
			logger.Debug("missing repository name in Docker Hub request body")
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					fmt.Errorf("missing repository name in request body"),
					http.StatusBadRequest,
				),
			)
			return
		}

		// Compose a Docker Hub repo URL for matching (e.g., docker.io/myorg/myimage)
		repoURL := fmt.Sprintf("docker.io/%s", payload.Repository.RepoName)
		logger = logger.WithValues("repoURL", repoURL)
		ctx = logging.ContextWithLogger(ctx, logger)

		result, err := refreshWarehouses(ctx, c, namespace, repoURL)
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

// BuildPayload is a docker hub build notice
// https://docs.docker.com/docker-hub/webhooks/
type BuildPayload struct {
	CallbackURL string `json:"callback_url"`
	PushData    struct {
		Images   []string `json:"images"`
		PushedAt float32  `json:"pushed_at"`
		Pusher   string   `json:"pusher"`
		Tag      string   `json:"tag"`
	} `json:"push_data"`
	Repository struct {
		CommentCount    int     `json:"comment_count"`
		DateCreated     float32 `json:"date_created"`
		Description     string  `json:"description"`
		Dockerfile      string  `json:"dockerfile"`
		FullDescription string  `json:"full_description"`
		IsOfficial      bool    `json:"is_official"`
		IsPrivate       bool    `json:"is_private"`
		IsTrusted       bool    `json:"is_trusted"`
		Name            string  `json:"name"`
		Namespace       string  `json:"namespace"`
		Owner           string  `json:"owner"`
		RepoName        string  `json:"repo_name"`
		RepoURL         string  `json:"repo_url"`
		StarCount       int     `json:"star_count"`
		Status          string  `json:"status"`
	} `json:"repository"`
}
