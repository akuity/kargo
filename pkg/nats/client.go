// Package nats provides common NATS client configuration and connection logic
// used by Kargo's control plane components. Anything specific to how a single
// component uses NATS belongs with that component rather than here.
package nats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/nats-io/nats.go"

	"github.com/akuity/kargo/pkg/logging"
)

// Config represents configuration for connecting to a NATS server.
type Config struct {
	// ServerURL is the URL of the NATS server to connect to, e.g.
	// "nats://<host>:4222". Multiple, comma-separated URLs may be given.
	ServerURL string `envconfig:"NATS_SERVER_URL"`
	// SeedKeyFile is the path to a file containing the nkey seed used to
	// authenticate with the NATS server. It is optional so that a server
	// running without authentication can be used.
	SeedKeyFile string `envconfig:"NATS_SEED_KEY_FILE"`
	// ClientName is the name given to the connection. The server reports it
	// in its monitoring endpoints, which makes it useful for identifying which
	// component (and which replica) a connection belongs to.
	ClientName string `envconfig:"NATS_CLIENT_NAME"`
	// ConnectTimeout is the timeout for establishing a connection to the NATS
	// server.
	ConnectTimeout time.Duration `envconfig:"NATS_CONNECT_TIMEOUT" default:"2s"`
}

// ConfigFromEnv returns a Config populated from environment variables.
func ConfigFromEnv() Config {
	cfg := Config{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// ConnectFromEnv reads NATS configuration from environment variables and uses
// it to connect to a NATS server. See Config.Connect.
func ConnectFromEnv(ctx context.Context, opts ...nats.Option) (*nats.Conn, error) {
	return ConfigFromEnv().Connect(ctx, opts...)
}

// Connect connects to the NATS server described by the Config. Any additional
// options provided are applied after those derived from the Config and
// therefore take precedence over them. The logger found in the provided
// context is used to report connection state changes for the lifetime of the
// connection.
func (c Config) Connect(ctx context.Context, opts ...nats.Option) (*nats.Conn, error) {
	natsOpts, err := c.options(logging.LoggerFromContext(ctx))
	if err != nil {
		return nil, err
	}
	conn, err := nats.Connect(c.ServerURL, append(natsOpts, opts...)...)
	if err != nil {
		return nil, fmt.Errorf("error connecting to NATS server: %w", err)
	}
	return conn, nil
}

// options returns the nats.Options derived from the Config.
func (c Config) options(logger *logging.Logger) ([]nats.Option, error) {
	if c.ServerURL == "" {
		return nil, errors.New("NATS_SERVER_URL is required")
	}
	timeout := c.ConnectTimeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	opts := []nats.Option{nats.Timeout(timeout)}
	if c.SeedKeyFile != "" {
		nkeyOpt, err := nats.NkeyOptionFromSeed(c.SeedKeyFile)
		if err != nil {
			return nil, fmt.Errorf("error loading NATS nkey seed: %w", err)
		}
		opts = append(opts, nkeyOpt)
	}
	if c.ClientName != "" {
		opts = append(opts, nats.Name(c.ClientName))
	}

	// Retry forever. The library otherwise gives up after 60 reconnect attempts
	// and permanently closes the connection, after which subscribers silently
	// stop receiving and publishes fail. We would rather ride out an outage of
	// any length and make it visible in the logs while it lasts.
	//
	// RetryOnFailedConnect is deliberately left off: a server that isn't
	// reachable at startup should fail loudly rather than let the component
	// come up in a reconnecting state and look healthy. This only governs a
	// connection that was established once and then lost.
	opts = append(
		opts,
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(conn *nats.Conn, err error) {
			if err == nil {
				// A nil error means the disconnect was initiated by us (e.g. Close)
				return
			}
			logger.Error(
				err, "disconnected from NATS server; reconnecting",
				"server", conn.ConnectedUrlRedacted(),
			)
		}),
		nats.ReconnectHandler(func(conn *nats.Conn) {
			logger.Info(
				"reconnected to NATS server",
				"server", conn.ConnectedUrlRedacted(),
			)
		}),
		nats.ClosedHandler(func(conn *nats.Conn) {
			if err := conn.LastError(); err != nil {
				logger.Error(err, "NATS connection closed")
				return
			}
			logger.Debug("NATS connection closed")
		}),
		nats.ErrorHandler(func(_ *nats.Conn, sub *nats.Subscription, err error) {
			var subject string
			if sub != nil {
				subject = sub.Subject
			}
			logger.Error(err, "asynchronous NATS error", "subject", subject)
		}),
	)
	return opts, nil
}
