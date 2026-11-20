//go:build integration && db

package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/db"
)

func TestOpen_integration(t *testing.T) {
	ctx := context.Background()

	t.Run("a schema that is behind is waited for, then refused", func(t *testing.T) {
		cfg := Config{
			URL:                  freshDatabase(t),
			ConnectTimeout:       2 * time.Second,
			MigrationLockTimeout: time.Minute,
		}
		start := time.Now()
		pool, err := Open(ctx, cfg, "kargo-test")
		require.ErrorIs(t, err, ErrSchemaOutOfDate)
		require.Nil(t, pool)
		require.GreaterOrEqual(t, time.Since(start), cfg.ConnectTimeout/2,
			"a schema that is merely behind is waited for, not refused at once")
	})

	t.Run("a current schema opens", func(t *testing.T) {
		cfg := Config{
			URL:                  freshDatabase(t),
			ConnectTimeout:       time.Minute,
			MigrationLockTimeout: time.Minute,
		}
		migrationPool, err := NewPool(ctx, cfg.URL, "kargo-test-migrate")
		require.NoError(t, err)
		defer migrationPool.Close()
		migrations, err := db.Migrations()
		require.NoError(t, err)
		m, err := NewMigrator(ctx, migrationPool, migrations, time.Minute)
		require.NoError(t, err)
		_, err = m.Up(ctx)
		require.NoError(t, err)

		pool, err := Open(ctx, cfg, "kargo-test")
		require.NoError(t, err)
		defer pool.Close()
		require.NoError(t, pool.Ping(ctx))
	})

	t.Run("an unusable config is refused", func(t *testing.T) {
		_, err := Open(ctx, Config{}, "kargo-test")
		require.ErrorContains(t, err, "DATABASE_URL or DATABASE_HOST")
	})
}
