package stages

import (
	"context"
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
)

// soakRequeueBuffer adds a small delay to avoid requeuing immediately after
// the soak time elapses to avoid getting no available Freight.
const soakRequeueBuffer = time.Second

// calculateNextSoakCheck computes how long the reconciler should wait before
// re-evaluating the given Stage for auto-promotion when one or more pieces of
// candidate Freight are still soaking in upstream Stages.
//
// It returns 0 when no soak deadline is pending.
func calculateNextSoakCheck(
	ctx context.Context,
	c client.Client,
	stage *kargoapi.Stage,
) (time.Duration, error) {
	var soonest time.Duration

	for _, req := range stage.Spec.RequestedFreight {
		if req.Sources.RequiredSoakTime == nil ||
			req.Sources.RequiredSoakTime.Duration <= 0 ||
			len(req.Sources.Stages) == 0 {
			continue
		}

		warehouse, err := api.GetWarehouse(ctx, c, client.ObjectKey{
			Namespace: stage.Namespace,
			Name:      req.Origin.Name,
		})
		if err != nil {
			return 0, fmt.Errorf("get Warehouse %q: %w", req.Origin.Name, err)
		}
		if warehouse == nil {
			continue
		}

		// List candidates with the soak filter disabled so we can see Freight
		// that is verified in upstream Stages but still soaking.
		candidates, err := api.ListFreightFromWarehouse(
			ctx,
			c,
			warehouse,
			&api.ListWarehouseFreightOptions{
				ApprovedFor:          stage.Name,
				VerifiedIn:           req.Sources.Stages,
				AvailabilityStrategy: req.Sources.AvailabilityStrategy,
			},
		)
		if err != nil {
			return 0, fmt.Errorf("list freight from Warehouse %q: %w", warehouse.Name, err)
		}

		for i := range candidates {
			remaining := remainingSoakForFreight(
				&candidates[i],
				req.Sources.Stages,
				req.Sources.RequiredSoakTime.Duration,
				req.Sources.AvailabilityStrategy,
			)
			if remaining <= 0 {
				continue
			}
			if soonest == 0 || remaining < soonest {
				soonest = remaining
			}
		}
	}

	if soonest == 0 {
		return 0, nil
	}
	return soonest + soakRequeueBuffer, nil
}

// remainingSoakForFreight returns how much longer the given Freight must soak
// before it satisfies the requirement of being soaked in the configured
// upstream Stages according to the given AvailabilityStrategy. A return value
// of 0 means the Freight either already satisfies the requirement or is not
// eligible to satisfy it (e.g. with the All strategy when the Freight has not
// been verified in every upstream Stage).
func remainingSoakForFreight(
	freight *kargoapi.Freight,
	upstreamStages []string,
	required time.Duration,
	strategy kargoapi.FreightAvailabilityStrategy,
) time.Duration {
	switch strategy {
	case kargoapi.FreightAvailabilityStrategyAll:
		// Freight must satisfy the soak in EVERY upstream Stage. The remaining
		// wait is the longest remaining wait across those Stages. If any
		// upstream Stage has not verified the Freight, the Freight will never
		// satisfy this requirement on its own; return 0 to indicate "no
		// deadline".
		var longest time.Duration
		for _, stage := range upstreamStages {
			if !freight.IsVerifiedIn(stage) {
				return 0
			}
			elapsed := freight.GetLongestSoak(stage)
			if elapsed >= required {
				continue
			}
			wait := required - elapsed
			if wait > longest {
				longest = wait
			}
		}
		return longest

	default:
		// OneOf (and empty/unset, which is treated as OneOf): the Freight
		// becomes eligible as soon as ANY upstream Stage's soak completes.
		// Return the smallest positive remaining wait across the upstream
		// Stages where the Freight is verified.
		var shortest time.Duration
		for _, stage := range upstreamStages {
			if !freight.IsVerifiedIn(stage) {
				continue
			}
			elapsed := freight.GetLongestSoak(stage)
			if elapsed >= required {
				// Already soaked somewhere — no deadline needed.
				return 0
			}
			wait := required - elapsed
			if shortest == 0 || wait < shortest {
				shortest = wait
			}
		}
		return shortest
	}
}
