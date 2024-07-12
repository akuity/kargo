package kubeclient

import (
	"context"
	"fmt"
	"slices"
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

	PromotionsByStageIndexField = "stage"

	RunningPromotionsByArgoCDApplicationsIndexField = "applications"

	StagesByAnalysisRunIndexField        = "analysisRun"
	StagesByArgoCDApplicationsIndexField = "applications"
	StagesByFreightIndexField            = "freight"
	StagesByUpstreamStagesIndexField     = "upstreamStages"
	StagesByWarehouseIndexField          = "warehouse"

	ServiceAccountsByOIDCEmailIndexField   = "email"
	ServiceAccountsByOIDCGroupIndexField   = "groups"
	ServiceAccountsByOIDCSubjectIndexField = "subjects"
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

		currentFC := stage.Status.FreightHistory.Current()
		if currentFC == nil {
			return nil
		}
		currentVI := currentFC.VerificationHistory.Current()
		if currentVI == nil || currentVI.AnalysisRun == nil {
			return nil
		}

		return []string{fmt.Sprintf(
			"%s:%s",
			currentVI.AnalysisRun.Namespace,
			currentVI.AnalysisRun.Name,
		)}
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
		FreightByWarehouseIndexer,
	)
}

// FreightByWarehouseIndexer is a client.IndexerFunc that indexes Freight by the
// Warehouse it is associated with.
func FreightByWarehouseIndexer(obj client.Object) []string {
	freight := obj.(*kargoapi.Freight) // nolint: forcetypeassert
	if freight.Origin.Kind == kargoapi.FreightOriginKindWarehouse {
		return []string{freight.Origin.Name}
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
		FreightByVerifiedStagesIndexer,
	)
}

// FreightByVerifiedStagesIndexer is a client.IndexerFunc that indexes Freight
// by the Stages in which it has been verified.
func FreightByVerifiedStagesIndexer(obj client.Object) []string {
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
		FreightApprovedForStagesIndexer,
	)
}

// FreightApprovedForStagesIndexer is a client.IndexerFunc that indexes Freight
// by the Stages for which it has been (manually) approved.
func FreightApprovedForStagesIndexer(obj client.Object) []string {
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

	current := stage.Status.FreightHistory.Current()
	if current == nil || len(current.Freight) == 0 {
		return nil
	}

	var freightIDs []string
	for _, freight := range current.Freight {
		freightIDs = append(freightIDs, freight.Name)
	}
	return freightIDs
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
	var upstreams []string
	for _, req := range stage.Spec.RequestedFreight {
		upstreams = append(upstreams, req.Sources.Stages...)
	}
	slices.Sort(upstreams)
	return slices.Compact(upstreams)
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
	var warehouses []string
	for _, req := range stage.Spec.RequestedFreight {
		if req.Origin.Kind == kargoapi.FreightOriginKindWarehouse && req.Sources.Direct {
			warehouses = append(warehouses, req.Origin.Name)
		}
	}
	return warehouses
}

func IndexServiceAccountsByOIDCEmail(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&corev1.ServiceAccount{},
		ServiceAccountsByOIDCEmailIndexField,
		indexServiceAccountsOIDCEmail,
	)
}

func indexServiceAccountsOIDCEmail(obj client.Object) []string {
	sa := obj.(*corev1.ServiceAccount) // nolint: forcetypeassert
	rawEmails := strings.TrimSpace(sa.GetAnnotations()[rbacapi.AnnotationKeyOIDCEmails])
	if rawEmails == "" {
		return nil
	}
	emails := strings.Split(rawEmails, ",")
	refinedEmails := make([]string, 0, len(emails))
	for _, e := range emails {
		if email := strings.TrimSpace(e); email != "" {
			refinedEmails = append(refinedEmails, email)
		}
	}
	return refinedEmails
}

func IndexServiceAccountsByOIDCGroups(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&corev1.ServiceAccount{},
		ServiceAccountsByOIDCGroupIndexField,
		indexServiceAccountsByOIDCGroups,
	)
}

func indexServiceAccountsByOIDCGroups(obj client.Object) []string {
	sa := obj.(*corev1.ServiceAccount) // nolint: forcetypeassert
	rawGroups := strings.TrimSpace(sa.GetAnnotations()[rbacapi.AnnotationKeyOIDCGroups])
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

func IndexServiceAccountsByOIDCSubjects(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&corev1.ServiceAccount{},
		ServiceAccountsByOIDCSubjectIndexField,
		indexServiceAccountsByOIDCSubjects,
	)
}

func indexServiceAccountsByOIDCSubjects(obj client.Object) []string {
	sa := obj.(*corev1.ServiceAccount) // nolint: forcetypeassert
	rawSubjects := strings.TrimSpace(sa.GetAnnotations()[rbacapi.AnnotationKeyOIDCSubjects])
	if rawSubjects == "" {
		return nil
	}
	subjects := strings.Split(rawSubjects, ",")
	refinedSubjects := make([]string, 0, len(subjects))
	for _, s := range subjects {
		if subject := strings.TrimSpace(s); subject != "" {
			refinedSubjects = append(refinedSubjects, subject)
		}
	}
	return refinedSubjects
}
