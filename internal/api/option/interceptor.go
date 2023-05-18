package option

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bufbuild/connect-go"
	log "github.com/sirupsen/logrus"
)

func NewHandlerOption(logger *log.Entry) connect.HandlerOption {
	return connect.WithHandlerOptions(
		connect.WithCodec(newJSONCodec("json")),
		connect.WithCodec(newJSONCodec("json; charset=utf-8")),
		connect.WithInterceptors(
			newLogInterceptor(logger, loggingIgnorableMethods),
		),
		connect.WithRecover(
			func(
				ctx context.Context,
				spec connect.Spec,
				header http.Header,
				r any,
			) error {
				return connect.NewError(
					connect.CodeInternal, fmt.Errorf("panic: %v", r))
			}),
	)
}
