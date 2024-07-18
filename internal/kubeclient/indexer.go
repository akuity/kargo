package kubeclient

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libargocd "github.com/akuity/kargo/internal/argocd"
	"github.com/akuity/kargo/internal/logging"
)

const (
	EventsByInvolvedObjectAPIGroupIndexField = "involvedObject.apiGroup"

	FreightByVerifiedStagesIndexField     = "verifiedIn"
	FreightApprovedForStagesIndexField    = "approvedFor"
	FreightByWarehouseIndexField          = "warehouse"
	PromotionsByStageAndFreightIndexField = "stageAndFreight"

	// Note: These two do not conflict with one another, because these two
	// indices are used by different components.
	PromotionsByStageIndexField            = "stage"
	NonTerminalPromotionsByStageIndexField = "stage"

	RunningPromotionsByArgoCDApplicationsIndexField = "applications"

	PromotionPoliciesByStageIndexField   = "stage"
	StagesByAnalysisRunIndexField        = "analysisRun"
	StagesByArgoCDApplicationsIndexField = "applications"
	StagesByFreightIndexField            = "freight"
	StagesByUpstreamStagesIndexField     = "upstreamStages"
	StagesByWarehouseIndexField          = "warehouse"

	ServiceAccountsByOIDCEmailIndexField   = "email"
	ServiceAccountsByOIDCGroupIndexField   = "groups"
	ServiceAccountsByOIDCSubjectIndexField = "subjects"
	ServiceAccountsByOIDCClaimIndexField   = "claims"
)

func IndexEventsByInvolvedObjectAPIGroup(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&corev1.Event{},
		EventsByInvolvedObjectAPIGroupIndexField,
		indexEventsByInvolvedObjectAPIGroup,
	)
}

func indexEventsByInvolvedObjectAPIGroup(obj client.Object) []string {
	event := obj.(*corev1.Event) // nolint: forcetypeassert
	// Ignore invalid APIVersion
	gv, _ := schema.ParseGroupVersion(event.InvolvedObject.APIVersion)
	if gv.Empty() {
		return nil
	}
	return []string{gv.Group}
}

func IndexStagesByAnalysisRun(ctx context.Context, mgr ctrl.Manager, shardName string) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Stage{},
		StagesByAnalysisRunIndexField,
		indexStagesByAnalysisRun(shardName))
}

func indexStagesByAnalysisRun(shardName string) client.IndexerFunc {
	return func(obj client.Object) []string {
		// Return early if:
		//
		// 1. This is the default controller, but the object is labeled for a
		//    specific shard.
		//
		// 2. This is a shard-specific controller, but the object is not labeled for
		//    this shard.
		objShardName, labeled := obj.GetLabels()[kargoapi.ShardLabelKey]
		if (shardName == "" && labeled) ||
			(shardName != "" && shardName != objShardName) {
			return nil
		}

		stage := obj.(*kargoapi.Stage) // nolint: forcetypeassert
		if stage.Status.CurrentFreight == nil ||
			stage.Status.CurrentFreight.VerificationInfo == nil ||
			stage.Status.CurrentFreight.VerificationInfo.AnalysisRun == nil {
			return nil
		}
		return []string{
			fmt.Sprintf(
				"%s:%s",
				stage.Status.CurrentFreight.VerificationInfo.AnalysisRun.Namespace,
				stage.Status.CurrentFreight.VerificationInfo.AnalysisRun.Name,
			),
		}
	}
}

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
		objShardName, labeled := obj.GetLabels()[kargoapi.ShardLabelKey]
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
			namespace := appCheck.AppNamespace
			if namespace == "" {
				namespace = libargocd.Namespace()
			}
			apps[i] = fmt.Sprintf("%s:%s", namespace, appCheck.AppName)
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

