package database

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"

	"github.com/akuity/kargo/pkg/telemetry"
)

func TestNewPoolConfig(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name       string
		connString string
		assert     func(*testing.T, map[string]string, error)
	}{
		{
			name:       "invalid connection string",
			connString: "postgres://kargo:kargo@localhost:5432/kargo?sslmode=bogus",
			assert: func(t *testing.T, _ map[string]string, err error) {
				require.ErrorContains(t, err, "error parsing database connection string")
			},
		},
		{
			name:       "application name is added",
			connString: "postgres://kargo:kargo@localhost:5432/kargo",
			assert: func(t *testing.T, params map[string]string, err error) {
				require.NoError(t, err)
				require.Equal(t, "kargo-test", params[applicationNameParam])
			},
		},
		{
			name:       "application name in connection string wins",
			connString: "postgres://kargo:kargo@localhost:5432/kargo?application_name=custom",
			assert: func(t *testing.T, params map[string]string, err error) {
				require.NoError(t, err)
				require.Equal(t, "custom", params[applicationNameParam])
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := newPoolConfig(testCase.connString, "kargo-test")
			var params map[string]string
			if cfg != nil {
				params = cfg.ConnConfig.RuntimeParams
			}
			testCase.assert(t, params, err)
		})
	}
}

func TestNewPoolConfig_tracer(t *testing.T) {
	// Not parallel: turns tracing on globally.
	testCases := []struct {
		name       string
		tracing    bool
		categories []string
		assert     func(*testing.T, pgx.QueryTracer)
	}{
		{
			name: "tracing off",
			assert: func(t *testing.T, tracer pgx.QueryTracer) {
				require.Nil(t, tracer)
			},
		},
		{
			name:    "tracing on, default categories",
			tracing: true,
			assert: func(t *testing.T, tracer pgx.QueryTracer) {
				pt, ok := tracer.(pgxTracer)
				require.True(t, ok)
				require.Contains(t, pt.connAttrs, semconv.DBNamespace("kargo"))
				require.Contains(t, pt.connAttrs, semconv.ServerAddress("localhost"))
				require.Contains(t, pt.connAttrs, semconv.ServerPort(5432))
			},
		},
		{
			name:       "database category selected",
			tracing:    true,
			categories: []string{string(telemetry.CategoryDatabase)},
			assert: func(t *testing.T, tracer pgx.QueryTracer) {
				require.IsType(t, pgxTracer{}, tracer)
			},
		},
		{
			name:       "database category not selected",
			tracing:    true,
			categories: []string{"none"},
			assert: func(t *testing.T, tracer pgx.QueryTracer) {
				require.Nil(t, tracer)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.tracing {
				enableTracing(t, testCase.categories...)
			}
			cfg, err := newPoolConfig("postgres://kargo:kargo@localhost:5432/kargo", "kargo-test")
			require.NoError(t, err)
			testCase.assert(t, cfg.ConnConfig.Tracer)
		})
	}
}

func TestNewPool(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name       string
		connString string
		wantErr    string
	}{
		{
			name:       "invalid connection string",
			connString: "postgres://kargo:kargo@localhost:5432/kargo?sslmode=bogus",
			wantErr:    "error parsing database connection string",
		},
		{
			name: "unreachable database fails at construction",
			// Port 1 is reserved and nothing listens on it, so the connection
			// attempt is refused immediately instead of timing out.
			connString: "postgres://kargo:kargo@127.0.0.1:1/kargo?sslmode=disable",
			wantErr:    "error connecting to database",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			pool, err := NewPool(context.Background(), testCase.connString, "kargo-test")
			require.ErrorContains(t, err, testCase.wantErr)
			require.Nil(t, pool)
		})
	}
}
