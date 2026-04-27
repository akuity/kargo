package webhook

import (
	"regexp"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	// KargoNamespace is the namespace in which Kargo is installed.
	KargoNamespace string `envconfig:"KARGO_NAMESPACE" required:"true"`
	// RawControlplaneUserRegex is a regular expression to match the username in
	// admission request to distinguish if the request is coming from controlplane.
	RawControlplaneUserRegex string         `envconfig:"CONTROLPLANE_USER_REGEX"`
	ControlplaneUserRegex    *regexp.Regexp `ignored:"true"`
	// ManagementControllerUsername is the exact username (typically a service
	// account name) of the management controller. This is used where only the
	// management controller (not the API server or other controlplane
	// components) should be permitted to act.
	ManagementControllerUsername string `envconfig:"MANAGEMENT_CONTROLLER_USERNAME"`
	// ExternalWebhooksServerUsername is the exact username (typically a service
	// account name) of the external webhooks server. When an admission request
	// originates from this subject, the "promote" verb authorization check is
	// bypassed, as the external webhooks server is permitted to refresh running
	// Promotions on behalf of webhook callers without holding that permission
	// itself.
	ExternalWebhooksServerUsername string `envconfig:"EXTERNAL_WEBHOOKS_SERVER_USERNAME"`
}

func ConfigFromEnv() Config {
	var cfg Config
	envconfig.MustProcess("", &cfg)

	if cfg.RawControlplaneUserRegex != "" {
		cfg.ControlplaneUserRegex = regexp.MustCompile(cfg.RawControlplaneUserRegex)
	}
	return cfg
}
