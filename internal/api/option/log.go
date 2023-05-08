package option

import (
	"context"
	"path"
	"time"

	"github.com/bufbuild/connect-go"
	log "github.com/sirupsen/logrus"

	"github.com/akuity/kargo/internal/logging"
)

var (
	loggingIgnorableMethods = map[string]bool{
		"/grpc.health.v1.Health/Check": true,
		"/grpc.health.v1.Health/Watch": true,
	}
)

var (
	_ connect.Interceptor = &logInterceptor{}
)

type logInterceptor struct {
	logger           *log.Entry
	ignorableMethods map[string]bool
}

func newLogInterceptor(logger *log.Entry, ignorableMethods map[string]bool) connect.Interceptor {
	return &logInterceptor{
		logger:           logger,
		ignorableMethods: ignorableMethods,
	}
}

func (i *logInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		start := time.Now()
		ctx = i.newLogger(ctx, req.Spec().Procedure, start)
		if !i.shouldLog(req.Spec().Procedure) {
			return next(ctx, req)
		}

		res, err := next(ctx, req)
		fields := log.Fields{
			"connect.duration": time.Since(start).String(),
		}
		level := log.InfoLevel
		if err != nil {
			level = log.ErrorLevel
			fields["connect.code"] = connect.CodeOf(err).String()
		}
		logging.
			LoggerFromContext(ctx).
			WithFields(fields).
			Log(level, "finished unary call")
		return res, err
	}
}

func (i *logInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	// TODO: Support streaming client
	return next
}

func (i *logInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		start := time.Now()
		ctx = i.newLogger(ctx, conn.Spec().Procedure, start)
		if !i.shouldLog(conn.Spec().Procedure) {
			return next(ctx, conn)
		}

		err := next(ctx, conn)
		fields := log.Fields{
			"connect.duration": time.Since(start),
		}
		level := log.InfoLevel
		if err != nil {
			level = log.ErrorLevel
			fields["connect.code"] = connect.CodeOf(err).String()
		}
		logging.
			LoggerFromContext(ctx).
			WithFields(fields).
			Log(level, "finished streaming call")
		return err
	}
}

func (i *logInterceptor) newLogger(ctx context.Context, procedure string, start time.Time) context.Context {
	service := path.Dir(procedure)[1:]
	method := path.Base(procedure)
	logger := i.logger.WithFields(log.Fields{
		"connect.service":    service,
		"connect.method":     method,
		"connect.start_time": start.Format(time.RFC3339),
	})
	return logging.ContextWithLogger(ctx, logger)
}

func (i *logInterceptor) shouldLog(procedure string) bool {
	return !i.ignorableMethods[procedure]
}
