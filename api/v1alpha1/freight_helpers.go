package v1alpha1

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
		return nil, errors.Wrapf(
			err,
			"error getting Freight %q in namespace %q",
			namespacedName.Name,
			namespacedName.Namespace,
		)
	}
	return &freight, nil
}

// GetAvailableFreight returns a pointer to the Freight resource specified by
// the namespacedName argument if it is found and:
//
//  1. No upstreamStages are specified
//     OR
//  2. The Freight has has been verified in ANY of the specified upstream stages
//     OR
//  3. The Freight is approved for the specified stage
//
// Note: The rationale for returning the found Freight (if any) instead of nil
// when no upstream stages are specified is that some Stages have no upstream
// Stages (e.g. a Stage that subscribes to a Warehouse), so any Freight that is
// found under those conditions is implicitly available.
func GetAvailableFreight(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
	upstreamStages []string,
	stage string,
) (*Freight, error) {
	freight, err := GetFreight(ctx, c, namespacedName)
	if err != nil {
		return nil, err
	}
	if freight == nil {
		return nil, nil
	}
	if len(upstreamStages) == 0 {
		return freight, nil
	}
	for _, stage := range upstreamStages {
		if _, ok := freight.Status.VerifiedIn[stage]; ok {
			return freight, nil
		}
	}
	if stage != "" {
		if _, ok := freight.Status.ApprovedFor[stage]; ok {
			return freight, nil
		}
	}
	return nil, nil
}
