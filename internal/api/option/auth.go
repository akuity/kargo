package option

import (
	"context"

	"github.com/bufbuild/connect-go"

	"github.com/akuity/kargo/internal/kubeclient"
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
				req.Header().Set("Authorization", cred)
			}
			return next(ctx, req)
		}

		if len(req.Header().Values("Authorization")) == 0 {
			return next(ctx, req)
		}

		cred := req.Header().Get("Authorization")
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
		conn.RequestHeader().Set("Authorization", cred)
		return conn
	}
}

func (a *authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if len(conn.RequestHeader().Values("Authorization")) == 0 {
			return next(ctx, conn)
		}
		cred := conn.RequestHeader().Get("Authorization")
		return next(kubeclient.SetCredentialToContext(ctx, cred), conn)
	}
}
