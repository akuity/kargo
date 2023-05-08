package option

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bufbuild/connect-go"
	grpchealth "github.com/bufbuild/connect-grpchealth-go"
	testlog "github.com/sirupsen/logrus/hooks/test"
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
			logger, hook := testlog.NewNullLogger()

			opt := connect.WithInterceptors(
				newLogInterceptor(logger.WithFields(nil), testSet.ignorableMethods))
			mux := http.NewServeMux()
			mux.Handle(grpchealth.NewHandler(grpchealth.NewStaticChecker(), opt))
			srv := httptest.NewServer(mux)
			srv.EnableHTTP2 = true
			t.Cleanup(srv.Close)

			client := connect.NewClient[grpc_health_v1.HealthCheckRequest, grpc_health_v1.HealthCheckResponse](
				srv.Client(),
				srv.URL+"/grpc.health.v1.Health/Check",
				connect.WithGRPC(),
			)
			_, err := client.CallUnary(context.Background(),
				connect.NewRequest[grpc_health_v1.HealthCheckRequest](&grpc_health_v1.HealthCheckRequest{}))
			require.NoError(t, err)

			if testSet.logExpected {
				entry := hook.LastEntry()
				require.NotNil(t, entry)
				for _, field := range []string{
					"connect.service",
					"connect.method",
					"connect.start_time",
					"connect.duration",
				} {
					require.Contains(t, entry.Data, field)
				}
			} else {
				require.Nil(t, hook.LastEntry())
			}
		})
	}
}

func TestStreamingServerLogging(t *testing.T) {
	// TODO
}
