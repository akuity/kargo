package option

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bufbuild/connect-go"
	log "github.com/sirupsen/logrus"

	"github.com/akuity/kargo/internal/config"
)

func NewHandlerOption(cfg config.APIConfig, logger *log.Entry) connect.HandlerOption {
	interceptors := []connect.Interceptor{
		newLogInterceptor(logger, loggingIgnorableMethods),
	}
	if !cfg.LocalMode {
		interceptors = append(interceptors, newAuthInterceptor())
	}
	return connect.WithHandlerOptions(
		connect.WithCodec(newJSONCodec("json")),
		connect.WithCodec(newJSONCodec("json; charset=utf-8")),
		connect.WithInterceptors(interceptors...),
		connect.WithRecover(
			func(ctx context.Context, spec connect.Spec, header http.Header, r any) error {
				return connect.NewError(
					connect.CodeInternal, fmt.Errorf("panic: %v", r))
			}),
	)
}
