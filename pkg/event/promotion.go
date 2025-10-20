package event

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/argocd"
	"github.com/akuity/kargo/pkg/expressions"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// Promotion is a struct that contains common fields for promotion-related events.
type Promotion struct {
	Freight      *Freight               `json:"freight,omitempty"`
	Name         string                 `json:"name"`
	StageName    string                 `json:"stageName"`
	CreateTime   time.Time              `json:"createTime"`
	Applications []types.NamespacedName `json:"applications,omitempty"`
}

func (p Promotion) GetName() string {
	return p.Name
}

func (p Promotion) Kind() string {
	return "Promotion"
}

// PromotionSucceeded is event data related to a successful promotion.
type PromotionSucceeded struct {
	Common
	Promotion
	VerificationPending *bool `json:"verificationPending,omitempty"`
}

func (p *PromotionSucceeded) Type() kargoapi.EventType {
	return kargoapi.EventTypePromotionSucceeded
}

// NOTE(thomastaylor312): Most of the promotion events are identical, but that could easily change
// in the future if we want to decorate with more data. That is why all of these are separate types,
// even though they are identical in structure right now

// PromotionFailed is event data related to a failed promotion.
type PromotionFailed struct {
	Common
	Promotion
}

func (p *PromotionFailed) Type() kargoapi.EventType {
	return kargoapi.EventTypePromotionFailed
}

// PromotionErrored is event data related to an errored promotion.
type PromotionErrored struct {
	Common
	Promotion
}

func (p *PromotionErrored) Type() kargoapi.EventType {
	return kargoapi.EventTypePromotionErrored
}

// PromotionAborted is event data related to an aborted promotion.
type PromotionAborted struct {
	Common
	Promotion
}

func (p *PromotionAborted) Type() kargoapi.EventType {
	return kargoapi.EventTypePromotionAborted
}

// PromotionCreated is event data related to a created promotion.
type PromotionCreated struct {
	Common
	Promotion
}

func (p *PromotionCreated) Type() kargoapi.EventType {
	return kargoapi.EventTypePromotionCreated
}

// NewPromotionCommon creates a new `Promotion` and `Common` event from the given promotion and
// freight data. Since these fields are common to all events, this is exposed for convenience. The
// given actor will be used if it is not empty, but it will be overridden if the promotion has an
// actor annotation.
func NewPromotionCommon(message,
	actor string, promotion *kargoapi.Promotion,
	freight *kargoapi.Freight,
) (Common, Promotion) {
	return newCommonFromPromotion(message, actor, promotion), newPromotion(promotion, freight)
}

// NewPromotionSucceeded creates a new PromotionSucceeded event from the given promotion and freight
// data. The given actor will be used if it is not empty, but it will be overridden if the promotion
// has an actor annotation.
func NewPromotionSucceeded(
	message, actor string, promotion *kargoapi.Promotion, freight *kargoapi.Freight,
) *PromotionSucceeded {
	common, promo := NewPromotionCommon(message, actor, promotion, freight)
	return &PromotionSucceeded{
		Common:              common,
		Promotion:           promo,
		VerificationPending: nil,
	}
}

// NewPromotionFailed creates a new PromotionFailed event from the given promotion and freight data.
// The given actor will be used if it is not empty, but it will be overridden if the promotion has
// an actor annotation.
func NewPromotionFailed(
	message, actor string, promotion *kargoapi.Promotion, freight *kargoapi.Freight,
) *PromotionFailed {
	common, promo := NewPromotionCommon(message, actor, promotion, freight)
	return &PromotionFailed{
		Common:    common,
		Promotion: promo,
	}
}

// NewPromotionErrored creates a new PromotionErrored event from the given promotion and freight data.
// The given actor will be used if it is not empty, but it will be overridden if the promotion has
// an actor annotation.
func NewPromotionErrored(
	message, actor string, promotion *kargoapi.Promotion, freight *kargoapi.Freight,
) *PromotionErrored {
	common, promo := NewPromotionCommon(message, actor, promotion, freight)
	return &PromotionErrored{
		Common:    common,
		Promotion: promo,
	}
}

