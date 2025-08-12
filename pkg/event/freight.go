package event

import (
	"encoding/json"
	"fmt"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// FreightEvent represents the data available for various events related to freights.
type FreightEvent struct {
	Project                string               `json:"project"`
	Name                   string               `json:"name"`
	StageName              string               `json:"stageName"`
	Message                string               `json:"message"`
	Actor                  *string              `json:"actor,omitempty"`
	FreightCreateTime      time.Time            `json:"freightCreateTime"`
	FreightAlias           *string              `json:"freightAlias,omitempty"`
	FreightCommits         []kargoapi.GitCommit `json:"freightCommits,omitempty"`
	FreightImages          []kargoapi.Image     `json:"freightImages,omitempty"`
	FreightCharts          []kargoapi.Chart     `json:"freightCharts,omitempty"`
	VerificationStartTime  *time.Time           `json:"verificationStartTime,omitempty"`
	VerificationFinishTime *time.Time           `json:"verificationFinishTime,omitempty"`
	AnalysisRunName        *string              `json:"analysisRunName,omitempty"`

	// AnalysisTriggeredByPromotion is the name of the promotion that triggered the analysis run.
	AnalysisTriggeredByPromotion *string `json:"analysisTriggeredByPromotion,omitempty"`
}

// NewFreightEvent creates a new FreightEvent from the given parameters. If the freight is nil, it
// returns an empty FreightEvent.
func NewFreightEvent(actor string, freight *kargoapi.Freight, stageName, message string) FreightEvent {
	if freight == nil {
		return FreightEvent{}
	}
	evt := FreightEvent{
		Project:           freight.Namespace,
		FreightCreateTime: freight.CreationTimestamp.Time,
		Name:              freight.Name,
		StageName:         stageName,
		Message:           message,
	}
	if actor != "" {
		evt.Actor = &actor
	}
	if freight.Alias != "" {
		evt.FreightAlias = &freight.Alias
	}
	if len(freight.Commits) > 0 {
		evt.FreightCommits = freight.Commits
	}
	if len(freight.Images) > 0 {
		evt.FreightImages = freight.Images
	}
	if len(freight.Charts) > 0 {
		evt.FreightCharts = freight.Charts
	}
	return evt
}

func (f *FreightEvent) MarshalAnnotations() map[string]string {
	// Note that we skip message here, as it is not used in the annotations.
	annotations := map[string]string{
		kargoapi.AnnotationKeyEventProject:           f.Project,
		kargoapi.AnnotationKeyEventFreightName:       f.Name,
		kargoapi.AnnotationKeyEventStageName:         f.StageName,
		kargoapi.AnnotationKeyEventFreightCreateTime: f.FreightCreateTime.Format(time.RFC3339),
	}

	if f.Actor != nil {
		annotations[kargoapi.AnnotationKeyEventActor] = *f.Actor
	}
	if f.FreightAlias != nil {
		annotations[kargoapi.AnnotationKeyEventFreightAlias] = *f.FreightAlias
	}
	if len(f.FreightCommits) > 0 {
		data, err := json.Marshal(f.FreightCommits)
		if err == nil {
			annotations[kargoapi.AnnotationKeyEventFreightCommits] = string(data)
		}
	}
	if len(f.FreightImages) > 0 {
		data, err := json.Marshal(f.FreightImages)
		if err == nil {
			annotations[kargoapi.AnnotationKeyEventFreightImages] = string(data)
		}
	}
	if len(f.FreightCharts) > 0 {
		data, err := json.Marshal(f.FreightCharts)
		if err == nil {
			annotations[kargoapi.AnnotationKeyEventFreightCharts] = string(data)
		}
	}
	if f.VerificationStartTime != nil {
		annotations[kargoapi.AnnotationKeyEventVerificationStartTime] =
			f.VerificationStartTime.Format(time.RFC3339)
	}
	if f.VerificationFinishTime != nil {
		annotations[kargoapi.AnnotationKeyEventVerificationFinishTime] =
			f.VerificationFinishTime.Format(time.RFC3339)
	}
	if f.AnalysisRunName != nil {
		annotations[kargoapi.AnnotationKeyEventAnalysisRunName] = *f.AnalysisRunName
	}
	// For compatibility with the current event types in Kubernetes, we use the promotion name annotation for this field
	if f.AnalysisTriggeredByPromotion != nil {
		annotations[kargoapi.AnnotationKeyEventPromotionName] = *f.AnalysisTriggeredByPromotion
	}
	return annotations
}

// ToCloudEvent converts the FreightEvent to a CloudEvent.
func (f *FreightEvent) ToCloudEvent(eventType kargoapi.EventType) cloudevents.Event {
	cloudEvent := cloudevents.NewEvent()
	cloudEvent.SetType(EventTypePrefix + string(eventType))
	cloudEvent.SetSource(Source(f.Project, "Freight", f.Name))
	// This ID will be ignored if used as a Kubernetes event. We can parse back the ID from the
	// Kubernetes event when parsing it.
	cloudEvent.SetID(uuid.NewString())
	// We control all this data, so serializing shouldn't fail. If this causes problems we can make
	// this function return an error.
	_ = cloudEvent.SetData(cloudevents.ApplicationJSON, f)
	cloudEvent.SetTime(time.Now())

	return cloudEvent
}

// UnmarshalAnnotations converts the given annotations into a FreightEvent. This is used by the
// main event handler to convert the data into a normal CloudEvent, but is exposed for convenience.
func UnmarshalFreightEventAnnotations(annotations map[string]string) (FreightEvent, error) {
	evt := FreightEvent{
		Project:           annotations[kargoapi.AnnotationKeyEventProject],
		Name:              annotations[kargoapi.AnnotationKeyEventFreightName],
		StageName:         annotations[kargoapi.AnnotationKeyEventStageName],
		FreightCreateTime: parseTime(annotations[kargoapi.AnnotationKeyEventFreightCreateTime]),
	}

	if actor, ok := annotations[kargoapi.AnnotationKeyEventActor]; ok {
		evt.Actor = &actor
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
	if analysisTriggeredByPromotion, ok := annotations[kargoapi.AnnotationKeyEventPromotionName]; ok {
		evt.AnalysisTriggeredByPromotion = &analysisTriggeredByPromotion
	}
	return evt, nil
}
