package event

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"sort"
	"strings"
	"time"

	cloudevent "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/argocd"
	"github.com/akuity/kargo/pkg/expressions"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// The reverse DNS event type prefix used for all Kargo CloudEvents.
const EventTypePrefix = "io.akuity.kargo.event."

// Source is a utility function that formats the source of an event (from a Kubernetes object). It combines
// the namespace, kind, and name of the event into a single string for use in a CloudEvent
func Source(namespace, kind, name string) string {
	return fmt.Sprintf("%s/%s/%s", namespace, kind, name)
}

// Meta is an interface for our built in event types that allows for easier conversion to
// CloudEvent types
type Meta interface {
	// Type returns the event type for the struct
	Type() kargoapi.EventType
	// Kind returns the object kind the event is related to (e.g. "Promotion", "Freight", etc.).
	// This is used for constructing a valid object reference for the event source
	Kind() string
	// GetProject returns the project associated with the event
	GetProject() string
	// GetName returns the name of the object associated with the event
	GetName() string
}

// Message is an interface for setting and getting the message of any built in event
type Message interface {
	// GetMessage returns the message of the event
	GetMessage() string
	// SetMessage sets the message of the event
	SetMessage(string)
}

// AnnotationMarshaler is an interface for any type that can marshal out to our custom annotation
// types
type AnnotationMarshaler interface {
	MarshalAnnotations() map[string]string
}

// ToCloudEvent converts the PromotionCreated to a CloudEvent. If nil is passed, it returns an empty
// CloudEvent.
func ToCloudEvent(p Meta) cloudevent.Event {
	if p == nil {
		return cloudevent.NewEvent()
	}
	cloudEvent := cloudevent.NewEvent()
	cloudEvent.SetType(EventTypePrefix + string(p.Type()))
	cloudEvent.SetSource(Source(p.GetProject(), p.Kind(), p.GetName()))
	// This ID will be ignored if used as a Kubernetes event. We can parse back the ID from the
	// Kubernetes event when parsing it.
	cloudEvent.SetID(uuid.NewString())
	// We control all this data, so serializing shouldn't fail. If this causes problems we can make
	// this function return an error.
	_ = cloudEvent.SetData(cloudevent.ApplicationJSON, p)
	cloudEvent.SetTime(time.Now())

	return cloudEvent
}

// Common is a struct that contains fields common to all events.
type Common struct {
	Project string  `json:"project"`
	Actor   *string `json:"actor,omitempty"`
	Message string  `json:"message"`
}

func (c Common) GetProject() string {
	return c.Project
}

func (c Common) GetMessage() string {
	return c.Message
}

func (c *Common) SetMessage(message string) {
	c.Message = message
}

// Promotion is a struct that contains common fields for promotion-related events.
type Promotion struct {
	*Freight
	Name         string                 `json:"promotionName"`
	StageName    string                 `json:"stageName"`
	CreateTime   time.Time              `json:"promotionCreateTime"`
	Applications []types.NamespacedName `json:"applications,omitempty"`
}

func (p Promotion) GetName() string {
	return p.Name
}

func (p Promotion) Kind() string {
	return "Promotion"
}

// Freight is a struct that contains common fields for freight-related events.
type Freight struct {
	Name       string               `json:"freightName"`
	StageName  string               `json:"stageName"`
	CreateTime time.Time            `json:"freightCreateTime,omitempty"`
	Alias      *string              `json:"freightAlias,omitempty"`
	Commits    []kargoapi.GitCommit `json:"freightCommits,omitempty"`
	Images     []kargoapi.Image     `json:"freightImages,omitempty"`
	Charts     []kargoapi.Chart     `json:"freightCharts,omitempty"`
}

func (f Freight) GetName() string {
	return f.Name
}

func (f Freight) Kind() string {
	return "Freight"
}

