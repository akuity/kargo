package event

import (
	"context"
	"encoding/json"
	"maps"
	"sort"
	"time"

	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libargocd "github.com/akuity/kargo/internal/argocd"
	"github.com/akuity/kargo/internal/expressions"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// NewPromotionAnnotations returns annotations for a Promotion related event.
// It may skip some fields when error occurred during serialization, to record event with best-effort.
func NewPromotionAnnotations(
	ctx context.Context,
	actor string,
	p *kargoapi.Promotion,
	f *kargoapi.Freight,
) map[string]string {
	logger := logging.LoggerFromContext(ctx)

	annotations := map[string]string{
		kargoapi.AnnotationKeyEventProject:             p.GetNamespace(),
		kargoapi.AnnotationKeyEventPromotionName:       p.GetName(),
		kargoapi.AnnotationKeyEventFreightName:         p.Spec.Freight,
		kargoapi.AnnotationKeyEventStageName:           p.Spec.Stage,
		kargoapi.AnnotationKeyEventPromotionCreateTime: p.GetCreationTimestamp().Format(time.RFC3339),
	}

	if actor != "" {
		annotations[kargoapi.AnnotationKeyEventActor] = actor
	}
	// All Promotion-related events are emitted after the promotion was created.
	// Therefore, if the promotion knows who triggered it, set them as an actor.
	if promoteActor, ok := p.Annotations[kargoapi.AnnotationKeyCreateActor]; ok {
		annotations[kargoapi.AnnotationKeyEventActor] = promoteActor
	}

	if f != nil {
		annotations[kargoapi.AnnotationKeyEventFreightCreateTime] = f.CreationTimestamp.Format(time.RFC3339)
		annotations[kargoapi.AnnotationKeyEventFreightAlias] = f.Alias
		if len(f.Commits) > 0 {
			data, err := json.Marshal(f.Commits)
			if err != nil {
				logger.Error(err, "marshal freight commits in JSON")
			} else {
				annotations[kargoapi.AnnotationKeyEventFreightCommits] = string(data)
			}
		}
		if len(f.Images) > 0 {
			data, err := json.Marshal(f.Images)
			if err != nil {
				logger.Error(err, "marshal freight images in JSON")
			} else {
				annotations[kargoapi.AnnotationKeyEventFreightImages] = string(data)
			}
		}
		if len(f.Charts) > 0 {
			data, err := json.Marshal(f.Charts)
			if err != nil {
				logger.Error(err, "marshal freight charts in JSON")
			} else {
				annotations[kargoapi.AnnotationKeyEventFreightCharts] = string(data)
			}
		}
	}

	baseEnv := map[string]any{
		"ctx": map[string]any{
			"project":   p.GetNamespace(),
			"promotion": p.GetName(),
			"stage":     p.Spec.Stage,
			"meta": map[string]any{
				"promotion": map[string]any{
					"actor": p.Annotations[kargoapi.AnnotationKeyCreateActor],
				},
			},
		},
	}
	if f != nil {
		targetFreight := map[string]any{
			"name": f.Name,
		}
		if f.Origin.Name != "" {
			targetFreight["origin"] = map[string]any{
				"name": f.Origin.Name,
			}
		}
		if ctx, ok := baseEnv["ctx"].(map[string]any); ok {
			ctx["targetFreight"] = targetFreight
		}
	}

	setVar := func(env map[string]any, vars map[string]any) {
		if _, ok := env["vars"]; !ok {
			env["vars"] = make(map[string]any)
		}
		if varsMap, ok := env["vars"].(map[string]any); ok {
			maps.Copy(varsMap, vars)
		}
	}

	var allApps []types.NamespacedName
	appSet := make(map[types.NamespacedName]struct{})
	promotionVars := calculatePromotionVars(p, baseEnv)
	setVar(baseEnv, promotionVars)

	for _, step := range p.Spec.Steps {
		if step.Uses != "argocd-update" || step.Config == nil {
			continue
		}
		stepEnv := make(map[string]any)
		maps.Copy(stepEnv, baseEnv)
		stepVars := calculateStepVars(step, stepEnv)
		setVar(stepEnv, stepVars)

		evaledConfig, err := expressions.EvaluateJSONTemplate(step.Config.Raw, stepEnv)
		if err != nil {
			logger.Error(err, "evaluate step config template")
			continue
		}

		var cfg builtin.ArgoCDUpdateConfig
		if err := json.Unmarshal(evaledConfig, &cfg); err != nil {
			logger.Error(err, "unmarshal evaluated ArgoCD update config")
			continue
		}

		for _, app := range cfg.Apps {
			namespacedName := types.NamespacedName{
				Namespace: app.Namespace,
				Name:      app.Name,
			}
			if namespacedName.Namespace == "" {
				namespacedName.Namespace = libargocd.Namespace()
			}
			if _, exists := appSet[namespacedName]; !exists {
				appSet[namespacedName] = struct{}{}
				allApps = append(allApps, namespacedName)
			}
		}
	}

	if len(allApps) == 0 {
		return annotations
	}

	sort.Slice(allApps, func(i, j int) bool {
		if allApps[i].Namespace != allApps[j].Namespace {
			return allApps[i].Namespace < allApps[j].Namespace
		}
		return allApps[i].Name < allApps[j].Name
	})

	if data, err := json.Marshal(allApps); err == nil {
		annotations[kargoapi.AnnotationKeyEventApplications] = string(data)
	}

	return annotations
}

func calculatePromotionVars(
	p *kargoapi.Promotion,
	baseEnv map[string]any,
) map[string]any {
	vars := make(map[string]any)

	for _, v := range p.Spec.Vars {
		env := make(map[string]any)
		maps.Copy(env, baseEnv)
		env["vars"] = vars

		newVar, err := expressions.EvaluateTemplate(v.Value, env)
		if err != nil {
			continue
		}
		vars[v.Name] = newVar
	}

	return vars
}

func calculateStepVars(
	step kargoapi.PromotionStep,
	baseEnv map[string]any,
) map[string]any {
	vars := make(map[string]any)

	for _, v := range step.Vars {
		env := make(map[string]any)
		maps.Copy(env, baseEnv)
		if existingVars, ok := baseEnv["vars"].(map[string]any); ok {
			envVars := make(map[string]any)
			env["vars"] = envVars
			for k, v := range existingVars {
				envVars[k] = v
			}
			for k, val := range vars {
				envVars[k] = val
			}
		} else {
			env["vars"] = vars
		}

		newVar, err := expressions.EvaluateTemplate(v.Value, env)
		if err != nil {
			continue
		}
		vars[v.Name] = newVar
	}

	return vars
}
