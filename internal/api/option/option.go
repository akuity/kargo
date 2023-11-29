package option

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
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
	}
	if !cfg.LocalMode {
		authInterceptor, err := newAuthInterceptor(ctx, cfg, internalClient)
		if err != nil {
			return nil,
				errors.Wrap(err, "error initializing authentication interceptor")
		}
		interceptors = append(interceptors, authInterceptor)
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
	), nil
}
