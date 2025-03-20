package main

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
)

const authHeaderKey = "Authorization"

// authInterceptor implements connect.Interceptor and is used to decorate
// outbound requests/connections with an appropriate Authorization header.
type authInterceptor struct {
	credential string
}

func (a *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if a.credential != "" {
			setAuthHeader(req.Header(), a.credential)
		}
		return next(ctx, req)
	}
}

func (a *authInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		if a.credential != "" {
			setAuthHeader(conn.RequestHeader(), a.credential)
		}
		return conn
	}
}

func (a *authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		// This is a no-op because this interceptor is only used with clients.
		return next(ctx, conn)
	}
}

func setAuthHeader(header http.Header, cred string) {
	if cred != "" {
		header.Set(authHeaderKey, fmt.Sprintf("Bearer %s", cred))
	}
}
