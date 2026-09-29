package database

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	spanRecorder     = tracetest.NewSpanRecorder()
	spanRecorderOnce sync.Once
)

// installSpanRecorder makes the global tracer provider record spans in memory
// and returns the recorder, cleared of anything recorded so far. The provider
// is installed once per test binary because this package's tracer is
// package-level and binds to the first provider the global sees.
func installSpanRecorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()
	spanRecorderOnce.Do(func() {
		otel.SetTracerProvider(sdktrace.NewTracerProvider(
			sdktrace.WithSpanProcessor(spanRecorder),
		))
	})
	spanRecorder.Reset()
	return spanRecorder
}

// attributesOf returns a span's attributes keyed by name for easy lookup.
func attributesOf(span sdktrace.ReadOnlySpan) map[attribute.Key]attribute.Value {
	attrs := map[attribute.Key]attribute.Value{}
	for _, kv := range span.Attributes() {
		attrs[kv.Key] = kv.Value
	}
	return attrs
}

// eventAttributesOf returns an event's attributes keyed by name.
func eventAttributesOf(event sdktrace.Event) map[attribute.Key]attribute.Value {
	attrs := map[attribute.Key]attribute.Value{}
	for _, kv := range event.Attributes {
		attrs[kv.Key] = kv.Value
	}
	return attrs
}

const testSQLCQuery = "-- name: GetProject :one\nSELECT id FROM projects WHERE id = $1\n"

func TestPgxTracer_Query(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	lockErr := &pgconn.PgError{Code: "55P03", Message: "lock not available"}
	testCases := []struct {
		name   string
		sql    string
		end    pgx.TraceQueryEndData
		assert func(*testing.T, sdktrace.ReadOnlySpan)
	}{
		{
			name: "sqlc query succeeds",
			sql:  testSQLCQuery,
			end:  pgx.TraceQueryEndData{CommandTag: pgconn.NewCommandTag("SELECT 1")},
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Equal(t, "GetProject", span.Name())
				require.Equal(t, trace.SpanKindClient, span.SpanKind())
				require.Equal(t, codes.Unset, span.Status().Code)
				attrs := attributesOf(span)
				require.Equal(t, "postgresql", attrs[semconv.DBSystemNameKey].AsString())
				require.Equal(t, testSQLCQuery, attrs[semconv.DBQueryTextKey].AsString())
				require.Equal(t, "GetProject", attrs[semconv.DBQuerySummaryKey].AsString())
				require.Equal(t, "SELECT", attrs[semconv.DBOperationNameKey].AsString())
				require.Equal(t, int64(1), attrs[semconv.DBResponseReturnedRowsKey].AsInt64())
				require.NotContains(t, attrs, affectedRowsKey)
			},
		},
		{
			name: "hand-written write is named by its operation and reports affected rows",
			sql:  "insert into projects (id) values ($1)",
			end:  pgx.TraceQueryEndData{CommandTag: pgconn.NewCommandTag("INSERT 0 1")},
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Equal(t, "INSERT", span.Name())
				attrs := attributesOf(span)
				require.NotContains(t, attrs, semconv.DBQuerySummaryKey)
				require.Equal(t, "INSERT", attrs[semconv.DBOperationNameKey].AsString())
				require.Equal(t, int64(1), attrs[affectedRowsKey].AsInt64())
				require.NotContains(t, attrs, semconv.DBResponseReturnedRowsKey)
			},
		},
		{
			name: "statement without a row count reports none",
			sql:  "BEGIN",
			end:  pgx.TraceQueryEndData{CommandTag: pgconn.NewCommandTag("BEGIN")},
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Equal(t, "BEGIN", span.Name())
				attrs := attributesOf(span)
				require.NotContains(t, attrs, semconv.DBResponseReturnedRowsKey)
				require.NotContains(t, attrs, affectedRowsKey)
			},
		},
		{
			name: "server error records the SQLSTATE code",
			sql:  testSQLCQuery,
			end:  pgx.TraceQueryEndData{Err: lockErr},
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Equal(t, codes.Error, span.Status().Code)
				require.Equal(t, lockErr.Error(), span.Status().Description)
				attrs := attributesOf(span)
				require.Equal(t, "55P03", attrs[semconv.DBResponseStatusCodeKey].AsString())
				require.NotContains(t, attrs, semconv.DBResponseReturnedRowsKey)
				require.NotContains(t, attrs, affectedRowsKey)
				require.Len(t, span.Events(), 1)
				require.Equal(t, "exception", span.Events()[0].Name)
			},
		},
		{
			name: "non-server error has no status code",
			sql:  testSQLCQuery,
			end:  pgx.TraceQueryEndData{Err: context.DeadlineExceeded},
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Equal(t, codes.Error, span.Status().Code)
				require.NotContains(t, attributesOf(span), semconv.DBResponseStatusCodeKey)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := installSpanRecorder(t)
			var tr pgxTracer
			ctx := tr.TraceQueryStart(
				context.Background(),
				nil,
				pgx.TraceQueryStartData{SQL: testCase.sql, Args: []any{"secret"}},
			)
			tr.TraceQueryEnd(ctx, nil, testCase.end)
			spans := recorder.Ended()
			require.Len(t, spans, 1)
			testCase.assert(t, spans[0])
			// Query parameters must never leak into a span.
			for _, kv := range spans[0].Attributes() {
				require.NotContains(t, kv.Value.String(), "secret")
			}
		})
	}
}

