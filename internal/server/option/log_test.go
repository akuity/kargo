package option

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	grpchealth "connectrpc.com/grpchealth"
	"github.com/bombsimon/logrusr/v4"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/akuity/kargo/internal/logging"
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
			// TODO: This is an odd spot where we're using logrus directly until we
			// figure out how to do the equivalent thing (assert that a message WAS
			// logged) with logr.
			logrusLogger, hook := testlog.NewNullLogger()
			logger := logging.Wrap(logrusr.New(logrusLogger))

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

func TestStreamingServerLogging(_ *testing.T) {
	// TODO
}
