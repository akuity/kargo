package database

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
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

// refusingServer is a connection string for a port nothing listens on.
const refusingServer = "postgres://kargo:kargo@127.0.0.1:1/kargo?sslmode=disable"

// rejectingServer listens on a loopback port and answers every connection's
// startup message with a PostgreSQL ErrorResponse carrying the given SQLSTATE
// and message, as a real server does for a bad password or while it is still
// starting. It returns a connection string for itself.
func rejectingServer(t *testing.T, sqlState string, message string) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	// ErrorResponse: 'E', int32 length (self included), then (type byte,
	// C string) fields ending with a zero byte.
	var fields []byte
	for _, f := range []struct {
		typ byte
		val string
	}{
		{'S', "FATAL"}, {'V', "FATAL"}, {'C', sqlState}, {'M', message},
	} {
		fields = append(fields, f.typ)
		fields = append(fields, f.val...)
		fields = append(fields, 0)
	}
	fields = append(fields, 0)
	response := append([]byte{'E'}, binary.BigEndian.AppendUint32(nil, uint32(len(fields)+4))...)
	response = append(response, fields...)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				// Consume the startup message: int32 length, then the body.
				var length [4]byte
				if _, err := io.ReadFull(conn, length[:]); err != nil {
					return
				}
				body := make([]byte, binary.BigEndian.Uint32(length[:])-4)
				if _, err := io.ReadFull(conn, body); err != nil {
					return
				}
				_, _ = conn.Write(response)
			}()
		}
	}()
	return fmt.Sprintf(
		"postgres://kargo:kargo@%s/kargo?sslmode=disable",
		listener.Addr().String(),
	)
}

func TestConnectWithRetry(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name       string
		ctx        func() (context.Context, context.CancelFunc)
		connString func(*testing.T) string
		timeout    time.Duration
		assert     func(*testing.T, error)
	}{
		{
			name:       "malformed connection string fails immediately",
			connString: func(*testing.T) string { return "postgres://kargo:kargo@[::1/kargo" },
			timeout:    time.Hour,
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error parsing database connection string")
			},
		},
		{
			name: "rejected credentials fail immediately",
			connString: func(t *testing.T) string {
				return rejectingServer(t, "28P01", "password authentication failed")
			},
			timeout: time.Hour,
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "password authentication failed")
				require.NotContains(t, err.Error(), "did not become reachable")
			},
		},
		{
			name: "a server that is still starting is retried until the timeout",
			connString: func(t *testing.T) string {
				return rejectingServer(t, sqlStateCannotConnectNow, "the database system is starting up")
			},
			timeout: 0,
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "database did not become reachable within 0s")
				require.ErrorContains(t, err, "the database system is starting up")
			},
		},
		{
			name:       "an unreachable server is retried until the timeout",
			connString: func(*testing.T) string { return refusingServer },
			timeout:    0,
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "database did not become reachable within 0s")
			},
		},
		{
			name: "stops when the context is canceled",
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 500*time.Millisecond)
			},
			connString: func(*testing.T) string { return refusingServer },
			timeout:    time.Hour,
			assert: func(t *testing.T, err error) {
				require.ErrorIs(t, err, context.DeadlineExceeded)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			if testCase.ctx != nil {
				ctx, cancel = testCase.ctx()
			}
			defer cancel()
			pool, err := ConnectWithRetry(
				ctx,
				testCase.connString(t),
				"kargo-test",
				testCase.timeout,
			)
			testCase.assert(t, err)
			require.Nil(t, pool)
		})
	}
}

func TestIsTransientConnectError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "network error",
			err:  &net.OpError{Op: "dial", Err: errors.New("connection refused")},
			want: true,
		},
		{
			name: "server still starting",
			err:  fmt.Errorf("wrapped: %w", &pgconn.PgError{Code: sqlStateCannotConnectNow}),
			want: true,
		},
		{
			name: "too many connections",
			err:  &pgconn.PgError{Code: sqlStateTooManyConnections},
			want: true,
		},
		{
			name: "invalid password",
			err:  fmt.Errorf("wrapped: %w", &pgconn.PgError{Code: "28P01"}),
			want: false,
		},
		{
			name: "unknown database",
			err:  &pgconn.PgError{Code: "3D000"},
			want: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.want, isTransientConnectError(testCase.err))
		})
	}
}
