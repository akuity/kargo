package telemetry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	utilnet "k8s.io/apimachinery/pkg/util/net"

	"github.com/akuity/kargo/pkg/telemetry/telemetrytest"
)

// bodyRoundTripper answers every request with a 200 and body.
type bodyRoundTripper struct{ body io.ReadCloser }

func (b bodyRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       b.body,
		Request:    r,
	}, nil
}

// okRoundTripper answers every request with an empty 200.
type okRoundTripper struct{}

func (okRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    r,
	}, nil
}

func newTestClientTransport(next http.RoundTripper) clientTransport {
	return clientTransport{
		next:   next,
		tracer: otel.GetTracerProvider().Tracer(clientScope),
		name: func(r *http.Request) (string, bool) {
			return "test " + r.Method, r.URL.Path != "/untraced"
		},
		attributes: func(r *http.Request) []attribute.KeyValue {
			return []attribute.KeyValue{semconv.URLFull(r.URL.String())}
		},
	}
}

func TestClientTransport(t *testing.T) {
	testCases := []struct {
		name   string
		url    string
		body   io.ReadCloser
		finish func(*testing.T, *http.Response)
	}{
		{
			name: "span ends when the body is read to its end",
			url:  "https://example.com:8443/x",
			body: io.NopCloser(strings.NewReader("data")),
			finish: func(t *testing.T, res *http.Response) {
				_, err := io.ReadAll(res.Body)
				require.NoError(t, err)
			},
		},
		{
			name: "span ends when the body is closed",
			url:  "https://example.com:8443/x",
			body: io.NopCloser(strings.NewReader("data")),
			finish: func(t *testing.T, res *http.Response) {
				require.NoError(t, res.Body.Close())
			},
		},
		{
			name: "no body",
			url:  "https://example.com:8443/x",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Not parallel: installs a global tracer provider.
			ended := telemetrytest.RecordSpans(t)
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, testCase.url, nil)
			require.NoError(t, err)
			res, err := newTestClientTransport(bodyRoundTripper{body: testCase.body}).RoundTrip(req)
			require.NoError(t, err)
			if res.Body != nil {
				defer res.Body.Close()
			}
			if testCase.finish != nil {
				require.Empty(t, ended(), "span ended before its body was done")
				testCase.finish(t, res)
			}
			span := telemetrytest.EndedSpan(t, ended(), "test GET")
			require.Equal(t, trace.SpanKindClient, span.SpanKind())
			require.Subset(
				t,
				span.Attributes(),
				[]attribute.KeyValue{
					semconv.HTTPRequestMethodKey.String(http.MethodGet),
					semconv.ServerAddress("example.com"),
					semconv.ServerPort(8443),
					semconv.HTTPResponseStatusCode(http.StatusOK),
					semconv.URLFull(testCase.url),
				},
			)
		})
	}
}

// failingBody fails every read.
type failingBody struct{}

func (failingBody) Read([]byte) (int, error) { return 0, errors.New("connection reset") }

func (failingBody) Close() error { return nil }

func TestClientTransport_bodyReadFails(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	ended := telemetrytest.RecordSpans(t)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com/x", nil)
	require.NoError(t, err)
	res, err := newTestClientTransport(bodyRoundTripper{body: failingBody{}}).RoundTrip(req)
	require.NoError(t, err)
	defer res.Body.Close()
	_, err = io.ReadAll(res.Body)
	require.Error(t, err)

	// The span ends with the failed read, not at Close.
	span := telemetrytest.EndedSpan(t, ended(), "test GET")
	require.Equal(t, codes.Error, span.Status().Code)
	require.Equal(t, "connection reset", span.Status().Description)
}

// idleClosingRoundTripper records CloseIdleConnections.
type idleClosingRoundTripper struct {
	okRoundTripper
	closed *bool
}

func (rt idleClosingRoundTripper) CloseIdleConnections() { *rt.closed = true }

func TestClientTransport_wrappedRoundTripper(t *testing.T) {
	t.Parallel()
	var closed bool
	utilnet.CloseIdleConnectionsFor(newTestClientTransport(idleClosingRoundTripper{closed: &closed}))
	require.True(t, closed)
}

func TestClientTransport_untraced(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	ended := telemetrytest.RecordSpans(t)
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"https://example.com/untraced",
		nil,
	)
	require.NoError(t, err)
	res, err := newTestClientTransport(okRoundTripper{}).RoundTrip(req)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	require.Empty(t, ended())
}

// roundTripperFunc is an http.RoundTripper backed by a function.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// respondWith answers every request with status and body.
func respondWith(status int, body io.Reader) roundTripperFunc {
	return func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Header:     http.Header{},
			Body:       io.NopCloser(body),
			Request:    r,
		}, nil
	}
}

