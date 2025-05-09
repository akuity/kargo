package external

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	gh "github.com/google/go-github/v71/github"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/logging"
)

var (
	// errMissingSignature is returned when the 'X-Hub-Signature-256'
	// header is not found or empty.
	errMissingSignature = errors.New("missing signature")
	// errSecretUnset is returned when the 'GH_WEBHOOK_SECRET'
	// environment variable is empty.
	errSecretUnset = errors.New("secret is unset")
)

// githubHandler handles push events for github.
// After the request has been authenticated,
// the kubeclient is queried for all warehouses that contain a subscription
// to the repo in question. Those warehouses are then patched with a special
// annotation that signals down stream logic to refresh the warehouse.
func githubHandler(c client.Client) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.LoggerFromContext(ctx)
		logger.Debug("identifying source repository")

		secret, ok := os.LookupEnv("GH_WEBHOOK_SECRET")
		if !ok {
			logger.Error(errSecretUnset, "environment misconfiguration")
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					// keep it vague, no need to leak config details
					errors.New("unexpected server error"),
					http.StatusInternalServerError,
				),
			)
			return
		}

		signature := r.Header.Get(gh.SHA256SignatureHeader)
		if signature == "" {
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					errMissingSignature,
					http.StatusUnauthorized,
				),
			)
			return
		}

		bodyBytes, code, err := limitRead(r.Body)
		if err != nil {
			xhttp.WriteErrorJSON(w,
				xhttp.Error(err, code),
			)
			return
		}

		if err = gh.ValidateSignature(
			signature,
			bodyBytes,
			[]byte(secret),
		); err != nil {
			logger.Error(err, "failed to validate signature")
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					// same here
					errors.New("unauthorized"),
					http.StatusUnauthorized,
				),
			)
			return
		}

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
		result, err := refresh(ctx, c, logger, repo)
		if err != nil {
			xhttp.WriteErrorJSON(w,
				xhttp.Error(err, http.StatusInternalServerError),
			)
			return
		}

		logger.Debug("execution complete",
			"total", result.totalWarehouses,
			"num-failures", result.numFailures,
		)

		if result.numFailures > 0 {
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					fmt.Errorf("failed to refresh %d of %d warehouses",
						result.numFailures,
						result.totalWarehouses,
					),
					http.StatusInternalServerError,
				),
			)
			return
		}

		xhttp.WriteResponseJSON(w,
			http.StatusOK,
			map[string]string{
				"msg": fmt.Sprintf("refreshed %d warehouses",
					result.totalWarehouses,
				),
			},
		)
	})
}
