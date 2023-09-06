package kubeclient

import (
	"context"
	"fmt"

	"golang.org/x/exp/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
)

const (
	StagesByArgoCDApplicationsIndexField   = "applications"
	PromotionsByStageIndexField            = "stage"
	OutstandingPromotionsByStageIndexField = PromotionPoliciesByStageIndexField
	PromotionPoliciesByStageIndexField     = "stage"
)

var (
	NonOutstandingPromotionPhases = []kargoapi.PromotionPhase{
		kargoapi.PromotionPhaseComplete,
		kargoapi.PromotionPhaseFailed,
	}
)

func IndexStagesByArgoCDApplications(ctx context.Context, mgr ctrl.Manager, shardName string) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Stage{},
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

		stage := obj.(*kargoapi.Stage) // nolint: forcetypeassert
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

// IndexPromotionsByStage creates Promotion index by Stage for which
// all the given predicates returns true for the Promotion.
func IndexPromotionsByStage(ctx context.Context, mgr ctrl.Manager, predicates ...func(*kargoapi.Promotion) bool) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Promotion{},
		PromotionsByStageIndexField,
		indexPromotionsByStage(predicates...))
}

// IndexOutstandingPromotionsByStage creates index for Promotions in non-terminal states by Stage
func IndexOutstandingPromotionsByStage(ctx context.Context, mgr ctrl.Manager) error {
	return IndexPromotionsByStage(ctx, mgr, filterNonOutstandingPromotionPhases)
}

func filterNonOutstandingPromotionPhases(promo *kargoapi.Promotion) bool {
	return !slices.Contains(NonOutstandingPromotionPhases, promo.GetStatus().Phase)
}

// indexPromotionsByStage indexes Promotion if all the given predicates
// returns true for the Promotion.
func indexPromotionsByStage(predicates ...func(*kargoapi.Promotion) bool) client.IndexerFunc {
	return func(obj client.Object) []string {
		promo, ok := obj.(*kargoapi.Promotion)
		if !ok {
			return nil
		}
		for _, predicate := range predicates {
			if !predicate(promo) {
				return nil
			}
		}
		return []string{promo.Spec.Stage}
	}
}

func IndexPromotionPoliciesByStage(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.PromotionPolicy{},
		PromotionPoliciesByStageIndexField,
		indexPromotionPoliciesByStage)
}

func indexPromotionPoliciesByStage(obj client.Object) []string {
	policy := obj.(*kargoapi.PromotionPolicy) // nolint: forcetypeassert
	return []string{policy.Stage}
}
