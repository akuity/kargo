package event

import (
	"encoding/json"
	"fmt"
	"maps"
	"sort"
	"strconv"
	"strings"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/argocd"
	"github.com/akuity/kargo/pkg/expressions"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// PromotionEvent represents the data available for various events related to promotions.
type PromotionEvent struct {
	Project                string                 `json:"project"`
	PromotionName          string                 `json:"promotionName"`
	FreightName            string                 `json:"freightName"`
	StageName              string                 `json:"stageName"`
	Message                string                 `json:"message"`
	PromotionCreateTime    time.Time              `json:"promotionCreateTime"`
	Actor                  *string                `json:"actor,omitempty"`
	FreightCreateTime      *time.Time             `json:"freightCreateTime,omitempty"`
	FreightAlias           *string                `json:"freightAlias,omitempty"`
	FreightCommits         []kargoapi.GitCommit   `json:"freightCommits,omitempty"`
	FreightImages          []kargoapi.Image       `json:"freightImages,omitempty"`
	FreightCharts          []kargoapi.Chart       `json:"freightCharts,omitempty"`
	Applications           []types.NamespacedName `json:"applications,omitempty"`
	VerificationPending    *bool                  `json:"verificationPending,omitempty"`
	VerificationStartTime  *time.Time             `json:"verificationStartTime,omitempty"`
	VerificationFinishTime *time.Time             `json:"verificationFinishTime,omitempty"`
	AnalysisRunName        *string                `json:"analysisRunName,omitempty"`
}

// NewPromotionEvent creates a new CloudEvent from the given promotion and freight data.
// The given actor will be used if it is not empty, but it will be overridden if the promotion has
// an actor annotation. The eventType comes from any of the `EventType` constants available in the
// kargo `api/v1alpha1` package.
func NewPromotionEvent(message,
	actor string, promotion *kargoapi.Promotion,
	freight *kargoapi.Freight,
) PromotionEvent {
	evt := PromotionEvent{
		Project:             promotion.GetNamespace(),
		PromotionName:       promotion.GetName(),
		FreightName:         promotion.Spec.Freight,
		StageName:           promotion.Spec.Stage,
		PromotionCreateTime: promotion.GetCreationTimestamp().Time,
		Message:             message,
	}
	if actor != "" {
		evt.Actor = &actor
	}
	// All Promotion-related events are emitted after the promotion was created.
	// Therefore, if the promotion knows who triggered it, set them as an actor.
	if promoteActor, ok := promotion.Annotations[kargoapi.AnnotationKeyCreateActor]; ok {
		evt.Actor = &promoteActor
	}
	if freight != nil {
		evt.FreightCreateTime = &freight.CreationTimestamp.Time
		evt.FreightAlias = &freight.Alias
		if len(freight.Commits) > 0 {
			evt.FreightCommits = freight.Commits
		}
		if len(freight.Images) > 0 {
			evt.FreightImages = freight.Images
		}
		if len(freight.Charts) > 0 {
			evt.FreightCharts = freight.Charts
		}
	}

	baseEnv := map[string]any{
		"ctx": map[string]any{
			"project":   promotion.GetNamespace(),
			"promotion": promotion.GetName(),
			"stage":     promotion.Spec.Stage,
			"meta": map[string]any{
				"promotion": map[string]any{
					"actor": promotion.Annotations[kargoapi.AnnotationKeyCreateActor],
				},
			},
		},
	}
	if freight != nil {
		targetFreight := map[string]any{
			"name": freight.Name,
		}
		if freight.Origin.Name != "" {
			targetFreight["origin"] = map[string]any{
				"name": freight.Origin.Name,
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
	promotionVars := calculatePromotionVars(promotion, baseEnv)
	setVar(baseEnv, promotionVars)

	// NOTE(thomastaylor312): Right now if there are any errors when trying to evaluate the
	// expressions or convert types, we just skip. Originally we logged this, but this library
	// doesn't have logging, so this retains the same behavior. If we want to log these errors, we
	// can add a logger to the context and use that here.
	for _, step := range promotion.Spec.Steps {
		if step.Uses != "argocd-update" || step.Config == nil {
			continue
		}
		var cfg builtin.ArgoCDUpdateConfig
		if err := json.Unmarshal(step.Config.Raw, &cfg); err != nil {
			continue
		}
		for _, app := range cfg.Apps {
			namespacedName := types.NamespacedName{
				Namespace: app.Namespace,
				Name:      app.Name,
			}

			if strings.Contains(namespacedName.Namespace, "${{") ||
				strings.Contains(namespacedName.Name, "${{") {
				stepEnv := make(map[string]any)
				maps.Copy(stepEnv, baseEnv)
				stepVars := calculateStepVars(step, stepEnv)
				setVar(stepEnv, stepVars)
				var ok bool
				namespaceAny, err := expressions.EvaluateTemplate(namespacedName.Namespace, stepEnv)
				if err != nil {
					continue
				}
				if namespacedName.Namespace, ok = namespaceAny.(string); !ok {
					continue
				}
				appNameAny, err := expressions.EvaluateTemplate(namespacedName.Name, stepEnv)
				if err != nil {
					continue
				}
				if namespacedName.Name, ok = appNameAny.(string); !ok {
					continue
				}
			}

			if namespacedName.Namespace == "" {
				namespacedName.Namespace = argocd.Namespace()
			}
			if _, exists := appSet[namespacedName]; !exists {
				appSet[namespacedName] = struct{}{}
				allApps = append(allApps, namespacedName)
			}
		}
	}
	if len(allApps) > 0 {
		sort.Slice(allApps, func(i, j int) bool {
			if allApps[i].Namespace != allApps[j].Namespace {
				return allApps[i].Namespace < allApps[j].Namespace
			}
			return allApps[i].Name < allApps[j].Name
		})

		evt.Applications = allApps
	}

	return evt
}

func (p *PromotionEvent) MarshalAnnotations() map[string]string {
	// Note that we skip message here, as it is not used in the annotations.
	annotations := map[string]string{
		kargoapi.AnnotationKeyEventProject:             p.Project,
		kargoapi.AnnotationKeyEventPromotionName:       p.PromotionName,
		kargoapi.AnnotationKeyEventFreightName:         p.FreightName,
		kargoapi.AnnotationKeyEventStageName:           p.StageName,
		kargoapi.AnnotationKeyEventPromotionCreateTime: p.PromotionCreateTime.Format(time.RFC3339),
	}

	if p.Actor != nil {
		annotations[kargoapi.AnnotationKeyEventActor] = *p.Actor
	}
	if p.FreightCreateTime != nil {
		annotations[kargoapi.AnnotationKeyEventFreightCreateTime] =
			p.FreightCreateTime.Format(time.RFC3339)
	}
	if p.FreightAlias != nil {
		annotations[kargoapi.AnnotationKeyEventFreightAlias] = *p.FreightAlias
	}
	if len(p.FreightCommits) > 0 {
		data, err := json.Marshal(p.FreightCommits)
		if err == nil {
			annotations[kargoapi.AnnotationKeyEventFreightCommits] = string(data)
		}
	}
	if len(p.FreightImages) > 0 {
		data, err := json.Marshal(p.FreightImages)
		if err == nil {
			annotations[kargoapi.AnnotationKeyEventFreightImages] = string(data)
		}
	}
	if len(p.FreightCharts) > 0 {
		data, err := json.Marshal(p.FreightCharts)
		if err == nil {
			annotations[kargoapi.AnnotationKeyEventFreightCharts] = string(data)
		}
	}
	if len(p.Applications) > 0 {
		data, err := json.Marshal(p.Applications)
		if err == nil {
			annotations[kargoapi.AnnotationKeyEventApplications] = string(data)
		}
	}
	if p.VerificationPending != nil {
		annotations[kargoapi.AnnotationKeyEventVerificationPending] =
			strconv.FormatBool(*p.VerificationPending)
	}
	if p.VerificationStartTime != nil {
		annotations[kargoapi.AnnotationKeyEventVerificationStartTime] =
			p.VerificationStartTime.Format(time.RFC3339)
	}
	if p.VerificationFinishTime != nil {
		annotations[kargoapi.AnnotationKeyEventVerificationFinishTime] =
			p.VerificationFinishTime.Format(time.RFC3339)
	}
	if p.AnalysisRunName != nil {
		annotations[kargoapi.AnnotationKeyEventAnalysisRunName] = *p.AnalysisRunName
	}
	return annotations
}

// ToCloudEvent converts the PromotionEvent to a CloudEvent.
func (p *PromotionEvent) ToCloudEvent(eventType kargoapi.EventType) cloudevents.Event {
	cloudEvent := cloudevents.NewEvent()
	cloudEvent.SetType(EventTypePrefix + string(eventType))
	cloudEvent.SetSource(Source(p.Project, "Promotion", p.PromotionName))
	// This ID will be ignored if used as a Kubernetes event. We can parse back the ID from the
	// Kubernetes event when parsing it.
	cloudEvent.SetID(uuid.NewString())
	// We control all this data, so serializing shouldn't fail. If this causes problems we can make
	// this function return an error.
	_ = cloudEvent.SetData(cloudevents.ApplicationJSON, p)
	cloudEvent.SetTime(time.Now())

	return cloudEvent
}

// UnmarshalAnnotations converts the given annotations into a PromotionEvent. This is used by the
// main event handler to convert the data into a normal CloudEvent, but is exposed for convenience.
func UnmarshalPromotionEventAnnotations(annotations map[string]string) (PromotionEvent, error) {
	evt := PromotionEvent{
		Project:             annotations[kargoapi.AnnotationKeyEventProject],
		PromotionName:       annotations[kargoapi.AnnotationKeyEventPromotionName],
		FreightName:         annotations[kargoapi.AnnotationKeyEventFreightName],
		StageName:           annotations[kargoapi.AnnotationKeyEventStageName],
		PromotionCreateTime: parseTime(annotations[kargoapi.AnnotationKeyEventPromotionCreateTime]),
	}

	if actor, ok := annotations[kargoapi.AnnotationKeyEventActor]; ok {
		evt.Actor = &actor
	}
	if freightCreateTime, ok := annotations[kargoapi.AnnotationKeyEventFreightCreateTime]; ok {
		t := parseTime(freightCreateTime)
		evt.FreightCreateTime = &t
	}
	if freightAlias, ok := annotations[kargoapi.AnnotationKeyEventFreightAlias]; ok {
		evt.FreightAlias = &freightAlias
	}
	if freightCommits, ok := annotations[kargoapi.AnnotationKeyEventFreightCommits]; ok {
		if err := json.Unmarshal([]byte(freightCommits), &evt.FreightCommits); err != nil {
			return evt, fmt.Errorf("failed to unmarshal freight commits: %w", err)
		}
	}
	if freightImages, ok := annotations[kargoapi.AnnotationKeyEventFreightImages]; ok {
		if err := json.Unmarshal([]byte(freightImages), &evt.FreightImages); err != nil {
			return evt, fmt.Errorf("failed to unmarshal freight images: %w", err)
		}
	}
	if freightCharts, ok := annotations[kargoapi.AnnotationKeyEventFreightCharts]; ok {
		if err := json.Unmarshal([]byte(freightCharts), &evt.FreightCharts); err != nil {
			return evt, fmt.Errorf("failed to unmarshal freight charts: %w", err)
		}
	}
	if applications, ok := annotations[kargoapi.AnnotationKeyEventApplications]; ok {
		var apps []types.NamespacedName
		if err := json.Unmarshal([]byte(applications), &apps); err != nil {
			return evt, fmt.Errorf("failed to unmarshal applications: %w", err)
		}
		evt.Applications = apps
	}
	if verificationPending, ok := annotations[kargoapi.AnnotationKeyEventVerificationPending]; ok {
		pending := verificationPending == "true"
		evt.VerificationPending = &pending
	}
	if verificationStartTime, ok := annotations[kargoapi.AnnotationKeyEventVerificationStartTime]; ok {
		t := parseTime(verificationStartTime)
		evt.VerificationStartTime = &t
	}
	if verificationFinishTime, ok := annotations[kargoapi.AnnotationKeyEventVerificationFinishTime]; ok {
		t := parseTime(verificationFinishTime)
		evt.VerificationFinishTime = &t
	}
	if analysisRunName, ok := annotations[kargoapi.AnnotationKeyEventAnalysisRunName]; ok {
		evt.AnalysisRunName = &analysisRunName
	}
	return evt, nil
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

func parseTime(value string) time.Time {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return t
}
