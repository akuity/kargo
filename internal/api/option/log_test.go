package option

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	grpchealth "connectrpc.com/grpchealth"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/health/grpc_health_v1"
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
			tmpWriteBuffer := bytes.NewBuffer(nil)
			logger := logr.FromSlogHandler(slog.NewTextHandler(tmpWriteBuffer, nil))

			opt := connect.WithInterceptors(
				newLogInterceptor(logger.WithValues(nil), testSet.ignorableMethods))
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

			if testSet.logExpected {
				entry := tmpWriteBuffer.String()
				require.NotNil(t, entry)
				for _, field := range []string{
					"connect.service",
					"connect.method",
					"connect.start_time",
					"connect.duration",
				} {
					require.Contains(t, entry, field)
				}
			} else {
				require.Empty(t, tmpWriteBuffer.String())
			}
		})
	}
}

func TestStreamingServerLogging(_ *testing.T) {
	// TODO
}
