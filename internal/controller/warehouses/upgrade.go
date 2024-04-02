package warehouses

import (
	"context"
	"errors"
	"fmt"

	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
	var warehouseCRD extv1.CustomResourceDefinition
	if err := r.client.Get(
		ctx,
		types.NamespacedName{
			Name: "warehouses.kargo.akuity.io",
		},
		&warehouseCRD,
	); err != nil {
		return ctrl.Result{}, err
	}
	if _, hasShardField := warehouseCRD.Spec.Versions[0].Schema.OpenAPIV3Schema.
		Properties["spec"].
		Properties["shard"]; !hasShardField {
		return ctrl.Result{},
			errors.New("warehouse CRD has no spec.shard field; waiting for update")
	}

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
