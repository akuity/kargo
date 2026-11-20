//go:build integration && db

package database

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestNewPool_tracing_integration(t *testing.T) {
	// Not parallel: turns tracing on globally.
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set")
	}
	testCases := []struct {
		name       string
		categories []string
		assert     func(*testing.T, []sdktrace.ReadOnlySpan)
	}{
		{
			name: "default categories",
			assert: func(t *testing.T, ended []sdktrace.ReadOnlySpan) {
				names := make([]string, len(ended))
				for i, span := range ended {
					names[i] = span.Name()
				}
				require.Contains(t, names, "SELECT")
				require.Contains(t, names, spanNameBatch)
			},
		},
		{
			name:       "database category not selected",
			categories: []string{"none"},
			assert: func(t *testing.T, ended []sdktrace.ReadOnlySpan) {
				require.Empty(t, ended)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			enableTracing(t, testCase.categories...)
			ctx := context.Background()
			pool, err := NewPool(ctx, dsn, "kargo-test")
			require.NoError(t, err)
			defer pool.Close()
			recorder := installSpanRecorder(t)

			ctx, parent := spanProvider.Tracer("test").Start(ctx, "parent")
			var one int
			require.NoError(t, pool.QueryRow(ctx, "SELECT 1").Scan(&one))
			batch := &pgx.Batch{}
			batch.Queue("SELECT 1")
			require.NoError(t, pool.SendBatch(ctx, batch).Close())

			ro, ok := parent.(sdktrace.ReadOnlySpan)
			require.True(t, ok)
			require.True(t, ro.EndTime().IsZero())
			require.Empty(t, ro.Attributes())
			require.Empty(t, ro.Events())
			require.Equal(t, "parent", ro.Name())
			testCase.assert(t, recorder.Ended())
			parent.End()
		})
	}
}