func IndexRunningPromotionsByArgoCDApplications(
	ctx context.Context,
	mgr ctrl.Manager,
	shardName string,
) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Promotion{},
		RunningPromotionsByArgoCDApplicationsIndexField,
		indexRunningPromotionsByArgoCDApplications(ctx, mgr.GetClient(), shardName),
	)
}

func indexRunningPromotionsByArgoCDApplications(
	ctx context.Context,
	c client.Client,
	shardName string,
) client.IndexerFunc {
	logger := logging.LoggerFromContext(ctx)

	return func(obj client.Object) []string {
		// Return early if:
		//
		// 1. This is the default controller, but the object is labeled for a
		//    specific shard.
		//
		// 2. This is a shard-specific controller, but the object is not labeled for
		//    this shard.
		objShardName, labeled := obj.GetLabels()[kargoapi.ShardLabelKey]
		if (shardName == "" && labeled) || (shardName != "" && shardName != objShardName) {
			return nil
		}

		promo, ok := obj.(*kargoapi.Promotion)
		if !ok {
			return nil
		}

		if promo.Status.Phase != kargoapi.PromotionPhaseRunning {
			// We are only interested in running Promotions.
			return nil
		}

		stage := kargoapi.Stage{}
		if err := c.Get(
			ctx,
			client.ObjectKey{
				Namespace: promo.Namespace,
				Name:      promo.Spec.Stage,
			},
			&stage,
		); err != nil {
			logger.Error(
				err, "failed to index running Promotion by Argo CD Applications; "+
					"can not get Stage for running Promotion",
				"promo", promo.Name,
				"namespace", promo.Namespace,
			)
			return nil
		}

		if stage.Spec.PromotionMechanisms == nil || len(stage.Spec.PromotionMechanisms.ArgoCDAppUpdates) == 0 {
			// If the Stage has no Argo CD Application promotion mechanisms,
			// then we have nothing to index.
			return nil
		}

		res := make([]string, len(stage.Spec.PromotionMechanisms.ArgoCDAppUpdates))
		for i, appUpdate := range stage.Spec.PromotionMechanisms.ArgoCDAppUpdates {
			namespace := appUpdate.AppNamespace
			if namespace == "" {
				namespace = libargocd.Namespace()
			}
			res[i] = fmt.Sprintf("%s:%s", namespace, appUpdate.AppName)
		}
		return res
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
	return []string{freight.Warehouse}
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

func IndexStagesByFreight(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Stage{},
		StagesByFreightIndexField,
		indexStagesByFreight,
	)
}

func indexStagesByFreight(obj client.Object) []string {
	stage := obj.(*kargoapi.Stage) // nolint: forcetypeassert
	if stage.Status.CurrentFreight != nil {
		if id := stage.Status.CurrentFreight.Name; id != "" {
			return []string{id}
		}
	}
	return nil
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

func IndexServiceAccountsByOIDCClaim(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&corev1.ServiceAccount{},
		ServiceAccountsByOIDCClaimIndexField,
		indexServiceAccountsOIDCClaim,
	)
}

func indexServiceAccountsOIDCClaim(obj client.Object) []string {
	sa := obj.(*corev1.ServiceAccount) // nolint: forcetypeassert
	rawClaimValues := []string{}
	for annotationKey, annotationValue := range sa.GetAnnotations() {
		if strings.HasPrefix(annotationKey, rbacapi.AnnotationKeyOIDCClaimNamePrefix) {
			rawClaimValue := strings.TrimSpace(annotationValue)
			if rawClaimValue == "" {
				continue
			}
			rawClaimValues = append(rawClaimValues, rawClaimValue)
		}
	}
	if len(rawClaimValues) == 0 {
		return nil
	}
	refinedClaimValues := []string{}
	for _, rawClaimValue := range rawClaimValues {
		claimValues := strings.Split(rawClaimValue, ",")
		for _, e := range claimValues {
			if email := strings.TrimSpace(e); email != "" {
				refinedClaimValues = append(refinedClaimValues, email)
			}
		}
	}
	return refinedClaimValues
}
