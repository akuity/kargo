package option

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/bufbuild/connect-go"

	"github.com/akuity/kargo/internal/kubeclient"
)

const (
	authHeaderKey = "Authorization"
	bearerPrefix  = "Bearer "
)

var (
	_ connect.Interceptor = &authInterceptor{}
)

type authInterceptor struct{}

func newAuthInterceptor() connect.Interceptor {
	return &authInterceptor{}
}

func (a *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Spec().IsClient {
			cred, ok := kubeclient.GetCredentialFromContext(ctx)
			if ok {
				setAuthHeader(req.Header(), cred)
			}
			return next(ctx, req)
		}

		cred := credFromAuthHeader(req.Header())
		if cred == "" {
			return next(ctx, req)
		}
		return next(kubeclient.SetCredentialToContext(ctx, cred), req)
	}
}

func (a *authInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		cred, ok := kubeclient.GetCredentialFromContext(ctx)
		if !ok {
			return next(ctx, spec)
		}

		conn := next(ctx, spec)
		setAuthHeader(conn.RequestHeader(), cred)
		return conn
	}
}

func (a *authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		cred := credFromAuthHeader(conn.RequestHeader())
		if cred == "" {
			return next(ctx, conn)
		}
		return next(kubeclient.SetCredentialToContext(ctx, cred), conn)
	}
}

func credFromAuthHeader(header http.Header) string {
	return strings.TrimPrefix(header.Get(authHeaderKey), bearerPrefix)
}

func setAuthHeader(header http.Header, cred string) {
	if cred != "" {
		header.Set(authHeaderKey, fmt.Sprintf("%s%s", bearerPrefix, cred))
	}
}
