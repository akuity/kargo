package event

import (
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// Freight is a struct that contains common fields for freight-related events.
type Freight struct {
	Name          string                       `json:"name"`
	StageName     string                       `json:"stageName"`
	WarehouseName string                       `json:"warehouseName,omitempty"`
	CreateTime    time.Time                    `json:"createTime"`
	Alias         *string                      `json:"alias,omitempty"`
	Commits       []kargoapi.GitCommit         `json:"commits,omitempty"`
	Images        []kargoapi.Image             `json:"images,omitempty"`
	Charts        []kargoapi.Chart             `json:"charts,omitempty"`
	Artifacts     []kargoapi.ArtifactReference `json:"artifacts,omitempty"`
}

// FreightVerification is a struct that contains common fields for a verification event
type FreightVerification struct {
	StartTime       *time.Time `json:"verificationStartTime,omitempty"`
	FinishTime      *time.Time `json:"verificationFinishTime,omitempty"`
	AnalysisRunName *string    `json:"analysisRunName,omitempty"`
	// AnalysisTriggeredByPromotion is the name of the promotion that triggered the analysis run.
	AnalysisTriggeredByPromotion *string `json:"analysisTriggeredByPromotion,omitempty"`
}

// newFreightVerification creates a new `FreightVerification` struct from a `VerificationInfo`.
func newFreightVerification(vi *kargoapi.VerificationInfo) FreightVerification {
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

// NOTE(thomastaylor312): Most of the promotion events are identical, but that could easily change
// in the future if we want to decorate with more data. That is why all of these are separate types,
// even though they are identical in structure right now

// FreightVerificationSucceeded is an event fired when a freight verification succeeds.
type FreightVerificationSucceeded struct {
	Common
	Freight
	FreightVerification
}

// FreightVerificationFailed is an event fired when a freight verification fails.
type FreightVerificationFailed struct {
	Common
	Freight
	FreightVerification
}

// FreightVerificationErrored is an event fired when a freight verification errors.
type FreightVerificationErrored struct {
	Common
	Freight
	FreightVerification
}

// FreightVerificationAborted is an event fired when a freight verification is aborted.
type FreightVerificationAborted struct {
	Common
	Freight
	FreightVerification
}

// FreightVerificationInconclusive is an event fired when a freight verification is inconclusive.
type FreightVerificationInconclusive struct {
	Common
	Freight
	FreightVerification
}

// FreightVerificationUnknown is an event fired when a freight verification is unknown.
type FreightVerificationUnknown struct {
	Common
	Freight
	FreightVerification
}

// FreightCreated is an event fired when a new piece of Freight is created by
// a Warehouse or API server
type FreightCreated struct {
	Common
	Freight
}

type FreightApproved struct {
	Common
	Freight
}

// NewFreightCommon creates a new `Freight` and `Common` event from the given freight data. Since
// these fields are common to all events, this is exposed for convenience.
func NewFreightCommon(message,
	actor, stageName string, freight *kargoapi.Freight,
) (Common, Freight) {
	return newCommonFromFreight(message, actor, freight), newFreight(freight, stageName)
}

// freightVerificationSucceededMessage is the message of every
// FreightVerificationSucceeded event. Verification only records a message
// when something went wrong, so a successful verification has none of its own.
const freightVerificationSucceededMessage = "Freight verification succeeded"

// NewFreightVerification creates a Freight verification event for the given
// verification of the given Freight in the given Stage. The event type (and
// therefore the type of its data) follows from the verification's phase, e.g.
// a successful verification yields a FreightVerificationSucceeded event. The
// message is taken from the verification, except for successful
// verifications, which get a fixed message. triggeredByPromotion names the
// Promotion whose AnalysisRun performed the verification, if any.
func NewFreightVerification(
	actor string,
	stageName string,
	freight *kargoapi.Freight,
	verification *kargoapi.VerificationInfo,
	triggeredByPromotion *string,
) (cloudevents.Event, error) {
	message := verification.Message
	if verification.Phase == kargoapi.VerificationPhaseSuccessful {
		message = freightVerificationSucceededMessage
	}
	common := newCommonFromFreight(message, actor, freight)
	freightEvent := newFreight(freight, stageName)
	freightVerification := newFreightVerification(verification)
	freightVerification.AnalysisTriggeredByPromotion = triggeredByPromotion

	var eventType kargoapi.EventType
	var data any
	switch verification.Phase {
	case kargoapi.VerificationPhaseSuccessful:
		eventType = kargoapi.EventTypeFreightVerificationSucceeded
		data = &FreightVerificationSucceeded{
			Common:              common,
			Freight:             freightEvent,
			FreightVerification: freightVerification,
		}
	case kargoapi.VerificationPhaseFailed:
		eventType = kargoapi.EventTypeFreightVerificationFailed
		data = &FreightVerificationFailed{
			Common:              common,
			Freight:             freightEvent,
			FreightVerification: freightVerification,
		}
	case kargoapi.VerificationPhaseError:
		eventType = kargoapi.EventTypeFreightVerificationErrored
		data = &FreightVerificationErrored{
			Common:              common,
			Freight:             freightEvent,
			FreightVerification: freightVerification,
		}
	case kargoapi.VerificationPhaseAborted:
		eventType = kargoapi.EventTypeFreightVerificationAborted
		data = &FreightVerificationAborted{
			Common:              common,
			Freight:             freightEvent,
			FreightVerification: freightVerification,
		}
	case kargoapi.VerificationPhaseInconclusive:
		eventType = kargoapi.EventTypeFreightVerificationInconclusive
		data = &FreightVerificationInconclusive{
			Common:              common,
			Freight:             freightEvent,
			FreightVerification: freightVerification,
		}
	default:
		eventType = kargoapi.EventTypeFreightVerificationUnknown
		data = &FreightVerificationUnknown{
			Common:              common,
			Freight:             freightEvent,
			FreightVerification: freightVerification,
		}
	}
	return newUserEvent(
		string(eventType),
		kindFreight,
		data,
	)
}

// NewFreightCreated creates a new `FreightCreated` event.
func NewFreightCreated(
	message, actor string,
	freight *kargoapi.Freight,
) (cloudevents.Event, error) {
	common := newCommonFromFreight(message, actor, freight)
	freightEvent := newFreight(freight, "")
	return newUserEvent(
		string(kargoapi.EventTypeFreightCreated),
		kindFreight,
		&FreightCreated{
			Common:  common,
			Freight: freightEvent,
		},
	)
}

// NewFreightApproved creates a new `FreightApproved` event.
func NewFreightApproved(
	message, actor, stageName string,
	freight *kargoapi.Freight,
) (cloudevents.Event, error) {
	common, freightEvent := NewFreightCommon(message, actor, stageName, freight)
	return newUserEvent(
		string(kargoapi.EventTypeFreightApproved),
		kindFreight,
		&FreightApproved{
			Common:  common,
			Freight: freightEvent,
		},
	)
}

func newFreight(freight *kargoapi.Freight, stageName string) Freight {
	if freight == nil {
		return Freight{}
	}
	evt := Freight{
		CreateTime: freight.EffectiveDiscoveredAt(),
		Name:       freight.Name,
		StageName:  stageName,
	}
	if freight.Origin.Name != "" {
		evt.WarehouseName = freight.Origin.Name
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
