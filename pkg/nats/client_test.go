package nats

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/stretchr/testify/require"
)

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("NATS_SERVER_URL", "nats://nats.example.com:4222")
	t.Setenv("NATS_SEED_KEY_FILE", "/etc/kargo/nats/seed.nk")
	t.Setenv("NATS_CLIENT_NAME", "fake-client")
	t.Setenv("NATS_CONNECT_TIMEOUT", "5s")
	require.Equal(
		t,
		Config{
			ServerURL:      "nats://nats.example.com:4222",
			SeedKeyFile:    "/etc/kargo/nats/seed.nk",
			ClientName:     "fake-client",
			ConnectTimeout: 5 * time.Second,
		},
		ConfigFromEnv(),
	)
}

func TestConfigConnect(t *testing.T) {
	// A server without authentication
	openSrv := natstest.RunRandClientPortServer()
	t.Cleanup(openSrv.Shutdown)

	// A server requiring nkey authentication
	userKey, err := nkeys.CreateUser()
	require.NoError(t, err)
	userPubKey, err := userKey.PublicKey()
	require.NoError(t, err)
	userSeed, err := userKey.Seed()
	require.NoError(t, err)
	seedFile := filepath.Join(t.TempDir(), "seed.nk")
	require.NoError(t, os.WriteFile(seedFile, userSeed, 0o600))
	authSrvOpts := natstest.DefaultTestOptions
	authSrvOpts.Port = natsserver.RANDOM_PORT
	authSrvOpts.Nkeys = []*natsserver.NkeyUser{{Nkey: userPubKey}}
	authSrv := natstest.RunServer(&authSrvOpts)
	t.Cleanup(authSrv.Shutdown)

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
			name: "seed key file does not exist",
			cfg: Config{
				ServerURL:   authSrv.ClientURL(),
				SeedKeyFile: filepath.Join(t.TempDir(), "missing.nk"),
			},
			assert: func(t *testing.T, _ *nats.Conn, err error) {
				require.ErrorContains(t, err, "error loading NATS nkey seed")
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
			name: "authentication required but no seed key file specified",
			cfg:  Config{ServerURL: authSrv.ClientURL()},
			assert: func(t *testing.T, _ *nats.Conn, err error) {
				require.ErrorContains(t, err, "error connecting to NATS server")
			},
		},
		{
			name: "success without authentication",
			cfg: Config{
				ServerURL:  openSrv.ClientURL(),
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
			name: "success with nkey authentication",
			cfg: Config{
				ServerURL:   authSrv.ClientURL(),
				SeedKeyFile: seedFile,
			},
			assert: func(t *testing.T, conn *nats.Conn, err error) {
				require.NoError(t, err)
				require.True(t, conn.IsConnected())
			},
		},
		{
			name: "caller-supplied options take precedence",
			cfg: Config{
				ServerURL:  openSrv.ClientURL(),
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
