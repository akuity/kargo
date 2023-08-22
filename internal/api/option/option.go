package option

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	log "github.com/sirupsen/logrus"

	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/logging"
)

func NewHandlerOption(ctx context.Context, cfg config.ServerConfig) connect.HandlerOption {
	interceptors := []connect.Interceptor{
		newLogInterceptor(logging.LoggerFromContext(ctx), loggingIgnorableMethods),
	}
	if !cfg.LocalMode {
		interceptors = append(interceptors, &authInterceptor{})
	}
	return connect.WithHandlerOptions(
		connect.WithCodec(newJSONCodec("json")),
		connect.WithCodec(newJSONCodec("json; charset=utf-8")),
		connect.WithInterceptors(interceptors...),
		connect.WithRecover(
			func(ctx context.Context, spec connect.Spec, header http.Header, r any) error {
				logging.LoggerFromContext(ctx).Log(log.ErrorLevel, takeStacktrace(defaultStackLength, 3))
				return connect.NewError(
					connect.CodeInternal, fmt.Errorf("panic: %v", r))
			}),
	)
}
