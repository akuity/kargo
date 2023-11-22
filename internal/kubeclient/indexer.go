package kubeclient

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
)

const (
	FreightByVerifiedStagesIndexField     = "verifiedIn"
	FreightApprovedForStagesIndexField    = "approvedFor"
	FreightByWarehouseIndexField          = "warehouse"
	PromotionsByStageAndFreightIndexField = "stageAndFreight"

	// Note: These two do not conflict with one another, because these two
	// indices are used by different components.
	PromotionsByStageIndexField            = "stage"
	NonTerminalPromotionsByStageIndexField = "stage"

	PromotionPoliciesByStageIndexField   = "stage"
	StagesByArgoCDApplicationsIndexField = "applications"
	StagesByUpstreamStagesIndexField     = "upstreamStages"
	StagesByWarehouseIndexField          = "warehouse"

	ServiceAccountsByFallbackIndexField = "fallback"
	ServiceAccountsByGroupIndexField    = "group"
	ServiceAccountsBySubjectIndexField  = "subject"
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
				fmt.Sprintf("%s:%s", appCheck.AppNamespaceOrDefault(), appCheck.AppName)
		}
		return apps
	}
}

// IndexPromotionsByStage creates Promotion index by Stage for which
// all the given predicates returns true for the Promotion.
func IndexPromotionsByStage(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Promotion{},
		PromotionsByStageIndexField,
		indexPromotionsByStage(),
	)
}

// IndexNonTerminalPromotionsByStage indexes Promotions in non-terminal states
// by Stage
func IndexNonTerminalPromotionsByStage(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Promotion{},
		NonTerminalPromotionsByStageIndexField,
		indexPromotionsByStage(isPromotionPhaseNonTerminal),
	)
}

func isPromotionPhaseNonTerminal(promo *kargoapi.Promotion) bool {
	return !promo.Status.Phase.IsTerminal()
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

// IndexPromotionsByStageAndFreight indexes Promotions by the Freight + Stage
// they reference.
func IndexPromotionsByStageAndFreight(
	ctx context.Context,
	mgr ctrl.Manager,
) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Promotion{},
		PromotionsByStageAndFreightIndexField,
		indexPromotionsByStageAndFreight,
	)
}

func indexPromotionsByStageAndFreight(obj client.Object) []string {
	promo := obj.(*kargoapi.Promotion) // nolint: forcetypeassert
	return []string{
		StageAndFreightKey(promo.Spec.Stage, promo.Spec.Freight),
	}
}

func StageAndFreightKey(stage, freight string) string {
	return fmt.Sprintf("%s:%s", stage, freight)
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

func IndexFreightByWarehouse(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Freight{},
		FreightByWarehouseIndexField,
		indexFreightByWarehouse,
	)
}

func indexFreightByWarehouse(obj client.Object) []string {
	freight := obj.(*kargoapi.Freight) // nolint: forcetypeassert
	for _, ownerRef := range freight.OwnerReferences {
		if ownerRef.APIVersion == kargoapi.GroupVersion.String() &&
			ownerRef.Kind == "Warehouse" {
			return []string{ownerRef.Name}
		}
	}
	return nil
}

// IndexFreightByVerifiedStages indexes Freight by the Stages in which it has
// been verified.
func IndexFreightByVerifiedStages(
	ctx context.Context,
	mgr ctrl.Manager,
) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Freight{},
		FreightByVerifiedStagesIndexField,
		indexFreightByVerifiedStages,
	)
}

func indexFreightByVerifiedStages(obj client.Object) []string {
	freight := obj.(*kargoapi.Freight) // nolint: forcetypeassert
	verifiedStages := make([]string, len(freight.Status.VerifiedIn))
	var i int
	for stage := range freight.Status.VerifiedIn {
		verifiedStages[i] = stage
		i++
	}
	return verifiedStages
}

// IndexFreightByApprovedStages indexes Freight by the Stages for which it has
// been (manually) approved.
func IndexFreightByApprovedStages(
	ctx context.Context,
	mgr ctrl.Manager,
) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Freight{},
		FreightApprovedForStagesIndexField,
		indexFreightByApprovedStages,
	)
}

