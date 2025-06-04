package external

import (
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
)

// route routes inbound webhook requests to a sender-specific handler.
func (s *server) route(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := logging.LoggerFromContext(ctx).WithValues("path", r.URL.Path)
	ctx = logging.ContextWithLogger(ctx, logger)

	logger.Debug("routing inbound webhook request")

	// Strategy: Search for a ProjectConfig that defines a webhook receiver
	// for the path. If none is found, check if ClusterConfig defines a
	// webhook receiver for the path. If neither is found, return a 404.
	//
	// Find the config for the webhook receiver that matches the path and
	// use that to obtain an appropriate handler to delegate the request to.

	var project string
	var secretsNamespace string
	var receiverCfgs []kargoapi.WebhookReceiverConfig
	var receivers []kargoapi.WebhookReceiverDetails

	projectConfigs := kargoapi.ProjectConfigList{}
	if err := s.client.List(
		ctx,
		&projectConfigs,
		client.MatchingFields{
			indexer.ProjectConfigsByWebhookReceiverPathsField: r.URL.Path,
		},
	); err != nil {
		logger.Error(err, "error listing ProjectConfigs")
		xhttp.WriteErrorJSON(w, err)
		return
	}

	// At most one ProjectConfig should match the path
	if len(projectConfigs.Items) == 1 {
		project = projectConfigs.Items[0].Namespace
		logger.Debug("found ProjectConfig", "project", project)
		secretsNamespace = project
		receiverCfgs = projectConfigs.Items[0].Spec.WebhookReceivers
		receivers = projectConfigs.Items[0].Status.WebhookReceivers
	} else {
		logger.Debug(
			"no ProjectConfigs define a WebhookReceiver for the path; "+
				"will check ClusterConfig",
			"path", r.URL.Path,
		)
		clusterCfg, err := api.GetClusterConfig(ctx, s.client)
		if err != nil {
			logger.Error(err, "error getting ClusterConfig")
			xhttp.WriteErrorJSON(w, err)
			return
		}
		if clusterCfg == nil {
			logger.Debug("found no ClusterConfig")
			xhttp.WriteErrorJSON(w, xhttp.Error(nil, http.StatusNotFound))
			return
		}
		logger.Debug("found ClusterConfig")
		secretsNamespace = s.cfg.ClusterSecretsNamespace
		receiverCfgs = clusterCfg.Spec.WebhookReceivers
		receivers = clusterCfg.Status.WebhookReceivers
	}

	receiverCfg := s.getWebhookReceiverConfig(receiverCfgs, receivers, r.URL.Path)
	if receiverCfg == nil {
		logger.Debug("found no WebhookReceiverConfig")
		xhttp.WriteErrorJSON(w, xhttp.Error(nil, http.StatusNotFound))
		return
	}
	logger.Debug("found WebhookReceiverConfig")

	receiver, err := NewReceiver(
		ctx,
		s.client,
		s.cfg.BaseURL,
		project,
		secretsNamespace,
		*receiverCfg,
	)
	if err != nil {
		logger.Error(err, "error creating WebhookReceiver")
		xhttp.WriteErrorJSON(w, err)
		return
	}

	receiver.GetHandler()(w, r)
}

// getWebhookReceiverConfig attempts to find and return WebhookReceiverConfig
// corresponding to the provided path, using the provided WebhookReceivers as
// associative entities. If no matching WebhookReceiverConfig is found, it
// returns nil.
func (s *server) getWebhookReceiverConfig(
	receiverCfgs []kargoapi.WebhookReceiverConfig,
	receivers []kargoapi.WebhookReceiverDetails,
	path string,
) *kargoapi.WebhookReceiverConfig {
	for _, receiver := range receivers {
		if path == receiver.Path {
			for _, receiverCfg := range receiverCfgs {
				if receiverCfg.Name == receiver.Name {
					return &receiverCfg
				}
			}
			return nil
		}
	}
	return nil
}
