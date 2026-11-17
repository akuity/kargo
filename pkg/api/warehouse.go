package api

import (
	"context"
	"fmt"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/client/apiserver/freightclient"
)

// GetWarehouse returns a pointer to the Warehouse resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func GetWarehouse(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*kargoapi.Warehouse, error) {
	warehouse := kargoapi.Warehouse{}
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
) (*kargoapi.Warehouse, error) {
	warehouse := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespacedName.Namespace,
			Name:      namespacedName.Name,
		},
	}
	if err := RefreshObject(ctx, c, warehouse); err != nil {
		return nil, fmt.Errorf("refresh: %w", err)
	}
	return warehouse, nil
}

// ListFreightFromWarehouse returns a list of all Freight resources that
// originated from the named Warehouse in the given project (namespace). It takes
// the Warehouse's name and project rather than a *Warehouse because those are the
// only values it needs, which lets callers that only know the name (e.g. a
// Freight origin) avoid fetching the Warehouse resource.
func ListFreightFromWarehouse(
	ctx context.Context,
	c client.Client,
	project string,
	warehouseName string,
	opts *freightclient.ListWarehouseFreightOptions,
) ([]kargoapi.Freight, error) {
	if opts == nil {
		opts = &freightclient.ListWarehouseFreightOptions{}
	}

	// Common selectors we'll use a few times in the code that follows:

	// Selects Freight from a specific Warehouse
	warehouseSelector := fields.OneTermEqualSelector("warehouse", warehouseName)

	// Selects Freight currently used by a specific Stage
	var currentlyInSelector fields.Selector
	if opts.CurrentlyIn != "" {
		currentlyInSelector = fields.OneTermEqualSelector("currentlyIn", opts.CurrentlyIn)
	}

	// Build a list of fields.Selectors we'll use to make multiple queries whose
	// results we will merge and de-dupe.
	fieldSelectors := make([]fields.Selector, 0)
	if opts.ApprovedFor == "" && len(opts.VerifiedIn) == 0 {
		// Just list all Freight resources that originated from the Warehouse
		sels := make([]fields.Selector, 1, 2)
		sels[0] = warehouseSelector
		if currentlyInSelector != nil {
			sels = append(sels, currentlyInSelector)
		}
		fieldSelectors = append(fieldSelectors, fields.AndSelectors(sels...))
	}
	if opts.ApprovedFor != "" {
		// List all Freight resources that are approved for the specified Stage
		sels := make([]fields.Selector, 2, 3)
		sels[0] = warehouseSelector
		sels[1] = fields.OneTermEqualSelector("approvedFor", opts.ApprovedFor)
		if currentlyInSelector != nil {
			sels = append(sels, currentlyInSelector)
		}
		fieldSelectors = append(fieldSelectors, fields.AndSelectors(sels...))
	}

	// Construct selectors for listing Freight using the configured AvailabilityStrategy
	// semantics.
	switch opts.AvailabilityStrategy {
	case kargoapi.FreightAvailabilityStrategyAll:
		// Query for Freight that is verified in ALL the VerifiedIn stages.
		sels := make([]fields.Selector, 1, len(opts.VerifiedIn)+2)
		sels[0] = warehouseSelector
		for _, stage := range opts.VerifiedIn {
			sels = append(
				sels,
				fields.OneTermEqualSelector("verifiedIn", stage),
			)
		}
		if currentlyInSelector != nil {
			sels = append(sels, currentlyInSelector)
		}
		fieldSelectors = append(fieldSelectors, fields.AndSelectors(sels...))
	case kargoapi.FreightAvailabilityStrategyOneOf, "":
		// Query for Freight that is verified in ANY of the VerifiedIn stages.
		for _, stage := range opts.VerifiedIn {
			sels := make([]fields.Selector, 2, 3)
			sels[0] = warehouseSelector
			sels[1] = fields.OneTermEqualSelector("verifiedIn", stage)
			if currentlyInSelector != nil {
				sels = append(sels, currentlyInSelector)
			}
			fieldSelectors = append(fieldSelectors, fields.AndSelectors(sels...))
		}
	default:
		return nil, fmt.Errorf(
			"unsupported AvailabilityStrategy: %s",
			opts.AvailabilityStrategy,
		)
	}

	var freight []kargoapi.Freight
	for _, fs := range fieldSelectors {
		res := &kargoapi.FreightList{}
		if err := c.List(
			ctx,
			res,
			&client.ListOptions{
				Namespace:     project,
				FieldSelector: fs,
			},
		); err != nil {
			return nil, fmt.Errorf(
				"error listing Freight for Warehouse %q in namespace %q: %w",
				warehouseName,
				project,
				err,
			)
		}
		freight = append(freight, res.Items...)
	}

	// Sort and de-dupe
	slices.SortFunc(freight, func(lhs, rhs kargoapi.Freight) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})
	freight = slices.CompactFunc(freight, func(lhs, rhs kargoapi.Freight) bool {
		return lhs.Name == rhs.Name
	})

	if len(opts.VerifiedIn) == 0 || opts.RequiredSoakTime == nil {
		// Nothing left to do
		return freight, nil
	}

	// Filter out Freight whose soak time has not yet elapsed
	filtered := make([]kargoapi.Freight, 0, len(freight))
	for _, f := range freight {
		if opts.ApprovedFor != "" {
			if f.IsApprovedFor(opts.ApprovedFor) {
				filtered = append(filtered, f)
				continue
			}
		}

		// Track set of VerifiedIn Stages in which the Freight has passed the
		// verification soak time.
		verifiedStages := sets.New[string]()
		for _, stage := range opts.VerifiedIn {
			if f.HasSoakedIn(stage, opts.RequiredSoakTime) {
				verifiedStages.Insert(stage)
			}
		}

		// Filter out Freight that has passed its verification soak time in ALL
		// the specified VerifiedIn Stages if AvailabilityStrategy is set to All.
		// Otherwise, include Freight if it has passed the soak time in a single
		// Stage.
		if opts.AvailabilityStrategy == kargoapi.FreightAvailabilityStrategyAll {
			// If Freight is verified in ALL upstream Stages, then it is
			// available.
			if verifiedStages.Equal(sets.New(opts.VerifiedIn...)) {
				filtered = append(filtered, f)
			}
			continue
		}

		// If Freight is verified in ANY upstream Stage, then it is available.
		if verifiedStages.Len() > 0 {
			filtered = append(filtered, f)
		}
	}
	return filtered, nil
}
