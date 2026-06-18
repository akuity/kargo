package server

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// listForWatchSeed lists Kargo resources straight from the Kubernetes API,
// bypassing the API server's controller-runtime read cache via the client's
// uncached APIReader.
//
// The cached client can return a list ResourceVersion of "0" or one that is
// older than the apiserver's compacted floor, which makes follow-up watches
// either replay the full set or fail with a "too old" error and force the
// client into a refetch loop. listForWatchSeed avoids that by reading through
// the uncached APIReader for list+watch seed endpoints, where the returned
// resourceVersion is used to start a follow-up watch.
//
// The APIReader does not enforce Kargo's RBAC, so we authorize the caller first
// with the same list SubjectAccessReview the cached client would perform.
// resource is the lowercase resource name in the Kargo API group (e.g.
// "stages", "warehouses", "promotions", "freights").
func (s *server) listForWatchSeed(
	ctx context.Context,
	resource string,
	list client.ObjectList,
	opts ...client.ListOption,
) error {
	if s.client == nil {
		return fmt.Errorf("kubernetes client is not configured")
	}
	if s.authorizeFn == nil {
		return fmt.Errorf("authorize function is not configured")
	}

	var listOpts client.ListOptions
	listOpts.ApplyOptions(opts)

	// Authorize the user before bypassing the cache. The APIReader runs with the
	// API server's own credentials, so without this check a caller could read
	// data they would otherwise be denied.
	if err := s.authorizeFn(
		ctx,
		"list",
		kargoapi.GroupVersion.WithResource(resource),
		"",
		client.ObjectKey{Namespace: listOpts.Namespace},
	); err != nil {
		return err
	}

	return s.client.APIReader().List(ctx, list, opts...)
}