// NewPromotionAborted creates a new PromotionAborted event from the given promotion and freight data.
// The given actor will be used if it is not empty, but it will be overridden if the promotion has
// an actor annotation.
func NewPromotionAborted(
	message, actor string, promotion *kargoapi.Promotion, freight *kargoapi.Freight,
) *PromotionAborted {
	common, promo := NewPromotionCommon(message, actor, promotion, freight)
	return &PromotionAborted{
		Common:    common,
		Promotion: promo,
	}
}

// NewPromotionCreated creates a new PromotionCreated event from the given promotion and freight data.
// The given actor will be used if it is not empty, but it will be overridden if the promotion has
// an actor annotation.
func NewPromotionCreated(
	message, actor string, promotion *kargoapi.Promotion, freight *kargoapi.Freight,
) *PromotionCreated {
	common, promo := NewPromotionCommon(message, actor, promotion, freight)
	return &PromotionCreated{
		Common:    common,
		Promotion: promo,
	}
}

func (p *Promotion) MarshalAnnotationsTo(annotations map[string]string) {
	annotations[kargoapi.AnnotationKeyEventPromotionName] = p.Name
	annotations[kargoapi.AnnotationKeyEventStageName] = p.StageName
	annotations[kargoapi.AnnotationKeyEventPromotionCreateTime] = p.CreateTime.Format(time.RFC3339)
	if len(p.Applications) > 0 {
		if data, err := json.Marshal(p.Applications); err == nil {
			annotations[kargoapi.AnnotationKeyEventApplications] = string(data)
		}
	}
	if p.Freight != nil {
		p.Freight.MarshalAnnotationsTo(annotations)
	}
}

func (p *PromotionSucceeded) MarshalAnnotations() map[string]string {
	// Note that we skip message here, as it is not used in the annotations.
	annotations := map[string]string{}
	if p.VerificationPending != nil {
		annotations[kargoapi.AnnotationKeyEventVerificationPending] = strconv.FormatBool(*p.VerificationPending)
	}
	p.Common.MarshalAnnotationsTo(annotations)
	p.Promotion.MarshalAnnotationsTo(annotations)
	return annotations
}

func (p *PromotionFailed) MarshalAnnotations() map[string]string {
	// Note that we skip message here, as it is not used in the annotations.
	annotations := map[string]string{}
	p.Common.MarshalAnnotationsTo(annotations)
	p.Promotion.MarshalAnnotationsTo(annotations)
	return annotations
}

func (p *PromotionErrored) MarshalAnnotations() map[string]string {
	// Note that we skip message here, as it is not used in the annotations.
	annotations := map[string]string{}
	p.Common.MarshalAnnotationsTo(annotations)
	p.Promotion.MarshalAnnotationsTo(annotations)
	return annotations
}

func (p *PromotionAborted) MarshalAnnotations() map[string]string {
	// Note that we skip message here, as it is not used in the annotations.
	annotations := map[string]string{}
	p.Common.MarshalAnnotationsTo(annotations)
	p.Promotion.MarshalAnnotationsTo(annotations)
	return annotations
}

func (p *PromotionCreated) MarshalAnnotations() map[string]string {
	// Note that we skip message here, as it is not used in the annotations.
	annotations := map[string]string{}
	p.Common.MarshalAnnotationsTo(annotations)
	p.Promotion.MarshalAnnotationsTo(annotations)
	return annotations
}

// UnmarshalPromotionAnnotations populates the Promotion fields from the given kubernetes annotations.
func UnmarshalPromotionAnnotations(annotations map[string]string) (Promotion, error) {
	var freight *Freight
	f, err := UnmarshalFreightAnnotations(annotations)
	if err != nil {
		return Promotion{}, fmt.Errorf("failed to unmarshal freight annotations: %w", err)
	}
	// If the returned Freight object is not the zero type (i.e. has data), then we include it
	if !reflect.ValueOf(f).IsZero() {
		freight = &f
	}
	createTime, err := parseTime(annotations[kargoapi.AnnotationKeyEventPromotionCreateTime])
	if err != nil {
		return Promotion{}, fmt.Errorf("failed to parse promotion create time: %w", err)
	}
	evt := Promotion{
		Freight:    freight,
		Name:       annotations[kargoapi.AnnotationKeyEventPromotionName],
		StageName:  annotations[kargoapi.AnnotationKeyEventStageName],
		CreateTime: createTime,
	}

	if applications, ok := annotations[kargoapi.AnnotationKeyEventApplications]; ok {
		var apps []types.NamespacedName
		if err := json.Unmarshal([]byte(applications), &apps); err != nil {
			return evt, fmt.Errorf("failed to unmarshal applications: %w", err)
		}
		evt.Applications = apps
	}
	return evt, nil
}

