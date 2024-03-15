package option

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	log "github.com/sirupsen/logrus"
	libClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/logging"
)

func NewHandlerOption(
	ctx context.Context,
	cfg config.ServerConfig,
	internalClient libClient.Client,
) (connect.HandlerOption, error) {
	interceptors := []connect.Interceptor{
		newLogInterceptor(logging.LoggerFromContext(ctx), loggingIgnorableMethods),
		newErrorInterceptor(),
	}
	if !cfg.LocalMode {
		authInterceptor, err := newAuthInterceptor(ctx, cfg, internalClient)
		if err != nil {
			return nil, fmt.Errorf("initialize authentication interceptor: %w", err)
		}
		interceptors = append(interceptors, authInterceptor)
	}
	return connect.WithHandlerOptions(
		connect.WithInterceptors(interceptors...),
		connect.WithRecover(
			func(ctx context.Context, _ connect.Spec, _ http.Header, r any) error {
				logging.LoggerFromContext(ctx).Log(log.ErrorLevel, takeStacktrace(defaultStackLength, 3))
				return connect.NewError(
					connect.CodeInternal, fmt.Errorf("panic: %v", r))
			}),
	), nil
}
