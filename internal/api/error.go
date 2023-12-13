package api

import (
	"net/http"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
)

func getCodeFromError(err error) connect.Code {
	var statusErr *kubeerr.StatusError
	if ok := errors.As(err, &statusErr); ok {
		return httpStatusToConnectCode(statusErr.Status().Code)
	}
	return connect.CodeUnknown
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