// UnmarshalPromotionSucceededAnnotations converts the given annotations into a PromotionSucceeded. This is used by the
// main event handler to convert the data into a normal structured event, but is exposed for convenience.
func UnmarshalPromotionSucceededAnnotations(
	eventID string, annotations map[string]string,
) (*PromotionSucceeded, error) {
	common, err := UnmarshalCommonAnnotations(eventID, annotations)
	if err != nil {
		return nil, err
	}
	promotion, err := UnmarshalPromotionAnnotations(annotations)
	if err != nil {
		return nil, err
	}
	evt := PromotionSucceeded{
		Common:    common,
		Promotion: promotion,
	}

	if verificationPending, ok := annotations[kargoapi.AnnotationKeyEventVerificationPending]; ok {
		pending := verificationPending == "true"
		evt.VerificationPending = &pending
	}
	return &evt, nil
}

// UnmarshalPromotionFailedAnnotations converts the given annotations into a PromotionFailed. This is used by the
// main event handler to convert the data into a normal structured event, but is exposed for convenience.
func UnmarshalPromotionFailedAnnotations(
	eventID string, annotations map[string]string,
) (*PromotionFailed, error) {
	common, err := UnmarshalCommonAnnotations(eventID, annotations)
	if err != nil {
		return nil, err
	}
	promotion, err := UnmarshalPromotionAnnotations(annotations)
	if err != nil {
		return nil, err
	}
	evt := PromotionFailed{
		Common:    common,
		Promotion: promotion,
	}
	return &evt, nil
}

// UnmarshalPromotionErroredAnnotations converts the given annotations into a PromotionErrored. This is used by the
// main event handler to convert the data into a normal structured event, but is exposed for convenience.
func UnmarshalPromotionErroredAnnotations(
	eventID string, annotations map[string]string,
) (*PromotionErrored, error) {
	common, err := UnmarshalCommonAnnotations(eventID, annotations)
	if err != nil {
		return nil, err
	}
	promotion, err := UnmarshalPromotionAnnotations(annotations)
	if err != nil {
		return nil, err
	}
	evt := PromotionErrored{
		Common:    common,
		Promotion: promotion,
	}
	return &evt, nil
}

// UnmarshalPromotionAbortedAnnotations converts the given annotations into a PromotionAborted. This is used by the
// main event handler to convert the data into a normal structured event, but is exposed for convenience.
func UnmarshalPromotionAbortedAnnotations(
	eventID string, annotations map[string]string,
) (*PromotionAborted, error) {
	common, err := UnmarshalCommonAnnotations(eventID, annotations)
	if err != nil {
		return nil, err
	}
	promotion, err := UnmarshalPromotionAnnotations(annotations)
	if err != nil {
		return nil, err
	}
	evt := PromotionAborted{
		Common:    common,
		Promotion: promotion,
	}
	return &evt, nil
}

// UnmarshalPromotionCreatedAnnotations converts the given annotations into a PromotionCreated. This is used by the
// main event handler to convert the data into a normal structured event, but is exposed for convenience.
func UnmarshalPromotionCreatedAnnotations(
	eventID string, annotations map[string]string,
) (*PromotionCreated, error) {
	common, err := UnmarshalCommonAnnotations(eventID, annotations)
	if err != nil {
		return nil, err
	}
	promotion, err := UnmarshalPromotionAnnotations(annotations)
	if err != nil {
		return nil, err
	}
	evt := PromotionCreated{
		Common:    common,
		Promotion: promotion,
	}
	return &evt, nil
}

func newPromotion(
	promotion *kargoapi.Promotion,
	freight *kargoapi.Freight,
) Promotion {
	evt := Promotion{
		Name:       promotion.GetName(),
		StageName:  promotion.Spec.Stage,
		CreateTime: promotion.GetCreationTimestamp().Time,
	}

	if freight != nil {
		evt.Freight = ptr.To(newFreight(freight, promotion.Spec.Stage))
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
		if step.Config == nil {
			continue
		}

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
