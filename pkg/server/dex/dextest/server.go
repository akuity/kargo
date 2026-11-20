// Package dextest provides utilities for testing code that talks to Dex over
// TLS.
package dextest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// RotatingTLSServer is a TLS test server whose certificate can be replaced
// while it runs, the way cert-manager reissues Dex's certificate.
type RotatingTLSServer struct {
	*httptest.Server
	cert atomic.Pointer[tls.Certificate]
}

// NewRotatingTLSServer starts a RotatingTLSServer serving handler with a new
// self-signed certificate and returns it along with that certificate,
// PEM-encoded. The server is closed when the test ends.
func NewRotatingTLSServer(
	t *testing.T,
	handler http.Handler,
) (*RotatingTLSServer, []byte) {
	t.Helper()
	srv := &RotatingTLSServer{Server: httptest.NewUnstartedServer(handler)}
	certPEM := srv.Rotate(t)
	// GetConfigForClient is consulted on every handshake, unlike
	// GetCertificate, which httptest would bypass with its own certificate.
	srv.TLS = &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) {
			return &tls.Config{
				MinVersion:   tls.VersionTLS12,
				Certificates: []tls.Certificate{*srv.cert.Load()},
			}, nil
		},
	}
	// Force a new handshake per request so each one sees the current cert.
	srv.Config.SetKeepAlivesEnabled(false)
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv, certPEM
}

// Rotate replaces the server's certificate with a new self-signed one and
// returns that certificate PEM-encoded. As with cert-manager's SelfSigned
// issuer, the certificate is its own trust anchor.
func (s *RotatingTLSServer) Rotate(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "kargo-dex-server"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	certDER, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&key.PublicKey,
		key,
	)
	require.NoError(t, err)
	s.cert.Store(&tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	})
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
}

// ReplaceFile replaces the file at path the way the kubelet updates a mounted
// Secret: by swapping in a new file rather than writing to the existing one.
func ReplaceFile(t *testing.T, path string, data []byte) {
	t.Helper()
	tmp := path + ".tmp"
	require.NoError(t, os.WriteFile(tmp, data, 0o600))
	require.NoError(t, os.Rename(tmp, path))
}
