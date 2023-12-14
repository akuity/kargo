package option

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
)

var (
	_ connect.Interceptor = &errorInterceptor{}
)

type errorInterceptor struct{}

func newErrorInterceptor() connect.Interceptor {
	return &errorInterceptor{}
}

func (i *errorInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		res, err := next(ctx, req)
		if err != nil {
			return nil, i.unwrapError(err)
		}
		return res, nil
	}
}

func (i *errorInterceptor) WrapStreamingClient(
	next connect.StreamingClientFunc,
) connect.StreamingClientFunc {
	// TODO: Support streaming client when necessary
	return next
}

func (i *errorInterceptor) WrapStreamingHandler(
	next connect.StreamingHandlerFunc,
) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if err := next(ctx, conn); err != nil {
			return i.unwrapError(err)
		}
		return nil
	}
}

func (i *errorInterceptor) unwrapError(err error) error {
	var connectErr *connect.Error
	// Extract underlying error if the error is a connect error
	if ok := errors.As(err, &connectErr); ok {
		// If the error code is other than Unknown, return error as is since it is intended.
		if connectErr.Code() != connect.CodeUnknown {
			return err
		}
		return i.unwrapError(connectErr.Unwrap())
	}
	var statusErr *kubeerr.StatusError
	if ok := errors.As(err, &statusErr); ok {
		return connect.NewError(httpStatusToConnectCode(statusErr.Status().Code), statusErr)
	}
	return err
}

func httpStatusToConnectCode(status int32) connect.Code {
	switch status {
	case http.StatusBadRequest:
		return connect.CodeInvalidArgument
	case http.StatusUnauthorized:
		return connect.CodeUnauthenticated
	case http.StatusForbidden:
		return connect.CodePermissionDenied
	case http.StatusNotFound:
		return connect.CodeNotFound
	case http.StatusConflict:
		return connect.CodeAlreadyExists
	case http.StatusGone:
		return connect.CodeNotFound
	case http.StatusTooManyRequests:
		return connect.CodeResourceExhausted
	case 499:
		return connect.CodeCanceled
	case http.StatusInternalServerError:
		return connect.CodeInternal
	case http.StatusNotImplemented:
		return connect.CodeUnimplemented
	case http.StatusServiceUnavailable:
		return connect.CodeUnavailable
	case http.StatusGatewayTimeout:
		return connect.CodeDeadlineExceeded
	default:
		return connect.CodeUnknown
	}
}
