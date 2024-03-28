package stages

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

// upgradeStage upgrades a Stage to be v0.5-compatible.
func (r *reconciler) upgradeStage(
	ctx context.Context,
	stage *kargoapi.Stage,
) (ctrl.Result, error) {
	// If there is a shard label, patch the spec to fill in the new shard field.
	if shard, ok := stage.Labels[kargoapi.ShardLabelKey]; stage.Spec.Shard == "" && ok {
		if err := r.kargoClient.Patch(
			ctx,
			stage,
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

	// In v0.6.0, the ID field of the FreightReference type will be removed and
	// replaced with a Name field. Both fields exist in v0.5, so we copy the value
	// from the ID field into the Name field.
	if stage.Status.CurrentFreight != nil {
		stage.Status.CurrentFreight.Name = stage.Status.CurrentFreight.ID
	}
	for i := range stage.Status.History {
		stage.Status.History[i].Name = stage.Status.History[i].ID
	}
	if err := r.kargoClient.Status().Update(ctx, stage); err != nil {
		return ctrl.Result{}, nil
	}

	if err :=
		kargoapi.AddV05CompatibilityLabel(ctx, r.kargoClient, stage); err != nil {
		return ctrl.Result{}, err
	}

	logging.LoggerFromContext(ctx).Debug("updated Stage for v0.5 compatibility")

	return ctrl.Result{
		Requeue: true,
	}, nil
}
