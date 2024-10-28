package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libargocd "github.com/akuity/kargo/internal/argocd"
	"github.com/akuity/kargo/internal/directives"
	"github.com/akuity/kargo/internal/logging"
)

const (
	EventsByInvolvedObjectAPIGroupIndexField = "involvedObject.apiGroup"

	FreightByVerifiedStagesIndexField     = "verifiedIn"
	FreightApprovedForStagesIndexField    = "approvedFor"
	FreightByWarehouseIndexField          = "warehouse"
	PromotionsByStageAndFreightIndexField = "stageAndFreight"
	PromotionsByTerminalIndexField        = "terminal"

	PromotionsByStageIndexField = "stage"

	RunningPromotionsByArgoCDApplicationsIndexField = "applications"

	StagesByAnalysisRunIndexField    = "analysisRun"
	StagesByFreightIndexField        = "freight"
	StagesByUpstreamStagesIndexField = "upstreamStages"
	StagesByWarehouseIndexField      = "warehouse"

	ServiceAccountsByOIDCClaimsIndexField = "claims"
)

// EventsByInvolvedObjectAPIGroup is a client.IndexerFunc that indexes
// Events by the API group of the involved object.
func EventsByInvolvedObjectAPIGroup(obj client.Object) []string {
	event := obj.(*corev1.Event) // nolint: forcetypeassert
	// Ignore invalid APIVersion
	gv, _ := schema.ParseGroupVersion(event.InvolvedObject.APIVersion)
	if gv.Empty() || gv.Group == "" {
		return nil
	}
	return []string{gv.Group}
}

// StagesByAnalysisRunIndexer is a client.IndexerFunc that indexes Stages by the
// AnalysisRun they are associated with.
func StagesByAnalysisRunIndexer(shardName string) client.IndexerFunc {
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

// PromotionsByStageIndexer returns a client.IndexerFunc that indexes Promotions
// by the Stage they reference. The provided predicates are used to further
// filter the Promotions that are indexed.
func PromotionsByStageIndexer(predicates ...func(*kargoapi.Promotion) bool) client.IndexerFunc {
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

// RunningPromotionsByArgoCDApplicationsIndexer returns a client.IndexerFunc that
// indexes running Promotions by the Argo CD Applications they are associated
// with.
//
// When the provided shardName is non-empty, only Promotions labeled with the
// provided shardName are indexed. When the provided shardName is empty, only
// Promotions not labeled with a shardName are indexed.
func RunningPromotionsByArgoCDApplicationsIndexer(
	ctx context.Context,
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

		// Extract the Argo CD Applications from the promotion steps.
		//
		// TODO(hidde): While this is arguably already better than the "legacy"
		// approach further down, which had to query the Stage to get the
		// Applications, it is still not ideal as it requires parsing the
		// directives and treating some of them as special cases. We should
		// consider a more general approach in the future.
		var res []string
		for i, step := range promo.Spec.Steps {
			if step.Uses != "argocd-update" || step.Config == nil {
				continue
			}

			config := directives.ArgoCDUpdateConfig{}
			if err := json.Unmarshal(step.Config.Raw, &config); err != nil {
				logger.Error(
					err,
					fmt.Sprintf(
						"failed to extract config from Promotion step %d:"+
							"ignoring any Argo CD Applications from this step",
						i,
					),
					"promo", promo.Name,
					"namespace", promo.Namespace,
				)
				continue
			}

			for _, app := range config.Apps {
				namespace := app.Namespace
				if namespace == "" {
					namespace = libargocd.Namespace()
				}
				res = append(res, fmt.Sprintf("%s:%s", namespace, app.Name))
			}
		}
		return res
	}
}

// PromotionsByStageAndFreightIndexer is a client.IndexerFunc that indexes
// Promotions by the Freight and Stage they reference.
func PromotionsByStageAndFreightIndexer(obj client.Object) []string {
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

// FreightByWarehouseIndexer is a client.IndexerFunc that indexes Freight by the
// Warehouse it is associated with.
func FreightByWarehouseIndexer(obj client.Object) []string {
	freight := obj.(*kargoapi.Freight) // nolint: forcetypeassert
	if freight.Origin.Kind == kargoapi.FreightOriginKindWarehouse {
		return []string{freight.Origin.Name}
	}
	return nil
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

// StagesByFreightIndexer is a client.IndexerFunc that indexes Stages by the
// Freight they reference.
func StagesByFreightIndexer(obj client.Object) []string {
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

// StagesByUpstreamStagesIndexer is a client.IndexerFunc that indexes Stages by
// the upstream Stages they reference.
func StagesByUpstreamStagesIndexer(obj client.Object) []string {
	stage := obj.(*kargoapi.Stage) // nolint: forcetypeassert
	var upstreams []string
	for _, req := range stage.Spec.RequestedFreight {
		upstreams = append(upstreams, req.Sources.Stages...)
	}
	slices.Sort(upstreams)
	return slices.Compact(upstreams)
}

// StagesByWarehouseIndexer is a client.IndexerFunc that indexes Stages by the
// Warehouse they are associated with.
func StagesByWarehouseIndexer(obj client.Object) []string {
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

// FormatClaim formats a claims name and values to be used by the
// IndexServiceAccountsByOIDCClaims index.
func FormatClaim(claimName string, claimValue string) string {
	return claimName + "/" + claimValue
}

// ServiceAccountsByOIDCClaimsIndexer is a client.IndexerFunc that indexes
// ServiceAccounts by the OIDC claims.
func ServiceAccountsByOIDCClaimsIndexer(obj client.Object) []string {
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

// PromotionsByTerminalIndexer is a client.IndexerFunc that indexes Promotions by
// whether or not their phase is terminal.
func PromotionsByTerminalIndexer(obj client.Object) []string {
	promo := obj.(*kargoapi.Promotion) // nolint: forcetypeassert
	return []string{strconv.FormatBool(isPromotionPhaseNonTerminal(promo))}
}

func isPromotionPhaseNonTerminal(promo *kargoapi.Promotion) bool {
	return !promo.Status.Phase.IsTerminal()
}
