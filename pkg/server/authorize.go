package server

import (
	"context"

	"github.com/akuity/kargo/pkg/server/auth/can"
)

// authorize checks that the identity bound to the context may perform the
// access, which the authorizing client cannot check implicitly, e.g. the
// custom "promote" verb, or a read the server performs with its own client.
func (s *server) authorize(ctx context.Context, access can.Access) error {
	return s.authorizeFn(
		ctx,
		access.Verb,
		access.Resource,
		access.Subresource,
		access.Key,
	)
}
