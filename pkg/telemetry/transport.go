package telemetry

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// WrapTransport makes every request through rt a client span,
// "<prefix> <method>", sending no trace context to the third party. It returns
// rt unchanged unless CategoryDiscovery is enabled.
func WrapTransport(rt http.RoundTripper, prefix string) http.RoundTripper {
	if !CategoryEnabled(CategoryDiscovery) {
		return rt
	}
	return clientTransport{
		next:   rt,
		tracer: otel.GetTracerProvider().Tracer(clientScope),
		name: func(r *http.Request) (string, bool) {
			return prefix + " " + r.Method, true
		},
		attributes: func(r *http.Request) []attribute.KeyValue {
			return []attribute.KeyValue{sanitizedURLFull(r)}
		},
	}
}

// sanitizedURLFull returns r's url.full without userinfo or query.
func sanitizedURLFull(r *http.Request) attribute.KeyValue {
	return semconv.URLFull(SanitizeURL(r.URL.String()))
}
