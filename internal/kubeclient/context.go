package kubeclient

import (
	"context"
)

type credentialKey struct {
	// explicitly empty
}

func SetCredentialToContext(ctx context.Context, cred string) context.Context {
	return context.WithValue(ctx, credentialKey{}, cred)
}

func GetCredentialFromContext(ctx context.Context) (string, bool) {
	cred, ok := ctx.Value(credentialKey{}).(string)
	return cred, ok
}
