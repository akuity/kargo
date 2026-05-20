package server

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func (s *server) listFresh(
	ctx context.Context,
	resource string,
	list client.ObjectList,
	opts ...client.ListOption,
) error {
	if s.cfg.RestConfig == nil {
		if s.client == nil {
			return fmt.Errorf("kubernetes client is not configured")
		}
		return s.client.List(ctx, list, opts...)
	}

	var listOpts client.ListOptions
	listOpts.ApplyOptions(opts)
	if s.authorizeFn == nil {
		return fmt.Errorf("authorize function is not configured")
	}
	if err := s.authorizeFn(
		ctx,
		"list",
		kargoapi.GroupVersion.WithResource(resource),
		"",
		client.ObjectKey{Namespace: listOpts.Namespace},
	); err != nil {
		return err
	}

	directClient, err := client.New(
		s.cfg.RestConfig,
		client.Options{
			Scheme: s.client.Scheme(),
			Mapper: s.client.RESTMapper(),
		},
	)
	if err != nil {
		return fmt.Errorf("create direct Kubernetes client: %w", err)
	}
	return directClient.List(ctx, list, opts...)
}
