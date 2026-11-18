package freightapi

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
)

// TODO: this will be removed when moving freights to DB
type K8sFreightWarehouseLister struct{ Client client.Client }

func (lister *K8sFreightWarehouseLister) ListFreightFromWarehouse(
	ctx context.Context,
	project, warehouse string,
	opts *ListWarehouseFreightOptions,
) ([]kargoapi.Freight, error) {
	return ListFreightFromWarehouse(ctx, lister.Client, project, warehouse, opts)
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
	// RequiredSoakTime optionally specifies a minimum duration that a piece of
	// Freight must have continuously remained in a Stage at any time after being
	// verified.
	RequiredSoakTime *metav1.Duration
	// AvailabilityStrategy specifies the semantics for how Freight is determined
	// to be available. If not set, the default is to consider Freight available
	// if it has been verified in any of the provided VerifiedIn stages.
	// IMPORTANT: This is also applied to Freight matched using the VerifiedBefore
	// condition.
	AvailabilityStrategy kargoapi.FreightAvailabilityStrategy

	// CurrentlyIn optionally specifies a Stage in which the Freight must
	// currently be used in order to be included in the list results.
	CurrentlyIn string
}

// GetFreightByNameOrAlias returns a pointer to the Freight resource specified
// by the project, and name OR alias arguments. If no such resource is found,
// nil is returned instead.
func GetFreightByNameOrAlias(
	ctx context.Context,
	c client.Client,
	project string,
	name string,
	alias string,
) (*kargoapi.Freight, error) {
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
) (*kargoapi.Freight, error) {
	freight := kargoapi.Freight{}
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
) (*kargoapi.Freight, error) {
	freightList := kargoapi.FreightList{}
	if err := c.List(
		ctx,
		&freightList,
		client.InNamespace(project),
		client.MatchingLabels{
			kargoapi.LabelKeyAlias: alias,
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

// ListFreightByCurrentStage returns a list of Freight resources that think
// they're currently in use by the Stage specified.
func ListFreightByCurrentStage(
	ctx context.Context,
	c client.Client,
	project, stageName string,
) ([]kargoapi.Freight, error) {
	freightList := kargoapi.FreightList{}
	if err := c.List(
		ctx,
		&freightList,
		client.InNamespace(project),
		client.MatchingFields{"currentlyIn": stageName},
	); err != nil {
		return nil, fmt.Errorf(
			"error listing Freight in namespace %q with current stage %q: %w",
			project,
			stageName,
			err,
		)
	}
	return freightList.Items, nil
}

// ListFreightReferencingStage returns the Freight whose status records the named
// Stage in its verifiedIn or approvedFor map, de-duplicated by name and ordered
// by name. It is used to find the Freight whose status must be cleaned up when a
// Stage is deleted.
func ListFreightReferencingStage(
	ctx context.Context,
	c client.Client,
	namespace string,
	stage string,
) ([]kargoapi.Freight, error) {
	// Field names match the indexer's verifiedIn and approvedFor indexes;
	// Index fields are referenced here to avoid circular dependency.
	// Indexers will be removed soon anyway, so it's not critical.
	byName := make(map[string]kargoapi.Freight)
	for _, field := range []string{"verifiedIn", "approvedFor", "currentlyIn"} {
		list := kargoapi.FreightList{}
		if err := c.List(
			ctx,
			&list,
			client.InNamespace(namespace),
			client.MatchingFieldsSelector{
				Selector: fields.OneTermEqualSelector(field, stage),
			},
		); err != nil {
			return nil, fmt.Errorf(
				"error listing Freight referencing Stage %q in namespace %q: %w",
				stage, namespace, err,
			)
		}
		for _, f := range list.Items {
			byName[f.Name] = f
		}
	}
	freight := make([]kargoapi.Freight, 0, len(byName))
	for _, f := range byName {
		freight = append(freight, f)
	}
	slices.SortFunc(freight, func(lhs, rhs kargoapi.Freight) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})
	return freight, nil
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
	opts *ListWarehouseFreightOptions,
) ([]kargoapi.Freight, error) {
	if opts == nil {
		opts = &ListWarehouseFreightOptions{}
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
