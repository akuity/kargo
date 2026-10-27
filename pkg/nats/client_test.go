package nats

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/stretchr/testify/require"
)

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("NATS_SERVER_URL", "nats://nats.example.com:4222")
	t.Setenv("NATS_SEED_KEY_FILE", "/etc/kargo/nats/seed.nk")
	t.Setenv("NATS_CREDS_FILE", "/etc/kargo/nats/user.creds")
	t.Setenv("NATS_TOKEN", "fake-token")
	t.Setenv("NATS_USERNAME", "fake-user")
	t.Setenv("NATS_PASSWORD", "fake-password")
	t.Setenv("NATS_TLS_CA_FILE", "/etc/kargo/nats/ca.crt")
	t.Setenv("NATS_TLS_CERT_FILE", "/etc/kargo/nats/tls.crt")
	t.Setenv("NATS_TLS_KEY_FILE", "/etc/kargo/nats/tls.key")
	t.Setenv("NATS_CLIENT_NAME", "fake-client")
	t.Setenv("NATS_CONNECT_TIMEOUT", "5s")
	require.Equal(
		t,
		Config{
			ServerURL:      "nats://nats.example.com:4222",
			SeedKeyFile:    "/etc/kargo/nats/seed.nk",
			CredsFile:      "/etc/kargo/nats/user.creds",
			Token:          "fake-token",
			Username:       "fake-user",
			Password:       "fake-password",
			TLSCAFile:      "/etc/kargo/nats/ca.crt",
			TLSCertFile:    "/etc/kargo/nats/tls.crt",
			TLSKeyFile:     "/etc/kargo/nats/tls.key",
			ClientName:     "fake-client",
			ConnectTimeout: 5 * time.Second,
		},
		ConfigFromEnv(),
	)
}

func TestConfigConnect(t *testing.T) {
	srv := natstest.RunRandClientPortServer()
	t.Cleanup(srv.Shutdown)

	testCases := []struct {
		name   string
		cfg    Config
		opts   []nats.Option
		assert func(*testing.T, *nats.Conn, error)
	}{
		{
			name: "server URL not specified",
			assert: func(t *testing.T, _ *nats.Conn, err error) {
				require.ErrorContains(t, err, "NATS_SERVER_URL is required")
			},
		},
		{
			name: "invalid authentication configuration",
			cfg: Config{
				ServerURL: srv.ClientURL(),
				Username:  "fake-user",
			},
			assert: func(t *testing.T, _ *nats.Conn, err error) {
				require.ErrorContains(
					t, err,
					"NATS_USERNAME and NATS_PASSWORD must be specified together",
				)
			},
		},
		{
			name: "server unreachable",
			cfg: Config{
				// Nothing listens on port 1
				ServerURL:      "nats://127.0.0.1:1",
				ConnectTimeout: 100 * time.Millisecond,
			},
			assert: func(t *testing.T, _ *nats.Conn, err error) {
				require.ErrorContains(t, err, "error connecting to NATS server")
			},
		},
		{
			name: "success",
			cfg: Config{
				ServerURL:  srv.ClientURL(),
				ClientName: "fake-client",
			},
			assert: func(t *testing.T, conn *nats.Conn, err error) {
				require.NoError(t, err)
				require.True(t, conn.IsConnected())
				require.Equal(t, "fake-client", conn.Opts.Name)
				// Reconnect attempts must never be exhausted
				require.Equal(t, -1, conn.Opts.MaxReconnect)
				// Default connect timeout applies when none is specified
				require.Equal(t, 2*time.Second, conn.Opts.Timeout)
			},
		},
		{
			name: "caller-supplied options take precedence",
			cfg: Config{
				ServerURL:  srv.ClientURL(),
				ClientName: "fake-client",
			},
			opts: []nats.Option{nats.Name("overridden")},
			assert: func(t *testing.T, conn *nats.Conn, err error) {
				require.NoError(t, err)
				require.Equal(t, "overridden", conn.Opts.Name)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			conn, err := testCase.cfg.Connect(context.Background(), testCase.opts...)
			if conn != nil {
				t.Cleanup(conn.Close)
			}
			testCase.assert(t, conn, err)
		})
	}
}

