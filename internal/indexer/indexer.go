package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libargocd "github.com/akuity/kargo/internal/argocd"
	"github.com/akuity/kargo/internal/expressions"
	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/promotion"
)

const (
	EventsByInvolvedObjectAPIGroupField = "involvedObject.apiGroup"

	FreightByWarehouseField       = "warehouse"
	FreightByCurrentStagesField   = "currentlyIn"
	FreightByVerifiedStagesField  = "verifiedIn"
	FreightApprovedForStagesField = "approvedFor"

	PromotionsByStageAndFreightField = "stageAndFreight"
	PromotionsByStageField           = "stage"

	RunningPromotionsByArgoCDApplicationsField = "applications"

	StagesByAnalysisRunField    = "analysisRun"
	StagesByFreightField        = "freight"
	StagesByUpstreamStagesField = "upstreamStages"
	StagesByWarehouseField      = "warehouse"

	ServiceAccountsByOIDCClaimsField = "claims"

	WarehouseRepoURLIndexKey = "subscriptions.repoURL"
)

// EventsByInvolvedObjectAPIGroup is a client.IndexerFunc that indexes
// Events by the API group of the involved object.
func EventsByInvolvedObjectAPIGroup(obj client.Object) []string {
	event, ok := obj.(*corev1.Event)
	if !ok {
		return nil
	}

	// Ignore invalid APIVersion
	gv, _ := schema.ParseGroupVersion(event.InvolvedObject.APIVersion)
	if gv.Empty() || gv.Group == "" {
		return nil
	}
	return []string{gv.Group}
}

// StagesByAnalysisRun is a client.IndexerFunc that indexes Stages by the
// AnalysisRun they are associated with.
func StagesByAnalysisRun(shardName string) client.IndexerFunc {
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

		stage, ok := obj.(*kargoapi.Stage)
		if !ok {
			return nil
		}

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

// PromotionsByStage returns a client.IndexerFunc that indexes Promotions
// by the Stage they reference.
func PromotionsByStage(obj client.Object) []string {
	promo, ok := obj.(*kargoapi.Promotion)
	if !ok {
		return nil
	}
	return []string{promo.Spec.Stage}
}

// RunningPromotionsByArgoCDApplications returns a client.IndexerFunc that
// indexes running Promotions by the Argo CD Applications they are associated
// with.
//
// When the provided shardName is non-empty, only Promotions labeled with the
// provided shardName are indexed. When the provided shardName is empty, only
// Promotions not labeled with a shardName are indexed.
func RunningPromotionsByArgoCDApplications(
	ctx context.Context,
	cl client.Client,
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

		// Build just enough context to extract the relevant config from the
		// argocd-update promotion step.
		promoCtx := promotion.Context{
			Project:   promo.Namespace,
			Stage:     promo.Spec.Stage,
			Promotion: promo.Name,
			State:     promo.Status.GetState(),
			Vars:      promo.Spec.Vars,
		}

		// Extract the Argo CD Applications from the promotion steps.
		//
		// TODO(hidde): This is not ideal as it requires parsing the step configs
		// and treating some of them as special cases. We should consider a more
		// general approach in the future.
		var res []string
		for i, step := range promo.Spec.Steps {
			if int64(i) > promo.Status.CurrentStep {
				// We are only interested in steps that have already been executed or
				// are about to be.
				break
			}
			if step.Uses != "argocd-update" || step.Config == nil {
				continue
			}

			dirStep := promotion.Step{
				Kind:   step.Uses,
				Alias:  step.As,
				Vars:   step.Vars,
				Config: step.Config.Raw,
			}

			// As step-level variables are allowed to reference to output, we
			// need to provide the state.
			vars, err := dirStep.GetVars(ctx, cl, promoCtx, promoCtx.State)
			if err != nil {
				logger.Error(
					err,
					fmt.Sprintf(
						"failed to extract relevant config from Promotion step %d:"+
							"ignoring any Argo CD Applications from this step",
						i,
					),
					"promo", promo.Name,
					"namespace", promo.Namespace,
				)
				continue
			}
			// Unpack the raw config into a map. We're not unpacking it into a struct
			// because:
			// 1. We don't want to evaluate expressions throughout the entire config
			//    because we may not have all context required to do so available.
			//    We will only evaluate expressions in specific fields.
			// 2. If there are expressions in the config, some fields that may not be
			//    strings in the struct may be strings in the unevaluated config and
			//    this could lead to unmarshaling errors.
			cfgMap := map[string]any{}
			if err = json.Unmarshal(step.Config.Raw, &cfgMap); err != nil {
				logger.Error(
					err,
					fmt.Sprintf(
						"failed to extract relevant config from Promotion step %d:"+
							"ignoring any Argo CD Applications from this step",
						i,
					),
					"promo", promo.Name,
					"namespace", promo.Namespace,
				)
				continue
			}
			// Dig through the map to find the names and namespaces of related Argo CD
			// Applications. Treat these as templates and evaluate expressions in
			// these individual fields without evaluating the entire config.
			if apps, ok := cfgMap["apps"]; ok {
				if appsList, ok := apps.([]any); ok {
					for _, app := range appsList {
						if app, ok := app.(map[string]any); ok {
							if nameTemplate, ok := app["name"].(string); ok {
								env := dirStep.BuildEnv(
									promoCtx,
									promotion.StepEnvWithOutputs(promoCtx.State),
									promotion.StepEnvWithTaskOutputs(dirStep.Alias, promoCtx.State),
									promotion.StepEnvWithVars(vars),
								)

								var namespace any = libargocd.Namespace()
								if namespaceTemplate, ok := app["namespace"].(string); ok {
									if namespace, err = expressions.EvaluateTemplate(namespaceTemplate, env); err != nil {
										logger.Error(
											err,
											fmt.Sprintf(
												"failed to extract relevant config from Promotion step %d:"+
													"ignoring any Argo CD Applications from this step",
												i,
											),
											"promo", promo.Name,
											"namespace", promo.Namespace,
										)
										continue
									}
								}
								name, err := expressions.EvaluateTemplate(nameTemplate, env)
								if err != nil {
									logger.Error(
										err,
										fmt.Sprintf(
											"failed to extract relevant config from Promotion step %d:"+
												"ignoring any Argo CD Applications from this step",
											i,
										),
										"promo", promo.Name,
										"namespace", promo.Namespace,
									)
									continue
								}
								res = append(res, fmt.Sprintf("%s:%s", namespace, name))
							}
						}
					}
				}
			}
		}
		return res
	}
}

