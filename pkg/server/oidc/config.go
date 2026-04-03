package oidc

import (
	"fmt"
	"strings"

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
	// UsernameClaim is the claim to use as the username for the user.
	UsernameClaim string `envconfig:"OIDC_USERNAME_CLAIM" default:"email"`

	// AdditionalParameters are any extra key/value parameters to pass to the
	// OIDC provider during the authorization code flow (e.g. audience,
	// connector_id, domain_hint). Configured as a comma-separated list of
	// key=value pairs: "audience=https://kubernetes.default.svc,domain_hint=example.com".
	AdditionalParameters AdditionalParameters `envconfig:"OIDC_ADDITIONAL_PARAMETERS"`

	// GlobalServiceAccountNamespaces is the list of namespaces to look up
	// for shared service accounts.
	GlobalServiceAccountNamespaces []string `envconfig:"GLOBAL_SERVICE_ACCOUNT_NAMESPACES"`
}

// AdditionalParameters is a map of extra authorization parameters to pass to
// an OIDC provider. It implements envconfig.Decoder to support KEY=VALUE pairs
// separated by commas, e.g. "audience=https://kubernetes.default.svc,domain_hint=corp.example.com".
// Splitting on the first '=' allows values to contain colons and other special characters.
type AdditionalParameters map[string]string

// Decode implements envconfig.Decoder.
func (p *AdditionalParameters) Decode(value string) error {
	m := make(AdditionalParameters)
	if value == "" {
		*p = m
		return nil
	}
	for pair := range strings.SplitSeq(value, ",") {
		k, v, ok := strings.Cut(pair, "=")
		if !ok {
			return fmt.Errorf(
				"invalid OIDC additional parameter %q: expected key=value format",
				pair,
			)
		}
		m[k] = v
	}
	*p = m
	return nil
}

// ConfigFromEnv returns a Config populated from environment variables.
func ConfigFromEnv() Config {
	cfg := Config{}
	envconfig.MustProcess("", &cfg)
	cfg.DefaultScopes = []string{"openid", "profile", "email"}
	return cfg
}
