package option

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/server/config"
)

func NewHandlerOption(
	ctx context.Context,
	cfg config.ServerConfig,
	kubeclient client.Client,
) (connect.HandlerOption, error) {
	interceptors := []connect.Interceptor{
		newLogInterceptor(logging.LoggerFromContext(ctx), loggingIgnorableMethods),
		newErrorInterceptor(),
	}
	if !cfg.LocalMode {
		interceptors = append(interceptors, newAuthInterceptor(ctx, cfg, kubeclient))
	}
	return connect.WithHandlerOptions(
		connect.WithInterceptors(interceptors...),
		connect.WithRecover(
			func(ctx context.Context, _ connect.Spec, _ http.Header, r any) error {
				logging.LoggerFromContext(ctx).Error(nil, takeStacktrace(defaultStackLength, 3))
				return connect.NewError(
					connect.CodeInternal, fmt.Errorf("panic: %v", r))
			}),
	), nil
}
