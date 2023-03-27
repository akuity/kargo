package kubeclient

import (
	"context"
)

type authCredentialContextKey struct {
	// explicitly empty
}

func ContextWithAuthCredential(ctx context.Context, cred string) context.Context {
	return context.WithValue(ctx, authCredentialContextKey{}, cred)
}

func AuthCredentialFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(authCredentialContextKey{}).(string)
	return v, ok
}
