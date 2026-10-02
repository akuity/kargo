package database

import (
	"context"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// unreachablePool returns a pool that has not connected and never will. The
// Migrator's offline behavior can be exercised without a database.
func unreachablePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(
		"postgres://kargo:kargo@127.0.0.1:1/kargo?sslmode=disable",
	)
	require.NoError(t, err)
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

func TestNewMigrator(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name       string
		migrations fstest.MapFS
		assert     func(*testing.T, Migrator, error)
	}{
		{
			name: "malformed migration is rejected up front",
			migrations: fstest.MapFS{
				"not-a-migration.sql": {Data: []byte("SELECT 1;")},
			},
			assert: func(t *testing.T, _ Migrator, err error) {
				require.ErrorContains(t, err, "error initializing migrations")
			},
		},
		{
			// Goose treats an empty set as a configuration error, which is what
			// we want: a binary that embeds no migrations should not be able to
			// declare a database current.
			name:       "no migrations is an error",
			migrations: fstest.MapFS{},
			assert: func(t *testing.T, _ Migrator, err error) {
				require.ErrorContains(t, err, "error initializing migrations")
			},
		},
		{
			name: "target is the highest embedded version",
			migrations: fstest.MapFS{
				"00001_first.sql":  {Data: []byte("-- +goose Up\nSELECT 1;\n")},
				"00003_third.sql":  {Data: []byte("-- +goose Up\nSELECT 3;\n")},
				"00002_second.sql": {Data: []byte("-- +goose Up\nSELECT 2;\n")},
			},
			assert: func(t *testing.T, m Migrator, err error) {
				require.NoError(t, err)
				impl, ok := m.(*migrator)
				require.True(t, ok)
				require.Equal(t, int64(3), impl.target())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			m, err := NewMigrator(
				context.Background(),
				unreachablePool(t),
				testCase.migrations,
				time.Minute,
			)
			testCase.assert(t, m, err)
		})
	}
}

func TestNewMigrator_lockTimeout(t *testing.T) {
	t.Parallel()
	migrations := fstest.MapFS{
		"00001_first.sql": {Data: []byte("-- +goose Up\nSELECT 1;\n")},
	}
	_, err := NewMigrator(context.Background(), unreachablePool(t), migrations, time.Second)
	require.ErrorContains(t, err, "lock timeout must be at least")
}

func TestConnectWithRetry(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		ctx     func() (context.Context, context.CancelFunc)
		timeout time.Duration
		wantErr string
	}{
		{
			name:    "gives up after the timeout",
			ctx:     func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) },
			timeout: 0,
			wantErr: "database did not become reachable within 0s",
		},
		{
			name: "stops when the context is canceled",
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 500*time.Millisecond)
			},
			timeout: time.Hour,
			wantErr: context.DeadlineExceeded.Error(),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := testCase.ctx()
			defer cancel()
			pool, err := ConnectWithRetry(
				ctx,
				"postgres://kargo:kargo@127.0.0.1:1/kargo?sslmode=disable",
				"kargo-test",
				testCase.timeout,
			)
			require.ErrorContains(t, err, testCase.wantErr)
			require.Nil(t, pool)
		})
	}
}
