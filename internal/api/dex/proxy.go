package dex

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
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
		return nil, errors.Wrapf(err, "error parsing URL %q", cfg.ServerAddr)
	}

	var caCertPool *x509.CertPool
	if cfg.CACertPath != "" {
		caCertBytes, err := os.ReadFile(cfg.CACertPath)
		if err != nil {
			return nil,
				errors.Wrapf(err, "error reading CA cert file %q", cfg.CACertPath)
		}
		if caCertPool, err = buildCACertPool(caCertBytes); err != nil {
			return nil, errors.Wrapf(err, "error building CA cert pool")
		}
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    caCertPool,
		},
	}

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
