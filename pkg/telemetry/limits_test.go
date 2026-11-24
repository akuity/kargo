package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestSpanLimits(t *testing.T) {
	// Not parallel: manipulates the environment.
	testCases := []struct {
		name        string
		env         map[string]string
		valueLength int
		linkCount   int
	}{
		{
			name:        "unset",
			valueLength: defaultAttributeValueLengthLimit,
			linkCount:   defaultLinkCountLimit,
		},
		{
			name: "general attribute limit",
			env: map[string]string{
				"OTEL_ATTRIBUTE_VALUE_LENGTH_LIMIT": "100",
				"OTEL_SPAN_LINK_COUNT_LIMIT":        "7",
			},
			valueLength: 100,
			linkCount:   7,
		},
		{
			name: "span attribute limit wins",
			env: map[string]string{
				"OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT": "50",
				"OTEL_ATTRIBUTE_VALUE_LENGTH_LIMIT":      "100",
			},
			valueLength: 50,
			linkCount:   defaultLinkCountLimit,
		},
		{
			name: "invalid values",
			env: map[string]string{
				"OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT": "lots",
				"OTEL_SPAN_LINK_COUNT_LIMIT":             "many",
			},
			valueLength: defaultAttributeValueLengthLimit,
			linkCount:   defaultLinkCountLimit,
		},
		{
			name: "invalid span attribute limit, valid general one",
			env: map[string]string{
				"OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT": "lots",
				"OTEL_ATTRIBUTE_VALUE_LENGTH_LIMIT":      "100",
			},
			valueLength: defaultAttributeValueLengthLimit,
			linkCount:   defaultLinkCountLimit,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			for _, key := range []string{
				"OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT",
				"OTEL_ATTRIBUTE_VALUE_LENGTH_LIMIT",
				"OTEL_SPAN_LINK_COUNT_LIMIT",
			} {
				t.Setenv(key, testCase.env[key])
			}
			limits := spanLimits()
			require.Equal(t, testCase.valueLength, limits.AttributeValueLengthLimit)
			require.Equal(t, testCase.linkCount, limits.LinkCountLimit)
		})
	}
}

func TestSetup_spanLimits(t *testing.T) {
	// Not parallel: changes global telemetry state.
	resetGlobals(t)
	for _, key := range []string{
		"OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT",
		"OTEL_ATTRIBUTE_VALUE_LENGTH_LIMIT",
		"OTEL_SPAN_LINK_COUNT_LIMIT",
	} {
		t.Setenv(key, "")
	}
	collector := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(collector.Close)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", collector.URL)
	shutdown, err := Setup(
		context.Background(),
		Config{Enabled: true, Protocol: ProtocolHTTPProtobuf},
		Service{Name: "test"},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, shutdown(ctx))
	})
	tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	require.True(t, ok)
	recorder := tracetest.NewSpanRecorder()
	tp.RegisterSpanProcessor(recorder)

	links := make([]trace.Link, defaultLinkCountLimit+8)
	for i := range links {
		links[i] = trace.Link{SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: trace.TraceID{byte(i + 1)},
			SpanID:  trace.SpanID{byte(i + 1)},
		})}
	}
	_, span := tp.Tracer("test").Start(
		context.Background(),
		"test",
		trace.WithLinks(links...),
		trace.WithAttributes(attribute.String(
			"long",
			strings.Repeat("x", defaultAttributeValueLengthLimit+100),
		)),
	)
	span.End()

	ended := recorder.Ended()
	require.Len(t, ended, 1)
	require.Len(t, ended[0].Attributes()[0].Value.AsString(), defaultAttributeValueLengthLimit)
	require.Len(t, ended[0].Links(), defaultLinkCountLimit)
}
