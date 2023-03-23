package context

import (
	"context"
)

type authCredentialContextKey struct {
	// explicitly empty
}

func SetAuthCredential(ctx context.Context, cred string) context.Context {
	return set[authCredentialContextKey, string](ctx, authCredentialContextKey{}, cred)
}

func GetAuthCredential(ctx context.Context) (string, bool) {
	return get[authCredentialContextKey, string](ctx, authCredentialContextKey{})
}

func set[K comparable, V any](ctx context.Context, k K, v V) context.Context {
	return context.WithValue(ctx, k, v)
}

func get[K comparable, V any](ctx context.Context, k K) (V, bool) {
	v, ok := ctx.Value(k).(V)
	return v, ok
}
