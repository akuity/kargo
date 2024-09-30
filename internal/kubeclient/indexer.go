package kubeclient

import (
	"context"
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"

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

	ServiceAccountsByOIDCClaimsIndexField = "claims"
)

// IndexEventsByInvolvedObjectAPIGroup sets up the indexing of Events by the
// API group of the involved object.
//
// It configures the field indexer of the provided cluster to allow querying
// Events by the API group of the involved object using the
// EventsByInvolvedObjectAPIGroupIndexField selector.
func IndexEventsByInvolvedObjectAPIGroup(ctx context.Context, clstr cluster.Cluster) error {
	return clstr.GetFieldIndexer().IndexField(
		ctx,
		&corev1.Event{},
		EventsByInvolvedObjectAPIGroupIndexField,
		indexEventsByInvolvedObjectAPIGroup,
	)
}

// indexEventsByInvolvedObjectAPIGroup is a client.IndexerFunc that indexes
// Events by the API group of the involved object.
func indexEventsByInvolvedObjectAPIGroup(obj client.Object) []string {
	event := obj.(*corev1.Event) // nolint: forcetypeassert
	// Ignore invalid APIVersion
	gv, _ := schema.ParseGroupVersion(event.InvolvedObject.APIVersion)
	if gv.Empty() || gv.Group == "" {
		return nil
	}
	return []string{gv.Group}
}

// IndexStagesByAnalysisRun sets up the indexing of Stages by the AnalysisRun
// they are associated with.
//
// It configures the field indexer of the provided cluster to allow querying
// Stages by the AnalysisRun they are associated with using the
// StagesByAnalysisRunIndexField selector.
func IndexStagesByAnalysisRun(ctx context.Context, clstr cluster.Cluster, shardName string) error {
	return clstr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Stage{},
		StagesByAnalysisRunIndexField,
		indexStagesByAnalysisRun(shardName))
}

// indexStagesByAnalysisRun is a client.IndexerFunc that indexes Stages by the
// AnalysisRun they are associated with.
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

// IndexStagesByArgoCDApplications sets up the indexing of Stages by the Argo CD
// Applications they are associated with.
//
// It configures the field indexer of the provided cluster to allow querying
// Stages by the Argo CD Applications they are associated with using the
// StagesByArgoCDApplicationsIndexField selector.
//
// When the provided shardName is non-empty, only Stages labeled with the
// provided shardName are indexed. When the provided shardName is empty, only
// Stages not labeled with a shardName are indexed.
func IndexStagesByArgoCDApplications(ctx context.Context, clstr cluster.Cluster, shardName string) error {
	return clstr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Stage{},
		StagesByArgoCDApplicationsIndexField,
		indexStagesByArgoCDApplications(shardName))
}

// indexStagesByArgoCDApplications returns a client.IndexerFunc that indexes
// Stages by the Argo CD Applications they are associated with.
//
// When the provided shardName is non-empty, only Stages labeled with the
// provided shardName are indexed. When the provided shardName is empty, only
// Stages not labeled with a shardName are indexed.
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
		// nolint: staticcheck
		if stage.Spec.PromotionMechanisms == nil || len(stage.Spec.PromotionMechanisms.ArgoCDAppUpdates) == 0 {
			return nil
		}
		// nolint: staticcheck
		apps := make([]string, len(stage.Spec.PromotionMechanisms.ArgoCDAppUpdates))
		// nolint: staticcheck
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

// IndexPromotionsByStage sets up the indexing of Promotions by the Stage they
// reference.
//
// It configures the field indexer of the provided cluster to allow querying
// Promotions by the Stage they reference using the PromotionsByStageIndexField.
func IndexPromotionsByStage(ctx context.Context, clstr cluster.Cluster) error {
	return clstr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Promotion{},
		PromotionsByStageIndexField,
		indexPromotionsByStage(),
	)
}

// indexPromotionsByStage returns a client.IndexerFunc that indexes Promotions
// by the Stage they reference. The provided predicates are used to further
// filter the Promotions that are indexed.
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

