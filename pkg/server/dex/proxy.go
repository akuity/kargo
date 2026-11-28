package dex

import (
	"fmt"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// ProxyConfig represents configuration for a reverse proxy to a Dex server.
type ProxyConfig struct {
	// ServerAddr is the address of the target Dex server, beginning with https://
	ServerAddr string `envconfig:"DEX_SERVER_ADDRESS" required:"true"`
	// CACertPath optionally specifies the path to a CA cert file used for
	// verifying the target Dex server's TLS certificate.
	CACertPath string `envconfig:"DEX_CA_CERT_PATH"`
	// CACertReloadInterval is the minimum time between checks of the file at
	// CACertPath for a new CA cert. Zero means the file is checked before
	// every request.
	CACertReloadInterval time.Duration `envconfig:"DEX_CA_CERT_RELOAD_INTERVAL" default:"30s"`
}

// ProxyConfigFromEnv returns a ProxyConfig populated from environment
// variables.
func ProxyConfigFromEnv() ProxyConfig {
	cfg := ProxyConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// NewProxy returns an *httputil.ReverseProxy that proxies requests to a Dex
// server.
func NewProxy(cfg ProxyConfig) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(cfg.ServerAddr)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL %q: %w", cfg.ServerAddr, err)
	}
	transport, err := NewTransport(cfg)
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport
	return proxy, nil
}
