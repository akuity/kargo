package oidc

import (
	"github.com/kelseyhightower/envconfig"
)

// Config represents configuration for an public/untrusted OpenID Connect
// client. The API server returns this configuration to clients that request it,
// thereby communicating how to proceed with the authorization code flow.
type Config struct {
	// IssuerURL is the issuer URL provided by the OpenID Connect identity
	// provider.
	IssuerURL string `envconfig:"OIDC_ISSUER_URL" required:"true"`
	// Client ID is the client ID provided by the OpenID Connect identity
	// provider.
	ClientID string `envconfig:"OIDC_CLIENT_ID" required:"true"`
	// CLIClientID is the client ID provided by the OpenID Connect identity
	// provider for CLI login.
	CLIClientID string `envconfig:"OIDC_CLI_CLIENT_ID"`
	// Scopes are the scopes to be requested during the authorization code flow.
	Scopes string `envconfig:"OIDC_SCOPES"`

	// GlobalServiceAccountNamespaces is the list of namespaces to look up
	// for shared service accounts.
	GlobalServiceAccountNamespaces []string `envconfig:"GLOBAL_SERVICE_ACCOUNT_NAMESPACES"`
}

// ConfigFromEnv returns a Config populated from environment variables.
func ConfigFromEnv() Config {
	cfg := Config{}
	envconfig.MustProcess("", &cfg)
	return cfg
}
