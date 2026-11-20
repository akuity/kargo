package dex

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/go-cleanhttp"

	"github.com/akuity/kargo/pkg/logging"
)

// NewTransport returns an http.RoundTripper for making requests to the Dex
// server described by cfg. If cfg specifies a CA cert file, the returned
// transport trusts only that CA and rereads the file, at most once per
// cfg.CACertReloadInterval, so that a reissued CA cert (e.g. one renewed by
// cert-manager and updated in a mounted Secret) is picked up without a
// restart.
func NewTransport(cfg ProxyConfig) (http.RoundTripper, error) {
	if cfg.CACertPath == "" {
		return newTLSTransport(nil), nil
	}
	caCertBytes, err := os.ReadFile(cfg.CACertPath)
	if err != nil {
		return nil, fmt.Errorf("error reading CA cert file %q: %w", cfg.CACertPath, err)
	}
	caCertPool, err := buildCACertPool(caCertBytes)
	if err != nil {
		return nil, fmt.Errorf("error building CA cert pool: %w", err)
	}
	return &caReloadingTransport{
		caCertPath:  cfg.CACertPath,
		interval:    cfg.CACertReloadInterval,
		lastChecked: time.Now(),
		caCertBytes: caCertBytes,
		transport:   newTLSTransport(caCertPool),
	}, nil
}

// caReloadingTransport is an http.RoundTripper that delegates to an
// *http.Transport trusting the CA cert in a file, and replaces that
// *http.Transport whenever the file's contents change.
type caReloadingTransport struct {
	caCertPath string
	// interval is the minimum time between checks of the CA cert file. Zero
	// means the file is checked before every request.
	interval time.Duration

	mu          sync.Mutex
	lastChecked time.Time
	caCertBytes []byte
	transport   *http.Transport
}

// RoundTrip implements http.RoundTripper.
func (c *caReloadingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return c.currentTransport(req.Context()).RoundTrip(req)
}

// currentTransport returns the *http.Transport to use for a request, first
// replacing it if the interval has elapsed and the CA cert file has changed.
// If the file cannot be read or does not contain a valid CA cert, the
// existing transport is kept so that a bad update cannot break a working
// connection to Dex.
func (c *caReloadingTransport) currentTransport(ctx context.Context) *http.Transport {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.lastChecked) < c.interval {
		return c.transport
	}
	c.lastChecked = time.Now()
	logger := logging.LoggerFromContext(ctx).WithValues("path", c.caCertPath)
	caCertBytes, err := os.ReadFile(c.caCertPath)
	if err != nil {
		logger.Error(err, "error reading Dex CA cert file; keeping current CA cert")
		return c.transport
	}
	if bytes.Equal(caCertBytes, c.caCertBytes) {
		return c.transport
	}
	caCertPool, err := buildCACertPool(caCertBytes)
	if err != nil {
		logger.Error(err, "error building Dex CA cert pool; keeping current CA cert")
		return c.transport
	}
	previous := c.transport
	c.transport = newTLSTransport(caCertPool)
	c.caCertBytes = caCertBytes
	previous.CloseIdleConnections()
	logger.Info("reloaded Dex CA cert")
	return c.transport
}

// newTLSTransport returns a pooled *http.Transport that verifies servers
// against caCertPool, or against the system roots if caCertPool is nil.
func newTLSTransport(caCertPool *x509.CertPool) *http.Transport {
	transport := cleanhttp.DefaultPooledTransport()
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    caCertPool,
	}
	return transport
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