// IndexRunningPromotionsByArgoCDApplications sets up the indexing of running
// Promotions by the Argo CD Applications they are associated with.
//
// It configures the field indexer of the provided cluster to allow querying
// running Promotions by the Argo CD Applications they are associated with using
// the RunningPromotionsByArgoCDApplicationsIndexField selector.
//
// When the provided shardName is non-empty, only Promotions labeled with the
// provided shardName are indexed. When the provided shardName is empty, only
// Promotions not labeled with a shardName are indexed.
func IndexRunningPromotionsByArgoCDApplications(
	ctx context.Context,
	clstr cluster.Cluster,
	shardName string,
) error {
	return clstr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Promotion{},
		RunningPromotionsByArgoCDApplicationsIndexField,
		indexRunningPromotionsByArgoCDApplications(ctx, clstr.GetClient(), shardName),
	)
}

// indexRunningPromotionsByArgoCDApplications returns a client.IndexerFunc that
// indexes running Promotions by the Argo CD Applications they are associated
// with.
//
// When the provided shardName is non-empty, only Promotions labeled with the
// provided shardName are indexed. When the provided shardName is empty, only
// Promotions not labeled with a shardName are indexed.
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

		// nolint: staticcheck
		if stage.Spec.PromotionMechanisms == nil || len(stage.Spec.PromotionMechanisms.ArgoCDAppUpdates) == 0 {
			// If the Stage has no Argo CD Application promotion mechanisms,
			// then we have nothing to index.
			return nil
		}

		// nolint: staticcheck
		res := make([]string, len(stage.Spec.PromotionMechanisms.ArgoCDAppUpdates))
		// nolint: staticcheck
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

// IndexPromotionsByStageAndFreight sets up indexing of Promotions by the Stage
// and Freight they reference.
//
// It configures the cluster's field indexer to allow querying Promotions using
// the PromotionsByStageAndFreightIndexField selector. The value of the index is
// the concatenation of the Stage and Freight keys, as returned by the
// StageAndFreightKey function.
func IndexPromotionsByStageAndFreight(
	ctx context.Context,
	clstr cluster.Cluster,
) error {
	return clstr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Promotion{},
		PromotionsByStageAndFreightIndexField,
		indexPromotionsByStageAndFreight,
	)
}

// indexPromotionsByStageAndFreight is a client.IndexerFunc that indexes
// Promotions by the Freight and Stage they reference.
func indexPromotionsByStageAndFreight(obj client.Object) []string {
	promo := obj.(*kargoapi.Promotion) // nolint: forcetypeassert
	return []string{
		StageAndFreightKey(promo.Spec.Stage, promo.Spec.Freight),
	}
}

// StageAndFreightKey returns a key that uniquely identifies a Stage and
// Freight.
func StageAndFreightKey(stage, freight string) string {
	return fmt.Sprintf("%s:%s", stage, freight)
}