// cancelingBody fails every read with a cancellation.
type cancelingBody struct{}

func (cancelingBody) Read([]byte) (int, error) { return 0, context.Canceled }

func TestClientTransport_outcome(t *testing.T) {
	testCases := []struct {
		name   string
		next   roundTripperFunc
		assert func(*testing.T, sdktrace.ReadOnlySpan, attribute.Set)
	}{
		{
			name: "4xx is not an error",
			next: respondWith(http.StatusNotFound, strings.NewReader("")),
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan, attrs attribute.Set) {
				require.Equal(t, codes.Unset, span.Status().Code)
				code, ok := attrs.Value(semconv.HTTPResponseStatusCodeKey)
				require.True(t, ok)
				require.Equal(t, int64(http.StatusNotFound), code.AsInt64())
				require.False(t, attrs.HasValue(semconv.ErrorTypeKey))
			},
		},
		{
			name: "5xx is an error",
			next: respondWith(http.StatusBadGateway, strings.NewReader("")),
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan, attrs attribute.Set) {
				require.Equal(t, codes.Error, span.Status().Code)
				require.Equal(t, "Bad Gateway", span.Status().Description)
				errType, ok := attrs.Value(semconv.ErrorTypeKey)
				require.True(t, ok)
				require.Equal(t, "502", errType.AsString())
			},
		},
		{
			name: "failed round trip is an error",
			next: func(*http.Request) (*http.Response, error) {
				return nil, errors.New("connection refused")
			},
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan, attrs attribute.Set) {
				require.Equal(t, codes.Error, span.Status().Code)
				require.Equal(t, "connection refused", span.Status().Description)
				require.True(t, attrs.HasValue(semconv.ErrorTypeKey))
				require.False(t, attrs.HasValue(semconv.HTTPResponseStatusCodeKey))
			},
		},
		{
			name: "canceled round trip is not an error",
			next: func(*http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("error dialing: %w", context.Canceled)
			},
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan, attrs attribute.Set) {
				require.Equal(t, codes.Unset, span.Status().Code)
				require.Empty(t, span.Events())
				require.False(t, attrs.HasValue(semconv.ErrorTypeKey))
			},
		},
		{
			name: "canceled body read is not an error",
			next: respondWith(http.StatusOK, cancelingBody{}),
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan, _ attribute.Set) {
				require.Equal(t, codes.Unset, span.Status().Code)
				require.Empty(t, span.Events())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Not parallel: installs a global tracer provider.
			ended := telemetrytest.RecordSpans(t)
			req, err := http.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				"https://example.com/x",
				nil,
			)
			require.NoError(t, err)
			res, err := newTestClientTransport(testCase.next).RoundTrip(req)
			if res != nil {
				_, _ = io.ReadAll(res.Body)
				require.NoError(t, res.Body.Close())
			}
			if err != nil {
				require.Nil(t, res)
			}
			span := telemetrytest.EndedSpan(t, ended(), "test GET")
			testCase.assert(t, span, attribute.NewSet(span.Attributes()...))
		})
	}
}

func TestClientTransport_propagate(t *testing.T) {
	testCases := []struct {
		name      string
		propagate bool
	}{
		{name: "propagates", propagate: true},
		{name: "does not propagate"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Not parallel: installs a global tracer provider.
			ended := telemetrytest.RecordSpans(t)
			var sent *http.Request
			transport := newTestClientTransport(roundTripperFunc(
				func(r *http.Request) (*http.Response, error) {
					sent = r
					return respondWith(http.StatusOK, strings.NewReader(""))(r)
				},
			))
			transport.propagate = testCase.propagate
			req, err := http.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				"https://example.com/x",
				nil,
			)
			require.NoError(t, err)
			res, err := transport.RoundTrip(req)
			require.NoError(t, err)
			require.NoError(t, res.Body.Close())

			require.Empty(t, req.Header.Get("traceparent"), "given request was modified")
			if !testCase.propagate {
				require.Same(t, req, sent)
				return
			}
			span := telemetrytest.EndedSpan(t, ended(), "test GET")
			require.Contains(
				t,
				sent.Header.Get("traceparent"),
				span.SpanContext().SpanID().String(),
			)
		})
	}
}

func TestClientTransport_notRecording(t *testing.T) {
	t.Parallel()
	transport := newTestClientTransport(okRoundTripper{})
	transport.tracer = noop.NewTracerProvider().Tracer(clientScope)
	transport.attributes = func(*http.Request) []attribute.KeyValue {
		t.Error("attributes called for a span that is not recording")
		return nil
	}
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"https://example.com/x",
		nil,
	)
	require.NoError(t, err)
	res, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
}
