package stages

import (
	"context"
	"errors"
	"fmt"
	"slices"

	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
)

// upgradeStage upgrades a Stage to be v0.8-compatible.
func (r *reconciler) upgradeStage(ctx context.Context, stage *kargoapi.Stage) (ctrl.Result, error) {
	// If we have already upgraded this Stage, requestedFreight will be set.
	var patchedSpec bool
	if len(stage.Spec.RequestedFreight) == 0 {
		// Check if the stages.kargo.akuity.io CRD has the requestedFreight field.
		// If it does not, we need to wait for the CRD to be updated.
		var stageCRD extv1.CustomResourceDefinition
		if err := r.kargoClient.Get(
			ctx,
			types.NamespacedName{
				Name: "stages.kargo.akuity.io",
			},
			&stageCRD,
		); err != nil {
			return ctrl.Result{}, err
		}

		if _, hasRequestedFreightField := stageCRD.Spec.Versions[0].Schema.OpenAPIV3Schema.
			Properties["spec"].
			Properties["requestedFreight"]; !hasRequestedFreightField {
			return ctrl.Result{},
				errors.New("stages.kargo.akuity.io does not have a requestedFreight field: waiting for update")
		}

		// Start the migration process.
		var requestedFreight kargoapi.FreightRequest
		switch stage.Spec.Subscriptions.Warehouse { // nolint: staticcheck
		case "":
			warehouses, err := r.resolveUpstreamSubscriptionsToWarehouse(
				ctx,
				stage.Namespace,
				stage.Spec.Subscriptions.UpstreamStages, // nolint: staticcheck
				nil,
			)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("unable to migrate: %w", err)
			}

			if len(warehouses) > 1 {
				return ctrl.Result{}, fmt.Errorf(
					"unable to migrate: upstream Stages resolve to more than one Warehouse: %v", warehouses,
				)
			}
			warehouse := warehouses[0]

			requestedFreight = kargoapi.FreightRequest{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: warehouse,
				},
			}
			for _, upstreamStage := range stage.Spec.Subscriptions.UpstreamStages { // nolint: staticcheck
				requestedFreight.Sources.Stages = append(requestedFreight.Sources.Stages, upstreamStage.Name)
			}
		default:
			requestedFreight = kargoapi.FreightRequest{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: stage.Spec.Subscriptions.Warehouse, // nolint: staticcheck
				},
				Sources: kargoapi.FreightSources{
					Direct: true,
				},
			}
		}

		// Update the Stage with the newly created requestedFreight, and clear
		// the deprecated subscriptions.
		stage.Spec.RequestedFreight = []kargoapi.FreightRequest{requestedFreight}
		stage.Spec.Subscriptions = kargoapi.Subscriptions{} // nolint: staticcheck

		// Update the Stage.
		if err := r.kargoClient.Update(ctx, stage); err != nil {
			return ctrl.Result{}, err
		}
		patchedSpec = true
	}

	// If the Stage has a history, we need to migrate it to the new format.
	var patchedStatus bool
	if len(stage.Status.History) > 0 { // nolint: staticcheck
		// Migrate the history to the new format.
		freightHistory := make(kargoapi.FreightHistory, 0, len(stage.Status.History)) // nolint: staticcheck
		for _, item := range stage.Status.History {                                   // nolint: staticcheck
			freightCollection := kargoapi.FreightCollection{
				VerificationHistory: item.VerificationHistory, // nolint: staticcheck
			}
			freightCollection.UpdateOrPush(item)
			freightHistory = append(freightHistory, &freightCollection)
		}

		// Update the Stage status.
		if err := kubeclient.PatchStatus(ctx, r.kargoClient, stage, func(status *kargoapi.StageStatus) {
			status.FreightHistory = freightHistory
			status.History = nil        // nolint: staticcheck
			status.CurrentFreight = nil // nolint: staticcheck
		}); err != nil {
			return ctrl.Result{}, err
		}
		patchedStatus = true
	}

	// If we have patched the spec or status, we need to requeue the Stage.
	if patchedSpec || patchedStatus {
		return ctrl.Result{Requeue: true}, nil
	}

	// No changes were made.
	return ctrl.Result{}, nil
}

// resolveUpstreamSubscriptionsToWarehouse resolves the Warehouse of a Stage by
// looking at its upstream Stages. This function will recursively resolve the
// Warehouse of each upstream Stage until it finds a Warehouse or reaches the
// end of the chain.
func (r *reconciler) resolveUpstreamSubscriptionsToWarehouse(
	ctx context.Context,
	namespace string,
	subscriptions []kargoapi.StageSubscription, // nolint: staticcheck
	visited map[string]struct{},
) ([]string, error) {
	if visited == nil {
		visited = make(map[string]struct{})
	}

	var warehouses []string
	for _, subscription := range subscriptions {
		// Prevent infinite loops by checking if we have already visited this
		// subscription.
		if _, found := visited[subscription.Name]; found {
			continue
		}
		visited[subscription.Name] = struct{}{}

		var upstreamStage kargoapi.Stage
		if err := r.kargoClient.Get(
			ctx,
			types.NamespacedName{
				Name:      subscription.Name,
				Namespace: namespace,
			},
			&upstreamStage,
		); err != nil {
			return nil, err
		}

		// Take into account that this resource may have already been upgraded.
		if freightReqNum := len(upstreamStage.Spec.RequestedFreight); freightReqNum > 0 {
			if freightReqNum > 1 {
				// If there is more than one requestedFreight, we cannot
				// resolve the Warehouse because we don't know which one
				// to choose.
				return nil, fmt.Errorf("upstream Stage %s has more than one requestedFreight", upstreamStage.Name)
			}

			// If the upstream Stage has a requestedFreight, we can
			// resolve the Warehouse by looking at the origin of the
			// requestedFreight.
			warehouses = append(warehouses, upstreamStage.Spec.RequestedFreight[0].Origin.Name)
			continue
		}

		// We found a direct match!
		if upstreamStage.Spec.Subscriptions.Warehouse != "" { // nolint: staticcheck
			warehouses = append(warehouses, upstreamStage.Spec.Subscriptions.Warehouse) // nolint: staticcheck
			continue
		}

		// This Stage by itself does not have a Warehouse, so we need to
		// continue checking upstream Stages.
		upstreamWarehouses, err := r.resolveUpstreamSubscriptionsToWarehouse(
			ctx,
			namespace,
			upstreamStage.Spec.Subscriptions.UpstreamStages, // nolint: staticcheck
			visited,
		)
		if err != nil {
			return nil, err
		}
		warehouses = append(warehouses, upstreamWarehouses...)
	}

	// Sort and compact the list of Warehouses.
	slices.Sort(warehouses)
	return slices.Compact(warehouses), nil
}
