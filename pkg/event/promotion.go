package event

import (
	"maps"
	"strconv"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/expressions"
)

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

func (p *PromotionSucceeded) MarshalAnnotations() map[string]string {
	// Note that we skip message here, as it is not used in the annotations.
	annotations := map[string]string{}
	if p.VerificationPending != nil {
		annotations[kargoapi.AnnotationKeyEventVerificationPending] =
			strconv.FormatBool(*p.VerificationPending)
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

// UnmarshalPromotionSucceededAnnotations converts the given annotations into a PromotionSucceeded. This is used by the
// main event handler to convert the data into a normal CloudEvent, but is exposed for convenience.
func UnmarshalPromotionSucceededAnnotations(annotations map[string]string) (*PromotionSucceeded, error) {
	common, err := UnmarshalCommonAnnotations(annotations)
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
// main event handler to convert the data into a normal CloudEvent, but is exposed for convenience.
func UnmarshalPromotionFailedAnnotations(annotations map[string]string) (*PromotionFailed, error) {
	common, err := UnmarshalCommonAnnotations(annotations)
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
// main event handler to convert the data into a normal CloudEvent, but is exposed for convenience.
func UnmarshalPromotionErroredAnnotations(annotations map[string]string) (*PromotionErrored, error) {
	common, err := UnmarshalCommonAnnotations(annotations)
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
// main event handler to convert the data into a normal CloudEvent, but is exposed for convenience.
func UnmarshalPromotionAbortedAnnotations(annotations map[string]string) (*PromotionAborted, error) {
	common, err := UnmarshalCommonAnnotations(annotations)
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
// main event handler to convert the data into a normal CloudEvent, but is exposed for convenience.
func UnmarshalPromotionCreatedAnnotations(annotations map[string]string) (*PromotionCreated, error) {
	common, err := UnmarshalCommonAnnotations(annotations)
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
