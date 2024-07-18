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
	// Subject is the unique identified of a non-admin user whose credentials have
	// been successfully verified by the server's authentication middleware.
	Subject string
	// Email is the verified email address of a non-admin user whose credentials
	// have been successfully verified by the server's authentication middleware.
	Email string
	// Groups are the group claims obtained from credentials that have been
	// successfully verified by the server's authentication middleware.
	Groups []string
	Claims map[string]any
	// BearerToken is set only in cases where the server's authentication
	// middleware could not verify the token it was presented with. In this case,
	// we assume the token to be a valid credential for a Kubernetes user. When
	// constructing an ad-hoc Kubernetes client, this token will be used directly.
	// When this is non-empty, all other fields should have an empty value.
	BearerToken string
	// ServiceAccountsByNamespace is the mapping of namespace names to sets of
	// ServiceAccounts that a user has been mapped to.
	ServiceAccountsByNamespace map[string]map[types.NamespacedName]struct{}
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
