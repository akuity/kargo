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

// GetQualifiedFreight returns a pointer to the Freight resource specified by
// the namespacedName argument if it is found and EITHER no Stages were
// specified in the function call OR the Freight has qualified for ANY of the
// specified Stages. If all other cases, nil is returned instead.
//
// Note: The rationale for returning the found Freight (if any) instead of nil
// when no Stages are specified is that the Stages provided are typically the
// names of Stages UPSTREAM from some other Stage. i.e. The typical use for this
// function is to answer whether a piece of Freight has qualified for any of a
// given Stage's UPSTREAM Stages. Some Stages have no upstream Stages, so any
// Freight that is found is implicitly qualified.
func GetQualifiedFreight(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
	stages []string,
) (*Freight, error) {
	freight, err := GetFreight(ctx, c, namespacedName)
	if err != nil {
		return nil, err
	}
	if freight == nil {
		return nil, nil
	}
	if len(stages) == 0 {
		return freight, nil
	}
	for qualifiedStage := range freight.Status.Qualifications {
		for _, stage := range stages {
			if qualifiedStage == stage {
				return freight, nil
			}
		}
	}
	return nil, nil
}