// UnmarshalCommonAnnotations populates the Common fields from the given kubernetes annotations.
func UnmarshalCommonAnnotations(annotations map[string]string) (Common, error) {
	evt := Common{
		Project: annotations[kargoapi.AnnotationKeyEventProject],
	}
	if actor, ok := annotations[kargoapi.AnnotationKeyEventActor]; ok {
		evt.Actor = &actor
	}
	return evt, nil
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

// UnmarshalFreightAnnotations populates the Freight fields from the given kubernetes annotations.
func UnmarshalFreightAnnotations(annotations map[string]string) (Freight, error) {
	evt := Freight{}

	// Only populate fields if freight name is present (indicating freight data exists)
	if name, ok := annotations[kargoapi.AnnotationKeyEventFreightName]; ok && name != "" {
		evt.Name = name
		evt.StageName = annotations[kargoapi.AnnotationKeyEventStageName]

		if createTimeStr, ok := annotations[kargoapi.AnnotationKeyEventFreightCreateTime]; ok && createTimeStr != "" {
			createTime, err := parseTime(createTimeStr)
			if err != nil {
				return Freight{}, fmt.Errorf("failed to parse freight create time: %w", err)
			}
			evt.CreateTime = createTime
		}
	}

	if alias, ok := annotations[kargoapi.AnnotationKeyEventFreightAlias]; ok {
		evt.Alias = &alias
	}
	if commits, ok := annotations[kargoapi.AnnotationKeyEventFreightCommits]; ok {
		var cs []kargoapi.GitCommit
		if err := json.Unmarshal([]byte(commits), &cs); err != nil {
			return evt, fmt.Errorf("failed to unmarshal freight commits: %w", err)
		}
		evt.Commits = cs
	}
	if images, ok := annotations[kargoapi.AnnotationKeyEventFreightImages]; ok {
		var is []kargoapi.Image
		if err := json.Unmarshal([]byte(images), &is); err != nil {
			return evt, fmt.Errorf("failed to unmarshal freight images: %w", err)
		}
		evt.Images = is
	}
	if charts, ok := annotations[kargoapi.AnnotationKeyEventFreightCharts]; ok {
		var cs []kargoapi.Chart
		if err := json.Unmarshal([]byte(charts), &cs); err != nil {
			return evt, fmt.Errorf("failed to unmarshal freight charts: %w", err)
		}
		evt.Charts = cs
	}
	return evt, nil
}

// NOTE(thomastaylor312): These marshal annotation functions all take pointers to an annotations map
// they should fill in because they are all meant for use as nested fields. So the main event type
// will have a `MarshalAnnotations` function that calls these with the appropriate map to fill in

func (c *Common) MarshalAnnotationsTo(annotations map[string]string) {
	annotations[kargoapi.AnnotationKeyEventProject] = c.Project
	if c.Actor != nil {
		annotations[kargoapi.AnnotationKeyEventActor] = *c.Actor
	}
	// Messages is skipped as it is passed to the k8s event directly
}

func (f *Freight) MarshalAnnotationsTo(annotations map[string]string) {
	annotations[kargoapi.AnnotationKeyEventFreightName] = f.Name
	annotations[kargoapi.AnnotationKeyEventFreightCreateTime] = f.CreateTime.Format(time.RFC3339)
	annotations[kargoapi.AnnotationKeyEventStageName] = f.StageName
	if f.Alias != nil {
		annotations[kargoapi.AnnotationKeyEventFreightAlias] = *f.Alias
	}
	if len(f.Commits) > 0 {
		if data, err := json.Marshal(f.Commits); err == nil {
			annotations[kargoapi.AnnotationKeyEventFreightCommits] = string(data)
		}
	}
	if len(f.Images) > 0 {
		if data, err := json.Marshal(f.Images); err == nil {
			annotations[kargoapi.AnnotationKeyEventFreightImages] = string(data)
		}
	}
	if len(f.Charts) > 0 {
		if data, err := json.Marshal(f.Charts); err == nil {
			annotations[kargoapi.AnnotationKeyEventFreightCharts] = string(data)
		}
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

func newCommonFromPromotion(message, actor string, promotion *kargoapi.Promotion) Common {
	if promotion == nil {
		return Common{}
	}
	evt := Common{
		Project: promotion.Namespace,
		Message: message,
	}
	if actor != "" {
		evt.Actor = &actor
	}
	// All Promotion-related events are emitted after the promotion was created.
	// Therefore, if the promotion knows who triggered it, set them as an actor.
	if promoteActor, ok := promotion.Annotations[kargoapi.AnnotationKeyCreateActor]; ok {
		evt.Actor = &promoteActor
	}
	return evt
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

func newCommonFromFreight(message,
	actor string, freight *kargoapi.Freight,
) Common {
	evt := Common{
		Message: message,
	}
	if freight != nil {
		evt.Project = freight.GetNamespace()
	}
	if actor != "" {
		evt.Actor = &actor
	}
	return evt
}

func newFreight(freight *kargoapi.Freight, stageName string) Freight {
	if freight == nil {
		return Freight{}
	}
	evt := Freight{
		CreateTime: freight.CreationTimestamp.Time,
		Name:       freight.Name,
		StageName:  stageName,
	}
	if freight.Alias != "" {
		evt.Alias = &freight.Alias
	}
	if len(freight.Commits) > 0 {
		evt.Commits = freight.Commits
	}
	if len(freight.Images) > 0 {
		evt.Images = freight.Images
	}
	if len(freight.Charts) > 0 {
		evt.Charts = freight.Charts
	}
	return evt
}

func parseTime(value string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}
