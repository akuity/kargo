package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetFreightByNameOrAlias returns a pointer to the Freight resource specified
// by the project, and name OR alias arguments. If no such resource is found,
// nil is returned instead.
func GetFreightByNameOrAlias(
	ctx context.Context,
	c client.Client,
	project string,
	name string,
	alias string,
) (*Freight, error) {
	if name != "" {
		return GetFreight(
			ctx,
			c,
			types.NamespacedName{
				Namespace: project,
				Name:      name,
			},
		)
	}
	return GetFreightByAlias(ctx, c, project, alias)
}

// GetFreight returns a pointer to the Freight resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func GetFreight(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*Freight, error) {
	freight := Freight{}
	if err := c.Get(ctx, namespacedName, &freight); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"error getting Freight %q in namespace %q: %w",
			namespacedName.Name,
			namespacedName.Namespace,
			err,
		)
	}
	return &freight, nil
}

// GetFreightByAlias returns a pointer to the Freight resource specified by the
// project and alias arguments. If no such resource is found, nil is returned
// instead.
func GetFreightByAlias(
	ctx context.Context,
	c client.Client,
	project string,
	alias string,
) (*Freight, error) {
	freightList := FreightList{}
	if err := c.List(
		ctx,
		&freightList,
		client.InNamespace(project),
		client.MatchingLabels{
			AliasLabelKey: alias,
		},
	); err != nil {
		return nil, fmt.Errorf(
			"error listing Freight with alias %q in namespace %q: %w",
			alias,
			project,
			err,
		)
	}
	if len(freightList.Items) == 0 {
		return nil, nil
	}
	return &freightList.Items[0], nil
}

// IsFreightAvailable answers whether the specified Freight is available to the
// specified Stage having the specified upstream stages. Freight is available
// if:
//
//  1. No upstreamStages are specified
//     OR
//  2. The Freight has has been verified in ANY of the specified upstream stages
//     OR
//  3. The Freight is approved for the specified stage
//
// Note: The rationale for returning true when no upstream stages are specified
// is that some Stages have no upstream Stages (e.g. a Stage that subscribes to
// a Warehouse), so ANY Freight is available to such a Stage.
func IsFreightAvailable(
	freight *Freight,
	stage string,
	upstreamStages []string,
) bool {
	if len(upstreamStages) == 0 {
		return true
	}
	for _, stage := range upstreamStages {
		if _, ok := freight.Status.VerifiedIn[stage]; ok {
			return true
		}
	}
	if stage != "" {
		if _, ok := freight.Status.ApprovedFor[stage]; ok {
			return true
		}
	}
	return false
}