func TestPgxTracer_Batch(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	batchErr := errors.New("batch failed")
	testCases := []struct {
		name    string
		queries []pgx.TraceBatchQueryData
		end     pgx.TraceBatchEndData
		assert  func(*testing.T, sdktrace.ReadOnlySpan)
	}{
		{
			name: "each query becomes an event",
			queries: []pgx.TraceBatchQueryData{
				{SQL: testSQLCQuery, CommandTag: pgconn.NewCommandTag("SELECT 1")},
				{SQL: "DELETE FROM projects", CommandTag: pgconn.NewCommandTag("DELETE 3")},
			},
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Equal(t, spanNameBatch, span.Name())
				require.Equal(t, codes.Unset, span.Status().Code)
				require.Equal(t, int64(2), attributesOf(span)[batchSizeKey].AsInt64())
				events := span.Events()
				require.Len(t, events, 2)
				require.Equal(t, "GetProject", events[0].Name)
				require.Equal(t, int64(1), eventAttributesOf(events[0])[semconv.DBResponseReturnedRowsKey].AsInt64())
				require.Equal(t, "DELETE", events[1].Name)
				require.Equal(t, int64(3), eventAttributesOf(events[1])[affectedRowsKey].AsInt64())
			},
		},
		{
			name: "failed query is recorded and the batch fails",
			queries: []pgx.TraceBatchQueryData{{
				SQL: testSQLCQuery,
				Err: &pgconn.PgError{Code: "40P01", Message: "deadlock detected"},
			}},
			end: pgx.TraceBatchEndData{Err: batchErr},
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Equal(t, codes.Error, span.Status().Code)
				require.Equal(t, batchErr.Error(), span.Status().Description)
				// One "exception" event from the query, one query event, and one
				// "exception" event from the batch ending in error.
				require.Len(t, span.Events(), 3)
				require.Equal(t, "exception", span.Events()[0].Name)
				require.Equal(t, "GetProject", span.Events()[1].Name)
				attrs := eventAttributesOf(span.Events()[1])
				require.Equal(t, "40P01", attrs[semconv.DBResponseStatusCodeKey].AsString())
				require.NotContains(t, attrs, semconv.DBResponseReturnedRowsKey)
				require.NotContains(t, attrs, affectedRowsKey)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := installSpanRecorder(t)
			var tr pgxTracer
			batch := &pgx.Batch{}
			for _, q := range testCase.queries {
				batch.Queue(q.SQL)
			}
			ctx := tr.TraceBatchStart(
				context.Background(),
				nil,
				pgx.TraceBatchStartData{Batch: batch},
			)
			for _, q := range testCase.queries {
				tr.TraceBatchQuery(ctx, nil, q)
			}
			tr.TraceBatchEnd(ctx, nil, testCase.end)
			spans := recorder.Ended()
			require.Len(t, spans, 1)
			testCase.assert(t, spans[0])
		})
	}
}

func TestPgxTracer_Acquire(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	testCases := []struct {
		name   string
		end    pgxpool.TraceAcquireEndData
		assert func(*testing.T, sdktrace.ReadOnlySpan)
	}{
		{
			name: "connection acquired",
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Equal(t, spanNameAcquire, span.Name())
				require.Equal(t, codes.Unset, span.Status().Code)
			},
		},
		{
			name: "acquire fails",
			end:  pgxpool.TraceAcquireEndData{Err: context.DeadlineExceeded},
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Equal(t, codes.Error, span.Status().Code)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := installSpanRecorder(t)
			var tr pgxTracer
			ctx := tr.TraceAcquireStart(
				context.Background(),
				nil,
				pgxpool.TraceAcquireStartData{},
			)
			tr.TraceAcquireEnd(ctx, nil, testCase.end)
			spans := recorder.Ended()
			require.Len(t, spans, 1)
			testCase.assert(t, spans[0])
		})
	}
}

func TestQuerySpanName(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		sql  string
		want string
	}{
		{
			name: "sqlc name",
			sql:  "-- name: ListProjects :many\nSELECT 1",
			want: "ListProjects",
		},
		{
			name: "sqlc name with surrounding whitespace",
			sql:  "\n  -- name:   UpsertProject   :exec\nINSERT INTO projects",
			want: "UpsertProject",
		},
		{
			name: "sqlc name comment with no name",
			sql:  "-- name: \nSELECT 1",
			want: "SELECT",
		},
		{
			name: "other comment then statement",
			sql:  "-- fetch everything\n\n  select * from projects",
			want: "SELECT",
		},
		{
			name: "single keyword statement",
			sql:  "BEGIN;",
			want: "BEGIN",
		},
		{
			name: "only comments",
			sql:  "-- nothing here",
			want: "db.query",
		},
		{
			name: "empty",
			sql:  "",
			want: "db.query",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.want, querySpanName(testCase.sql))
		})
	}
}
