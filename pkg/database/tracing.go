package database

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("github.com/akuity/kargo/pkg/database")

const (
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
)

// pgxTracer records OpenTelemetry spans for the pgx operations it is attached
// to. It implements pgx.QueryTracer and pgx.BatchTracer, which together cover
// every query issued through a connection. Query parameters are never
// recorded.
//
// Time spent waiting for a connection from the pool is deliberately not a
// span: it precedes every query, is near zero unless the pool is saturated,
// and is better observed as a trend through the pool's statistics.
//
// The spans use the OpenTelemetry API only, so a pool built with one of these
// costs next to nothing until a component enables tracing.
type pgxTracer struct {
	// connAttrs describes the database every span concerns. It is built once
	// from the pool's configuration; pgx.Conn.Config() returns a deep copy,
	// which is too expensive to take on every query.
	connAttrs []attribute.KeyValue
}

// newPgxTracer returns a tracer for connections made with cfg.
func newPgxTracer(cfg *pgx.ConnConfig) pgxTracer {
	return pgxTracer{
		connAttrs: []attribute.KeyValue{
			semconv.DBSystemNamePostgreSQL,
			semconv.DBNamespace(cfg.Database),
			semconv.ServerAddress(cfg.Host),
			semconv.ServerPort(int(cfg.Port)),
		},
	}
}

// pgx only requires a QueryTracer; it discovers BatchTracer with a runtime
// type assertion and silently skips it if not found. These assertions turn a
// drifted method signature into a compile error rather than missing spans.
var (
	_ pgx.QueryTracer = pgxTracer{}
	_ pgx.BatchTracer = pgxTracer{}
)

// TraceQueryStart implements pgx.QueryTracer.
func (t pgxTracer) TraceQueryStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	ctx, _ = tracer.Start(
		ctx,
		querySpanName(data.SQL),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(t.queryAttributes(data.SQL)...),
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
func (t pgxTracer) TraceBatchStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceBatchStartData,
) context.Context {
	attrs := t.connectionAttributes()
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
func (t pgxTracer) queryAttributes(sql string) []attribute.KeyValue {
	attrs := append(
		t.connectionAttributes(),
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

// connectionAttributes returns a fresh slice describing the database, safe
// for callers to append to. A zero-value tracer, as used in tests, reports
// only the database system.
func (t pgxTracer) connectionAttributes() []attribute.KeyValue {
	if t.connAttrs == nil {
		return []attribute.KeyValue{semconv.DBSystemNamePostgreSQL}
	}
	return append(make([]attribute.KeyValue, 0, len(t.connAttrs)+4), t.connAttrs...)
}

// rowCountAttributes returns the row count a statement's command tag carries.
// SELECT, INSERT, UPDATE, and DELETE report one; other statements (BEGIN, SET,
// DDL, and so on) do not, and reporting their zero would be misleading. The
// span's operation name says whether the count is rows returned or written.
func rowCountAttributes(tag pgconn.CommandTag) []attribute.KeyValue {
	if !tag.Select() && !tag.Insert() && !tag.Update() && !tag.Delete() {
		return nil
	}
	return []attribute.KeyValue{
		semconv.DBResponseReturnedRows(int(tag.RowsAffected())),
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
