package external

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/expr-lang/expr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/git"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/logging"
)

const (
	generic              = "generic"
	genericSecretDataKey = "secret"
)

func init() {
	registry.register(
		generic,
		webhookReceiverRegistration{
			predicate: func(cfg kargoapi.WebhookReceiverConfig) bool {
				return cfg.Generic != nil
			},
			factory: newGenericWebhookReceiver,
		},
	)
}

// genericWebhookReceiver is an implementation of WebhookReceiver that handles
// inbound webhooks from any source using user-defined logic.
type genericWebhookReceiver struct {
	*baseWebhookReceiver
	cfg kargoapi.GenericWebhookReceiverConfig
}

// newGenericWebhookReceiver returns a new instance of
// genericWebhookReceiver.
func newGenericWebhookReceiver(
	c client.Client,
	project string,
	cfg kargoapi.WebhookReceiverConfig,
) WebhookReceiver {
	return &genericWebhookReceiver{
		baseWebhookReceiver: &baseWebhookReceiver{
			client:     c,
			project:    project,
			secretName: cfg.Bitbucket.SecretRef.Name,
		},
		cfg: *cfg.Generic,
	}
}

// getReceiverType implements WebhookReceiver.
func (g *genericWebhookReceiver) getReceiverType() string {
	return generic
}

// getSecretValues implements WebhookReceiver.
func (g *genericWebhookReceiver) getSecretValues(
	secretData map[string][]byte,
) ([]string, error) {
	secretValue, ok := secretData[genericSecretDataKey]
	if !ok {
		return nil,
			errors.New("Secret data is not valid for a Generic WebhookReceiver")
	}
	return []string{string(secretValue)}, nil
}

// getHandler implements WebhookReceiver.
func (g *genericWebhookReceiver) getHandler(requestBody []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger := logging.LoggerFromContext(ctx)

		env := map[string]any{
			"request": map[string]any{
				"header":  r.Header.Get,
				"headers": r.Header,
			},
		}

		var parsedBody any
		if err := json.Unmarshal(requestBody, &parsedBody); err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("invalid request body"), http.StatusBadRequest),
			)
			return
		}
		env["request"].(map[string]any)["body"] = parsedBody // nolint: forcetypeassert

		program, err := expr.Compile(g.cfg.WarehouseRefresh.Predicate)
		if err != nil {
			logger.Error(err, "error compiling warehouse refresh predicate")
			xhttp.WriteErrorJSON(w, err)
			return
		}

		resultAny, err := expr.Run(program, env)
		if err != nil {
			logger.Error(err, "error evaluating warehouse refresh predicate")
			xhttp.WriteErrorJSON(w, err)
			return
		}
		result, ok := resultAny.(bool)
		if !ok {
			logger.Error(nil, "predicate result is not a boolean")
			xhttp.WriteErrorJSON(w, nil)
			return
		}
		if !result {
			xhttp.WriteResponseJSON(
				w,
				http.StatusOK,
				map[string]string{"msg": "no action taken"},
			)
			return
		}

		program, err = expr.Compile(g.cfg.WarehouseRefresh.RepoURL)
		if err != nil {
			logger.Error(err, "error compiling repo URL expression")
			xhttp.WriteErrorJSON(w, err)
			return
		}

		if resultAny, err = expr.Run(program, env); err != nil {
			logger.Error(err, "error evaluating repo URL expression")
			xhttp.WriteErrorJSON(w, err)
			return
		}

		repoURL, ok := resultAny.(string)
		if !ok {
			logger.Error(nil, "repo URL expression did not evaluate to a string")
			xhttp.WriteErrorJSON(w, nil)
			return
		}

		repoURL = git.NormalizeURL(repoURL)

		logger = logger.WithValues("repoURL", repoURL)
		ctx = logging.ContextWithLogger(ctx, logger)

		refreshWarehouses(ctx, w, g.client, g.project, repoURL)
	})
}
