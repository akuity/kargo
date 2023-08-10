package option

import (
	"context"
	"net/http"
	"strings"

	"github.com/bufbuild/connect-go"

	"github.com/akuity/kargo/internal/kubeclient"
)

const authHeaderKey = "Authorization"

// authInterceptor implements connect.Interceptor and is used to retrieve the
// value of the Authorization header from inbound requests/connections and
// store it in the context.
type authInterceptor struct{}

func (a *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		cred := credFromAuthHeader(req.Header())
		if cred != "" {
			// TODO: Determine what kind of credential this is. If it's a JWT
			// we should verify it and map it to a Kubernetes service account. If it's
			// not a JWT, we'll assume it might be a bearer token for the Kubernetes
			// API server and no mapping is needed.
			ctx = kubeclient.SetCredentialToContext(ctx, cred)
		}
		return next(ctx, req)
	}
}

func (a *authInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		// This is a no-op because this interceptor is only used with handlers.
		return next(ctx, spec)
	}
}

func (a *authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		cred := credFromAuthHeader(conn.RequestHeader())
		if cred != "" {
			// TODO: Determine what kind of credential this is. If it's a JWT
			// we should verify it and map it to a Kubernetes service account. If it's
			// not a JWT, we'll assume it might be a bearer token for the Kubernetes
			// API server and no mapping is needed.
			ctx = kubeclient.SetCredentialToContext(ctx, cred)
		}
		return next(ctx, conn)
	}
}

func credFromAuthHeader(header http.Header) string {
	return strings.TrimPrefix(header.Get(authHeaderKey), "Bearer ")
}