func TestConfigAuthOptions(t *testing.T) {
	dir := t.TempDir()

	userKey, err := nkeys.CreateUser()
	require.NoError(t, err)
	userPubKey, err := userKey.PublicKey()
	require.NoError(t, err)
	userSeed, err := userKey.Seed()
	require.NoError(t, err)
	seedFile := filepath.Join(dir, "seed.nk")
	require.NoError(t, os.WriteFile(seedFile, userSeed, 0o600))

	// The library does not read this until it connects, so its content is
	// irrelevant here.
	credsFile := filepath.Join(dir, "user.creds")
	require.NoError(t, os.WriteFile(credsFile, nil, 0o600))

	// A self-signed certificate serves as both a CA and a client certificate.
	certFile, keyFile := writeSelfSignedCert(t, dir)

	testCases := []struct {
		name   string
		cfg    Config
		assert func(*testing.T, nats.Options, error)
	}{
		{
			name: "no authentication",
			assert: func(t *testing.T, opts nats.Options, err error) {
				require.NoError(t, err)
				require.Equal(t, nats.GetDefaultOptions(), opts)
			},
		},
		{
			name: "multiple authentication methods specified",
			cfg: Config{
				SeedKeyFile: seedFile,
				CredsFile:   credsFile,
				Token:       "fake-token",
				Username:    "fake-user",
				Password:    "fake-password",
			},
			assert: func(t *testing.T, _ nats.Options, err error) {
				require.ErrorContains(t, err, "at most one of")
			},
		},
		{
			name: "username specified without password",
			cfg:  Config{Username: "fake-user"},
			assert: func(t *testing.T, _ nats.Options, err error) {
				require.ErrorContains(
					t, err,
					"NATS_USERNAME and NATS_PASSWORD must be specified together",
				)
			},
		},
		{
			name: "password specified without username",
			cfg:  Config{Password: "fake-password"},
			assert: func(t *testing.T, _ nats.Options, err error) {
				require.ErrorContains(
					t, err,
					"NATS_USERNAME and NATS_PASSWORD must be specified together",
				)
			},
		},
		{
			name: "client certificate specified without key",
			cfg:  Config{TLSCertFile: certFile},
			assert: func(t *testing.T, _ nats.Options, err error) {
				require.ErrorContains(
					t, err,
					"NATS_TLS_CERT_FILE and NATS_TLS_KEY_FILE must be specified together",
				)
			},
		},
		{
			name: "seed key file does not exist",
			cfg:  Config{SeedKeyFile: filepath.Join(dir, "missing.nk")},
			assert: func(t *testing.T, _ nats.Options, err error) {
				require.ErrorContains(t, err, "error loading NATS nkey seed")
			},
		},
		{
			name: "nkey",
			cfg:  Config{SeedKeyFile: seedFile},
			assert: func(t *testing.T, opts nats.Options, err error) {
				require.NoError(t, err)
				require.Equal(t, userPubKey, opts.Nkey)
			},
		},
		{
			name: "creds file does not exist",
			cfg:  Config{CredsFile: filepath.Join(dir, "missing.creds")},
			assert: func(t *testing.T, _ nats.Options, err error) {
				require.ErrorContains(t, err, "error loading NATS credentials")
			},
		},
		{
			name: "creds file",
			cfg:  Config{CredsFile: credsFile},
			assert: func(t *testing.T, opts nats.Options, err error) {
				require.NoError(t, err)
				require.NotNil(t, opts.UserJWT)
				require.NotNil(t, opts.SignatureCB)
			},
		},
		{
			name: "token",
			cfg:  Config{Token: "fake-token"},
			assert: func(t *testing.T, opts nats.Options, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-token", opts.Token)
			},
		},
		{
			name: "username and password",
			cfg: Config{
				Username: "fake-user",
				Password: "fake-password",
			},
			assert: func(t *testing.T, opts nats.Options, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-user", opts.User)
				require.Equal(t, "fake-password", opts.Password)
			},
		},
		{
			name: "TLS combined with other credentials",
			cfg: Config{
				CredsFile:   credsFile,
				TLSCAFile:   certFile,
				TLSCertFile: certFile,
				TLSKeyFile:  keyFile,
			},
			assert: func(t *testing.T, opts nats.Options, err error) {
				require.NoError(t, err)
				require.NotNil(t, opts.UserJWT)
				require.True(t, opts.Secure)
				require.NotNil(t, opts.RootCAsCB)
				require.NotNil(t, opts.TLSCertCB)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			opts := nats.GetDefaultOptions()
			natsOpts, err := testCase.cfg.authOptions()
			for _, opt := range natsOpts {
				require.NoError(t, opt(&opts))
			}
			testCase.assert(t, opts, err)
		})
	}
}

// writeSelfSignedCert writes a PEM-encoded self-signed certificate and its
// private key to the given directory and returns their paths.
func writeSelfSignedCert(t *testing.T, dir string) (string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "fake"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&key.PublicKey,
		key,
	)
	require.NoError(t, err)
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	certFile := filepath.Join(dir, "tls.crt")
	require.NoError(t, os.WriteFile(
		certFile,
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}),
		0o600,
	))
	keyFile := filepath.Join(dir, "tls.key")
	require.NoError(t, os.WriteFile(
		keyFile,
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}),
		0o600,
	))
	return certFile, keyFile
}
