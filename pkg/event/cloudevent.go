package event

import (
	"errors"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
)

const (
	// sourcePrefix is prepended to the name of the Kargo component emitting an event to form the
	// event's source.
	sourcePrefix = "/kargo/"

	// KindExtension is the CloudEvent extension attribute holding the kind of
	// object (e.g. "Freight") an event is about, so consumers can look the object
	// up without decoding the event's data.
	KindExtension = "kind"

	// Kinds of objects that built-in events are about.
	kindFreight   = "Freight"
	kindPromotion = "Promotion"
	// API tokens are ServiceAccount token Secrets, so that is what an API token
	// event's object is looked up as.
	kindSecret = "Secret"
)

// ErrMissingEventType is returned when an event type is not provided to NewCloudEvent.
var ErrMissingEventType = errors.New("event type is required")

// NewCloudEvent is a convenience function that creates a new CloudEvent with the given type and
// data. All data is serialized as JSON.
func NewCloudEvent(eventType string, data any) (cloudevents.Event, error) {
	if eventType == "" {
		return cloudevents.Event{}, ErrMissingEventType
	}
	evt := cloudevents.New()
	evt.SetType(eventType)
	if err := evt.SetData(cloudevents.ApplicationJSON, data); err != nil {
		return cloudevents.Event{}, err
	}
	return evt, nil
}

// ComponentSource returns the CloudEvent source identifying the Kargo
// component with the given name (e.g. "api" or "stage-controller").
func ComponentSource(component string) string {
	return sourcePrefix + component
}

// KindOf returns the kind of object the given CloudEvent is about, or "" if
// the event doesn't record one.
func KindOf(evt cloudevents.Event) string {
	kind, _ := evt.Extensions()[KindExtension].(string)
	return kind
}

// newUserEvent is a convenience function for internal use with user-facing events as a convenience
// to set the kind extension attribute.
//
// In the event we group the current user events into their own package, this should go with it
func newUserEvent(eventType, kind string, data any) (cloudevents.Event, error) {
	evt, err := NewCloudEvent(eventType, data)
	if err != nil {
		return cloudevents.Event{}, err
	}
	if kind != "" {
		evt.SetExtension(KindExtension, kind)
	}
	return evt, nil
}
