package interceptor

import (
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	loggableMethods = map[string]bool{
		"/grpc.health.v1.Health/Check": false,
		"/grpc.health.v1.Health/Watch": false,
	}
)

func NewStreamInterceptor(logger *logrus.Entry) grpc.ServerOption {
	return grpc.ChainStreamInterceptor(
		grpc_logrus.StreamServerInterceptor(logger, newLogDecider()),
		recovery.StreamServerInterceptor(),
	)
}

func NewUnaryInterceptor(logger *logrus.Entry) grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(
		grpc_logrus.UnaryServerInterceptor(logger, newLogDecider()),
		recovery.UnaryServerInterceptor(),
	)
}

func newLogDecider() grpc_logrus.Option {
	return grpc_logrus.WithDecider(func(fullMethodName string, err error) bool {
		skip, ok := loggableMethods[fullMethodName]
		if !ok {
			return true
		}
		// Log error even if this method should be skipped
		return err != nil || skip
	})
}
