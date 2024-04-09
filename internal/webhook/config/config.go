package config

import (
	"strings"

	"github.com/kelseyhightower/envconfig"
	"k8s.io/apimachinery/pkg/types"
)

type WebhookConfig struct {
	// RawControlplaneServiceAccounts is a list of controlplane service accounts.
	// Each service accounts should be formatted in "namespace/name" format and separated by commas.
	RawControlplaneServiceAccounts []string                          `envconfig:"CONTROLPLANE_SERVICE_ACCOUNTS"`
	ControlplaneServiceAccounts    map[types.NamespacedName]struct{} `envconfig:"-"`
}

func WebhookConfigFromEnv() WebhookConfig {
	var cfg WebhookConfig
	envconfig.MustProcess("", &cfg)

	cfg.ControlplaneServiceAccounts = make(map[types.NamespacedName]struct{}, len(cfg.RawControlplaneServiceAccounts))
	for _, account := range cfg.RawControlplaneServiceAccounts {
		account = strings.TrimSpace(account)
		if account == "" {
			continue
		}
		segments := strings.Split(account, "/")
		if len(segments) != 2 {
			panic("invalid controlplane service account: " + account)
		}
		cfg.ControlplaneServiceAccounts[types.NamespacedName{
			Namespace: segments[0],
			Name:      segments[1],
		}] = struct{}{}
	}

	return cfg
}
