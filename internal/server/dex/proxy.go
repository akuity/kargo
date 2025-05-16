package dex

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/kelseyhightower/envconfig"
)

// ProxyConfig represents configuration for a reverse proxy to a Dex server.
type ProxyConfig struct {
	// ServerAddr is the address of the target Dex server, beginning with https://
	ServerAddr string `envconfig:"DEX_SERVER_ADDRESS" required:"true"`
	// CACertPath optionally specifies the path to a CA cert file used for
	// verifying the target Dex server's TLS certificate.
	CACertPath string `envconfig:"DEX_CA_CERT_PATH"`
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

	var caCertPool *x509.CertPool
	if cfg.CACertPath != "" {
		caCertBytes, err := os.ReadFile(cfg.CACertPath)
		if err != nil {
			return nil, fmt.Errorf("error reading CA cert file %q: %w", cfg.CACertPath, err)
		}
		if caCertPool, err = buildCACertPool(caCertBytes); err != nil {
			return nil, fmt.Errorf("error building CA cert pool: %w", err)
		}
	}

	transport := cleanhttp.DefaultPooledTransport()
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    caCertPool,
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport

	return proxy, nil
}

// buildCACertPool returns a *x509.CertPool built from the provided bytes, which
// are assumed to represent a PEM-encoded CA certificate.
func buildCACertPool(caCertBytes []byte) (*x509.CertPool, error) {
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCertBytes); !ok {
		return nil, errors.New("invalid CA cert data")
	}
	return caCertPool, nil
}
