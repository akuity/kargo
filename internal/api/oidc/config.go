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
	// DefaultScopes are the scopes to always be requested during the authorization code flow.
	DefaultScopes []string
	// AdditionalScopes are any more scopes to be requested during the authorization code flow
	// on top of the default Scopes.
	AdditionalScopes []string `envconfig:"OIDC_ADDITIONAL_SCOPES"`

	// GlobalServiceAccountNamespaces is the list of namespaces to look up
	// for shared service accounts.
	GlobalServiceAccountNamespaces []string `envconfig:"GLOBAL_SERVICE_ACCOUNT_NAMESPACES"`
}

// ConfigFromEnv returns a Config populated from environment variables.
func ConfigFromEnv() Config {
	cfg := Config{}
	envconfig.MustProcess("", &cfg)
	cfg.DefaultScopes = []string{"openid", "profile", "email", "groups"}
	return cfg
}