// PromotionsByStageAndFreight is a client.IndexerFunc that indexes Promotions
// by the Freight and Stage they reference.
func PromotionsByStageAndFreight(obj client.Object) []string {
	promo, ok := obj.(*kargoapi.Promotion)
	if !ok {
		return nil
	}

	return []string{
		StageAndFreightKey(promo.Spec.Stage, promo.Spec.Freight),
	}
}

// StageAndFreightKey returns a key that uniquely identifies a Stage and
// Freight.
func StageAndFreightKey(stage, freight string) string {
	return fmt.Sprintf("%s:%s", stage, freight)
}

// FreightByWarehouse is a client.IndexerFunc that indexes Freight by the
// Warehouse it is associated with.
func FreightByWarehouse(obj client.Object) []string {
	freight, ok := obj.(*kargoapi.Freight)
	if !ok {
		return nil
	}

	if freight.Origin.Kind == kargoapi.FreightOriginKindWarehouse {
		return []string{freight.Origin.Name}
	}
	return nil
}

// FreightByCurrentStages is a client.IndexerFunc that indexes Freight by the
// Stages in which it is currently in use.
func FreightByCurrentStages(obj client.Object) []string {
	freight, ok := obj.(*kargoapi.Freight)
	if !ok {
		return nil
	}

	currentStages := make([]string, len(freight.Status.CurrentlyIn))
	var i int
	for stage := range freight.Status.CurrentlyIn {
		currentStages[i] = stage
		i++
	}
	return currentStages
}

// FreightByVerifiedStages is a client.IndexerFunc that indexes Freight by the
// Stages in which it has been verified.
func FreightByVerifiedStages(obj client.Object) []string {
	freight, ok := obj.(*kargoapi.Freight)
	if !ok {
		return nil
	}

	verifiedStages := make([]string, len(freight.Status.VerifiedIn))
	var i int
	for stage := range freight.Status.VerifiedIn {
		verifiedStages[i] = stage
		i++
	}
	return verifiedStages
}

// FreightApprovedForStages is a client.IndexerFunc that indexes Freight by the
// Stages for which it has been (manually) approved.
func FreightApprovedForStages(obj client.Object) []string {
	freight, ok := obj.(*kargoapi.Freight)
	if !ok {
		return nil
	}

	approvedStages := make([]string, len(freight.Status.ApprovedFor))
	var i int
	for stages := range freight.Status.ApprovedFor {
		approvedStages[i] = stages
		i++
	}
	return approvedStages
}

// StagesByFreight is a client.IndexerFunc that indexes Stages by the Freight
// they reference.
func StagesByFreight(obj client.Object) []string {
	stage, ok := obj.(*kargoapi.Stage)
	if !ok {
		return nil
	}

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

// StagesByUpstreamStages is a client.IndexerFunc that indexes Stages by the
// upstream Stages they reference.
func StagesByUpstreamStages(obj client.Object) []string {
	stage, ok := obj.(*kargoapi.Stage)
	if !ok {
		return nil
	}

	var upstreams []string
	for _, req := range stage.Spec.RequestedFreight {
		upstreams = append(upstreams, req.Sources.Stages...)
	}
	slices.Sort(upstreams)
	return slices.Compact(upstreams)
}

// StagesByWarehouse is a client.IndexerFunc that indexes Stages by the
// Warehouse they are associated with.
func StagesByWarehouse(obj client.Object) []string {
	stage, ok := obj.(*kargoapi.Stage)
	if !ok {
		return nil
	}

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

// ServiceAccountsByOIDCClaims is a client.IndexerFunc that indexes
// ServiceAccounts by their OIDC claims.
func ServiceAccountsByOIDCClaims(obj client.Object) []string {
	sa, ok := obj.(*corev1.ServiceAccount)
	if !ok {
		return nil
	}

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

// WarehousesByRepoURL is a client.IndexerFunc that indexes Warehouses by the
// the RepoURLs they are associated with.
func WarehousesByRepoURL(obj client.Object) []string {
	warehouse, ok := obj.(*kargoapi.Warehouse)
	if !ok {
		return nil
	}

	var repoURLs []string
	for _, sub := range warehouse.Spec.Subscriptions {
		if sub.Git != nil && sub.Git.RepoURL != "" {
			repoURLs = append(repoURLs,
				git.NormalizeURL(sub.Git.RepoURL),
			)
		}
		if sub.Chart != nil && sub.Chart.RepoURL != "" {
			repoURLs = append(repoURLs,
				helm.NormalizeChartRepositoryURL(sub.Chart.RepoURL),
			)
		}
		if sub.Image != nil && sub.Image.RepoURL != "" {
			repoURLs = append(repoURLs,
				// The normalization of Helm chart repository URLs can also be used here
				// to ensure the uniqueness of the image reference as it does the job of
				// ensuring lower-casing, etc. without introducing unwanted side effects.
				helm.NormalizeChartRepositoryURL(sub.Image.RepoURL),
			)
		}
	}
	return repoURLs
}
