package database

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config represents how a component reaches the database. Either URL is set,
// which is used verbatim, or the individual fields are, from which a
// connection string is composed with the password properly escaped. The
// Helm chart uses the individual fields for the bundled PostgreSQL, since it
// cannot escape a password it reads from a Secret, and URL for an external
// database whose Secret already holds a complete connection string.
type Config struct {
	// URL is a complete connection string, for example
	// postgres://user:password@host:5432/dbname?sslmode=require.
	URL string `envconfig:"DATABASE_URL"`
	// Host is the database server's host name, used when URL is empty.
	Host string `envconfig:"DATABASE_HOST"`
	// Port is the database server's port, used when URL is empty.
	Port int `envconfig:"DATABASE_PORT" default:"5432"`
	// Name is the database to connect to, used when URL is empty.
	Name string `envconfig:"DATABASE_NAME" default:"kargo"`
	// User is the user to connect as, used when URL is empty.
	User string `envconfig:"DATABASE_USER" default:"kargo"`
	// Password is the user's password, used when URL is empty.
	Password string `envconfig:"DATABASE_PASSWORD"`
	// SSLMode is the libpq sslmode to use, used when URL is empty.
	SSLMode string `envconfig:"DATABASE_SSL_MODE" default:"disable"`
	// ConnectTimeout bounds how long to keep retrying the first connection.
	// The database may still be starting when a component does, so refusing
	// to connect is retried rather than treated as fatal until this elapses.
	ConnectTimeout time.Duration `envconfig:"DATABASE_CONNECT_TIMEOUT" default:"5m"`
	// MigrationLockTimeout bounds how long a migration run waits for another
	// run to release the database-level lock before giving up. It must be
	// longer than the slowest migration, since that is exactly what a waiting
	// runner is waiting on.
	MigrationLockTimeout time.Duration `envconfig:"DATABASE_MIGRATION_LOCK_TIMEOUT" default:"1h"`
}

// ConfigFromEnv returns a Config populated from environment variables.
func ConfigFromEnv() Config {
	cfg := Config{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// ConnString returns the connection string described by the Config.
func (c Config) ConnString() (string, error) {
	if c.URL != "" {
		return c.URL, nil
	}
	if c.Host == "" {
		return "", fmt.Errorf(
			"either DATABASE_URL or DATABASE_HOST must be set",
		)
	}
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Password),
		Host:     c.Host + ":" + strconv.Itoa(c.Port),
		Path:     "/" + c.Name,
		RawQuery: url.Values{"sslmode": {c.SSLMode}}.Encode(),
	}
	return u.String(), nil
}