// IndexFreightByWarehouse sets up indexing of Freight by the Warehouse they are
// associated with.
//
// It configures the cluster's field indexer to allow querying Freight using the
// FreightByWarehouseIndexField selector.
func IndexFreightByWarehouse(ctx context.Context, clstr cluster.Cluster) error {
	return clstr.GetFieldIndexer().IndexField(
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

// IndexFreightByVerifiedStages sets up indexing of Freight by the Stages that
// have verified it.
//
// It configures the cluster's field indexer to allow querying Freight using
// the FreightByVerifiedStagesIndexField selector.
func IndexFreightByVerifiedStages(ctx context.Context, clstr cluster.Cluster) error {
	return clstr.GetFieldIndexer().IndexField(
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

// IndexFreightByApprovedStages sets up indexing of Freight by the Stages for
// which it has been (manually) approved.
//
// It configures the cluster's field indexer to allow querying Freight using
// the FreightApprovedForStagesIndexField selector.
func IndexFreightByApprovedStages(ctx context.Context, clstr cluster.Cluster) error {
	return clstr.GetFieldIndexer().IndexField(
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

// IndexStagesByFreight sets up indexing of Stages by the Freight they
// reference.
//
// It configures the cluster's field indexer to allow querying Stages using the
// StagesByFreightIndexField selector.
func IndexStagesByFreight(ctx context.Context, clstr cluster.Cluster) error {
	return clstr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Stage{},
		StagesByFreightIndexField,
		indexStagesByFreight,
	)
}

// indexStagesByFreight is a client.IndexerFunc that indexes Stages by the
// Freight they reference.
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
	slices.Sort(freightIDs)
	return freightIDs
}

// IndexStagesByUpstreamStages sets up indexing of Stages by the upstream Stages
// they reference.
//
// It configures the cluster's field indexer to allow querying Stages using the
// StagesByUpstreamStagesIndexField selector.
func IndexStagesByUpstreamStages(ctx context.Context, clstr cluster.Cluster) error {
	return clstr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Stage{},
		StagesByUpstreamStagesIndexField,
		indexStagesByUpstreamStages,
	)
}

// indexStagesByUpstreamStages is a client.IndexerFunc that indexes Stages by
// the upstream Stages they reference.
func indexStagesByUpstreamStages(obj client.Object) []string {
	stage := obj.(*kargoapi.Stage) // nolint: forcetypeassert
	var upstreams []string
	for _, req := range stage.Spec.RequestedFreight {
		upstreams = append(upstreams, req.Sources.Stages...)
	}
	slices.Sort(upstreams)
	return slices.Compact(upstreams)
}

// IndexStagesByWarehouse sets up indexing of Stages by the Warehouse they are
// associated with.
//
// It configures the cluster's field indexer to allow querying Stages using the
// StagesByWarehouseIndexField selector.
func IndexStagesByWarehouse(ctx context.Context, clstr cluster.Cluster) error {
	return clstr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Stage{},
		StagesByWarehouseIndexField,
		indexStagesByWarehouse,
	)
}

// indexStagesByWarehouse is a client.IndexerFunc that indexes Stages by the
// Warehouse they are associated with.
func indexStagesByWarehouse(obj client.Object) []string {
	stage := obj.(*kargoapi.Stage) // nolint: forcetypeassert
	var warehouses []string
	for _, req := range stage.Spec.RequestedFreight {
		if req.Origin.Kind == kargoapi.FreightOriginKindWarehouse && req.Sources.Direct {
			warehouses = append(warehouses, req.Origin.Name)
		}
	}
	slices.Sort(warehouses)
	return warehouses
}

// A helper function to format a claims name and values
// to be used by the IndexServiceAccountsByOIDCClaims index.
func FormatClaim(claimName string, claimValue string) string {
	return claimName + "/" + claimValue
}

// IndexServiceAccountsByOIDCClaims sets up indexing of ServiceAccounts by
// their OIDC claim annotations.
//
// It configures the manager's field indexer to allow querying ServiceAccounts
// using the ServiceAccountsByOIDCClaimIndexField selector.
func IndexServiceAccountsByOIDCClaims(ctx context.Context, clstr cluster.Cluster) error {
	return clstr.GetFieldIndexer().IndexField(
		ctx,
		&corev1.ServiceAccount{},
		ServiceAccountsByOIDCClaimsIndexField,
		indexServiceAccountsByOIDCClaims,
	)
}

// indexServiceAccountsByOIDCClaims is a client.IndexerFunc that indexes
// ServiceAccounts by the OIDC claims.
func indexServiceAccountsByOIDCClaims(obj client.Object) []string {
	sa := obj.(*corev1.ServiceAccount) // nolint: forcetypeassert
	refinedClaimValues := []string{}
	for annotationKey, annotationValue := range sa.GetAnnotations() {
		if strings.HasPrefix(annotationKey, rbacapi.AnnotationKeyOIDCClaimNamePrefix) {
			rawClaimName := strings.TrimPrefix(annotationKey, rbacapi.AnnotationKeyOIDCClaimNamePrefix)
			rawClaimValue := strings.TrimSpace(annotationValue)
			if rawClaimValue == "" {
				continue
			}
			claimValues := strings.Split(rawClaimValue, ",")
			for _, e := range claimValues {
				if claimValue := strings.TrimSpace(e); claimValue != "" {
					refinedClaimValues = append(refinedClaimValues, FormatClaim(rawClaimName, claimValue))
				}
			}
		}
	}
	if len(refinedClaimValues) == 0 {
		return nil
	}
	return refinedClaimValues
}

func isPromotionPhaseNonTerminal(promo *kargoapi.Promotion) bool {
	return !promo.Status.Phase.IsTerminal()
}
