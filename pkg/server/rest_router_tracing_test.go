package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/akuity/kargo/pkg/server/config"
)

func TestSetupRESTRouter_tracing(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name   string
		cfg    config.ServerConfig
		assert func(*testing.T, *tracetest.SpanRecorder)
	}{
		{
			name: "tracing disabled",
			cfg:  config.ServerConfig{TracingEnabled: false},
			assert: func(t *testing.T, recorder *tracetest.SpanRecorder) {
				require.Empty(t, recorder.Ended())
			},
		},
		{
			name: "tracing enabled",
			cfg:  config.ServerConfig{TracingEnabled: true},
			assert: func(t *testing.T, recorder *tracetest.SpanRecorder) {
				ended := recorder.Ended()
				require.Len(t, ended, 1)
				span := ended[0]
				// Named after the route pattern, not the concrete path.
				require.Equal(t, "GET /v1beta1/system/server-version", span.Name())
				require.Equal(t, trace.SpanKindServer, span.SpanKind())
				require.Subset(
					t,
					span.Attributes(),
					[]any{
						semconv.HTTPRequestMethodGet,
						semconv.HTTPRouteKey.String("/v1beta1/system/server-version"),
						semconv.HTTPResponseStatusCode(http.StatusOK),
					},
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			prev := otel.GetTracerProvider()
			t.Cleanup(func() { otel.SetTracerProvider(prev) })
			recorder := tracetest.NewSpanRecorder()
			otel.SetTracerProvider(sdktrace.NewTracerProvider(
				sdktrace.WithSpanProcessor(recorder),
			))

			s := &server{cfg: testCase.cfg}
			router := s.setupRESTRouter(t.Context())

			w := httptest.NewRecorder()
			req := httptest.NewRequest(
				http.MethodGet,
				"/v1beta1/system/server-version",
				nil,
			)
			router.ServeHTTP(w, req)
			require.Equal(t, http.StatusOK, w.Code)

			testCase.assert(t, recorder)
		})
	}
}
