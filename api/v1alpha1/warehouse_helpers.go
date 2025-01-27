package v1alpha1

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetWarehouse returns a pointer to the Warehouse resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func GetWarehouse(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*Warehouse, error) {
	warehouse := Warehouse{}
	if err := c.Get(ctx, namespacedName, &warehouse); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"error getting Warehouse %q in namespace %q: %w",
			namespacedName.Name,
			namespacedName.Namespace,
			err,
		)
	}
	return &warehouse, nil
}

// RefreshWarehouse forces reconciliation of a Warehouse by setting an annotation
// on the Warehouse, causing the controller to reconcile it. Currently, the
// annotation value is the timestamp of the request, but might in the
// future include additional metadata/context necessary for the request.
func RefreshWarehouse(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*Warehouse, error) {
	warehouse := &Warehouse{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespacedName.Namespace,
			Name:      namespacedName.Name,
		},
	}
	if err := patchAnnotation(
		ctx,
		c,
		warehouse,
		AnnotationKeyRefresh,
		time.Now().Format(time.RFC3339),
	); err != nil {
		return nil, fmt.Errorf("refresh: %w", err)
	}
	return warehouse, nil
}

// ListWarehouseFreightOptions is a struct that can be used to specify filtering
// criteria when listing Freight resources that originated from a Warehouse.
//
// +protobuf=false
// +k8s:deepcopy-gen=false
// +k8s:openapi-gen=false
type ListWarehouseFreightOptions struct {
	// ApprovedFor names a Stage for which all Freight resources that have been
	// approved for that Stage should be included in the list results.
	//
	// IMPORTANT: This is OR'ed with the VerifiedIn field.
	ApprovedFor string
	// VerifiedIn names zero or more Stages for which all Freight resources that
	// have been verified for those Stages should be included in the list results
	// AS long as they were verified before the VerifiedBefore time (if set).
	//
	// IMPORTANT: This is OR'ed with the ApprovedFor field.
	VerifiedIn []string
	// VerifiedBefore optionally specifies a time before which a Freight verified
	// in any of the Stages named in the VerifiedIn field must have been verified.
	// This is useful for filtering out Freight whose soak time has not yet
	// elapsed.
	VerifiedBefore *metav1.Time
}

// ListFreight returns a list of all Freight resources that originated from the
// Warehouse.
func (w *Warehouse) ListFreight(
	ctx context.Context,
	c client.Client,
	opts *ListWarehouseFreightOptions,
) ([]Freight, error) {
	if opts == nil {
		opts = &ListWarehouseFreightOptions{}
	}

	// Build a list of list options to make multiple queries whose results we will
	// merge and de-dupe.
	fieldSelectors := make([]fields.Selector, 0, 1+len(opts.VerifiedIn))
	warehouseSelector := fields.OneTermEqualSelector("warehouse", w.Name)
	if opts.ApprovedFor == "" && len(opts.VerifiedIn) == 0 {
		// Just list all Freight resources that originated from the Warehouse
		fieldSelectors = append(fieldSelectors, warehouseSelector)
	}
	if opts.ApprovedFor != "" {
		// List all Freight resources that are approved for the specified Stage
		fieldSelectors = append(
			fieldSelectors,
			fields.AndSelectors(
				warehouseSelector,
				fields.OneTermEqualSelector("approvedFor", opts.ApprovedFor),
			),
		)
	}
	for _, stage := range opts.VerifiedIn {
		// List all Freight resources that are verified in the specified Stage
		fieldSelectors = append(
			fieldSelectors,
			fields.AndSelectors(
				warehouseSelector,
				fields.OneTermEqualSelector("verifiedIn", stage),
			),
		)
	}

	freight := []Freight{}
	for _, fs := range fieldSelectors {
		res := &FreightList{}
		if err := c.List(
			ctx,
			res,
			&client.ListOptions{
				Namespace:     w.Namespace,
				FieldSelector: fs,
			},
		); err != nil {
			return nil, fmt.Errorf(
				"error listing Freight for Warehouse %q in namespace %q: %w",
				w.Name,
				w.Namespace,
				err,
			)
		}
		freight = append(freight, res.Items...)
	}

	// Sort and de-dupe
	slices.SortFunc(freight, func(lhs, rhs Freight) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})
	freight = slices.CompactFunc(freight, func(lhs, rhs Freight) bool {
		return lhs.Name == rhs.Name
	})

	if len(opts.VerifiedIn) == 0 || opts.VerifiedBefore == nil {
		// Nothing left to do
		return freight, nil
	}

	// Filter out Freight whose soak time has not yet elapsed
	filtered := make([]Freight, 0, len(freight))
freightLoop:
	for _, f := range freight {
		if opts.ApprovedFor != "" {
			if f.IsApprovedFor(opts.ApprovedFor) {
				filtered = append(filtered, f)
				continue
			}
		}
		for _, ver := range f.Status.VerifiedIn {
			if verifiedAt := ver.VerifiedAt; verifiedAt != nil {
				if verifiedAt.Time.Before(opts.VerifiedBefore.Time) {
					filtered = append(filtered, f)
					continue freightLoop
				}
			}
		}
	}
	return filtered, nil
}
