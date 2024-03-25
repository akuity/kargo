package option

import (
	"context"
	"log/slog"
	"path"
	"time"

	"connectrpc.com/connect"
	"github.com/go-logr/logr"

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
	logger           logr.Logger
	ignorableMethods map[string]bool
}

func newLogInterceptor(
	logger logr.Logger,
	ignorableMethods map[string]bool,
) connect.Interceptor {
	return &logInterceptor{
		logger:           logger,
		ignorableMethods: ignorableMethods,
	}
}

func (i *logInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(
		ctx context.Context,
		req connect.AnyRequest,
	) (connect.AnyResponse, error) {
		start := time.Now()
		ctx = i.newLogger(ctx, req.Spec().Procedure, start)
		if !i.shouldLog(req.Spec().Procedure) {
			return next(ctx, req)
		}

		res, err := next(ctx, req)
		fields := map[string]any{
			"connect.duration": time.Since(start).String(),
		}
		level := slog.LevelInfo
		if err != nil {
			level = slog.LevelError
			fields["connect.code"] = connect.CodeOf(err).String()
		}
		logging.
			LoggerFromContext(ctx).
			WithValues(fields).
			V(int(level)).Info("finished unary call")
		return res, err
	}
}

func (i *logInterceptor) WrapStreamingClient(
	next connect.StreamingClientFunc) connect.StreamingClientFunc {
	// TODO: Support streaming client
	return next
}

func (i *logInterceptor) WrapStreamingHandler(
	next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		start := time.Now()
		ctx = i.newLogger(ctx, conn.Spec().Procedure, start)
		if !i.shouldLog(conn.Spec().Procedure) {
			return next(ctx, conn)
		}

		err := next(ctx, conn)
		fields := map[string]any{
			"connect.duration": time.Since(start),
		}
		level := slog.LevelInfo
		if err != nil {
			level = slog.LevelInfo
			fields["connect.code"] = connect.CodeOf(err).String()
		}
		logging.
			LoggerFromContext(ctx).
			WithValues(fields).
			V(int(level)).
			Info("finished streaming call")
		return err
	}
}

func (i *logInterceptor) newLogger(
	ctx context.Context, procedure string, start time.Time) context.Context {
	service := path.Dir(procedure)[1:]
	method := path.Base(procedure)
	logger := i.logger.WithValues(
		"connect.service", service,
		"connect.method", method,
		"connect.start_time", start.Format(time.RFC3339),
	)
	return logging.ContextWithLogger(ctx, logger)
}

func (i *logInterceptor) shouldLog(procedure string) bool {
	return !i.ignorableMethods[procedure]
}
