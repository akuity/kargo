package http

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	const defaultPort = 8080
	const overriddenPort = 1234
	testCases := []struct {
		name       string
		config     *ServerConfig
		assertions func(s *server)
	}{
		{
			name: "without optional config",
			assertions: func(s *server) {
				require.NotNil(t, s.config)
				require.Equal(t, defaultPort, s.config.Port)
			},
		},
		{
			name:   "with port not specified",
			config: &ServerConfig{},
			assertions: func(s *server) {
				require.NotNil(t, s.config)
				require.Equal(t, defaultPort, s.config.Port)
			},
		},
		{
			name: "with port specified",
			config: &ServerConfig{
				Port: overriddenPort,
			},
			assertions: func(s *server) {
				require.NotNil(t, s.config)
				require.Equal(t, overriddenPort, s.config.Port)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server, ok := NewServer(
				http.HandlerFunc(
					func(rw http.ResponseWriter, r *http.Request) {},
				),
				testCase.config,
			).(*server)
			require.True(t, ok)
			require.NotNil(t, server.handler)
			testCase.assertions(server)
		})
	}
}

func TestListenAndServe(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func() *ServerConfig
		assertions func(ctx context.Context, err error)
	}{
		{
			name: "TLS enabled; missing cert path",
			setup: func() *ServerConfig {
				return &ServerConfig{
					TLSEnabled: true,
				}
			},
			assertions: func(ctx context.Context, err error) {
				require.Error(t, err)
				require.Equal(
					t,
					"TLS was enabled, but no certificate path was specified",
					err.Error(),
				)
			},
		},
		{
			name: "TLS enabled; missing key path",
			setup: func() *ServerConfig {
				return &ServerConfig{
					TLSEnabled:  true,
					TLSCertPath: "/app/certs/tls.crt",
				}
			},
			assertions: func(ctx context.Context, err error) {
				require.Error(t, err)
				require.Equal(
					t,
					"TLS was enabled, but no key path was specified",
					err.Error(),
				)
			},
		},
		{
			name: "TLS enabled; cert not found",
			setup: func() *ServerConfig {
				return &ServerConfig{
					TLSEnabled:  true,
					TLSCertPath: "/app/certs/tls.crt",
					TLSKeyPath:  "/app/certs/tls.key",
				}
			},
			assertions: func(ctx context.Context, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"no TLS certificate found at path",
				)
			},
		},
		{
			name: "TLS enabled; key not found",
			setup: func() *ServerConfig {
				certFile, err := os.CreateTemp("", "tls-*.crt")
				require.NoError(t, err)
				return &ServerConfig{
					TLSEnabled:  true,
					TLSCertPath: certFile.Name(),
					TLSKeyPath:  "/app/certs/tls.key",
				}
			},
			assertions: func(ctx context.Context, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"no TLS key found at path",
				)
			},
		},
		{
			name: "TLS enabled; invalid cert",
			setup: func() *ServerConfig {
				certFile, err := os.CreateTemp("", "tls-*.crt")
				require.NoError(t, err)
				keyFile, err := os.CreateTemp("", "tls-*.key")
				require.NoError(t, err)
				return &ServerConfig{
					Port:        8082,
					TLSEnabled:  true,
					TLSCertPath: certFile.Name(),
					TLSKeyPath:  keyFile.Name(),
				}
			},
			assertions: func(ctx context.Context, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"failed to find any PEM data in certificate input",
				)
			},
		},
		{
			name: "TLS enabled",
			setup: func() *ServerConfig {
				cert, key := generateCert(t)

				certFile, err := os.CreateTemp("", "tls-*.crt")
				require.NoError(t, err)
				defer certFile.Close()
				_, err = certFile.Write(cert)
				require.NoError(t, err)

				keyFile, err := os.CreateTemp("", "tls-*.key")
				require.NoError(t, err)
				defer keyFile.Close()
				_, err = keyFile.Write(key)
				require.NoError(t, err)

				return &ServerConfig{
					Port:        8081,
					TLSEnabled:  true,
					TLSCertPath: certFile.Name(),
					TLSKeyPath:  keyFile.Name(),
				}
			},
			assertions: func(ctx context.Context, err error) {
				require.Error(t, err)
				require.Equal(t, ctx.Err(), err)
			},
		},
		{
			name: "TLS not enabled",
			setup: func() *ServerConfig {
				return nil
			},
			assertions: func(ctx context.Context, err error) {
				require.Error(t, err)
				require.Equal(t, ctx.Err(), err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.setup()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			s := NewServer(nil, testCase.setup())
			err := s.ListenAndServe(ctx)
			testCase.assertions(ctx, err)
		})
	}
}

// generateCert generates and returns a PEM encoded, self-signed x.509v3 cert
// and corresponding PEM encoded private key. This cert and corresponding key
// are adequate for test purposes.
func generateCert(t *testing.T) ([]byte, []byte) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 4096)
	require.NoError(t, err)

	serialNumberUpperBound := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberUpperBound)
	require.NoError(t, err)

	certTemplate := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		NotBefore: time.Now().UTC(),
		NotAfter:  time.Now().UTC().Add(time.Hour * 24),
	}

	certBytes, err := x509.CreateCertificate(
		rand.Reader,
		certTemplate,
		certTemplate,
		&key.PublicKey,
		key,
	)
	require.NoError(t, err)

	return pem.EncodeToMemory(
			&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: certBytes,
			},
		), pem.EncodeToMemory(
			&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(key),
			},
		)
}
