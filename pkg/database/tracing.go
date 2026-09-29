package database

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("github.com/akuity/kargo/pkg/database")

const (
	// spanNameAcquire is the name of the span recorded while waiting for a
	// connection from the pool. Its duration is the time a caller spent
	// blocked on pool capacity, which is the leading indicator that the pool
	// is too small for the load.
	spanNameAcquire = "db.pool.acquire"
	// spanNameBatch is the name of the span recorded around a pipelined batch
	// of queries.
	spanNameBatch = "db.batch"
	// sqlcNamePrefix begins the comment sqlc places at the top of every query
	// it generates, e.g. "-- name: GetProject :one". The name that follows is
	// the most useful identifier a span can carry.
	sqlcNamePrefix = "-- name: "
	// batchSizeKey is the attribute under which the number of queries in a
	// batch is recorded.
	batchSizeKey = attribute.Key("db.operation.batch.size")
	// affectedRowsKey is the attribute under which the number of rows an
	// INSERT, UPDATE, or DELETE touched is recorded. Semantic conventions
	// define an attribute for rows returned but none for rows written.
	affectedRowsKey = attribute.Key("db.response.affected_rows")
)

// pgxTracer records OpenTelemetry spans for the pgx operations it is attached
// to. It implements pgx.QueryTracer and pgx.BatchTracer, which cover every
// query issued through a connection, and pgxpool.AcquireTracer, which covers
// time spent waiting for a connection. Query parameters are never recorded.
//
// The spans use the OpenTelemetry API only, so a pool built with one of these
// costs next to nothing until a component enables tracing.
type pgxTracer struct{}

// pgx only requires a QueryTracer; it discovers the other tracer interfaces
// with runtime type assertions and silently skips any it does not find. These
// assertions turn a drifted method signature into a compile error rather than
// missing spans.
var (
	_ pgx.QueryTracer       = pgxTracer{}
	_ pgx.BatchTracer       = pgxTracer{}
	_ pgxpool.AcquireTracer = pgxTracer{}
)

// TraceQueryStart implements pgx.QueryTracer.
func (pgxTracer) TraceQueryStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	ctx, _ = tracer.Start(
		ctx,
		querySpanName(data.SQL),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(queryAttributes(conn, data.SQL)...),
	)
	return ctx
}

// TraceQueryEnd implements pgx.QueryTracer.
func (pgxTracer) TraceQueryEnd(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryEndData,
) {
	span := trace.SpanFromContext(ctx)
	defer span.End()
	if data.Err != nil {
		recordError(span, data.Err)
		return
	}
	span.SetAttributes(rowCountAttributes(data.CommandTag)...)
}

