package user

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
)

type userInfoKey struct{}

// Info represents information about an API user. This is bound to the context
// by the server's authentication middleware and later retrieved and used to
// create an ad-hoc Kubernetes client that has the correct level of permissions
// for the user.
type Info struct {
	// IsAdmin indicates whether the user represented by this struct has been
	// verified as the Kargo API server's admin user. When this is true, all
	// other fields should have an empty value.
	IsAdmin bool
	// Claims is a map of claims from an identity provider of a
	// non-admin user whose credentials have
	// been successfully verified by the server's authentication middleware.
	Claims map[string]any
	// BearerToken is the raw bearer token presented in the Authorization header
	// of any request requiring authentication.
	BearerToken string
	// ServiceAccountsByNamespace is the mapping of namespace names to sets of
	// ServiceAccounts that a user has been mapped to.
	ServiceAccountsByNamespace map[string]map[types.NamespacedName]struct{}
	// Username is the username of the user. This is typically the email address
	// of the user, but may be different depending on the identity provider.
	Username string
}

// ContextWithInfo returns a context.Context that has been augmented with
// the provided Info.
func ContextWithInfo(ctx context.Context, u Info) context.Context {
	return context.WithValue(ctx, userInfoKey{}, u)
}

// InfoFromContext extracts a userInfo from the provided context.Context and
// returns it. If no Info is found, a zero-value Info is returned. A
// boolean is also returned to indicate the success or failure of the call.
func InfoFromContext(ctx context.Context) (Info, bool) {
	val := ctx.Value(userInfoKey{})
	if val == nil {
		return Info{}, false
	}
	u, ok := val.(Info)
	return u, ok
}
