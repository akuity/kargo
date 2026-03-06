package option

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	grpchealth "connectrpc.com/grpchealth"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/akuity/kargo/pkg/logging"
)

func TestUnaryServerLogging(t *testing.T) {
	testSets := map[string]struct {
		ignorableMethods map[string]bool
		logExpected      bool
	}{
		"loggable method": {
			logExpected: true,
		},
		"ignorable method": {
			ignorableMethods: map[string]bool{
				"/grpc.health.v1.Health/Check": true,
			},
			logExpected: false,
		},
	}
	for name, testSet := range testSets {
		t.Run(name, func(t *testing.T) {
			// Use zap's observer to capture log entries. This is how we can verify
			// that our logging interceptor actually logs something.
			core, logs := observer.New(zapcore.InfoLevel)
			zapLogger := zap.New(core)
			logger := logging.Wrap(zapLogger)

			opt := connect.WithInterceptors(
				newLogInterceptor(logger, testSet.ignorableMethods))
			mux := http.NewServeMux()
			mux.Handle(grpchealth.NewHandler(grpchealth.NewStaticChecker(), opt))
			srv := httptest.NewServer(mux)
			srv.EnableHTTP2 = true
			t.Cleanup(srv.Close)

			client := connect.NewClient[
				grpc_health_v1.HealthCheckRequest,
				grpc_health_v1.HealthCheckResponse](
				srv.Client(),
				srv.URL+"/grpc.health.v1.Health/Check",
				connect.WithGRPC(),
			)
			_, err := client.CallUnary(context.Background(),
				connect.NewRequest[grpc_health_v1.HealthCheckRequest](
					&grpc_health_v1.HealthCheckRequest{}))
			require.NoError(t, err)

			allLogs := logs.All()
			if testSet.logExpected {
				require.NotEmpty(t, allLogs)
				entry := allLogs[len(allLogs)-1]
				for _, field := range []string{
					"connect.service",
					"connect.method",
					"connect.start_time",
					"connect.duration",
				} {
					found := false
					for _, f := range entry.Context {
						if f.Key == field {
							found = true
							break
						}
					}
					require.True(t, found, "expected field %q in log entry", field)
				}
			} else {
				require.Empty(t, allLogs)
			}
		})
	}
}

func TestStreamingServerLogging(_ *testing.T) {
	// TODO
}