// TraceBatchStart implements pgx.BatchTracer.
func (pgxTracer) TraceBatchStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceBatchStartData,
) context.Context {
	attrs := connectionAttributes(conn)
	if data.Batch != nil {
		attrs = append(attrs, batchSizeKey.Int(data.Batch.Len()))
	}
	ctx, _ = tracer.Start(
		ctx,
		spanNameBatch,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
	return ctx
}

// TraceBatchQuery implements pgx.BatchTracer. The queries in a batch are
// pipelined, so their individual timings are not meaningful; each is recorded
// as an event on the batch's span instead of as a span of its own.
func (pgxTracer) TraceBatchQuery(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceBatchQueryData,
) {
	span := trace.SpanFromContext(ctx)
	attrs := []attribute.KeyValue{
		semconv.DBQuerySummary(querySpanName(data.SQL)),
	}
	if data.Err != nil {
		span.RecordError(data.Err)
		attrs = append(attrs, statusCodeAttributes(data.Err)...)
	} else {
		attrs = append(attrs, rowCountAttributes(data.CommandTag)...)
	}
	span.AddEvent(querySpanName(data.SQL), trace.WithAttributes(attrs...))
}

// TraceBatchEnd implements pgx.BatchTracer.
func (pgxTracer) TraceBatchEnd(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceBatchEndData,
) {
	span := trace.SpanFromContext(ctx)
	defer span.End()
	if data.Err != nil {
		recordError(span, data.Err)
	}
}

// TraceAcquireStart implements pgxpool.AcquireTracer.
func (pgxTracer) TraceAcquireStart(
	ctx context.Context,
	_ *pgxpool.Pool,
	_ pgxpool.TraceAcquireStartData,
) context.Context {
	ctx, _ = tracer.Start(
		ctx,
		spanNameAcquire,
		trace.WithAttributes(semconv.DBSystemNamePostgreSQL),
	)
	return ctx
}

// TraceAcquireEnd implements pgxpool.AcquireTracer.
func (pgxTracer) TraceAcquireEnd(
	ctx context.Context,
	_ *pgxpool.Pool,
	data pgxpool.TraceAcquireEndData,
) {
	span := trace.SpanFromContext(ctx)
	defer span.End()
	if data.Err != nil {
		recordError(span, data.Err)
	}
}

// querySpanName returns the name a span for the given SQL should carry. sqlc
// prefixes every query it generates with a comment naming it, and that name
// (e.g. "GetProject") is used when present. Otherwise the leading SQL keyword
// (e.g. "SELECT") is used, so that spans for hand-written statements are at
// least grouped by operation rather than by their full text.
func querySpanName(sql string) string {
	if name, ok := sqlcQueryName(sql); ok {
		return name
	}
	if op := operationName(sql); op != "" {
		return op
	}
	return "db.query"
}

// sqlcQueryName extracts the query name from the "-- name:" comment sqlc
// places at the top of the queries it generates.
func sqlcQueryName(sql string) (string, bool) {
	first, _, _ := strings.Cut(strings.TrimSpace(sql), "\n")
	rest, ok := strings.CutPrefix(first, sqlcNamePrefix)
	if !ok {
		return "", false
	}
	name, _, _ := strings.Cut(strings.TrimSpace(rest), " ")
	return name, name != ""
}

// operationName returns the first keyword of the SQL statement, uppercased,
// skipping any leading line comments.
func operationName(sql string) string {
	for line := range strings.SplitSeq(sql, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		op, _, _ := strings.Cut(line, " ")
		return strings.ToUpper(strings.TrimRight(op, ";"))
	}
	return ""
}

// queryAttributes returns the attributes recorded on a query span. The full
// query text is included; sqlc queries are parameterized, so it never
// contains values.
func queryAttributes(conn *pgx.Conn, sql string) []attribute.KeyValue {
	attrs := append(
		connectionAttributes(conn),
		semconv.DBQueryText(sql),
	)
	if name, ok := sqlcQueryName(sql); ok {
		attrs = append(attrs, semconv.DBQuerySummary(name))
	}
	if op := operationName(sql); op != "" {
		attrs = append(attrs, semconv.DBOperationName(op))
	}
	return attrs
}

// connectionAttributes describes the database a connection talks to.
func connectionAttributes(conn *pgx.Conn) []attribute.KeyValue {
	attrs := []attribute.KeyValue{semconv.DBSystemNamePostgreSQL}
	if conn == nil {
		return attrs
	}
	cfg := conn.Config()
	return append(
		attrs,
		semconv.DBNamespace(cfg.Database),
		semconv.ServerAddress(cfg.Host),
		semconv.ServerPort(int(cfg.Port)),
	)
}

// rowCountAttributes returns the row count a statement's command tag carries,
// under the attribute that says what the count means. A SELECT reports rows
// returned; INSERT, UPDATE, and DELETE report rows written. Other statements
// (BEGIN, SET, DDL, and so on) carry no count, and reporting their zero
// would be misleading.
func rowCountAttributes(tag pgconn.CommandTag) []attribute.KeyValue {
	rows := int(tag.RowsAffected())
	switch {
	case tag.Select():
		return []attribute.KeyValue{semconv.DBResponseReturnedRows(rows)}
	case tag.Insert(), tag.Update(), tag.Delete():
		return []attribute.KeyValue{affectedRowsKey.Int(rows)}
	default:
		return nil
	}
}

// recordError marks the span as failed and records what went wrong.
func recordError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	span.SetAttributes(statusCodeAttributes(err)...)
}

// statusCodeAttributes returns the server's SQLSTATE code for errors that
// carry one. Lock timeouts, deadlocks, and serialization failures each have
// their own code, so this is what distinguishes them from other failures.
func statusCodeAttributes(err error) []attribute.KeyValue {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return []attribute.KeyValue{semconv.DBResponseStatusCode(pgErr.Code)}
	}
	return nil
}
