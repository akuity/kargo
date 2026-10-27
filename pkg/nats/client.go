// Package nats provides common NATS client configuration and connection logic
// used by Kargo's control plane components. Anything specific to how a single
// component uses NATS belongs with that component rather than here.
package nats

import (
	"context"
	"errors"
	"fmt"
	"os"
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
	//
	// At most one of SeedKeyFile, CredsFile, Token, or Username/Password may
	// be specified.
	SeedKeyFile string `envconfig:"NATS_SEED_KEY_FILE"`
	// CredsFile is the path to a credentials file containing a user JWT and
	// the nkey seed used to sign server challenges, as used by servers in
	// decentralized (operator) mode. Unlike the other credentials, it is
	// re-read on every (re)connect, so it can be rotated without a restart.
	CredsFile string `envconfig:"NATS_CREDS_FILE"`
	// Token is a token used to authenticate with the NATS server.
	Token string `envconfig:"NATS_TOKEN"`
	// Username is the username used to authenticate with the NATS server.
	// Password must also be specified.
	Username string `envconfig:"NATS_USERNAME"`
	// Password is the password used to authenticate with the NATS server.
	// Username must also be specified.
	Password string `envconfig:"NATS_PASSWORD"`
	// TLSCAFile is the path to a file containing PEM-encoded CA certificates
	// used to verify the NATS server's certificate. When unspecified, the
	// system's root CAs are used.
	TLSCAFile string `envconfig:"NATS_TLS_CA_FILE"`
	// TLSCertFile is the path to a file containing a PEM-encoded client
	// certificate presented to the NATS server. This is required by servers
	// that verify client certificates, including those that authenticate
	// clients by them. TLSKeyFile must also be specified.
	TLSCertFile string `envconfig:"NATS_TLS_CERT_FILE"`
	// TLSKeyFile is the path to a file containing the PEM-encoded private key
	// for TLSCertFile. TLSCertFile must also be specified.
	TLSKeyFile string `envconfig:"NATS_TLS_KEY_FILE"`
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
	natsOpts, err := c.options(ctx)
	if err != nil {
		return nil, err
	}
	conn, err := nats.Connect(c.ServerURL, append(natsOpts, opts...)...)
	if err != nil {
		return nil, fmt.Errorf("error connecting to NATS server: %w", err)
	}
	return conn, nil
}

// options returns the nats.Options derived from the Config. Connection state
// change handlers log using the logger found in the provided context.
func (c Config) options(ctx context.Context) ([]nats.Option, error) {
	if c.ServerURL == "" {
		return nil, errors.New("NATS_SERVER_URL is required")
	}
	logger := logging.LoggerFromContext(ctx)
	timeout := c.ConnectTimeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	opts := []nats.Option{nats.Timeout(timeout)}
	authOpts, err := c.authOptions()
	if err != nil {
		return nil, err
	}
	opts = append(opts, authOpts...)
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

// authOptions returns the nats.Options for authenticating with, and securing
// the connection to, the NATS server.
func (c Config) authOptions() ([]nats.Option, error) {
	var methods int
	for _, set := range []bool{
		c.SeedKeyFile != "",
		c.CredsFile != "",
		c.Token != "",
		c.Username != "" || c.Password != "",
	} {
		if set {
			methods++
		}
	}
	// The server accepts a single set of credentials, and which one the
	// library would send if given several is not obvious.
	if methods > 1 {
		return nil, errors.New(
			"at most one of NATS_SEED_KEY_FILE, NATS_CREDS_FILE, NATS_TOKEN, " +
				"or NATS_USERNAME/NATS_PASSWORD may be specified",
		)
	}
	if (c.Username == "") != (c.Password == "") {
		return nil, errors.New(
			"NATS_USERNAME and NATS_PASSWORD must be specified together",
		)
	}
	if (c.TLSCertFile == "") != (c.TLSKeyFile == "") {
		return nil, errors.New(
			"NATS_TLS_CERT_FILE and NATS_TLS_KEY_FILE must be specified together",
		)
	}

	var opts []nats.Option
	switch {
	case c.SeedKeyFile != "":
		nkeyOpt, err := nats.NkeyOptionFromSeed(c.SeedKeyFile)
		if err != nil {
			return nil, fmt.Errorf("error loading NATS nkey seed: %w", err)
		}
		opts = append(opts, nkeyOpt)
	case c.CredsFile != "":
		// The library reads this lazily, on each connection attempt. Check it
		// up front so that a missing file is reported clearly rather than as a
		// failure to connect.
		if _, err := os.Stat(c.CredsFile); err != nil {
			return nil, fmt.Errorf("error loading NATS credentials: %w", err)
		}
		opts = append(opts, nats.UserCredentials(c.CredsFile))
	case c.Token != "":
		opts = append(opts, nats.Token(c.Token))
	case c.Username != "":
		opts = append(opts, nats.UserInfo(c.Username, c.Password))
	}
	// Either of these also requires TLS for the connection.
	if c.TLSCAFile != "" {
		opts = append(opts, nats.RootCAs(c.TLSCAFile))
	}
	if c.TLSCertFile != "" {
		opts = append(opts, nats.ClientCert(c.TLSCertFile, c.TLSKeyFile))
	}
	return opts, nil
}
