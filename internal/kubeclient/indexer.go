package kubeclient

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
)

const (
	StagesByArgoCDApplicationsIndexField   = "applications"
	OutstandingPromotionsByStageIndexField = "stage"
	PromotionPoliciesByStageIndexField     = "stage"
)

func IndexStagesByArgoCDApplications(ctx context.Context, mgr ctrl.Manager, shardName string) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&api.Stage{},
		StagesByArgoCDApplicationsIndexField,
		indexStagesByArgoCDApplications(shardName))
}

func indexStagesByArgoCDApplications(shardName string) client.IndexerFunc {
	return func(obj client.Object) []string {
		// Return early if:
		//
		// 1. This is the default controller, but the object is labeled for a
		//    specific shard.
		//
		// 2. This is a shard-specific controller, but the object is not labeled for
		//    this shard.
		objShardName, labeled := obj.GetLabels()[controller.ShardLabelKey]
		if (shardName == "" && labeled) ||
			(shardName != "" && shardName != objShardName) {
			return nil
		}

		stage := obj.(*api.Stage) // nolint: forcetypeassert
		if stage.Spec.PromotionMechanisms == nil ||
			len(stage.Spec.PromotionMechanisms.ArgoCDAppUpdates) == 0 {
			return nil
		}
		apps := make([]string, len(stage.Spec.PromotionMechanisms.ArgoCDAppUpdates))
		for i, appCheck := range stage.Spec.PromotionMechanisms.ArgoCDAppUpdates {
			apps[i] =
				fmt.Sprintf("%s:%s", appCheck.AppNamespace, appCheck.AppName)
		}
		return apps
	}
}

// IndexOutstandingPromotionsByStage creates index for Promotions in non-terminal states by Stage
func IndexOutstandingPromotionsByStage(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&api.Promotion{},
		OutstandingPromotionsByStageIndexField,
		indexOutstandingPromotionsByStage)
}

func indexOutstandingPromotionsByStage(obj client.Object) []string {
	promo := obj.(*api.Promotion) // nolint: forcetypeassert
	switch promo.Status.Phase {
	case api.PromotionPhaseComplete, api.PromotionPhaseFailed:
		return nil
	}
	return []string{promo.Spec.Stage}
}

func IndexPromotionPoliciesByStage(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&api.PromotionPolicy{},
		PromotionPoliciesByStageIndexField,
		indexPromotionPoliciesByStage)
}

func indexPromotionPoliciesByStage(obj client.Object) []string {
	policy := obj.(*api.PromotionPolicy) // nolint: forcetypeassert
	return []string{policy.Stage}
}
