package event

import (
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// Freight is a struct that contains common fields for freight-related events.
type Freight struct {
	Name       string                       `json:"name"`
	StageName  string                       `json:"stageName"`
	CreateTime time.Time                    `json:"createTime"`
	Alias      *string                      `json:"alias,omitempty"`
	Commits    []kargoapi.GitCommit         `json:"commits,omitempty"`
	Images     []kargoapi.Image             `json:"images,omitempty"`
	Charts     []kargoapi.Chart             `json:"charts,omitempty"`
	Artifacts  []kargoapi.ArtifactReference `json:"artifacts,omitempty"`
}

func (f Freight) GetName() string {
	return f.Name
}

func (f Freight) Kind() string {
	return "Freight"
}

// FreightVerificationEventMeta is an interface for metadata associated with freight verification
// events. It is an extension of the Meta interface with a few verification-specific methods
type FreightVerificationEventMeta interface {
	Meta
	GetTriggeredByPromotion() *string
	SetTriggeredByPromotion(promotionName *string)
}

// FreightVerification is a struct that contains common fields for a verification event
type FreightVerification struct {
	StartTime       *time.Time `json:"verificationStartTime,omitempty"`
	FinishTime      *time.Time `json:"verificationFinishTime,omitempty"`
	AnalysisRunName *string    `json:"analysisRunName,omitempty"`
	// AnalysisTriggeredByPromotion is the name of the promotion that triggered the analysis run.
	AnalysisTriggeredByPromotion *string `json:"analysisTriggeredByPromotion,omitempty"`
}

func (f *FreightVerification) GetTriggeredByPromotion() *string {
	return f.AnalysisTriggeredByPromotion
}

func (f *FreightVerification) SetTriggeredByPromotion(promotionName *string) {
	f.AnalysisTriggeredByPromotion = promotionName
}

// NewFreightVerification creates a new `FreightVerification` struct from a `VerificationInfo`. This
// is mostly a convenience method if you're constructing an event yourself
func NewFreightVerification(vi *kargoapi.VerificationInfo) FreightVerification {
	evt := FreightVerification{}
	if vi == nil {
		return evt
	}
	if vi.StartTime != nil {
		evt.StartTime = &vi.StartTime.Time
	}
	if vi.FinishTime != nil {
		evt.FinishTime = &vi.FinishTime.Time
	}
	if vi.HasAnalysisRun() {
		evt.AnalysisRunName = &vi.AnalysisRun.Name
	}
	return evt
}

func (f *FreightVerification) MarshalAnnotationsTo(annotations map[string]string) {
	if f.StartTime != nil {
		annotations[kargoapi.AnnotationKeyEventVerificationStartTime] = f.StartTime.Format(time.RFC3339)
	}
	if f.FinishTime != nil {
		annotations[kargoapi.AnnotationKeyEventVerificationFinishTime] = f.FinishTime.Format(time.RFC3339)
	}
	if f.AnalysisRunName != nil {
		annotations[kargoapi.AnnotationKeyEventAnalysisRunName] = *f.AnalysisRunName
	}
	// For compatibility with the current event types in Kubernetes, we use the promotion name annotation for this field
	if f.AnalysisTriggeredByPromotion != nil {
		annotations[kargoapi.AnnotationKeyEventPromotionName] = *f.AnalysisTriggeredByPromotion
	}
}

func UnmarshalFreightVerificationAnnotations(annotations map[string]string) (FreightVerification, error) {
	var f FreightVerification

	if v, ok := annotations[kargoapi.AnnotationKeyEventVerificationStartTime]; ok {
		startTime, err := parseTime(v)
		if err != nil {
			return FreightVerification{}, fmt.Errorf(
				"failed to parse verification start time: %w", err)
		}
		f.StartTime = ptr.To(startTime)
	}
	if v, ok := annotations[kargoapi.AnnotationKeyEventVerificationFinishTime]; ok {
		finishTime, err := parseTime(v)
		if err != nil {
			return FreightVerification{}, fmt.Errorf(
				"failed to parse verification finish time: %w", err)
		}
		f.FinishTime = ptr.To(finishTime)
	}
	if v, ok := annotations[kargoapi.AnnotationKeyEventAnalysisRunName]; ok {
		f.AnalysisRunName = &v
	}
	if v, ok := annotations[kargoapi.AnnotationKeyEventPromotionName]; ok {
		f.AnalysisTriggeredByPromotion = &v
	}

	return f, nil
}

// NOTE(thomastaylor312): Most of the promotion events are identical, but that could easily change
// in the future if we want to decorate with more data. That is why all of these are separate types,
// even though they are identical in structure right now

// FreightVerificationSucceeded is an event fired when a freight verification succeeds.
type FreightVerificationSucceeded struct {
	Common
	Freight
	FreightVerification
}

func (f *FreightVerificationSucceeded) Type() kargoapi.EventType {
	return kargoapi.EventTypeFreightVerificationSucceeded
}

// FreightVerificationFailed is an event fired when a freight verification fails.
type FreightVerificationFailed struct {
	Common
	Freight
	FreightVerification
}

func (f *FreightVerificationFailed) Type() kargoapi.EventType {
	return kargoapi.EventTypeFreightVerificationFailed
}

// FreightVerificationErrored is an event fired when a freight verification errors.
type FreightVerificationErrored struct {
	Common
	Freight
	FreightVerification
}

func (f *FreightVerificationErrored) Type() kargoapi.EventType {
	return kargoapi.EventTypeFreightVerificationErrored
}

// FreightVerificationAborted is an event fired when a freight verification is aborted.
type FreightVerificationAborted struct {
	Common
	Freight
	FreightVerification
}

func (f *FreightVerificationAborted) Type() kargoapi.EventType {
	return kargoapi.EventTypeFreightVerificationAborted
}

// FreightVerificationInconclusive is an event fired when a freight verification is inconclusive.
type FreightVerificationInconclusive struct {
	Common
	Freight
	FreightVerification
}

func (f *FreightVerificationInconclusive) Type() kargoapi.EventType {
	return kargoapi.EventTypeFreightVerificationInconclusive
}

// FreightVerificationUnknown is an event fired when a freight verification is unknown.
type FreightVerificationUnknown struct {
	Common
	Freight
	FreightVerification
}

func (f *FreightVerificationUnknown) Type() kargoapi.EventType {
	return kargoapi.EventTypeFreightVerificationUnknown
}

type FreightApproved struct {
	Common
	Freight
}

func (f *FreightApproved) Type() kargoapi.EventType {
	return kargoapi.EventTypeFreightApproved
}

// NewFreightCommon creates a new `Freight` and `Common` event from the given freight data. Since
// these fields are common to all events, this is exposed for convenience.
func NewFreightCommon(message,
	actor, stageName string, freight *kargoapi.Freight,
) (Common, Freight) {
	return newCommonFromFreight(message, actor, freight), newFreight(freight, stageName)
}

// This assembles the common parts for a freight verification event. The message is automatically
// derived from the message on the `VerificationInfo`
func newFreightVerificationParts(actor, stageName string, freight *kargoapi.Freight,
	verification *kargoapi.VerificationInfo,
) (Common, Freight, FreightVerification) {
	freightVerification := NewFreightVerification(verification)
	freightEvent := newFreight(freight, stageName)
	common := newCommonFromFreight(verification.Message, actor, freight)

	return common, freightEvent, freightVerification
}

// NewFreightVerificationSucceeded creates a new `FreightVerificationSucceeded` event.
func NewFreightVerificationSucceeded(actor, stageName string, freight *kargoapi.Freight,
	verification *kargoapi.VerificationInfo,
) *FreightVerificationSucceeded {
	common, freightEvent, freightVerification := newFreightVerificationParts(actor, stageName, freight, verification)
	return &FreightVerificationSucceeded{
		Common:              common,
		Freight:             freightEvent,
		FreightVerification: freightVerification,
	}
}

// NewFreightVerificationFailed creates a new `FreightVerificationFailed` event.
func NewFreightVerificationFailed(actor, stageName string, freight *kargoapi.Freight,
	verification *kargoapi.VerificationInfo,
) *FreightVerificationFailed {
	common, freightEvent, freightVerification := newFreightVerificationParts(actor, stageName, freight, verification)
	return &FreightVerificationFailed{
		Common:              common,
		Freight:             freightEvent,
		FreightVerification: freightVerification,
	}
}

// NewFreightVerificationAborted creates a new `FreightVerificationAborted` event.
func NewFreightVerificationAborted(actor, stageName string, freight *kargoapi.Freight,
	verification *kargoapi.VerificationInfo,
) *FreightVerificationAborted {
	common, freightEvent, freightVerification := newFreightVerificationParts(actor, stageName, freight, verification)
	return &FreightVerificationAborted{
		Common:              common,
		Freight:             freightEvent,
		FreightVerification: freightVerification,
	}
}

// NewFreightVerificationUnknown creates a new `FreightVerificationUnknown` event.
func NewFreightVerificationUnknown(actor, stageName string, freight *kargoapi.Freight,
	verification *kargoapi.VerificationInfo,
) *FreightVerificationUnknown {
	common, freightEvent, freightVerification := newFreightVerificationParts(actor, stageName, freight, verification)
	return &FreightVerificationUnknown{
		Common:              common,
		Freight:             freightEvent,
		FreightVerification: freightVerification,
	}
}

// NewFreightVerificationErrored creates a new `FreightVerificationErrored` event.
func NewFreightVerificationErrored(actor, stageName string, freight *kargoapi.Freight,
	verification *kargoapi.VerificationInfo,
) *FreightVerificationErrored {
	common, freightEvent, freightVerification := newFreightVerificationParts(actor, stageName, freight, verification)
	return &FreightVerificationErrored{
		Common:              common,
		Freight:             freightEvent,
		FreightVerification: freightVerification,
	}
}

// NewFreightVerificationInconclusive creates a new `FreightVerificationInconclusive` event.
func NewFreightVerificationInconclusive(actor, stageName string, freight *kargoapi.Freight,
	verification *kargoapi.VerificationInfo,
) *FreightVerificationInconclusive {
	common, freightEvent, freightVerification := newFreightVerificationParts(actor, stageName, freight, verification)
	return &FreightVerificationInconclusive{
		Common:              common,
		Freight:             freightEvent,
		FreightVerification: freightVerification,
	}
}

// NewFreightApproved creates a new `FreightApproved` event.
func NewFreightApproved(message, actor, stageName string, freight *kargoapi.Freight,
) *FreightApproved {
	common, freightEvent := NewFreightCommon(message, actor, stageName, freight)
	return &FreightApproved{
		Common:  common,
		Freight: freightEvent,
	}
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
	if len(f.Artifacts) > 0 {
		if data, err := json.Marshal(f.Artifacts); err == nil {
			annotations[kargoapi.AnnotationKeyEventFreightArtifacts] = string(data)
		}
	}
}

func (f *FreightVerificationSucceeded) MarshalAnnotations() map[string]string {
	annotations := map[string]string{}
	f.Common.MarshalAnnotationsTo(annotations)
	f.Freight.MarshalAnnotationsTo(annotations)
	f.FreightVerification.MarshalAnnotationsTo(annotations)
	return annotations
}

func (f *FreightVerificationFailed) MarshalAnnotations() map[string]string {
	annotations := map[string]string{}
	f.Common.MarshalAnnotationsTo(annotations)
	f.Freight.MarshalAnnotationsTo(annotations)
	f.FreightVerification.MarshalAnnotationsTo(annotations)
	return annotations
}

func (f *FreightVerificationUnknown) MarshalAnnotations() map[string]string {
	annotations := map[string]string{}
	f.Common.MarshalAnnotationsTo(annotations)
	f.Freight.MarshalAnnotationsTo(annotations)
	f.FreightVerification.MarshalAnnotationsTo(annotations)
	return annotations
}

func (f *FreightVerificationErrored) MarshalAnnotations() map[string]string {
	annotations := map[string]string{}
	f.Common.MarshalAnnotationsTo(annotations)
	f.Freight.MarshalAnnotationsTo(annotations)
	f.FreightVerification.MarshalAnnotationsTo(annotations)
	return annotations
}

func (f *FreightVerificationAborted) MarshalAnnotations() map[string]string {
	annotations := map[string]string{}
	f.Common.MarshalAnnotationsTo(annotations)
	f.Freight.MarshalAnnotationsTo(annotations)
	f.FreightVerification.MarshalAnnotationsTo(annotations)
	return annotations
}

func (f *FreightVerificationInconclusive) MarshalAnnotations() map[string]string {
	annotations := map[string]string{}
	f.Common.MarshalAnnotationsTo(annotations)
	f.Freight.MarshalAnnotationsTo(annotations)
	f.FreightVerification.MarshalAnnotationsTo(annotations)
	return annotations
}

func (f *FreightApproved) MarshalAnnotations() map[string]string {
	annotations := map[string]string{}
	f.Common.MarshalAnnotationsTo(annotations)
	f.Freight.MarshalAnnotationsTo(annotations)
	return annotations
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
	if artifacts, ok := annotations[kargoapi.AnnotationKeyEventFreightArtifacts]; ok {
		var ars []kargoapi.ArtifactReference
		if err := json.Unmarshal([]byte(artifacts), &ars); err != nil {
			return evt, fmt.Errorf("failed to unmarshal freight artifacts: %w", err)
		}
		evt.Artifacts = ars
	}
	return evt, nil
}

// UnmarshalFreightVerificationSucceededAnnotations converts the given annotations into a
// FreightVerificationSucceeded event. This is used by the main event handler to convert the data
// into a normal structured event, but is exposed for convenience.
func UnmarshalFreightVerificationSucceededAnnotations(
	eventID string,
	annotations map[string]string,
) (*FreightVerificationSucceeded, error) {
	freight, err := UnmarshalFreightAnnotations(annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal freight annotations: %w", err)
	}
	common, err := UnmarshalCommonAnnotations(eventID, annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal common annotations: %w", err)
	}
	verification, err := UnmarshalFreightVerificationAnnotations(annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal freight verification annotations: %w", err)
	}
	evt := FreightVerificationSucceeded{
		Common:              common,
		Freight:             freight,
		FreightVerification: verification,
	}
	return &evt, nil
}

// UnmarshalFreightVerificationFailedAnnotations converts the given annotations into a
// FreightVerificationFailed event. This is used by the main event handler to convert the data
// into a normal structured event, but is exposed for convenience.
func UnmarshalFreightVerificationFailedAnnotations(
	eventID string,
	annotations map[string]string,
) (*FreightVerificationFailed, error) {
	freight, err := UnmarshalFreightAnnotations(annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal freight annotations: %w", err)
	}
	common, err := UnmarshalCommonAnnotations(eventID, annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal common annotations: %w", err)
	}
	verification, err := UnmarshalFreightVerificationAnnotations(annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal freight verification annotations: %w", err)
	}
	evt := FreightVerificationFailed{
		Common:              common,
		Freight:             freight,
		FreightVerification: verification,
	}
	return &evt, nil
}

// UnmarshalFreightVerificationInconclusiveAnnotations converts the given annotations into a
// FreightVerificationInconclusive event. This is used by the main event handler to convert the data
// into a normal structured event, but is exposed for convenience.
func UnmarshalFreightVerificationInconclusiveAnnotations(
	eventID string,
	annotations map[string]string,
) (*FreightVerificationInconclusive, error) {
	freight, err := UnmarshalFreightAnnotations(annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal freight annotations: %w", err)
	}
	common, err := UnmarshalCommonAnnotations(eventID, annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal common annotations: %w", err)
	}
	verification, err := UnmarshalFreightVerificationAnnotations(annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal freight verification annotations: %w", err)
	}
	evt := FreightVerificationInconclusive{
		Common:              common,
		Freight:             freight,
		FreightVerification: verification,
	}
	return &evt, nil
}

// UnmarshalFreightVerificationErroredAnnotations converts the given annotations into a
// FreightVerificationErrored event. This is used by the main event handler to convert the data
// into a normal structured event, but is exposed for convenience.
func UnmarshalFreightVerificationErroredAnnotations(
	eventID string,
	annotations map[string]string,
) (*FreightVerificationErrored, error) {
	freight, err := UnmarshalFreightAnnotations(annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal freight annotations: %w", err)
	}
	common, err := UnmarshalCommonAnnotations(eventID, annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal common annotations: %w", err)
	}
	verification, err := UnmarshalFreightVerificationAnnotations(annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal freight verification annotations: %w", err)
	}
	evt := FreightVerificationErrored{
		Common:              common,
		Freight:             freight,
		FreightVerification: verification,
	}
	return &evt, nil
}

// UnmarshalFreightVerificationUnknownAnnotations converts the given annotations into a
// FreightVerificationUnknown event. This is used by the main event handler to convert the data
// into a normal structured event, but is exposed for convenience.
func UnmarshalFreightVerificationUnknownAnnotations(
	eventID string,
	annotations map[string]string,
) (*FreightVerificationUnknown, error) {
	freight, err := UnmarshalFreightAnnotations(annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal freight annotations: %w", err)
	}
	common, err := UnmarshalCommonAnnotations(eventID, annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal common annotations: %w", err)
	}
	verification, err := UnmarshalFreightVerificationAnnotations(annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal freight verification annotations: %w", err)
	}
	evt := FreightVerificationUnknown{
		Common:              common,
		Freight:             freight,
		FreightVerification: verification,
	}
	return &evt, nil
}

// UnmarshalFreightVerificationAbortedAnnotations converts the given annotations into a
// FreightVerificationAborted event. This is used by the main event handler to convert the data
// into a normal structured event, but is exposed for convenience.
func UnmarshalFreightVerificationAbortedAnnotations(
	eventID string,
	annotations map[string]string,
) (*FreightVerificationAborted, error) {
	freight, err := UnmarshalFreightAnnotations(annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal freight annotations: %w", err)
	}
	common, err := UnmarshalCommonAnnotations(eventID, annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal common annotations: %w", err)
	}
	verification, err := UnmarshalFreightVerificationAnnotations(annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal freight verification annotations: %w", err)
	}
	evt := FreightVerificationAborted{
		Common:              common,
		Freight:             freight,
		FreightVerification: verification,
	}
	return &evt, nil
}

// UnmarshalFreightApprovedAnnotations converts the given annotations into a
// FreightApproved event. This is used by the main event handler to convert the data
// into a normal structured event, but is exposed for convenience.
func UnmarshalFreightApprovedAnnotations(
	eventID string,
	annotations map[string]string,
) (*FreightApproved, error) {
	freight, err := UnmarshalFreightAnnotations(annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal freight annotations: %w", err)
	}
	common, err := UnmarshalCommonAnnotations(eventID, annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal common annotations: %w", err)
	}
	evt := FreightApproved{
		Common:  common,
		Freight: freight,
	}
	return &evt, nil
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
	if len(freight.Artifacts) > 0 {
		evt.Artifacts = freight.Artifacts
	}
	return evt
}
