package telemetry

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
	utilnet "k8s.io/apimachinery/pkg/util/net"
)

// clientScope is the instrumentation scope of client spans.
const clientScope = "github.com/akuity/kargo/pkg/telemetry"

// clientTransport makes each request through next a client span, with the
// context's attributes, ending once the body is read or closed. Only a 5xx or
// a failed round trip is the span's error; a 4xx is the caller's to judge, and
// a cancellation is not a failure.
type clientTransport struct {
	next http.RoundTripper
	// tracer comes from the provider installed when the transport is built.
	tracer trace.Tracer
	// name returns a request's span name, and whether it is traced at all.
	name func(*http.Request) (string, bool)
	// attributes returns a request's attributes, including url.full. It is
	// only called for a recording span.
	attributes func(*http.Request) []attribute.KeyValue
	// propagate is whether the request carries the span's trace context.
	propagate bool
}

var _ utilnet.RoundTripperWrapper = clientTransport{}

// WrappedRoundTripper returns next, so client-go can reach the transport
// beneath, e.g. to close its idle connections.
func (t clientTransport) WrappedRoundTripper() http.RoundTripper {
	return t.next
}

func (t clientTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	name, traced := t.name(r)
	if !traced {
		return t.next.RoundTrip(r)
	}
	ctx, span := StartSpan(r.Context(), t.tracer, name, trace.WithSpanKind(trace.SpanKindClient))
	if span.IsRecording() {
		span.SetAttributes(
			semconv.HTTPRequestMethodKey.String(r.Method),
			semconv.ServerAddress(r.URL.Hostname()),
		)
		if port, err := strconv.Atoi(r.URL.Port()); err == nil {
			span.SetAttributes(semconv.ServerPort(port))
		}
		span.SetAttributes(t.attributes(r)...)
	}
	if t.propagate {
		// A RoundTripper must not modify the request it was given.
		r = r.Clone(ctx)
		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))
	}
	res, err := t.next.RoundTrip(r)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			span.SetAttributes(semconv.ErrorType(err))
			RecordError(span, err)
		}
		span.End()
		return res, err
	}
	span.SetAttributes(semconv.HTTPResponseStatusCode(res.StatusCode))
	if res.StatusCode >= http.StatusInternalServerError {
		span.SetAttributes(semconv.ErrorTypeKey.String(strconv.Itoa(res.StatusCode)))
		span.SetStatus(codes.Error, http.StatusText(res.StatusCode))
	}
	if res.Body == nil {
		span.End()
		return res, nil
	}
	res.Body = &spanEndingBody{ReadCloser: res.Body, span: span}
	return res, nil
}

// spanEndingBody ends span once the body is read to its end, fails or is
// closed.
type spanEndingBody struct {
	io.ReadCloser
	span trace.Span
	once sync.Once
}

func (b *spanEndingBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err != nil {
		if err != io.EOF && !errors.Is(err, context.Canceled) {
			RecordError(b.span, err)
		}
		b.end()
	}
	return n, err
}

func (b *spanEndingBody) Close() error {
	b.end()
	return b.ReadCloser.Close()
}

func (b *spanEndingBody) end() {
	b.once.Do(func() { b.span.End() })
}