func indexFreightByApprovedStages(obj client.Object) []string {
	freight := obj.(*kargoapi.Freight) // nolint: forcetypeassert
	approvedStages := make([]string, len(freight.Status.ApprovedFor))
	var i int
	for stages := range freight.Status.ApprovedFor {
		approvedStages[i] = stages
		i++
	}
	return approvedStages
}

func IndexStagesByUpstreamStages(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Stage{},
		StagesByUpstreamStagesIndexField,
		indexStagesByUpstreamStages,
	)
}

func indexStagesByUpstreamStages(obj client.Object) []string {
	stage := obj.(*kargoapi.Stage) // nolint: forcetypeassert
	if stage.Spec.Subscriptions.UpstreamStages == nil {
		return nil
	}
	upstreamStages := make([]string, len(stage.Spec.Subscriptions.UpstreamStages))
	for i, upstreamStage := range stage.Spec.Subscriptions.UpstreamStages {
		upstreamStages[i] = upstreamStage.Name
	}
	return upstreamStages
}

func IndexStagesByWarehouse(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Stage{},
		StagesByWarehouseIndexField,
		indexStagesByWarehouse,
	)
}

func indexStagesByWarehouse(obj client.Object) []string {
	stage := obj.(*kargoapi.Stage) // nolint: forcetypeassert
	if stage.Spec.Subscriptions.Warehouse != "" {
		return []string{stage.Spec.Subscriptions.Warehouse}
	}
	return nil
}

func IndexServiceAccountsByFallbackAnnotation(
	ctx context.Context,
	mgr ctrl.Manager,
	kargoNamespace string,
) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&corev1.ServiceAccount{},
		ServiceAccountsByFallbackIndexField,
		indexServiceAccountsByFallbackAnnotation(kargoNamespace),
	)
}

func indexServiceAccountsByFallbackAnnotation(kargoNamespace string) client.IndexerFunc {
	return func(obj client.Object) []string {
		sa := obj.(*corev1.ServiceAccount) // nolint: forcetypeassert
		if sa.Namespace != kargoNamespace {
			return nil
		}
		if sa.GetAnnotations()[kargoapi.AnnotationKeyRBACFallbackServiceAccount] ==
			kargoapi.AnnotationValueTrue {
			return []string{kargoapi.AnnotationValueTrue}
		}
		return nil
	}
}

func IndexServiceAccountsByRBACGroups(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&corev1.ServiceAccount{},
		ServiceAccountsByGroupIndexField,
		indexServiceAccountsByRBACGroups,
	)
}

func indexServiceAccountsByRBACGroups(obj client.Object) []string {
	sa := obj.(*corev1.ServiceAccount) // nolint: forcetypeassert
	rawGroups := strings.TrimSpace(sa.GetAnnotations()[kargoapi.AnnotationKeyRBACGroups])
	if rawGroups == "" {
		return nil
	}
	groups := strings.Split(rawGroups, ",")
	refinedGroups := make([]string, 0, len(groups))
	for _, g := range groups {
		if group := strings.TrimSpace(g); group != "" {
			refinedGroups = append(refinedGroups, group)
		}
	}
	return refinedGroups
}

func IndexServiceAccountsByRBACSubjects(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&corev1.ServiceAccount{},
		ServiceAccountsBySubjectIndexField,
		indexServiceAccountsByRBACSubjects,
	)
}

func indexServiceAccountsByRBACSubjects(obj client.Object) []string {
	sa := obj.(*corev1.ServiceAccount) // nolint: forcetypeassert
	rawGroups := strings.TrimSpace(sa.GetAnnotations()[kargoapi.AnnotationKeyRBACSubjects])
	if rawGroups == "" {
		return nil
	}
	subjects := strings.Split(rawGroups, ",")
	refinedSubjects := make([]string, 0, len(subjects))
	for _, s := range subjects {
		if subject := strings.TrimSpace(s); subject != "" {
			refinedSubjects = append(refinedSubjects, subject)
		}
	}
	return refinedSubjects
}
