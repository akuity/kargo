package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const operationTimeout = 5 * time.Second

// NewPool creates a bounded pool without requiring a reachable database.
// Connections are established lazily, so database outages do not prevent startup.
func NewPool(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		// Parse errors may include credentials from the connection string.
		return nil, errors.New("invalid database connection configuration")
	}
	cfg.MaxConns = 8
	cfg.MinConns = 0
	cfg.ConnConfig.ConnectTimeout = operationTimeout
	return pgxpool.NewWithConfig(ctx, cfg)
}
