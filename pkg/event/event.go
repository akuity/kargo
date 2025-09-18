package event

import (
	"time"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// A list of all known event types
var KnownEventTypes = []kargoapi.EventType{
	// NOTE(thomastaylor312): This list should be updated if new event types are added
	kargoapi.EventTypePromotionCreated,
	kargoapi.EventTypePromotionSucceeded,
	kargoapi.EventTypePromotionFailed,
	kargoapi.EventTypePromotionErrored,
	kargoapi.EventTypePromotionAborted,
	kargoapi.EventTypeFreightApproved,
	kargoapi.EventTypeFreightVerificationSucceeded,
	kargoapi.EventTypeFreightVerificationFailed,
	kargoapi.EventTypeFreightVerificationErrored,
	kargoapi.EventTypeFreightVerificationAborted,
	kargoapi.EventTypeFreightVerificationInconclusive,
	kargoapi.EventTypeFreightVerificationUnknown,
}

// Meta is an interface for our built in event types that all of them implement
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
	// GetID returns a unique identifier for the event
	GetID() string
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

// Common is a struct that contains fields common to all events.
type Common struct {
	Project string  `json:"project"`
	Actor   *string `json:"actor,omitempty"`
	Message string  `json:"message"`
	ID      string  `json:"id"`
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

func (c Common) GetID() string {
	return c.ID
}

// UnmarshalCommonAnnotations populates the Common fields from the given kubernetes annotations and
// event ID
func UnmarshalCommonAnnotations(eventID string, annotations map[string]string) (Common, error) {
	evt := Common{
		Project: annotations[kargoapi.AnnotationKeyEventProject],
		ID:      eventID,
	}
	if actor, ok := annotations[kargoapi.AnnotationKeyEventActor]; ok {
		evt.Actor = &actor
	}
	return evt, nil
}

// NOTE(thomastaylor312): These `MarshalAnnotationsTo` functions all take pointers to an annotations map
// they should fill in because they are all meant for use as nested fields. So the main event type
// will have a `MarshalAnnotations` function that calls these with the appropriate map to fill in

func (c *Common) MarshalAnnotationsTo(annotations map[string]string) {
	annotations[kargoapi.AnnotationKeyEventProject] = c.Project
	if c.Actor != nil {
		annotations[kargoapi.AnnotationKeyEventActor] = *c.Actor
	}
	// Message is skipped as it is passed to the k8s event directly
	// ID is skipped as it is not needed in annotations and is pulled from the event UUID
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

func parseTime(value string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}
