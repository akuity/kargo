package warehouses

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

// upgradeStage upgrades a Warehouse to be v0.5-compatible.
func (r *reconciler) upgradeWarehouse(
	ctx context.Context,
	warehouse *kargoapi.Warehouse,
) (ctrl.Result, error) {
	// If there is a shard label, patch the spec to fill in the new shard field.
	if shard, ok := warehouse.Labels[kargoapi.ShardLabelKey]; warehouse.Spec.Shard == "" && ok {
		if err := r.client.Patch(
			ctx,
			warehouse,
			client.RawPatch(
				types.MergePatchType,
				[]byte(
					fmt.Sprintf(`{"spec":{"shard":"%s"}}`, shard),
				),
			),
		); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err :=
		kargoapi.AddV05CompatibilityLabel(ctx, r.client, warehouse); err != nil {
		return ctrl.Result{}, err
	}

	logging.LoggerFromContext(ctx).Debug("updated Warehouse for v0.5 compatibility")

	return ctrl.Result{
		Requeue: true,
	}, nil
}
